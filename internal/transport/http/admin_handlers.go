package http

import (
	"net/http"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminHandler handles admin-specific HTTP requests.
type AdminHandler struct {
	userService domain.UserService
	logger      *observability.Logger
}

// NewAdminHandler creates a new AdminHandler instance.
func NewAdminHandler(userService domain.UserService, logger *observability.Logger) *AdminHandler {
	return &AdminHandler{
		userService: userService,
		logger:      logger,
	}
}

// ListUsers handles GET /api/v1/admin/users
// Lists all users with pagination.
func (h *AdminHandler) ListUsers(c *gin.Context) {
	var req AdminListUsersRequest
	req.Limit = 20  // default
	req.Offset = 0  // default

	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid list users request")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"limit":  req.Limit,
		"offset": req.Offset,
	}).Info("Admin: Processing list users request")

	users, total, err := h.userService.ListUsers(c.Request.Context(), req.Limit, req.Offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list users")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve users",
		})
		return
	}

	adminUsers := make([]AdminUserDTO, len(users))
	for i, user := range users {
		adminUsers[i] = toAdminUserDTO(user)
	}

	c.JSON(http.StatusOK, AdminUsersListResponse{
		Users:  adminUsers,
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
}

// SearchUsers handles GET /api/v1/admin/users/search
// Searches users by email, first name, or last name.
func (h *AdminHandler) SearchUsers(c *gin.Context) {
	var req AdminSearchUsersRequest
	req.Limit = 20  // default
	req.Offset = 0  // default

	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid search users request")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"query":  req.Query,
		"limit":  req.Limit,
		"offset": req.Offset,
	}).Info("Admin: Processing search users request")

	users, err := h.userService.SearchUsers(c.Request.Context(), req.Query, req.Limit, req.Offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to search users")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to search users",
		})
		return
	}

	adminUsers := make([]AdminUserDTO, len(users))
	for i, user := range users {
		adminUsers[i] = toAdminUserDTO(user)
	}

	c.JSON(http.StatusOK, AdminUsersListResponse{
		Users:  adminUsers,
		Total:  int64(len(adminUsers)),
		Limit:  req.Limit,
		Offset: req.Offset,
	})
}

// GetUser handles GET /api/v1/admin/users/:id
// Gets a specific user by ID (including deleted users).
func (h *AdminHandler) GetUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid user ID")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID format",
		})
		return
	}

	h.logger.WithField("user_id", userID).Info("Admin: Processing get user request")

	user, err := h.userService.GetUserByIDAdmin(c.Request.Context(), userID)
	if err != nil {
		if err == domain.ErrUserNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "user_not_found",
				Message: "User not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to get user")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve user",
		})
		return
	}

	c.JSON(http.StatusOK, toAdminUserDTO(user))
}

// UpdateUserRole handles PUT /api/v1/admin/users/:id/role
// Updates a user's role.
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid user ID")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID format",
		})
		return
	}

	var req AdminUpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid update role request")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id": userID,
		"role":    req.Role,
	}).Info("Admin: Processing update role request")

	user, err := h.userService.UpdateUserRole(c.Request.Context(), userID, domain.Role(req.Role))
	if err != nil {
		if err == domain.ErrUserNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "user_not_found",
				Message: "User not found",
			})
			return
		}
		if err == domain.ErrInvalidRole {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_role",
				Message: "Invalid role specified",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to update user role")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update user role",
		})
		return
	}

	c.JSON(http.StatusOK, toAdminUserDTO(user))
}

// GetAllSessions handles GET /api/v1/admin/sessions
// Gets all active sessions across all users.
func (h *AdminHandler) GetAllSessions(c *gin.Context) {
	var req AdminListUsersRequest
	req.Limit = 50  // default
	req.Offset = 0  // default

	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid get all sessions request")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"limit":  req.Limit,
		"offset": req.Offset,
	}).Info("Admin: Processing get all sessions request")

	sessions, total, err := h.userService.GetAllActiveSessions(c.Request.Context(), req.Limit, req.Offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get all sessions")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve sessions",
		})
		return
	}

	// Note: sessions from GetAllActiveSessions don't have user details
	// You'd need to modify the repository query to join with users table
	// For now, returning basic session info
	adminSessions := make([]AdminSessionDTO, len(sessions))
	for i, session := range sessions {
		adminSessions[i] = AdminSessionDTO{
			Token:     session.Token,
			UserID:    session.UserID,
			IPAddress: session.IPAddress,
			UserAgent: session.UserAgent,
			CreatedAt: session.CreatedAt,
			ExpiresAt: session.ExpiresAt,
		}
	}

	c.JSON(http.StatusOK, AdminSessionsResponse{
		Sessions: adminSessions,
		Total:    total,
		Limit:    req.Limit,
		Offset:   req.Offset,
	})
}

// ForceLogout handles POST /api/v1/admin/sessions/revoke
// Force logout a specific session by token.
func (h *AdminHandler) ForceLogout(c *gin.Context) {
	var req AdminForceLogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid force logout request")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Admin: Processing force logout request")

	err := h.userService.ForceLogout(c.Request.Context(), req.Token)
	if err != nil {
		if err == domain.ErrTokenNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "token_not_found",
				Message: "Token not found or already revoked",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to force logout")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to revoke session",
		})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Session revoked successfully",
	})
}

// GetSystemStats handles GET /api/v1/admin/stats
// Gets system statistics for admin dashboard.
func (h *AdminHandler) GetSystemStats(c *gin.Context) {
	h.logger.Info("Admin: Processing get system stats request")

	stats, err := h.userService.GetSystemStats(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get system stats")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve system statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}
