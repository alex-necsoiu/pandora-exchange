package http

import (
	"net/http"
	"strings"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/gin-gonic/gin"
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
