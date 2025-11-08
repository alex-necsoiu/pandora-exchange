// Package http provides HTTP transport layer for the User Service using Gin framework.
package http

import (
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/google/uuid"
)

// RegisterRequest represents the request body for user registration.
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

// LoginRequest represents the request body for user login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshTokenRequest represents the request body for token refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// LogoutRequest represents the request body for logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// UpdateKYCRequest represents the request body for updating KYC status.
type UpdateKYCRequest struct {
	Status string `json:"status" binding:"required,oneof=none pending approved rejected"`
}

// UpdateProfileRequest represents the request body for updating user profile.
type UpdateProfileRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

// AuthResponse represents the response body for authentication operations.
type AuthResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	User         UserDTO    `json:"user"`
	ExpiresAt    time.Time  `json:"expires_at"`
}

// UserDTO represents a user in API responses.
type UserDTO struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	KYCStatus string    `json:"kyc_status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SessionDTO represents an active session in API responses.
type SessionDTO struct {
	Token     string    `json:"token"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionsResponse represents the response body for getting active sessions.
type SessionsResponse struct {
	Sessions []SessionDTO `json:"sessions"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// MessageResponse represents a simple message response.
type MessageResponse struct {
	Message string `json:"message"`
}

// toUserDTO converts a domain User to a UserDTO.
func toUserDTO(user *domain.User) UserDTO {
	return UserDTO{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		KYCStatus: user.KYCStatus.String(),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// toSessionDTO converts a domain RefreshToken to a SessionDTO.
func toSessionDTO(token *domain.RefreshToken) SessionDTO {
	return SessionDTO{
		Token:     token.Token,
		IPAddress: token.IPAddress,
		UserAgent: token.UserAgent,
		CreatedAt: token.CreatedAt,
		ExpiresAt: token.ExpiresAt,
	}
}

// Admin-specific DTOs

// AdminListUsersRequest represents query parameters for listing users (admin).
type AdminListUsersRequest struct {
	Limit  int `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset int `form:"offset" binding:"omitempty,min=0"`
}

// AdminSearchUsersRequest represents query parameters for searching users (admin).
type AdminSearchUsersRequest struct {
	Query  string `form:"query" binding:"required,min=1"`
	Limit  int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset int    `form:"offset" binding:"omitempty,min=0"`
}

// AdminUserDTO represents detailed user information for admin panel.
type AdminUserDTO struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      string    `json:"role"`
	KYCStatus string    `json:"kyc_status"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// AdminUsersListResponse represents the response for list users endpoint.
type AdminUsersListResponse struct {
	Users []AdminUserDTO `json:"users"`
	Total int64          `json:"total"`
	Limit int            `json:"limit"`
	Offset int           `json:"offset"`
}

// AdminSessionDTO represents a user session with user details (admin).
type AdminSessionDTO struct {
	Token     string    `json:"token"`
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AdminSessionsResponse represents the response for admin sessions endpoint.
type AdminSessionsResponse struct {
	Sessions []AdminSessionDTO `json:"sessions"`
	Total    int64             `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

// AdminForceLogoutRequest represents the request to force logout a user.
type AdminForceLogoutRequest struct {
	Token string `json:"token" binding:"required"`
}

// AdminUpdateRoleRequest represents the request to update a user's role.
type AdminUpdateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// AdminStatsResponse represents system statistics for admin dashboard.
type AdminStatsResponse struct {
	TotalUsers      int64 `json:"total_users"`
	ActiveSessions  int64 `json:"active_sessions"`
	VerifiedUsers   int64 `json:"verified_users,omitempty"`
	PendingKYC      int64 `json:"pending_kyc,omitempty"`
}

// toAdminUserDTO converts a domain User to an AdminUserDTO.
func toAdminUserDTO(user *domain.User) AdminUserDTO {
	return AdminUserDTO{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role.String(),
		KYCStatus: user.KYCStatus.String(),
		IsActive:  !user.IsDeleted(),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		DeletedAt: user.DeletedAt,
	}
}
