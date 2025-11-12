package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRateLimiter(t *testing.T) (*RateLimiter, *miniredis.Miniredis, func()) {
	t.Helper()

	// Create mini redis server
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Create rate limiter
	config := RateLimiterConfig{
		RequestsPerWindow:     10,
		WindowDuration:        time.Minute,
		EnablePerUserLimits:   false,
		UserRequestsPerWindow: 0,
	}
	limiter := NewRateLimiter(redisClient, config)

	cleanup := func() {
		redisClient.Close()
		mr.Close()
	}

	return limiter, mr, cleanup
}

func TestRateLimiter_Allow(t *testing.T) {
	limiter, _, cleanup := setupTestRateLimiter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:key"

	// First request should be allowed
	allowed, err := limiter.Allow(ctx, key, limiter.config.RequestsPerWindow)
	assert.NoError(t, err)
	assert.True(t, allowed)

	// Requests within limit should be allowed
	for i := 0; i < 8; i++ {
		allowed, err = limiter.Allow(ctx, key, limiter.config.RequestsPerWindow)
		assert.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+2)
	}

	// 10th request should be allowed (reaches limit)
	allowed, err = limiter.Allow(ctx, key, limiter.config.RequestsPerWindow)
	assert.NoError(t, err)
	assert.True(t, allowed)

	// 11th request should be blocked (exceeds limit)
	allowed, err = limiter.Allow(ctx, key, limiter.config.RequestsPerWindow)
	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestRateLimiter_SlidingWindow(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	// Create limiter with 1-second window
	config := RateLimiterConfig{
		RequestsPerWindow: 5,
		WindowDuration:    time.Second,
	}
	limiter := NewRateLimiter(redisClient, config)

	ctx := context.Background()
	key := "test:sliding"

	// Make 5 requests (reach limit)
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, key, limiter.config.RequestsPerWindow)
		assert.NoError(t, err)
		assert.True(t, allowed)
	}

	// 6th request should be blocked
	allowed, err := limiter.Allow(ctx, key, limiter.config.RequestsPerWindow)
	assert.NoError(t, err)
	assert.False(t, allowed)

	// Wait for window to expire (real time, not fast forward)
	time.Sleep(1100 * time.Millisecond)

	// After window expires, requests should be allowed again
	allowed, err = limiter.Allow(ctx, key, limiter.config.RequestsPerWindow)
	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestRateLimitMiddleware_IPBased(t *testing.T) {
	limiter, _, cleanup := setupTestRateLimiter(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make requests up to the limit
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "request %d should succeed", i+1)
		assert.Contains(t, w.Header().Get("X-RateLimit-Limit"), "10")
	}

	// 11th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestRateLimitMiddleware_PerUser(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	config := RateLimiterConfig{
		RequestsPerWindow:     100, // High limit for IPs
		WindowDuration:        time.Minute,
		EnablePerUserLimits:   true,
		UserRequestsPerWindow: 5, // Strict limit for users
	}
	limiter := NewRateLimiter(redisClient, config)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Simulate authenticated user
		c.Set("user_id", "user-123")
		c.Next()
	})
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// User should be limited to 5 requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "request %d should succeed", i+1)
	}

	// 6th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestEndpointRateLimitMiddleware(t *testing.T) {
	limiter, _, cleanup := setupTestRateLimiter(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Different endpoints with different limits
	router.POST("/login", EndpointRateLimitMiddleware(limiter, 3, time.Minute), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "login success"})
	})
	router.GET("/data", EndpointRateLimitMiddleware(limiter, 10, time.Minute), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "data"})
	})

	// Login endpoint: 3 requests allowed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 4th login should be blocked
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Data endpoint should still work (separate limit)
	req = httptest.NewRequest(http.MethodGet, "/data", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_GetRateLimitInfo(t *testing.T) {
	limiter, _, cleanup := setupTestRateLimiter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:info"

	// Make 3 requests
	for i := 0; i < 3; i++ {
		_, err := limiter.Allow(ctx, key, limiter.config.RequestsPerWindow)
		require.NoError(t, err)
	}

	used, remaining, resetAt, err := limiter.GetRateLimitInfo(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), used)
	assert.Equal(t, int64(7), remaining)
	assert.True(t, resetAt.After(time.Now()))
}

func TestRateLimiter_ResetRateLimit(t *testing.T) {
	limiter, _, cleanup := setupTestRateLimiter(t)
	defer cleanup()

	ctx := context.Background()
	key := "test:reset"

	// Make 10 requests (reach limit)
	for i := 0; i < 10; i++ {
		_, err := limiter.Allow(ctx, key, limiter.config.RequestsPerWindow)
		require.NoError(t, err)
	}

	// Should be blocked
	allowed, err := limiter.Allow(ctx, key, limiter.config.RequestsPerWindow)
	assert.NoError(t, err)
	assert.False(t, allowed)

	// Reset rate limit
	err = limiter.ResetRateLimit(ctx, key)
	assert.NoError(t, err)

	// Should be allowed again
	allowed, err = limiter.Allow(ctx, key, limiter.config.RequestsPerWindow)
	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestRateLimitMiddleware_Headers(t *testing.T) {
	limiter, _, cleanup := setupTestRateLimiter(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "10", w.Header().Get("X-RateLimit-Limit"))
	assert.Contains(t, w.Header().Get("X-RateLimit-Remaining"), "9")
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
}

func TestRateLimitMiddleware_DifferentIPs(t *testing.T) {
	limiter, _, cleanup := setupTestRateLimiter(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make 10 requests from IP1
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// IP1 should be blocked
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// IP2 should still be allowed (separate rate limit)
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func BenchmarkRateLimiter_Allow(b *testing.B) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	config := RateLimiterConfig{
		RequestsPerWindow: 1000,
		WindowDuration:    time.Minute,
	}
	limiter := NewRateLimiter(redisClient, config)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench:key:%d", i%100)
		_, _ = limiter.Allow(ctx, key, limiter.config.RequestsPerWindow)
	}
}
