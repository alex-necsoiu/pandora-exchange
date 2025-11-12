package http

import (
	"regexp"

	"github.com/alex-necsoiu/pandora-exchange/internal/config"
	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	grpcTransport "github.com/alex-necsoiu/pandora-exchange/internal/transport/grpc"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/alex-necsoiu/pandora-exchange/docs" // Import generated docs
)

// ServiceRegistry is the interface for gRPC service registry operations
// It's defined here to avoid circular imports between http and grpc packages
type ServiceRegistry interface {
	ListServices() []*grpcTransport.ServiceInfo
}

// ValidateParamMiddleware returns a middleware that validates a named param against provided regex.
func ValidateParamMiddleware(param string, re *regexp.Regexp) gin.HandlerFunc {
	return func(c *gin.Context) {
		val := c.Param(param)
		if val == "" {
			// no param present — continue
			c.Next()
			return
		}
		if !re.MatchString(val) {
			c.AbortWithStatusJSON(400, gin.H{"error": "invalid parameter"})
			return
		}
		c.Next()
	}
}

// SetupUserRouter configures and returns a Gin router for user-facing endpoints only.
func SetupUserRouter(
	userService domain.UserService,
	jwtManager *auth.JWTManager,
	auditRepo domain.AuditRepository,
	cfg *config.Config,
	logger *observability.Logger,
	mode string, // "release" or "debug"
	tracingEnabled bool,
) *gin.Engine {
	// Set Gin mode
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware (order matters: Recovery first, then tracing, then audit, then error, then logging, then CORS)
	router.Use(RecoveryMiddleware(logger))
	
	// Add tracing middleware if enabled
	if tracingEnabled {
		router.Use(TracingMiddleware("user-service"))
	}
	
	// Audit middleware - logs all requests to audit_logs table
	router.Use(AuditMiddleware(auditRepo, cfg, logger))
	
	// Error middleware - converts domain errors to HTTP responses
	router.Use(ErrorMiddleware())
	
	router.Use(LoggingMiddleware(logger))
	router.Use(CORSMiddleware())

	// Create handler
	handler := NewHandler(userService, logger)

	// Health check (no auth required)
	router.GET("/health", handler.HealthCheck)

	// Swagger documentation (no auth required)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes (user-facing)
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

			// KYC update (only numeric/uuid id allowed) - validate id param
			uuidRe := regexp.MustCompile(`^[a-f0-9-]{36}$`)
			users.PUT("/:id/kyc", ValidateParamMiddleware("id", uuidRe), AdminMiddleware(logger), handler.UpdateKYC)
		}
	}

	return router
}

// SetupAdminRouter configures and returns a Gin router for admin-only endpoints.
// This router is intended to be started as a separate HTTP server (different port) so
// admin routes never share the same server instance or path space with user routes.
func SetupAdminRouter(
	userService domain.UserService,
	jwtManager *auth.JWTManager,
	auditRepo domain.AuditRepository,
	cfg *config.Config,
	logger *observability.Logger,
	mode string,
	tracingEnabled bool,
	registry ServiceRegistry,
) *gin.Engine {
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	
	// Global middleware (order matters: Recovery first, then tracing, then audit, then error, then logging, then CORS)
	router.Use(RecoveryMiddleware(logger))
	
	// Add tracing middleware if enabled
	if tracingEnabled {
		router.Use(TracingMiddleware("admin-service"))
	}
	
	// Audit middleware - logs all requests to audit_logs table (CRITICAL for admin actions)
	router.Use(AuditMiddleware(auditRepo, cfg, logger))
	
	// Error middleware - converts domain errors to HTTP responses
	router.Use(ErrorMiddleware())
	
	router.Use(LoggingMiddleware(logger))
	router.Use(CORSMiddleware())

	adminHandler := NewAdminHandler(userService, logger)
	adminAuthHandler := NewAdminAuthHandler(userService, logger)

	// Health check for service registry (no auth required)
	router.GET("/health/services", func(c *gin.Context) {
		services := registry.ListServices()
		c.JSON(200, gin.H{
			"services": services,
			"count":    len(services),
		})
	})

	// Admin auth routes (NO authentication required - this is the login endpoint)
	auth := router.Group("/admin/auth")
	{
		auth.POST("/login", adminAuthHandler.AdminLogin)
		auth.POST("/refresh", adminAuthHandler.AdminRefreshToken)
	}

	// Admin routes are mounted under /admin to keep separation of concerns.
	// All routes require authentication + admin role.
	admin := router.Group("/admin")
	admin.Use(AuthMiddleware(jwtManager, logger))
	admin.Use(AdminMiddleware(logger))
	{
		// Validate UUID params using a conservative regex
		uuidRe := regexp.MustCompile(`^[a-f0-9-]{36}$`)

		admin.GET("/users", adminHandler.ListUsers)
		admin.GET("/users/search", adminHandler.SearchUsers)
		admin.GET("/users/:id", ValidateParamMiddleware("id", uuidRe), adminHandler.GetUser)
		admin.PUT("/users/:id/role", ValidateParamMiddleware("id", uuidRe), adminHandler.UpdateUserRole)

		admin.GET("/sessions", adminHandler.GetAllSessions)
		admin.POST("/sessions/revoke", adminHandler.ForceLogout)

		admin.GET("/stats", adminHandler.GetSystemStats)
	}

	return router
}
