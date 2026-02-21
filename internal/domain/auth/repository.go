package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TokenRepository defines the interface for refresh token persistence.
// Handles token storage, retrieval, and revocation for session management.
type TokenRepository interface {
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
