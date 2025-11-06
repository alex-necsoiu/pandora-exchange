// Package repository implements the domain repository interfaces using sqlc-generated code.
// This layer bridges the domain and database, converting between domain models and database types.
// Following Clean Architecture: infrastructure implements domain-defined interfaces.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// UserRepository implements domain.UserRepository using sqlc-generated queries.
// Never exposes sqlc types to the domain layer - all conversions happen here.
type UserRepository struct {
	queries *postgres.Queries
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(db postgres.DBTX) *UserRepository {
	return &UserRepository{
		queries: postgres.New(db),
	}
}

// Create creates a new user with the provided email, full name, and hashed password.
// Returns domain.ErrUserAlreadyExists if email already exists.
func (r *UserRepository) Create(ctx context.Context, email, fullName, hashedPassword string) (*domain.User, error) {
	var fullNamePtr *string
	if fullName != "" {
		fullNamePtr = &fullName
	}

	dbUser, err := r.queries.CreateUser(ctx, postgres.CreateUserParams{
		Email:          email,
		FullName:       fullNamePtr,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		// Check for unique constraint violation (duplicate email)
		if isDuplicateKeyError(err) {
			return nil, domain.ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return dbUserToDomain(&dbUser), nil
}

// GetByID retrieves a user by their unique ID.
// Returns domain.ErrUserNotFound if user doesn't exist or is soft-deleted.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	dbUser, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return dbUserToDomain(&dbUser), nil
}

// GetByEmail retrieves a user by their email address.
// Returns domain.ErrUserNotFound if user doesn't exist or is soft-deleted.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	dbUser, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return dbUserToDomain(&dbUser), nil
}

// UpdateKYCStatus updates the KYC verification status for a user.
// Returns domain.ErrUserNotFound if user doesn't exist.
// Returns domain.ErrInvalidKYCStatus if status is invalid.
func (r *UserRepository) UpdateKYCStatus(ctx context.Context, id uuid.UUID, status domain.KYCStatus) (*domain.User, error) {
	if !status.IsValid() {
		return nil, domain.ErrInvalidKYCStatus
	}

	dbUser, err := r.queries.UpdateUserKYCStatus(ctx, postgres.UpdateUserKYCStatusParams{
		ID:        id,
		KycStatus: status.String(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to update KYC status: %w", err)
	}

	return dbUserToDomain(&dbUser), nil
}

// UpdateProfile updates the user's profile information (full name).
// Returns domain.ErrUserNotFound if user doesn't exist.
func (r *UserRepository) UpdateProfile(ctx context.Context, id uuid.UUID, fullName string) (*domain.User, error) {
	var fullNamePtr *string
	if fullName != "" {
		fullNamePtr = &fullName
	}

	dbUser, err := r.queries.UpdateUserProfile(ctx, postgres.UpdateUserProfileParams{
		ID:       id,
		FullName: fullNamePtr,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}

	return dbUserToDomain(&dbUser), nil
}

// SoftDelete marks a user as deleted without removing the record.
// Returns domain.ErrUserNotFound if user doesn't exist or is already deleted.
func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := r.queries.SoftDeleteUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return domain.ErrUserNotFound
		}
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

// List retrieves a paginated list of active users.
// Returns empty slice if no users found.
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	dbUsers, err := r.queries.ListUsers(ctx, postgres.ListUsersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]*domain.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		users[i] = dbUserToDomain(&dbUser)
	}

	return users, nil
}

// Count returns the total count of active (non-deleted) users.
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.queries.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// dbUserToDomain converts a database User model to a domain User model.
// Handles conversion of database-specific types (pgtype) to Go standard types.
func dbUserToDomain(dbUser *postgres.User) *domain.User {
	user := &domain.User{
		ID:             dbUser.ID,
		Email:          dbUser.Email,
		HashedPassword: dbUser.HashedPassword,
		KYCStatus:      domain.KYCStatus(dbUser.KycStatus),
		CreatedAt:      pgTimestampToTime(dbUser.CreatedAt),
		UpdatedAt:      pgTimestampToTime(dbUser.UpdatedAt),
	}

	// Handle optional full name
	if dbUser.FullName != nil {
		user.FullName = *dbUser.FullName
	}

	// Handle optional deleted_at (soft delete)
	if dbUser.DeletedAt.Valid {
		deletedAt := pgTimestampToTime(dbUser.DeletedAt)
		user.DeletedAt = &deletedAt
	}

	return user
}

// pgTimestampToTime converts pgtype.Timestamptz to time.Time.
func pgTimestampToTime(ts pgtype.Timestamptz) time.Time {
	if ts.Valid {
		return ts.Time
	}
	return time.Time{}
}

// isDuplicateKeyError checks if the error is a unique constraint violation.
// PostgreSQL error code 23505 indicates unique_violation.
func isDuplicateKeyError(err error) bool {
	// pgx error codes for unique constraint violations
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "unique constraint") ||
		strings.Contains(errMsg, "duplicate key") ||
		strings.Contains(errMsg, "23505")
}
