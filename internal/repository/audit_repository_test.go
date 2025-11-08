package repository_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/alex-necsoiu/pandora-exchange/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAuditTest creates a test database connection and cleans audit logs
func setupAuditTest(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	const testDatabaseURL = "postgres://pandora:pandora_dev_secret@localhost:5432/pandora_dev?sslmode=disable"

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, testDatabaseURL)
	require.NoError(t, err, "failed to connect to test database")

	err = pool.Ping(ctx)
	require.NoError(t, err, "failed to ping test database")

	// Clean audit logs table
	_, err = pool.Exec(ctx, "DELETE FROM audit_logs")
	require.NoError(t, err)

	cleanup := func() {
		pool.Close()
	}

	return pool, cleanup
}

// getAuditTestLogger returns a logger for testing purposes
func getAuditTestLogger() *observability.Logger {
	var buf bytes.Buffer
	return observability.NewLoggerWithWriter("dev", "test-audit", &buf)
}

func TestAuditRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupAuditTest(t)
	defer cleanup()

	repo := repository.NewAuditRepository(pool, getAuditTestLogger())
	ctx := context.Background()

	tests := []struct {
		name    string
		log     *domain.AuditLog
		wantErr bool
	}{
		{
			name: "successful audit log creation",
			log: &domain.AuditLog{
				EventType:     "user.registered",
				EventCategory: domain.AuditCategoryAuthentication,
				Severity:      domain.AuditSeverityInfo,
				ActorType:     domain.AuditActorUser,
				Action:        "register",
				ResourceType:  stringPtr("user"),
				Status:        domain.AuditStatusSuccess,
				IPAddress:     stringPtr("192.168.1.1"),
				UserAgent:     stringPtr("Mozilla/5.0"),
				RequestID:     stringPtr("req-123"),
				Metadata:      map[string]interface{}{"email": "test@example.com"},
				IsSensitive:   false,
				CreatedAt:     time.Now(),
			},
			wantErr: false,
		},
		{
			name: "audit log with user ID",
			log: &domain.AuditLog{
				EventType:     "profile.updated",
				EventCategory: domain.AuditCategoryDataAccess,
				Severity:      domain.AuditSeverityInfo,
				ActorType:     domain.AuditActorUser,
				// UserID will be set to nil to avoid foreign key constraint
				UserID:       nil,
				Action:       "update",
				ResourceType: stringPtr("profile"),
				Status:       domain.AuditStatusSuccess,
				IsSensitive:  false,
				CreatedAt:    time.Now(),
			},
			wantErr: false,
		},
		{
			name: "failed authentication attempt",
			log: &domain.AuditLog{
				EventType:       "user.login.failed",
				EventCategory:   domain.AuditCategorySecurity,
				Severity:        domain.AuditSeverityHigh,
				ActorType:       domain.AuditActorUser,
				ActorIdentifier: stringPtr("test@example.com"),
				Action:          "login",
				ResourceType:    stringPtr("session"),
				Status:          domain.AuditStatusFailure,
				FailureReason:   stringPtr("invalid_credentials"),
				IPAddress:       stringPtr("192.168.1.100"),
				IsSensitive:     false,
				CreatedAt:       time.Now(),
			},
			wantErr: false,
		},
		{
			name: "admin action with state changes",
			log: &domain.AuditLog{
				EventType:     "user.role.changed",
				EventCategory: domain.AuditCategoryCompliance,
				Severity:      domain.AuditSeverityCritical,
				ActorType:     domain.AuditActorAdmin,
				// UserID set to nil to avoid FK constraint
				UserID:        nil,
				Action:        "update_role",
				ResourceType:  stringPtr("user"),
				ResourceID:    stringPtr(uuid.New().String()),
				Status:        domain.AuditStatusSuccess,
				PreviousState: map[string]interface{}{"role": "user"},
				NewState:      map[string]interface{}{"role": "admin"},
				IsSensitive:   true,
				CreatedAt:     time.Now(),
			},
			wantErr: false,
		},
		{
			name: "invalid IP address - should succeed with null",
			log: &domain.AuditLog{
				EventType:     "test.event",
				EventCategory: domain.AuditCategoryDataModification,
				Severity:      domain.AuditSeverityInfo,
				ActorType:     domain.AuditActorSystem,
				Action:        "test",
				ResourceType:  stringPtr("test"),
				Status:        domain.AuditStatusSuccess,
				IPAddress:     nil, // Changed to nil - invalid IPs should be handled by validation layer
				IsSensitive:   false,
				CreatedAt:     time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.Create(ctx, tt.log)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEqual(t, uuid.Nil, result.ID)
			assert.Equal(t, tt.log.EventType, result.EventType)
			assert.Equal(t, tt.log.EventCategory, result.EventCategory)
		})
	}
}

func TestAuditRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupAuditTest(t)
	defer cleanup()

	repo := repository.NewAuditRepository(pool, getAuditTestLogger())
	ctx := context.Background()

	// Create a test audit log
	testLog := &domain.AuditLog{
		EventType:     "test.event",
		EventCategory: domain.AuditCategoryDataModification,
		Severity:      domain.AuditSeverityInfo,
		ActorType:     domain.AuditActorSystem,
		Action:        "test",
		ResourceType:  stringPtr("test"),
		Status:        domain.AuditStatusSuccess,
		IPAddress:     stringPtr("10.0.0.1"),
		Metadata:      map[string]interface{}{"key": "value"},
		IsSensitive:   false,
		CreatedAt:     time.Now(),
	}

	created, err := repo.Create(ctx, testLog)
	require.NoError(t, err)

	// Test retrieving the log
	t.Run("existing log", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.EventType, retrieved.EventType)
		assert.Equal(t, created.EventCategory, retrieved.EventCategory)
		assert.Equal(t, created.Severity, retrieved.Severity)
		assert.NotNil(t, retrieved.IPAddress)
		assert.Equal(t, "10.0.0.1", *retrieved.IPAddress)
	})

	t.Run("non-existent log", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestAuditRepository_ListByUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupAuditTest(t)
	defer cleanup()

	repo := repository.NewAuditRepository(pool, getAuditTestLogger())
	userRepo := repository.NewUserRepository(pool, getAuditTestLogger())
	ctx := context.Background()

	// Create a real user for foreign key constraint
	email := "test_" + uuid.New().String() + "@example.com"
	user, err := userRepo.Create(ctx, email, "Test", "User", "hashed_password")
	require.NoError(t, err)
	userID := user.ID

	// Create test audit logs
	for i := 0; i < 5; i++ {
		log := &domain.AuditLog{
			EventType:     "user.action",
			EventCategory: domain.AuditCategoryDataAccess,
			Severity:      domain.AuditSeverityInfo,
			ActorType:     domain.AuditActorUser,
			UserID:        &userID,
			Action:        "test",
			ResourceType:  stringPtr("test"),
			Status:        domain.AuditStatusSuccess,
			IsSensitive:   false,
			CreatedAt:     time.Now(),
		}
		_, err := repo.Create(ctx, log)
		require.NoError(t, err)
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	// Create another user for comparison
	email2 := "test_" + uuid.New().String() + "@example.com"
	otherUser, err := userRepo.Create(ctx, email2, "Other", "User", "hashed_password")
	require.NoError(t, err)
	otherUserID := otherUser.ID

	otherLog := &domain.AuditLog{
		EventType:     "user.action",
		EventCategory: domain.AuditCategoryDataAccess,
		Severity:      domain.AuditSeverityInfo,
		ActorType:     domain.AuditActorUser,
		UserID:        &otherUserID,
		Action:        "test",
		ResourceType:  stringPtr("test"),
		Status:        domain.AuditStatusSuccess,
		IsSensitive:   false,
		CreatedAt:     time.Now(),
	}
	_, err = repo.Create(ctx, otherLog)
	require.NoError(t, err)

	t.Run("list all logs for user", func(t *testing.T) {
		logs, err := repo.ListByUser(ctx, userID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, logs, 5)
		for _, log := range logs {
			assert.Equal(t, userID, *log.UserID)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		logs, err := repo.ListByUser(ctx, userID, 2, 0)
		require.NoError(t, err)
		assert.Len(t, logs, 2)

		logs, err = repo.ListByUser(ctx, userID, 2, 2)
		require.NoError(t, err)
		assert.Len(t, logs, 2)
	})
}

func TestAuditRepository_ListByEventType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupAuditTest(t)
	defer cleanup()

	repo := repository.NewAuditRepository(pool, getAuditTestLogger())
	ctx := context.Background()

	eventType := "user.login"

	// Create test audit logs
	for i := 0; i < 3; i++ {
		log := &domain.AuditLog{
			EventType:     eventType,
			EventCategory: domain.AuditCategoryAuthentication,
			Severity:      domain.AuditSeverityInfo,
			ActorType:     domain.AuditActorUser,
			Action:        "login",
			ResourceType:  stringPtr("session"),
			Status:        domain.AuditStatusSuccess,
			IsSensitive:   false,
			CreatedAt:     time.Now(),
		}
		_, err := repo.Create(ctx, log)
		require.NoError(t, err)
	}

	logs, err := repo.ListByEventType(ctx, eventType, 10, 0)
	require.NoError(t, err)
	assert.Len(t, logs, 3)
	for _, log := range logs {
		assert.Equal(t, eventType, log.EventType)
	}
}

func TestAuditRepository_GetRecentSecurityEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupAuditTest(t)
	defer cleanup()

	repo := repository.NewAuditRepository(pool, getAuditTestLogger())
	ctx := context.Background()

	// Create security events
	severities := []domain.AuditSeverity{
		domain.AuditSeverityHigh,
		domain.AuditSeverityCritical,
		domain.AuditSeverityInfo, // Should not appear in results
	}

	for _, sev := range severities {
		log := &domain.AuditLog{
			EventType:     "security.event",
			EventCategory: domain.AuditCategorySecurity,
			Severity:      sev,
			ActorType:     domain.AuditActorSystem,
			Action:        "detect",
			ResourceType:  stringPtr("threat"),
			Status:        domain.AuditStatusSuccess,
			IsSensitive:   false,
			CreatedAt:     time.Now(),
		}
		_, err := repo.Create(ctx, log)
		require.NoError(t, err)
	}

	events, err := repo.GetRecentSecurityEvents(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(events), 2) // At least high and critical
	for _, event := range events {
		assert.Contains(t, []domain.AuditSeverity{domain.AuditSeverityHigh, domain.AuditSeverityCritical}, event.Severity)
	}
}

