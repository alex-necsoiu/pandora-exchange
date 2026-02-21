package user

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for user data persistence.
// This interface is implemented by the infrastructure layer (repository package).
// Following Clean Architecture: domain defines the interface, infrastructure implements it.
type Repository interface {
	// Create creates a new user with the provided email, first name, last name, and hashed password.
	// Returns error if email already exists or database operation fails.
	Create(ctx context.Context, email, firstName, lastName, hashedPassword string) (*User, error)

	// GetByID retrieves a user by their unique ID.
	// Returns ErrNotFound if user doesn't exist or is soft-deleted.
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)

	// GetByEmail retrieves a user by their email address.
	// Returns ErrNotFound if user doesn't exist or is soft-deleted.
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
