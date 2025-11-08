package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/config"
	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// AuthMiddleware provides JWT authentication for protected routes.
func AuthMiddleware(jwtManager *auth.JWTManager, logger *observability.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn("Missing authorization header")
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "unauthorized",
				Message: "missing authorization header",
			})
			c.Abort()
			return
		}

		// Extract Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Warn("Invalid authorization header format")
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "unauthorized",
				Message: "invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate access token
		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Warn("Invalid access token")
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "unauthorized",
				Message: "invalid or expired access token",
			})
			c.Abort()
			return
		}

		// Set user ID and email in context for handlers
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("user_role", claims.Role) // Set role for authorization

		logger.WithFields(map[string]interface{}{
			"user_id": claims.UserID,
			"email":   claims.Email,
		}).Debug("User authenticated")

		c.Next()
	}
}

// LoggingMiddleware logs HTTP requests and responses.
func LoggingMiddleware(logger *observability.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log request
		logger.WithFields(map[string]interface{}{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"ip_address": c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}).Info("HTTP request received")

		c.Next()

		// Log response
		statusCode := c.Writer.Status()
		logger.WithFields(map[string]interface{}{
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"status_code": statusCode,
		}).Info("HTTP request completed")
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RecoveryMiddleware recovers from panics and logs them.
func RecoveryMiddleware(logger *observability.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.WithFields(map[string]interface{}{
					"error":  err,
					"method": c.Request.Method,
					"path":   c.Request.URL.Path,
				}).Error("Panic recovered")

				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Error:   "internal_error",
					Message: "an unexpected error occurred",
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}

// TracingMiddleware returns OpenTelemetry tracing middleware for Gin
// It creates a span for each HTTP request with relevant attributes
func TracingMiddleware(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName)
}

// AuditMiddleware logs HTTP requests to the audit_logs table
// Should be placed after AuthMiddleware to capture user information
func AuditMiddleware(auditRepo domain.AuditRepository, cfg *config.Config, logger *observability.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Capture start time
		startTime := time.Now()

		// Process request
		c.Next()

		// Skip audit logging for health checks and non-critical endpoints
		if shouldSkipAudit(c.Request.URL.Path) {
			return
		}

		// Create audit log asynchronously to avoid blocking the response
		go func() {
			if err := createAuditLog(c, auditRepo, cfg, startTime, logger); err != nil {
				logger.WithError(err).Error("Failed to create audit log")
			}
		}()
	}
}

// shouldSkipAudit determines if a path should be excluded from audit logging
func shouldSkipAudit(path string) bool {
	skipPaths := []string{
		"/health",
		"/metrics",
		"/ready",
		"/live",
	}

	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// createAuditLog builds and stores the audit log entry
func createAuditLog(c *gin.Context, auditRepo domain.AuditRepository, cfg *config.Config, startTime time.Time, logger *observability.Logger) error {
	// Extract user information from context (set by AuthMiddleware)
	var userID *uuid.UUID
	var actorType domain.AuditActorType
	var actorIdentifier *string

	if userIDVal, exists := c.Get("user_id"); exists {
		if uid, ok := userIDVal.(uuid.UUID); ok {
			userID = &uid
			actorType = domain.AuditActorUser

			// Check if admin
			if roleVal, roleExists := c.Get("user_role"); roleExists {
				if role, ok := roleVal.(string); ok && role == "admin" {
					actorType = domain.AuditActorAdmin
				}
			}

			// Set actor identifier to user email if available
			if emailVal, emailExists := c.Get("email"); emailExists {
				if email, ok := emailVal.(string); ok {
					actorIdentifier = &email
				}
			}
		}
	} else {
		// No authenticated user - anonymous or system
		actorType = domain.AuditActorAPI
		anonymous := "anonymous"
		actorIdentifier = &anonymous
	}

	// Determine event type and category based on endpoint
	eventType, eventCategory := categorizeRequest(c)

	// Determine severity
	severity := determineSeverity(c)

	// Determine status
	status := determineStatus(c)

	// Build metadata
	metadata := map[string]interface{}{
		"method":      c.Request.Method,
		"path":        c.Request.URL.Path,
		"status_code": c.Writer.Status(),
		"duration_ms": time.Since(startTime).Milliseconds(),
	}

	// Add query parameters if present
	if len(c.Request.URL.RawQuery) > 0 {
		metadata["query"] = c.Request.URL.RawQuery
	}

	// Add request ID if present
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = c.GetString("request_id")
	}

	// Get IP address
	ipAddress := c.ClientIP()
	
	// Get user agent
	userAgent := c.Request.UserAgent()

	// Calculate retention date based on environment
	retentionDays := cfg.Audit.RetentionDays
	if retentionDays == 0 {
		retentionDays = 90 // Default fallback
	}
	retentionUntil := time.Now().Add(time.Duration(retentionDays) * 24 * time.Hour)

	// Create audit log
	auditLog := &domain.AuditLog{
		EventType:      eventType,
		EventCategory:  eventCategory,
		Severity:       severity,
		ActorType:      actorType,
		UserID:         userID,
		ActorIdentifier: actorIdentifier,
		Action:         buildAction(c),
		Status:         status,
		IPAddress:      &ipAddress,
		UserAgent:      &userAgent,
		RequestID:      stringPtr(requestID),
		Metadata:       metadata,
		RetentionUntil: &retentionUntil,
		IsSensitive:    isSensitiveEndpoint(c.Request.URL.Path),
	}

	// Store audit log
	_, err := auditRepo.Create(c.Request.Context(), auditLog)
	return err
}

