// Package domain contains the core business models and interfaces for the User Service.
// This layer is independent of infrastructure and transport concerns.
// Following Clean Architecture principles, domain never imports infrastructure packages.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a user's role in the system for authorization purposes.
type Role string

const (
	// RoleUser is the default role for regular users.
	RoleUser Role = "user"
	// RoleAdmin is the role for administrators with elevated privileges.
	RoleAdmin Role = "admin"
)

// IsValid checks if the role is one of the allowed values.
func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleAdmin:
		return true
	default:
		return false
	}
}

// String returns the string representation of Role.
func (r Role) String() string {
	return string(r)
}

// KYCStatus represents the Know Your Customer verification status.
type KYCStatus string

const (
	// KYCStatusPending indicates KYC verification is pending.
	KYCStatusPending KYCStatus = "pending"
	// KYCStatusVerified indicates KYC verification is complete and approved.
	KYCStatusVerified KYCStatus = "verified"
	// KYCStatusRejected indicates KYC verification was rejected.
	KYCStatusRejected KYCStatus = "rejected"
)

// IsValid checks if the KYC status is one of the allowed values.
func (s KYCStatus) IsValid() bool {
	switch s {
	case KYCStatusPending, KYCStatusVerified, KYCStatusRejected:
		return true
	default:
		return false
	}
}

// String returns the string representation of KYCStatus.
func (s KYCStatus) String() string {
	return string(s)
}

// User represents a user in the Pandora Exchange platform.
// This is the domain model, separate from database and API representations.
// Never expose database-specific types (like pgtype.Timestamptz) in this model.
type User struct {
	ID             uuid.UUID
	Email          string
	FirstName      string // User's first name
	LastName       string // User's last name
	HashedPassword string // Argon2id hashed password
	Role           Role   // User role for authorization (user or admin)
	KYCStatus      KYCStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time // nil if not deleted
}

// IsDeleted returns true if the user has been soft-deleted.
func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

// IsKYCVerified returns true if the user's KYC status is verified.
func (u *User) IsKYCVerified() bool {
	return u.KYCStatus == KYCStatusVerified
}

// IsAdmin returns true if the user has admin role.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// TokenPair represents an access token and refresh token pair.
// Used for JWT-based authentication.
type TokenPair struct {
	User         *User
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

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
