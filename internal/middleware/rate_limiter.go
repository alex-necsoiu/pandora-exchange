// Package middleware provides rate limiting functionality for HTTP endpoints.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiterConfig holds rate limiting configuration.
type RateLimiterConfig struct {
	// RequestsPerWindow is the maximum number of requests allowed per window
	RequestsPerWindow int
	
	// WindowDuration is the time window for rate limiting
	WindowDuration time.Duration
	
	// EnablePerUserLimits enables stricter limits for authenticated users
	EnablePerUserLimits bool
	
	// UserRequestsPerWindow is requests per window for authenticated users
	// If 0, uses RequestsPerWindow
	UserRequestsPerWindow int
}

// RateLimiter implements Redis-backed rate limiting with sliding window algorithm.
type RateLimiter struct {
	redis  *redis.Client
	config RateLimiterConfig
}

// NewRateLimiter creates a new rate limiter instance.
func NewRateLimiter(redisClient *redis.Client, config RateLimiterConfig) *RateLimiter {
	return &RateLimiter{
		redis:  redisClient,
		config: config,
	}
}

// Allow checks if a request should be allowed based on rate limits.
// Returns true if allowed, false if rate limit exceeded.
func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int) (bool, error) {
	now := time.Now().UnixNano()
	windowStart := now - rl.config.WindowDuration.Nanoseconds()

	pipe := rl.redis.Pipeline()

		// Remove old requests outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// Count requests in current window (before adding current request)
	countCmd := pipe.ZCard(ctx, key)

	// Execute pipeline to get current count
	if _, err := pipe.Exec(ctx); err != nil {
		return false, fmt.Errorf("rate limiter redis error: %w", err)
	}

	// Check if we're at or over the limit
	count := countCmd.Val()
	if count >= int64(limit) {
		return false, nil
	}

	// Add current request and set expiry
	pipe2 := rl.redis.Pipeline()
	pipe2.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: fmt.Sprintf("%d", now),
	})
	pipe2.Expire(ctx, key, rl.config.WindowDuration+time.Minute)

	if _, err := pipe2.Exec(ctx); err != nil {
		return false, fmt.Errorf("rate limiter redis error: %w", err)
	}

	return true, nil
}

// RateLimitMiddleware returns a Gin middleware that enforces rate limiting.
//
// Rate limiting strategy:
//   - Per-IP rate limiting for all requests
//   - Optional per-user rate limiting for authenticated requests
//   - Uses Redis sorted sets with sliding window algorithm
//   - Configurable request limits and time windows
//
// Headers added to response:
//   - X-RateLimit-Limit: Maximum requests per window
//   - X-RateLimit-Remaining: Remaining requests in current window
//   - X-RateLimit-Reset: Unix timestamp when limit resets
//
// Response when rate limit exceeded:
//   - Status: 429 Too Many Requests
//   - Retry-After header with seconds to wait
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		
		// Determine rate limit key (IP-based by default)
		ip := c.ClientIP()
		key := fmt.Sprintf("ratelimit:ip:%s", ip)
		limit := limiter.config.RequestsPerWindow

		// If user is authenticated, use per-user limit
		if userID, exists := c.Get("user_id"); exists && limiter.config.EnablePerUserLimits {
			key = fmt.Sprintf("ratelimit:user:%v", userID)
			if limiter.config.UserRequestsPerWindow > 0 {
				limit = limiter.config.UserRequestsPerWindow
			}
		}

		// Check rate limit
		allowed, err := limiter.Allow(ctx, key, limit)
		if err != nil {
			// On Redis error, log and allow request (fail open)
			c.Error(fmt.Errorf("rate limiter error: %w", err))
			c.Next()
			return
		}

		// Get remaining count
		now := time.Now().UnixNano()
		windowStart := now - limiter.config.WindowDuration.Nanoseconds()
		count, err := limiter.redis.ZCount(ctx, key, fmt.Sprintf("%d", windowStart), "+inf").Result()
		if err != nil {
			count = 0
		}

		remaining := int64(limit) - count
		if remaining < 0 {
			remaining = 0
		}

		// Calculate reset time
		resetTime := time.Now().Add(limiter.config.WindowDuration).Unix()

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))

		if !allowed {
			// Rate limit exceeded
			retryAfter := int(limiter.config.WindowDuration.Seconds())
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please try again later.",
				"retry_after_seconds": retryAfter,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// EndpointRateLimitMiddleware returns a middleware for endpoint-specific rate limiting.
// This allows different endpoints to have different rate limits.
//
// Example usage:
//
//	router.POST("/login", EndpointRateLimitMiddleware(limiter, 5, time.Minute), handler.Login)
//
// This limits login endpoint to 5 requests per minute.
func EndpointRateLimitMiddleware(limiter *RateLimiter, requestsPerWindow int, window time.Duration) gin.HandlerFunc {
	endpointLimiter := &RateLimiter{
		redis: limiter.redis,
		config: RateLimiterConfig{
			RequestsPerWindow:     requestsPerWindow,
			WindowDuration:        window,
			EnablePerUserLimits:   false,
		},
	}

	return RateLimitMiddleware(endpointLimiter)
}

// GetRateLimitInfo returns current rate limit information for a key.
// Useful for debugging and monitoring.
func (rl *RateLimiter) GetRateLimitInfo(ctx context.Context, key string) (used int64, remaining int64, resetAt time.Time, err error) {
	now := time.Now()
	windowStart := now.Add(-rl.config.WindowDuration).UnixNano()

	count, err := rl.redis.ZCount(ctx, key, fmt.Sprintf("%d", windowStart), "+inf").Result()
	if err != nil {
		return 0, 0, time.Time{}, err
	}

	used = count
	remaining = int64(rl.config.RequestsPerWindow) - count
	if remaining < 0 {
		remaining = 0
	}
	resetAt = now.Add(rl.config.WindowDuration)

	return used, remaining, resetAt, nil
}

// ResetRateLimit clears rate limit data for a specific key.
// Useful for admin operations or testing.
func (rl *RateLimiter) ResetRateLimit(ctx context.Context, key string) error {
	return rl.redis.Del(ctx, key).Err()
}
