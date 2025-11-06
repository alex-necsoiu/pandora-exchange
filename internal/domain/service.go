package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserService defines the interface for user business logic.
// This is the core service layer that orchestrates domain operations.
// Following ARCHITECTURE.md section 7 (Domain Interfaces).
type UserService interface {
	// Register creates a new user account with the provided email and password.
	// Password is hashed using Argon2id before storage.
	// Returns error if email already exists or validation fails.
	Register(ctx context.Context, email, password, fullName string) (*User, error)

	// Login authenticates a user with email and password.
	// Returns a token pair (access + refresh) if credentials are valid.
	// Returns error if credentials are invalid or account is deleted.
	Login(ctx context.Context, email, password, ipAddress, userAgent string) (*TokenPair, error)

	// RefreshToken validates a refresh token and issues a new token pair.
	// Old refresh token is revoked and a new one is issued (token rotation).
	// Returns error if refresh token is invalid, expired, or revoked.
	RefreshToken(ctx context.Context, refreshToken, ipAddress, userAgent string) (*TokenPair, error)

	// Logout revokes the provided refresh token.
	// Returns error if token doesn't exist.
	Logout(ctx context.Context, refreshToken string) error

	// LogoutAll revokes all refresh tokens for a user.
	// Logs out the user from all devices.
	LogoutAll(ctx context.Context, userID uuid.UUID) error

	// GetByID retrieves a user by their unique ID.
	// Returns error if user doesn't exist or is deleted.
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)

	// UpdateKYC updates the KYC verification status for a user.
	// Emits a kyc.updated event to Redis Streams.
	// Returns error if user doesn't exist or status is invalid.
	UpdateKYC(ctx context.Context, id uuid.UUID, status KYCStatus) (*User, error)

	// UpdateProfile updates the user's profile information.
	// Returns error if user doesn't exist.
	UpdateProfile(ctx context.Context, id uuid.UUID, fullName string) (*User, error)

	// DeleteAccount soft-deletes a user account.
	// Revokes all active refresh tokens for the user.
	// Returns error if user doesn't exist or is already deleted.
	DeleteAccount(ctx context.Context, id uuid.UUID) error

	// GetActiveSessions retrieves all active sessions (refresh tokens) for a user.
	// Useful for "active devices" feature in user dashboard.
	GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]*RefreshToken, error)
}
