package repository_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/alex-necsoiu/pandora-exchange/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDatabaseURL = "postgres://pandora:pandora_dev_secret@localhost:5432/pandora_dev?sslmode=disable"
)

// getTestLogger returns a logger for testing purposes
func getTestLogger() *observability.Logger {
	var buf bytes.Buffer
	return observability.NewLoggerWithWriter("dev", "test-service", &buf)
}

// TestUserRepository_Create tests user creation.
func TestUserRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewUserRepository(pool, getTestLogger())
	ctx := context.Background()

	t.Run("create user successfully", func(t *testing.T) {
		email := generateTestEmail()
		user, err := repo.Create(ctx, email, "John Doe", "hashed_password_123")

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, user.ID)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, "John Doe", user.FullName)
		assert.Equal(t, "hashed_password_123", user.HashedPassword)
		assert.Equal(t, domain.KYCStatusPending, user.KYCStatus)
		assert.False(t, user.CreatedAt.IsZero())
		assert.False(t, user.UpdatedAt.IsZero())
		assert.Nil(t, user.DeletedAt)
		assert.False(t, user.IsDeleted())
		assert.False(t, user.IsKYCVerified())
	})

	t.Run("create user with empty full name", func(t *testing.T) {
		email := generateTestEmail()
		user, err := repo.Create(ctx, email, "", "hashed_password_456")

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, user.ID)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, "", user.FullName)
		assert.Equal(t, "hashed_password_456", user.HashedPassword)
	})

	t.Run("create user with duplicate email returns error", func(t *testing.T) {
		email := generateTestEmail()

		// Create first user
		_, err := repo.Create(ctx, email, "User One", "password1")
		require.NoError(t, err)

		// Attempt to create second user with same email
		_, err = repo.Create(ctx, email, "User Two", "password2")
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
	})
}