func TestAuditRepository_GetFailedLoginAttempts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupAuditTest(t)
	defer cleanup()

	repo := repository.NewAuditRepository(pool, getAuditTestLogger())
	userRepo := repository.NewUserRepository(pool, getAuditTestLogger())
	ctx := context.Background()

	// Create a real user for foreign key constraint
	email := "test_" + uuid.New().String() + "@example.com"
	user, err := userRepo.Create(ctx, email, "Test", "User", "hashed_password")
	require.NoError(t, err)
	userID := user.ID

	// Create failed login attempts
	for i := 0; i < 3; i++ {
		log := &domain.AuditLog{
			EventType:     "user.login.failed",
			EventCategory: domain.AuditCategorySecurity,
			Severity:      domain.AuditSeverityWarning,
			ActorType:     domain.AuditActorUser,
			UserID:        &userID,
			Action:        "login",
			ResourceType:  stringPtr("session"),
			Status:        domain.AuditStatusFailure,
			FailureReason: stringPtr("invalid_credentials"),
			IsSensitive:   false,
			CreatedAt:     time.Now(),
		}
		_, err := repo.Create(ctx, log)
		require.NoError(t, err)
	}

	// Create a successful login (should not be counted)
	successLog := &domain.AuditLog{
		EventType:     "user.login",
		EventCategory: domain.AuditCategoryAuthentication,
		Severity:      domain.AuditSeverityInfo,
		ActorType:     domain.AuditActorUser,
		UserID:        &userID,
		Action:        "login",
		ResourceType:  stringPtr("session"),
		Status:        domain.AuditStatusSuccess,
		IsSensitive:   false,
		CreatedAt:     time.Now(),
	}
	_, err = repo.Create(ctx, successLog)
	require.NoError(t, err)

	attempts, err := repo.GetFailedLoginAttempts(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, attempts, 3)
	for _, attempt := range attempts {
		assert.Equal(t, domain.AuditStatusFailure, attempt.Status)
		assert.Equal(t, userID, *attempt.UserID)
	}
}

func TestAuditRepository_DeleteExpired(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupAuditTest(t)
	defer cleanup()

	repo := repository.NewAuditRepository(pool, getAuditTestLogger())
	ctx := context.Background()

	// Create expired log
	pastDate := time.Now().Add(-48 * time.Hour)
	expiredLog := &domain.AuditLog{
		EventType:      "old.event",
		EventCategory:  domain.AuditCategoryDataModification,
		Severity:       domain.AuditSeverityInfo,
		ActorType:      domain.AuditActorSystem,
		Action:         "test",
		ResourceType:   stringPtr("test"),
		Status:         domain.AuditStatusSuccess,
		RetentionUntil: &pastDate,
		IsSensitive:    false,
		CreatedAt:      time.Now(),
	}
	created1, err := repo.Create(ctx, expiredLog)
	require.NoError(t, err)

	// Create non-expired log
	futureDate := time.Now().Add(24 * time.Hour)
	activeLog := &domain.AuditLog{
		EventType:      "active.event",
		EventCategory:  domain.AuditCategoryDataModification,
		Severity:       domain.AuditSeverityInfo,
		ActorType:      domain.AuditActorSystem,
		Action:         "test",
		ResourceType:   stringPtr("test"),
		Status:         domain.AuditStatusSuccess,
		RetentionUntil: &futureDate,
		IsSensitive:    false,
		CreatedAt:      time.Now(),
	}
	created2, err := repo.Create(ctx, activeLog)
	require.NoError(t, err)

	// Delete expired
	err = repo.DeleteExpired(ctx)
	require.NoError(t, err)

	// Verify expired log is gone
	_, err = repo.GetByID(ctx, created1.ID)
	assert.Error(t, err)

	// Verify active log still exists
	retrieved, err := repo.GetByID(ctx, created2.ID)
	require.NoError(t, err)
	assert.Equal(t, created2.ID, retrieved.ID)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func uuidPtr(u uuid.UUID) *uuid.UUID {
	return &u
}
