package dto

import (
	"fmt"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// RegisterRequest represents the request body for user registration.
type RegisterRequest struct {
	Email     string `json:"email" binding:"required"`
	Password  string `json:"password" binding:"required"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

// Validate validates the RegisterRequest fields.
func (r *RegisterRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)

	if r.Email == "" {
		return fmt.Errorf("email is required")
	}

	if !emailRegex.MatchString(r.Email) {
		return fmt.Errorf("invalid email format")
	}

	if r.Password == "" {
		return fmt.Errorf("password is required")
	}

	if len(r.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	if r.FirstName == "" {
		return fmt.Errorf("first name is required")
	}

	if r.LastName == "" {
		return fmt.Errorf("last name is required")
	}

	return nil
}

// LoginRequest represents the request body for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Validate validates the LoginRequest fields.
func (r *LoginRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)

	if r.Email == "" {
		return fmt.Errorf("email is required")
	}

	if !emailRegex.MatchString(r.Email) {
		return fmt.Errorf("invalid email format")
	}

	if r.Password == "" {
		return fmt.Errorf("password is required")
	}

	return nil
}

// UserResponse represents the user data returned in API responses.
// It exposes only safe, non-sensitive user information.
type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	KYCStatus string `json:"kyc_status"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// TokenResponse represents the authentication token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

// RefreshTokenRequest represents the request body for token refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// UpdateProfileRequest represents the request body for profile updates.
type UpdateProfileRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
}

// UpdateKYCRequest represents the request body for admin KYC status updates.
type UpdateKYCRequest struct {
	KYCStatus string `json:"kyc_status" binding:"required,oneof=pending verified rejected"`
}