// categorizeRequest determines the event type and category based on the request
func categorizeRequest(c *gin.Context) (string, domain.AuditEventCategory) {
	path := c.Request.URL.Path
	method := c.Request.Method

	// Admin endpoints (check FIRST before other patterns)
	if strings.HasPrefix(path, "/admin") {
		return "admin." + strings.ToLower(method), domain.AuditCategorySecurity
	}

	// Authentication endpoints
	if strings.Contains(path, "/auth/register") {
		return "user.register", domain.AuditCategoryAuthentication
	}
	if strings.Contains(path, "/auth/login") {
		return "user.login", domain.AuditCategoryAuthentication
	}
	if strings.Contains(path, "/auth/refresh") {
		return "token.refresh", domain.AuditCategoryAuthentication
	}
	if strings.Contains(path, "/auth/logout") {
		return "user.logout", domain.AuditCategoryAuthentication
	}

	// KYC endpoints
	if strings.Contains(path, "/kyc") {
		if method == "PUT" {
			return "user.kyc_update", domain.AuditCategoryCompliance
		}
		return "user.kyc_view", domain.AuditCategoryDataAccess
	}

	// User management
	if strings.Contains(path, "/users") {
		switch method {
		case "GET":
			return "user.view", domain.AuditCategoryDataAccess
		case "PUT", "PATCH":
			return "user.update", domain.AuditCategoryDataModification
		case "DELETE":
			return "user.delete", domain.AuditCategoryDataModification
		case "POST":
			return "user.create", domain.AuditCategoryDataModification
		}
	}

	// Default
	return "http." + strings.ToLower(method), domain.AuditCategoryDataAccess
}

// determineSeverity assigns severity based on status code and endpoint
func determineSeverity(c *gin.Context) domain.AuditSeverity {
	statusCode := c.Writer.Status()
	path := c.Request.URL.Path

	// Critical: Admin actions or authentication failures
	if strings.HasPrefix(path, "/admin") {
		return domain.AuditSeverityCritical
	}
	if strings.Contains(path, "/auth") && statusCode >= 400 {
		return domain.AuditSeverityHigh
	}

	// High: KYC changes, user deletions
	if strings.Contains(path, "/kyc") && c.Request.Method != "GET" {
		return domain.AuditSeverityHigh
	}
	if c.Request.Method == "DELETE" {
		return domain.AuditSeverityHigh
	}

	// Warning: Client errors
	if statusCode >= 400 && statusCode < 500 {
		return domain.AuditSeverityWarning
	}

	// Info: Successful operations
	return domain.AuditSeverityInfo
}

// determineStatus maps HTTP status code to audit status
func determineStatus(c *gin.Context) domain.AuditStatus {
	statusCode := c.Writer.Status()

	if statusCode >= 200 && statusCode < 300 {
		return domain.AuditStatusSuccess
	}
	if statusCode >= 500 {
		return domain.AuditStatusError
	}
	return domain.AuditStatusFailure
}

// buildAction creates a human-readable action description
func buildAction(c *gin.Context) string {
	method := c.Request.Method
	path := c.Request.URL.Path

	// Simplify path for readability
	simplePath := path
	if strings.Contains(path, "/users/") {
		simplePath = "/users/:id"
	}

	return method + " " + simplePath
}

// isSensitiveEndpoint determines if the endpoint handles sensitive data
func isSensitiveEndpoint(path string) bool {
	sensitivePatterns := []string{
		"/auth/login",
		"/auth/register",
		"/kyc",
		"/admin",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// stringPtr returns a pointer to the string if non-empty, otherwise nil
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

