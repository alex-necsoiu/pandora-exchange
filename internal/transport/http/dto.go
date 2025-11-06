// Package http provides HTTP transport layer for the User Service using Gin framework.
package http

import (
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/google/uuid"
)

// RegisterRequest represents the request body for user registration.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"omitempty"`
}

// LoginRequest represents the request body for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest represents the request body for token refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
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
	FullName string `json:"full_name" binding:"required"`
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
	FullName  string    `json:"full_name,omitempty"`
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
		FullName:  user.FullName,
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
