package http

import (
	"net/http"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/gin-gonic/gin"
)

// AdminAuthHandler handles admin authentication HTTP requests.
// This handler is mounted on the admin server (separate port) to enforce
// separation between user and admin authentication endpoints.
type AdminAuthHandler struct {
	userService domain.UserService
	logger      *observability.Logger
}

// NewAdminAuthHandler creates a new AdminAuthHandler instance.
func NewAdminAuthHandler(userService domain.UserService, logger *observability.Logger) *AdminAuthHandler {
	return &AdminAuthHandler{
		userService: userService,
		logger:      logger,
	}
}

// AdminLogin authenticates an admin user and returns JWT tokens.
// Only users with admin role can successfully authenticate.
//
// POST /admin/auth/login
//
// Request body:
//
//	{
//	  "email": "admin@example.com",
//	  "password": "SecurePassword123"
//	}
//
// Response 200:
//
//	{
//	  "user": { ... },
//	  "access_token": "eyJ...",
//	  "refresh_token": "abc...",
//	  "expires_at": "2024-01-01T00:00:00Z"
//	}
//
// Errors:
//   - 400: Invalid request body
//   - 401: Invalid credentials or not an admin
//   - 500: Internal server error
func (h *AdminAuthHandler) AdminLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("invalid login request body")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request body",
		})
		return
	}

	// Validate request - this should not be needed with binding:"required" tags,
	// but kept for explicit validation
	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "email and password are required",
		})
		return
	}

	// Get client IP and user agent for audit logging
	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	// Call admin login service method (enforces admin role check)
	tokenPair, err := h.userService.AdminLogin(c.Request.Context(), req.Email, req.Password, ipAddress, userAgent)
	if err != nil {
		if err == domain.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "invalid credentials",
			})
			return
		}
		// Check for admin access error
		if err.Error() == "admin access required" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "admin access required",
			})
			return
		}
		// Check for deleted account
		if err.Error() == "account is deleted" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "account is deleted",
			})
			return
		}

		h.logger.WithError(err).Error("admin login failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		User:         toUserDTO(tokenPair.User),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	})
}

// AdminRefreshToken refreshes an admin's access token.
// Validates the refresh token and issues a new token pair.
//
// POST /admin/auth/refresh
//
// Request body:
//
//	{
//	  "refresh_token": "abc..."
//	}
//
// Response 200:
//
//	{
//	  "user": { ... },
//	  "access_token": "eyJ...",
//	  "refresh_token": "xyz...",
//	  "expires_at": "2024-01-01T00:00:00Z"
//	}
//
// Errors:
//   - 400: Invalid request body
//   - 401: Invalid or expired refresh token
//   - 500: Internal server error
//
// Note: This uses the same RefreshToken service method as regular users.
// The admin role is preserved in the new access token from the user's stored role.
func (h *AdminAuthHandler) AdminRefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("invalid refresh token request body")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request body",
		})
		return
	}

	// Validate request - this should not be needed with binding:"required" tag,
	// but kept for explicit validation
	if req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "refresh_token is required",
		})
		return
	}

	// Get client IP and user agent for audit logging
	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	// Call refresh token service method (same as regular users)
	// The user's admin role is preserved in the database and included in new tokens
	tokenPair, err := h.userService.RefreshToken(c.Request.Context(), req.RefreshToken, ipAddress, userAgent)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "invalid or expired refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		User:         toUserDTO(tokenPair.User),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	})
}
