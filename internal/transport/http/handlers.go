package http

import (
	"errors"
	"net/http"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for the user service.
type Handler struct {
	userService domain.UserService
	logger      *observability.Logger
}

// NewHandler creates a new HTTP handler.
func NewHandler(userService domain.UserService, logger *observability.Logger) *Handler {
	return &Handler{
		userService: userService,
		logger:      logger,
	}
}

// Register handles user registration requests.
//
//	@Summary		Register a new user
//	@Description	Create a new user account and return authentication tokens
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RegisterRequest	true	"User registration details"
//	@Success		201		{object}	AuthResponse	"User registered successfully"
//	@Failure		400		{object}	ErrorResponse	"Invalid request"
//	@Failure		409		{object}	ErrorResponse	"User already exists"
//	@Failure		500		{object}	ErrorResponse	"Internal server error"
//	@Router			/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Warn("Invalid registration request")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"email": req.Email,
	}).Info("Processing registration request")

	// Register the user
	user, err := h.userService.Register(
		c.Request.Context(),
		req.Email,
		req.Password,
		req.FirstName,
		req.LastName,
	)
	if err != nil {
		h.handleServiceError(c, err, "registration failed")
		return
	}

	// Login the newly registered user to get tokens
	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	tokenPair, err := h.userService.Login(
		c.Request.Context(),
		req.Email,
		req.Password,
		ipAddress,
		userAgent,
	)
	if err != nil {
		h.handleServiceError(c, err, "auto-login after registration failed")
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
	}).Info("User registered successfully")

	c.JSON(http.StatusCreated, AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         toUserDTO(tokenPair.User),
		ExpiresAt:    tokenPair.ExpiresAt,
	})
}

// Login handles user login requests.
//
//	@Summary		Login user
//	@Description	Authenticate user and return access tokens
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		LoginRequest	true	"User login credentials"
//	@Success		200		{object}	AuthResponse	"Login successful"
//	@Failure		400		{object}	ErrorResponse	"Invalid request"
//	@Failure		401		{object}	ErrorResponse	"Invalid credentials"
//	@Failure		500		{object}	ErrorResponse	"Internal server error"
//	@Router			/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid login request")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	h.logger.WithField("email", req.Email).Info("Processing login request")

	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	tokenPair, err := h.userService.Login(
		c.Request.Context(),
		req.Email,
		req.Password,
		ipAddress,
		userAgent,
	)
	if err != nil {
		h.handleServiceError(c, err, "login failed")
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id": tokenPair.User.ID,
		"email":   tokenPair.User.Email,
	}).Info("User logged in successfully")

	c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         toUserDTO(tokenPair.User),
		ExpiresAt:    tokenPair.ExpiresAt,
	})
}

// RefreshToken handles token refresh requests.
//
//	@Summary		Refresh access token
//	@Description	Exchange a refresh token for a new access token
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RefreshTokenRequest	true	"Refresh token"
//	@Success		200		{object}	AuthResponse		"Token refreshed successfully"
//	@Failure		400		{object}	ErrorResponse		"Invalid request"
//	@Failure		401		{object}	ErrorResponse		"Invalid or expired refresh token"
//	@Failure		500		{object}	ErrorResponse		"Internal server error"
//	@Router			/auth/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid refresh token request")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	h.logger.Debug("Processing token refresh request")

	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	tokenPair, err := h.userService.RefreshToken(
		c.Request.Context(),
		req.RefreshToken,
		ipAddress,
		userAgent,
	)
	if err != nil {
		h.handleServiceError(c, err, "token refresh failed")
		return
	}

	h.logger.WithField("user_id", tokenPair.User.ID).Info("Token refreshed successfully")

	c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         toUserDTO(tokenPair.User),
		ExpiresAt:    tokenPair.ExpiresAt,
	})
}

// Logout handles user logout requests.
//
//	@Summary		Logout user
//	@Description	Invalidate refresh token and logout user from current device
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		LogoutRequest	true	"Refresh token to invalidate"
//	@Success		200		{object}	MessageResponse	"Logout successful"
//	@Failure		400		{object}	ErrorResponse	"Invalid request"
//	@Failure		401		{object}	ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	ErrorResponse	"Internal server error"
//	@Router			/users/me/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid logout request")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	userID := getUserIDFromContext(c)
	h.logger.WithField("user_id", userID).Info("Processing logout request")

	if err := h.userService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		h.handleServiceError(c, err, "logout failed")
		return
	}

	h.logger.WithField("user_id", userID).Info("User logged out successfully")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "logged out successfully",
	})
}

// LogoutAll handles logout from all devices.
//
//	@Summary		Logout from all devices
//	@Description	Invalidate all refresh tokens for the user
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	MessageResponse	"Logout successful"
//	@Failure		401	{object}	ErrorResponse	"Unauthorized"
//	@Failure		500	{object}	ErrorResponse	"Internal server error"
//	@Router			/users/me/logout-all [post]
func (h *Handler) LogoutAll(c *gin.Context) {
	userID := getUserIDFromContext(c)
	h.logger.WithField("user_id", userID).Info("Processing logout all request")

	if err := h.userService.LogoutAll(c.Request.Context(), userID); err != nil {
		h.handleServiceError(c, err, "logout all failed")
		return
	}

	h.logger.WithField("user_id", userID).Info("User logged out from all devices")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "logged out from all devices successfully",
	})
}

