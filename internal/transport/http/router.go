package http

import (
	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/gin-gonic/gin"
)

// SetupRouter configures and returns a Gin router with all routes and middleware.
func SetupRouter(
	userService domain.UserService,
	jwtManager *auth.JWTManager,
	logger *observability.Logger,
	mode string, // "release" or "debug"
) *gin.Engine {
	// Set Gin mode
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(RecoveryMiddleware(logger))
	router.Use(LoggingMiddleware(logger))
	router.Use(CORSMiddleware())

	// Create handler
	handler := NewHandler(userService, logger)

	// Health check (no auth required)
	router.GET("/health", handler.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public auth routes (no authentication required)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handler.Register)
			auth.POST("/login", handler.Login)
			auth.POST("/refresh", handler.RefreshToken)
		}

		// Protected user routes (authentication required)
		users := v1.Group("/users")
		users.Use(AuthMiddleware(jwtManager, logger))
		{
			// Current user endpoints
			users.GET("/me", handler.GetProfile)
			users.PUT("/me", handler.UpdateProfile)
			users.DELETE("/me", handler.DeleteAccount)
			users.GET("/me/sessions", handler.GetActiveSessions)
			users.POST("/me/logout", handler.Logout)
			users.POST("/me/logout-all", handler.LogoutAll)

			// Admin endpoints (KYC management)
			// TODO: Add admin role middleware
			users.PUT("/:id/kyc", handler.UpdateKYC)
		}
	}

	return router
}
