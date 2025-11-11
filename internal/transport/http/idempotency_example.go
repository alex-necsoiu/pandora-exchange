package http

import (
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/middleware"
	"github.com/gin-gonic/gin"
)

// Example: How to integrate idempotency middleware into your router
//
// This file demonstrates various ways to apply the idempotency middleware
// to protect against duplicate operations.

// setupIdempotencyMiddleware creates and configures the idempotency middleware
// with recommended settings for production use.
func setupIdempotencyMiddleware() gin.HandlerFunc {
	// For production, use Redis-backed store
	// For development/testing, use in-memory store
	store := middleware.NewInMemoryStore()

	config := middleware.IdempotencyConfig{
		Store:       store,
		TTL:         24 * time.Hour, // Responses cached for 24 hours
		IncludeBody: true,           // Include body hash in cache key
	}

	return middleware.IdempotencyMiddleware(config)
}

// ExampleGlobalIdempotency shows how to apply idempotency globally
// and skip for non-mutating endpoints
func ExampleGlobalIdempotency(router *gin.Engine) {
	// Apply idempotency middleware globally
	router.Use(setupIdempotencyMiddleware())

	// Health checks and metrics don't need idempotency keys
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Mutating operations benefit from idempotency
	router.POST("/users", func(c *gin.Context) {
		// User creation logic
		c.JSON(201, gin.H{"id": "user-123"})
	})
}

// ExamplePerEndpointIdempotency shows how to apply idempotency
// to specific endpoints with custom TTLs
func ExamplePerEndpointIdempotency(router *gin.Engine) {
	// Short TTL for session operations (1 hour)
	sessionIdempotency := middleware.IdempotencyMiddleware(middleware.IdempotencyConfig{
		Store: middleware.NewInMemoryStore(),
		TTL:   1 * time.Hour,
	})

	// Long TTL for critical financial operations (7 days)
	paymentIdempotency := middleware.IdempotencyMiddleware(middleware.IdempotencyConfig{
		Store: middleware.NewInMemoryStore(),
		TTL:   7 * 24 * time.Hour,
	})

	// Apply to specific routes
	router.POST("/auth/login", sessionIdempotency, func(c *gin.Context) {
		c.JSON(200, gin.H{"token": "abc123"})
	})

	router.POST("/payments", paymentIdempotency, func(c *gin.Context) {
		c.JSON(201, gin.H{"payment_id": "pay-123"})
	})

	// Read operations don't need idempotency
	router.GET("/users/:id", func(c *gin.Context) {
		c.JSON(200, gin.H{"user": "data"})
	})
}

// ExampleCustomKeyGenerator shows how to use a custom key generator
// that includes user context in the cache key
func ExampleCustomKeyGenerator(router *gin.Engine) {
	customKeyGen := func(c *gin.Context, idempotencyKey string) string {
		// Include user ID from JWT claims in the cache key
		userID, _ := c.Get("user_id")
		return idempotencyKey + ":user:" + userID.(string)
	}

	idempotencyMiddleware := middleware.IdempotencyMiddleware(middleware.IdempotencyConfig{
		Store:        middleware.NewInMemoryStore(),
		TTL:          24 * time.Hour,
		KeyGenerator: customKeyGen,
	})

	router.POST("/users/me/orders", idempotencyMiddleware, func(c *gin.Context) {
		c.JSON(201, gin.H{"order_id": "order-123"})
	})
}

// ExampleGroupedRoutes shows how to apply idempotency to route groups
func ExampleGroupedRoutes(router *gin.Engine) {
	idempotency := setupIdempotencyMiddleware()

	// API v1 group
	v1 := router.Group("/api/v1")
	{
		// Public routes without idempotency
		auth := v1.Group("/auth")
		{
			auth.POST("/login", func(c *gin.Context) {
				c.JSON(200, gin.H{"token": "abc123"})
			})
		}

		// Protected routes with idempotency
		users := v1.Group("/users")
		users.Use(idempotency) // Apply to all routes in this group
		{
			users.POST("/", func(c *gin.Context) {
				c.JSON(201, gin.H{"id": "user-123"})
			})

			users.PUT("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{"updated": true})
			})
		}
	}
}

// Recommended configuration for the Pandora Exchange user service:
//
// 1. User Registration/Login: 24 hour TTL
//    - Prevents duplicate account creation
//    - Allows retry window for network issues
//
// 2. KYC Submissions: 7 day TTL
//    - Critical compliance operation
//    - Long retry window for document upload failures
//
// 3. Account Operations: 1 hour TTL
//    - Profile updates, password changes
//    - Short window for transactional consistency
//
// 4. Session Management: 1 hour TTL
//    - Login, logout operations
//    - Short window matches session lifecycle

// IntegrateWithUserRouter shows the recommended integration
// with the existing Pandora Exchange user router
func IntegrateWithUserRouter(router *gin.Engine) {
	// Create shared store (in production, use Redis)
	store := middleware.NewInMemoryStore()

	// Standard TTL for most operations
	standardIdempotency := middleware.IdempotencyMiddleware(middleware.IdempotencyConfig{
		Store: store,
		TTL:   24 * time.Hour,
	})

	// Short TTL for session operations
	sessionIdempotency := middleware.IdempotencyMiddleware(middleware.IdempotencyConfig{
		Store: store,
		TTL:   1 * time.Hour,
	})

	// Long TTL for KYC operations
	kycIdempotency := middleware.IdempotencyMiddleware(middleware.IdempotencyConfig{
		Store: store,
		TTL:   7 * 24 * time.Hour,
	})

	v1 := router.Group("/api/v1")
	{
		// Auth routes with session TTL
		auth := v1.Group("/auth")
		{
			auth.POST("/register", standardIdempotency, func(c *gin.Context) {
				// Registration handler
			})
			auth.POST("/login", sessionIdempotency, func(c *gin.Context) {
				// Login handler
			})
			auth.POST("/refresh", sessionIdempotency, func(c *gin.Context) {
				// Refresh token handler
			})
		}

		// User routes
		users := v1.Group("/users")
		{
			// Profile updates with standard TTL
			users.PUT("/me", standardIdempotency, func(c *gin.Context) {
				// Update profile handler
			})

			// KYC with long TTL
			users.PUT("/:id/kyc", kycIdempotency, func(c *gin.Context) {
				// KYC update handler
			})

			// Session management with short TTL
			users.POST("/me/logout", sessionIdempotency, func(c *gin.Context) {
				// Logout handler
			})
			users.POST("/me/logout-all", sessionIdempotency, func(c *gin.Context) {
				// Logout all sessions handler
			})
		}
	}
}

// Usage in client applications:
//
// POST /api/v1/auth/register
// Headers:
//   Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000
//   Content-Type: application/json
// Body:
//   {"email": "user@example.com", "password": "secure123"}
//
// First request: Creates user, returns 201 Created
// Retry request: Returns cached 201 response with X-Idempotency-Replay: true
//
// Benefits:
// - Network failures don't create duplicate users
// - Client can safely retry without side effects
// - Consistent response for debugging