// TestUserRepository_GetByID tests retrieving users by ID.
func TestUserRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewUserRepository(pool, getTestLogger())
	ctx := context.Background()

	t.Run("get existing user by ID", func(t *testing.T) {
		// Create user
		email := generateTestEmail()
		created, err := repo.Create(ctx, email, "Jane Doe", "hashed_pass")
		require.NoError(t, err)

		// Retrieve user
		user, err := repo.GetByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, user.ID)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, "Jane Doe", user.FullName)
	})

	t.Run("get non-existent user returns error", func(t *testing.T) {
		randomID := uuid.New()
		_, err := repo.GetByID(ctx, randomID)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("get soft-deleted user returns error", func(t *testing.T) {
		// Create and then soft delete user
		email := generateTestEmail()
		created, err := repo.Create(ctx, email, "Deleted User", "pass")
		require.NoError(t, err)

		err = repo.SoftDelete(ctx, created.ID)
		require.NoError(t, err)

		// Try to get deleted user
		_, err = repo.GetByID(ctx, created.ID)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

// TestUserRepository_GetByEmail tests retrieving users by email.
func TestUserRepository_GetByEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewUserRepository(pool, getTestLogger())
	ctx := context.Background()

	t.Run("get existing user by email", func(t *testing.T) {
		email := generateTestEmail()
		created, err := repo.Create(ctx, email, "Email User", "hashed")
		require.NoError(t, err)

		user, err := repo.GetByEmail(ctx, email)
		require.NoError(t, err)
		assert.Equal(t, created.ID, user.ID)
		assert.Equal(t, email, user.Email)
	})

	t.Run("get non-existent email returns error", func(t *testing.T) {
		_, err := repo.GetByEmail(ctx, "nonexistent@example.com")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("get soft-deleted user by email returns error", func(t *testing.T) {
		email := generateTestEmail()
		created, err := repo.Create(ctx, email, "Soon Deleted", "pass")
		require.NoError(t, err)

		err = repo.SoftDelete(ctx, created.ID)
		require.NoError(t, err)

		_, err = repo.GetByEmail(ctx, email)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

// TestUserRepository_UpdateKYCStatus tests KYC status updates.
func TestUserRepository_UpdateKYCStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewUserRepository(pool, getTestLogger())
	ctx := context.Background()

	t.Run("update KYC status to verified", func(t *testing.T) {
		email := generateTestEmail()
		created, err := repo.Create(ctx, email, "KYC User", "pass")
		require.NoError(t, err)
		assert.Equal(t, domain.KYCStatusPending, created.KYCStatus)

		updated, err := repo.UpdateKYCStatus(ctx, created.ID, domain.KYCStatusVerified)
		require.NoError(t, err)
		assert.Equal(t, domain.KYCStatusVerified, updated.KYCStatus)
		assert.True(t, updated.IsKYCVerified())
		assert.True(t, updated.UpdatedAt.After(created.UpdatedAt))
	})

	t.Run("update KYC status to rejected", func(t *testing.T) {
		email := generateTestEmail()
		created, err := repo.Create(ctx, email, "Rejected User", "pass")
		require.NoError(t, err)

		updated, err := repo.UpdateKYCStatus(ctx, created.ID, domain.KYCStatusRejected)
		require.NoError(t, err)
		assert.Equal(t, domain.KYCStatusRejected, updated.KYCStatus)
		assert.False(t, updated.IsKYCVerified())
	})

	t.Run("update with invalid KYC status returns error", func(t *testing.T) {
		email := generateTestEmail()
		created, err := repo.Create(ctx, email, "Invalid KYC", "pass")
		require.NoError(t, err)

		_, err = repo.UpdateKYCStatus(ctx, created.ID, domain.KYCStatus("invalid"))
		assert.ErrorIs(t, err, domain.ErrInvalidKYCStatus)
	})

	t.Run("update non-existent user returns error", func(t *testing.T) {
		randomID := uuid.New()
		_, err := repo.UpdateKYCStatus(ctx, randomID, domain.KYCStatusVerified)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

// TestUserRepository_UpdateProfile tests profile updates.
func TestUserRepository_UpdateProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewUserRepository(pool, getTestLogger())
	ctx := context.Background()

	t.Run("update full name", func(t *testing.T) {
		email := generateTestEmail()
		created, err := repo.Create(ctx, email, "Old Name", "pass")
		require.NoError(t, err)

		updated, err := repo.UpdateProfile(ctx, created.ID, "New Name")
		require.NoError(t, err)
		assert.Equal(t, "New Name", updated.FullName)
		assert.True(t, updated.UpdatedAt.After(created.UpdatedAt))
	})

	t.Run("update to empty full name", func(t *testing.T) {
		email := generateTestEmail()
		created, err := repo.Create(ctx, email, "Has Name", "pass")
		require.NoError(t, err)

		updated, err := repo.UpdateProfile(ctx, created.ID, "")
		require.NoError(t, err)
		assert.Equal(t, "", updated.FullName)
	})

	t.Run("update non-existent user returns error", func(t *testing.T) {
		randomID := uuid.New()
		_, err := repo.UpdateProfile(ctx, randomID, "New Name")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

// TestUserRepository_SoftDelete tests soft deletion.
func TestUserRepository_SoftDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewUserRepository(pool, getTestLogger())
	ctx := context.Background()

	t.Run("soft delete user", func(t *testing.T) {
		email := generateTestEmail()
		created, err := repo.Create(ctx, email, "To Delete", "pass")
		require.NoError(t, err)

		err = repo.SoftDelete(ctx, created.ID)
		require.NoError(t, err)

		// Verify user cannot be retrieved
		_, err = repo.GetByID(ctx, created.ID)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("soft delete non-existent user returns error", func(t *testing.T) {
		randomID := uuid.New()
		err := repo.SoftDelete(ctx, randomID)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("soft delete already deleted user returns error", func(t *testing.T) {
		email := generateTestEmail()
		created, err := repo.Create(ctx, email, "Double Delete", "pass")
		require.NoError(t, err)

		// First deletion
		err = repo.SoftDelete(ctx, created.ID)
		require.NoError(t, err)

		// Second deletion should fail
		err = repo.SoftDelete(ctx, created.ID)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

// TestUserRepository_List tests listing users with pagination.
func TestUserRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewUserRepository(pool, getTestLogger())
	ctx := context.Background()

	t.Run("list users with pagination", func(t *testing.T) {
		// Create 5 test users
		for i := 0; i < 5; i++ {
			email := generateTestEmail()
			_, err := repo.Create(ctx, email, "Test User", "pass")
			require.NoError(t, err)
		}

		// Get first page (2 users)
		users, err := repo.List(ctx, 2, 0)
		require.NoError(t, err)
		assert.Len(t, users, 2)

		// Get second page (2 users)
		users, err = repo.List(ctx, 2, 2)
		require.NoError(t, err)
		assert.Len(t, users, 2)

		// Get third page (1 user)
		users, err = repo.List(ctx, 2, 4)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), 1)
	})

	t.Run("list excludes soft-deleted users", func(t *testing.T) {
		// Create and delete a user
		email := generateTestEmail()
		created, err := repo.Create(ctx, email, "Will Delete", "pass")
		require.NoError(t, err)

		err = repo.SoftDelete(ctx, created.ID)
		require.NoError(t, err)

		// List should not include deleted user
		users, err := repo.List(ctx, 100, 0)
		require.NoError(t, err)

		for _, user := range users {
			assert.NotEqual(t, created.ID, user.ID)
			assert.False(t, user.IsDeleted())
		}
	})

	t.Run("list empty result when no users", func(t *testing.T) {
		// This test relies on cleanup between runs
		// In a real scenario, you might want to use a separate test database
		users, err := repo.List(ctx, 10, 10000)
		require.NoError(t, err)
		assert.NotNil(t, users)
	})
}

// TestUserRepository_Count tests counting active users.
func TestUserRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewUserRepository(pool, getTestLogger())
	ctx := context.Background()

	t.Run("count active users", func(t *testing.T) {
		initialCount, err := repo.Count(ctx)
		require.NoError(t, err)

		// Create 3 users
		for i := 0; i < 3; i++ {
			email := generateTestEmail()
			_, err := repo.Create(ctx, email, "Count User", "pass")
			require.NoError(t, err)
		}

		newCount, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, initialCount+3, newCount)
	})

	t.Run("count excludes soft-deleted users", func(t *testing.T) {
		beforeCount, err := repo.Count(ctx)
		require.NoError(t, err)

		// Create user
		email := generateTestEmail()
		created, err := repo.Create(ctx, email, "Delete for Count", "pass")
		require.NoError(t, err)

		afterCreate, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, beforeCount+1, afterCreate)

		// Delete user
		err = repo.SoftDelete(ctx, created.ID)
		require.NoError(t, err)

		afterDelete, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, beforeCount, afterDelete)
	})
}

// Helper functions

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, testDatabaseURL)
	require.NoError(t, err, "failed to connect to test database")

	// Verify connection
	err = pool.Ping(ctx)
	require.NoError(t, err, "failed to ping test database")

	// Cleanup function to close pool and clean up test data
	cleanup := func() {
		pool.Close()
	}

	return pool, cleanup
}

func generateTestEmail() string {
	return "test_" + uuid.New().String() + "@example.com"
}
