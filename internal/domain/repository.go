package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UserRepository defines the interface for user data persistence.
// This interface is implemented by the infrastructure layer (repository package).
// Following Clean Architecture: domain defines the interface, infrastructure implements it.
type UserRepository interface {
	// Create creates a new user with the provided email, first name, last name, and hashed password.
	// Returns error if email already exists or database operation fails.
	Create(ctx context.Context, email, firstName, lastName, hashedPassword string) (*User, error)

	// GetByID retrieves a user by their unique ID.
	// Returns ErrUserNotFound if user doesn't exist or is soft-deleted.
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)

	// GetByEmail retrieves a user by their email address.
	// Returns ErrUserNotFound if user doesn't exist or is soft-deleted.
	GetByEmail(ctx context.Context, email string) (*User, error)

	// UpdateKYCStatus updates the KYC verification status for a user.
	// Returns error if user doesn't exist or status is invalid.
	UpdateKYCStatus(ctx context.Context, id uuid.UUID, status KYCStatus) (*User, error)

	// UpdateProfile updates the user's profile information (first name and last name).
	// Returns error if user doesn't exist.
	UpdateProfile(ctx context.Context, id uuid.UUID, firstName, lastName string) (*User, error)

	// SoftDelete marks a user as deleted without removing the record.
	// Returns error if user doesn't exist or is already deleted.
	SoftDelete(ctx context.Context, id uuid.UUID) error

	// List retrieves a paginated list of active users.
	// Returns empty slice if no users found.
	List(ctx context.Context, limit, offset int) ([]*User, error)

	// Count returns the total count of active (non-deleted) users.
	Count(ctx context.Context) (int64, error)

	// SearchUsers searches users by email, first name, or last name with pagination.
	// Admin-only operation for user management.
	SearchUsers(ctx context.Context, query string, limit, offset int) ([]*User, error)

	// UpdateRole updates a user's role (admin-only operation).
	// Returns error if user doesn't exist or role is invalid.
	UpdateRole(ctx context.Context, id uuid.UUID, role Role) (*User, error)

	// GetByIDIncludeDeleted retrieves a user by ID including soft-deleted users.
	// Admin-only operation for user recovery or audit purposes.
	GetByIDIncludeDeleted(ctx context.Context, id uuid.UUID) (*User, error)
}

// RefreshTokenRepository defines the interface for refresh token persistence.
// Handles token storage, retrieval, and revocation for session management.
type RefreshTokenRepository interface {
	// Create stores a new refresh token for a user.
	// Includes audit information (IP address and user agent).
	Create(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time, ipAddress, userAgent string) (*RefreshToken, error)

	// GetByToken retrieves a refresh token by its value.
	// Returns error if token not found or revoked.
	GetByToken(ctx context.Context, token string) (*RefreshToken, error)

	// Revoke marks a refresh token as revoked.
	// Returns error if token doesn't exist.
	Revoke(ctx context.Context, token string) error

	// RevokeAllForUser revokes all active refresh tokens for a user.
	// Used when user logs out from all devices or password changes.
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error

	// GetActiveTokensForUser retrieves all active tokens for a user.
	// Useful for session management and "active devices" feature.
	GetActiveTokensForUser(ctx context.Context, userID uuid.UUID) ([]*RefreshToken, error)

	// CountActiveForUser returns the number of active sessions for a user.
	CountActiveForUser(ctx context.Context, userID uuid.UUID) (int64, error)

	// DeleteExpired removes expired refresh tokens from the database.
	// Should be run periodically as a cleanup job.
	DeleteExpired(ctx context.Context) error

	// GetAllActiveSessions retrieves all active sessions across all users with pagination.
	// Admin-only operation for monitoring and audit purposes.
	GetAllActiveSessions(ctx context.Context, limit, offset int) ([]*RefreshToken, error)

	// CountAllActiveSessions returns the total count of active sessions across all users.
	// Admin-only operation for analytics.
	CountAllActiveSessions(ctx context.Context) (int64, error)

	// RevokeToken revokes a specific token by its value.
	// Admin-only operation for force logout.
	RevokeToken(ctx context.Context, token string) error
}
