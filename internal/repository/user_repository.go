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
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/alex-necsoiu/pandora-exchange/internal/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// UserRepository implements domain.UserRepository using sqlc-generated queries.
// Never exposes sqlc types to the domain layer - all conversions happen here.
type UserRepository struct {
	queries *postgres.Queries
	logger  *observability.Logger
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(db postgres.DBTX, logger *observability.Logger) *UserRepository {
	logger.Info("UserRepository initialized")
	return &UserRepository{
		queries: postgres.New(db),
		logger:  logger,
	}
}

// Create creates a new user with the provided email, first name, last name, and hashed password.
// Returns domain.ErrUserAlreadyExists if email already exists.
func (r *UserRepository) Create(ctx context.Context, email, firstName, lastName, hashedPassword string) (*domain.User, error) {
	r.logger.WithField("email", email).Debug("Creating user")

	// Default role is 'user' (handled by SQL query COALESCE)
	dbUser, err := r.queries.CreateUser(ctx, postgres.CreateUserParams{
		Email:          email,
		FirstName:      firstName,
		LastName:       lastName,
		HashedPassword: hashedPassword,
		Column5:        "user", // role column
	})
	if err != nil {
		// Check for unique constraint violation (duplicate email)
		if isDuplicateKeyError(err) {
			r.logger.WithField("email", email).Warn("User creation failed: email already exists")
			return nil, domain.ErrUserAlreadyExists
		}
		r.logger.WithFields(map[string]interface{}{
			"email": email,
			"error": err.Error(),
		}).Error("Failed to create user in database")
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	r.logger.WithFields(map[string]interface{}{
		"user_id": dbUser.ID,
		"email":   email,
	}).Info("User created successfully")
	return dbUserToDomain(&dbUser), nil
}

// GetByID retrieves a user by their unique ID.
// Returns domain.ErrUserNotFound if user doesn't exist or is soft-deleted.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	r.logger.WithField("user_id", id).Debug("Getting user by ID")
	
	dbUser, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.WithField("user_id", id).Debug("User not found")
			return nil, domain.ErrUserNotFound
		}
		r.logger.WithFields(map[string]interface{}{
			"user_id": id,
			"error":   err.Error(),
		}).Error("Failed to get user by ID")
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return dbUserToDomain(&dbUser), nil
}

// GetByEmail retrieves a user by their email address.
// Returns domain.ErrUserNotFound if user doesn't exist or is soft-deleted.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.logger.WithField("email", email).Debug("Getting user by email")
	
	dbUser, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.WithField("email", email).Debug("User not found")
			return nil, domain.ErrUserNotFound
		}
		r.logger.WithFields(map[string]interface{}{
			"email": email,
			"error": err.Error(),
		}).Error("Failed to get user by email")
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return dbUserToDomain(&dbUser), nil
}

// UpdateKYCStatus updates the KYC verification status for a user.
// Returns domain.ErrUserNotFound if user doesn't exist.
// Returns domain.ErrInvalidKYCStatus if status is invalid.
func (r *UserRepository) UpdateKYCStatus(ctx context.Context, id uuid.UUID, status domain.KYCStatus) (*domain.User, error) {
	if !status.IsValid() {
		r.logger.WithFields(map[string]interface{}{
			"user_id": id,
			"status":  status,
		}).Warn("Invalid KYC status")
		return nil, domain.ErrInvalidKYCStatus
	}

	r.logger.WithFields(map[string]interface{}{
		"user_id": id,
		"status":  status,
	}).Debug("Updating KYC status")

	dbUser, err := r.queries.UpdateUserKYCStatus(ctx, postgres.UpdateUserKYCStatusParams{
		ID:        id,
		KycStatus: status.String(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.WithField("user_id", id).Debug("User not found for KYC update")
			return nil, domain.ErrUserNotFound
		}
		r.logger.WithFields(map[string]interface{}{
			"user_id": id,
			"status":  status,
			"error":   err.Error(),
		}).Error("Failed to update KYC status")
		return nil, fmt.Errorf("failed to update KYC status: %w", err)
	}

	r.logger.WithFields(map[string]interface{}{
		"user_id": id,
		"status":  status,
	}).Info("KYC status updated successfully")
	return dbUserToDomain(&dbUser), nil
}

// UpdateProfile updates the user's profile information (first name and last name).
// Returns domain.ErrUserNotFound if user doesn't exist.
func (r *UserRepository) UpdateProfile(ctx context.Context, id uuid.UUID, firstName, lastName string) (*domain.User, error) {
	r.logger.WithField("user_id", id).Debug("Updating user profile")

	dbUser, err := r.queries.UpdateUserProfile(ctx, postgres.UpdateUserProfileParams{
		ID:        id,
		FirstName: firstName,
		LastName:  lastName,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.WithField("user_id", id).Debug("User not found for profile update")
			return nil, domain.ErrUserNotFound
		}
		r.logger.WithFields(map[string]interface{}{
			"user_id": id,
			"error":   err.Error(),
		}).Error("Failed to update user profile")
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}

	r.logger.WithField("user_id", id).Info("User profile updated successfully")
	return dbUserToDomain(&dbUser), nil
}

// SoftDelete marks a user as deleted without removing the record.
// Returns domain.ErrUserNotFound if user doesn't exist or is already deleted.
func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	r.logger.WithField("user_id", id).Debug("Soft deleting user")
	
	rowsAffected, err := r.queries.SoftDeleteUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			r.logger.WithField("user_id", id).Debug("User not found for deletion")
			return domain.ErrUserNotFound
		}
		r.logger.WithFields(map[string]interface{}{
			"user_id": id,
			"error":   err.Error(),
		}).Error("Failed to soft delete user")
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.WithField("user_id", id).Debug("User not found for deletion (no rows affected)")
		return domain.ErrUserNotFound
	}

	r.logger.WithField("user_id", id).Info("User soft deleted successfully")
	return nil
}

