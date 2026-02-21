// Package auth contains authentication and session management domain models.
// This package follows Clean Architecture principles, remaining independent
// of infrastructure and transport concerns.
package auth

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a stored refresh token with metadata.
// Used for session management and token rotation.
type RefreshToken struct {
	Token     string
	UserID    uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time // nil if active
	IPAddress string
	UserAgent string
}

// IsActive returns true if the token is not revoked and not expired.
func (rt *RefreshToken) IsActive() bool {
	return rt.RevokedAt == nil && time.Now().Before(rt.ExpiresAt)
}

// IsExpired returns true if the token has passed its expiration time.
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsRevoked returns true if the token has been explicitly revoked.
func (rt *RefreshToken) IsRevoked() bool {
	return rt.RevokedAt != nil
}
