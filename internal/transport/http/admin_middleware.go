package http

import (
	"net/http"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminMiddleware checks if the authenticated user has admin role.
// Must be used after AuthMiddleware as it depends on the user context.
func AdminMiddleware(logger *observability.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user role from context (set by AuthMiddleware)
		roleValue, exists := c.Get("user_role")
		if !exists {
			logger.Warn("Admin middleware: user role not found in context")
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "Admin access required",
			})
			c.Abort()
			return
		}

		role, ok := roleValue.(string)
		if !ok {
			logger.Warn("Admin middleware: invalid role type in context")
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "Admin access required",
			})
			c.Abort()
			return
		}

		// Check if user has admin role
		if domain.Role(role) != domain.RoleAdmin {
			userID, _ := c.Get("user_id")
			logger.WithFields(map[string]interface{}{
				"user_id": userID,
				"role":    role,
			}).Warn("Non-admin user attempted to access admin endpoint")
			
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "forbidden",
				Message: "Admin access required",
			})
			c.Abort()
			return
		}

		// User is admin, continue
		c.Next()
	}
}

// GetUserIDFromContext extracts the user ID from the Gin context.
// Returns error if user ID is not found or invalid.
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, domain.ErrUnauthorized
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		return uuid.Nil, domain.ErrUnauthorized
	}

	return userID, nil
}

// GetUserRoleFromContext extracts the user role from the Gin context.
func GetUserRoleFromContext(c *gin.Context) (domain.Role, error) {
	roleValue, exists := c.Get("user_role")
	if !exists {
		return "", domain.ErrUnauthorized
	}

	role, ok := roleValue.(string)
	if !ok {
		return "", domain.ErrUnauthorized
	}

	return domain.Role(role), nil
}