// List retrieves a paginated list of active users.
// Returns empty slice if no users found.
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	r.logger.WithFields(map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}).Debug("Listing users")
	
	// Validate pagination parameters to prevent integer overflow
	if limit < 0 || limit > 1000 {
		return nil, fmt.Errorf("invalid limit: must be between 0 and 1000")
	}
	if offset < 0 {
		return nil, fmt.Errorf("invalid offset: must be non-negative")
	}
	
	dbUsers, err := r.queries.ListUsers(ctx, postgres.ListUsersParams{
		Limit:  int32(limit),  // #nosec G115 -- validated above
		Offset: int32(offset), // #nosec G115 -- validated above
	})
	if err != nil {
		r.logger.WithFields(map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"error":  err.Error(),
		}).Error("Failed to list users")
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]*domain.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		users[i] = dbUserToDomain(&dbUser)
	}

	r.logger.WithFields(map[string]interface{}{
		"count":  len(users),
		"limit":  limit,
		"offset": offset,
	}).Debug("Users listed successfully")
	return users, nil
}

// Count returns the total count of active (non-deleted) users.
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	r.logger.Debug("Counting users")
	
	count, err := r.queries.CountUsers(ctx)
	if err != nil {
		r.logger.WithField("error", err.Error()).Error("Failed to count users")
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	r.logger.WithField("count", count).Debug("Users counted successfully")
	return count, nil
}

// dbUserToDomain converts a database User model to a domain User model.
// Handles conversion of database-specific types (pgtype) to Go standard types.
func dbUserToDomain(dbUser *postgres.User) *domain.User {
	user := &domain.User{
		ID:             dbUser.ID,
		Email:          dbUser.Email,
		FirstName:      dbUser.FirstName,
		LastName:       dbUser.LastName,
		HashedPassword: dbUser.HashedPassword,
		Role:           domain.Role(dbUser.Role),
		KYCStatus:      domain.KYCStatus(dbUser.KycStatus),
		CreatedAt:      pgTimestampToTime(dbUser.CreatedAt),
		UpdatedAt:      pgTimestampToTime(dbUser.UpdatedAt),
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

// SearchUsers searches users by email, first name, or last name with pagination.
// Admin-only operation for user management.
func (r *UserRepository) SearchUsers(ctx context.Context, query string, limit, offset int) ([]*domain.User, error) {
	r.logger.WithFields(map[string]interface{}{
		"query":  query,
		"limit":  limit,
		"offset": offset,
	}).Debug("Searching users")

	// Validate pagination parameters to prevent integer overflow
	if limit < 0 || limit > 1000 {
		return nil, fmt.Errorf("invalid limit: must be between 0 and 1000")
	}
	if offset < 0 {
		return nil, fmt.Errorf("invalid offset: must be non-negative")
	}

	dbUsers, err := r.queries.SearchUsers(ctx, postgres.SearchUsersParams{
		Column1: &query,
		Limit:   int32(limit),  // #nosec G115 -- validated above
		Offset:  int32(offset), // #nosec G115 -- validated above
	})
	if err != nil {
		r.logger.WithError(err).Error("Failed to search users")
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	users := make([]*domain.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		users[i] = dbUserToDomain(&dbUser)
	}

	r.logger.WithField("count", len(users)).Debug("Users search completed")
	return users, nil
}

// UpdateRole updates a user's role (admin-only operation).
func (r *UserRepository) UpdateRole(ctx context.Context, id uuid.UUID, role domain.Role) (*domain.User, error) {
	// Validate role before database operation
	if !role.IsValid() {
		r.logger.WithFields(map[string]interface{}{
			"user_id": id,
			"role":    role,
		}).Warn("Invalid role")
		return nil, domain.ErrInvalidRole
	}

	r.logger.WithFields(map[string]interface{}{
		"user_id": id,
		"role":    role,
	}).Debug("Updating user role")

	dbUser, err := r.queries.UpdateUserRole(ctx, postgres.UpdateUserRoleParams{
		ID:   id,
		Role: string(role),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.WithField("user_id", id).Debug("User not found for role update")
			return nil, domain.ErrUserNotFound
		}
		r.logger.WithError(err).Error("Failed to update user role")
		return nil, fmt.Errorf("failed to update user role: %w", err)
	}

	r.logger.WithField("user_id", id).Info("User role updated successfully")
	return dbUserToDomain(&dbUser), nil
}

// GetByIDIncludeDeleted retrieves a user by ID including soft-deleted users.
// Admin-only operation for user recovery or audit purposes.
func (r *UserRepository) GetByIDIncludeDeleted(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	r.logger.WithField("user_id", id).Debug("Getting user by ID (include deleted)")

	dbUser, err := r.queries.GetUserByIDIncludeDeleted(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.WithField("user_id", id).Debug("User not found")
			return nil, domain.ErrUserNotFound
		}
		r.logger.WithError(err).Error("Failed to get user")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return dbUserToDomain(&dbUser), nil
}