// GetProfile handles get user profile requests.
//
//	@Summary		Get user profile
//	@Description	Get current user's profile information
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	UserDTO			"User profile"
//	@Failure		401	{object}	ErrorResponse	"Unauthorized"
//	@Failure		404	{object}	ErrorResponse	"User not found"
//	@Failure		500	{object}	ErrorResponse	"Internal server error"
//	@Router			/users/me [get]
func (h *Handler) GetProfile(c *gin.Context) {
	userID := getUserIDFromContext(c)
	h.logger.WithField("user_id", userID).Debug("Getting user profile")

	user, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		h.handleServiceError(c, err, "failed to get user profile")
		return
	}

	c.JSON(http.StatusOK, toUserDTO(user))
}

// UpdateKYC handles KYC status update requests (admin only).
// PUT /api/v1/users/:id/kyc
func (h *Handler) UpdateKYC(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid user ID")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_user_id",
			Message: "invalid user ID format",
		})
		return
	}

	var req UpdateKYCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid KYC update request")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id": userID,
		"status":  req.Status,
	}).Info("Processing KYC status update")

	kycStatus := domain.KYCStatus(req.Status)
	user, err := h.userService.UpdateKYC(c.Request.Context(), userID, kycStatus)
	if err != nil {
		h.handleServiceError(c, err, "KYC update failed")
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"user_id": userID,
		"status":  req.Status,
	}).Info("KYC status updated successfully")

	c.JSON(http.StatusOK, toUserDTO(user))
}

// UpdateProfile handles user profile update requests.
//
//	@Summary		Update user profile
//	@Description	Update current user's profile information
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		UpdateProfileRequest	true	"Profile update data"
//	@Success		200		{object}	UserDTO					"Updated user profile"
//	@Failure		400		{object}	ErrorResponse			"Invalid request"
//	@Failure		401		{object}	ErrorResponse			"Unauthorized"
//	@Failure		500		{object}	ErrorResponse			"Internal server error"
//	@Router			/users/me [put]
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID := getUserIDFromContext(c)

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithField("error", err.Error()).Warn("Invalid profile update request")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	h.logger.WithField("user_id", userID).Info("Processing profile update")

	user, err := h.userService.UpdateProfile(c.Request.Context(), userID, req.FirstName, req.LastName)
	if err != nil {
		h.handleServiceError(c, err, "profile update failed")
		return
	}

	h.logger.WithField("user_id", userID).Info("Profile updated successfully")

	c.JSON(http.StatusOK, toUserDTO(user))
}

// DeleteAccount handles account deletion requests.
// DELETE /api/v1/users/me
func (h *Handler) DeleteAccount(c *gin.Context) {
	userID := getUserIDFromContext(c)
	h.logger.WithField("user_id", userID).Info("Processing account deletion")

	if err := h.userService.DeleteAccount(c.Request.Context(), userID); err != nil {
		h.handleServiceError(c, err, "account deletion failed")
		return
	}

	h.logger.WithField("user_id", userID).Info("Account deleted successfully")

	c.JSON(http.StatusOK, MessageResponse{
		Message: "account deleted successfully",
	})
}

// GetActiveSessions handles getting active sessions requests.
// GET /api/v1/users/me/sessions
func (h *Handler) GetActiveSessions(c *gin.Context) {
	userID := getUserIDFromContext(c)
	h.logger.WithField("user_id", userID).Debug("Getting active sessions")

	sessions, err := h.userService.GetActiveSessions(c.Request.Context(), userID)
	if err != nil {
		h.handleServiceError(c, err, "failed to get active sessions")
		return
	}

	sessionDTOs := make([]SessionDTO, len(sessions))
	for i, session := range sessions {
		sessionDTOs[i] = toSessionDTO(session)
	}

	c.JSON(http.StatusOK, SessionsResponse{
		Sessions: sessionDTOs,
	})
}

// HealthCheck handles health check requests.
// GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// handleServiceError converts domain errors to appropriate HTTP responses.
func (h *Handler) handleServiceError(c *gin.Context, err error, context string) {
	h.logger.WithFields(map[string]interface{}{
		"error":   err.Error(),
		"context": context,
	}).Error("Service error")

	var statusCode int
	var errorCode string
	var message string

	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		statusCode = http.StatusNotFound
		errorCode = "user_not_found"
		message = "user not found"
	case errors.Is(err, domain.ErrUserAlreadyExists):
		statusCode = http.StatusConflict
		errorCode = "user_already_exists"
		message = "user with this email already exists"
	case errors.Is(err, domain.ErrInvalidCredentials):
		statusCode = http.StatusUnauthorized
		errorCode = "invalid_credentials"
		message = "invalid email or password"
	case errors.Is(err, domain.ErrRefreshTokenNotFound),
		errors.Is(err, domain.ErrRefreshTokenExpired),
		errors.Is(err, domain.ErrRefreshTokenRevoked):
		statusCode = http.StatusUnauthorized
		errorCode = "invalid_refresh_token"
		message = "invalid or expired refresh token"
	case errors.Is(err, domain.ErrInvalidKYCStatus):
		statusCode = http.StatusBadRequest
		errorCode = "invalid_kyc_status"
		message = "invalid KYC status"
	case errors.Is(err, domain.ErrInvalidEmail):
		statusCode = http.StatusBadRequest
		errorCode = "invalid_email"
		message = "invalid email format"
	case errors.Is(err, domain.ErrWeakPassword):
		statusCode = http.StatusBadRequest
		errorCode = "weak_password"
		message = "password does not meet security requirements"
	default:
		statusCode = http.StatusInternalServerError
		errorCode = "internal_error"
		message = "an internal error occurred"
	}

	c.JSON(statusCode, ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}

// getUserIDFromContext extracts the user ID from the Gin context.
// This is set by the authentication middleware.
func getUserIDFromContext(c *gin.Context) uuid.UUID {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}
	return userID.(uuid.UUID)
}
