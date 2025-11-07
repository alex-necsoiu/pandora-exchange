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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getRefreshTokenTestLogger returns a logger for testing purposes
func getRefreshTokenTestLogger() *observability.Logger {
	var buf bytes.Buffer
	return observability.NewLoggerWithWriter("dev", "test-service", &buf)
}

// TestRefreshTokenRepository_Create tests refresh token creation.
func TestRefreshTokenRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := repository.NewUserRepository(pool, getRefreshTokenTestLogger())
	tokenRepo := repository.NewRefreshTokenRepository(pool, getRefreshTokenTestLogger())
	ctx := context.Background()

	t.Run("create refresh token successfully", func(t *testing.T) {
		// Create test user
		user, err := userRepo.Create(ctx, generateTestEmail(), "Token", "User", "pass")
		require.NoError(t, err)

		// Create refresh token
		tokenString := "test_token_" + uuid.New().String()
		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		ipAddress := "192.168.1.1"
		userAgent := "Mozilla/5.0"

		token, err := tokenRepo.Create(ctx, tokenString, user.ID, expiresAt, ipAddress, userAgent)
		require.NoError(t, err)
		assert.Equal(t, tokenString, token.Token)
		assert.Equal(t, user.ID, token.UserID)
		assert.Equal(t, ipAddress, token.IPAddress)
		assert.Equal(t, userAgent, token.UserAgent)
		assert.False(t, token.ExpiresAt.IsZero())
		assert.False(t, token.CreatedAt.IsZero())
		assert.Nil(t, token.RevokedAt)
		assert.False(t, token.IsRevoked())
		assert.False(t, token.IsExpired())
		assert.True(t, token.IsActive())
	})

	t.Run("create token with empty audit fields", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "User", "2", "pass")
		require.NoError(t, err)

		tokenString := "test_token_" + uuid.New().String()
		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		token, err := tokenRepo.Create(ctx, tokenString, user.ID, expiresAt, "", "")
		require.NoError(t, err)
		assert.Equal(t, "", token.IPAddress)
		assert.Equal(t, "", token.UserAgent)
	})
}

// TestRefreshTokenRepository_Get tests retrieving tokens.
func TestRefreshTokenRepository_Get(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := repository.NewUserRepository(pool, getRefreshTokenTestLogger())
	tokenRepo := repository.NewRefreshTokenRepository(pool, getRefreshTokenTestLogger())
	ctx := context.Background()

	t.Run("get existing token", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "Get", "User", "pass")
		require.NoError(t, err)

		tokenString := "get_token_" + uuid.New().String()
		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		created, err := tokenRepo.Create(ctx, tokenString, user.ID, expiresAt, "1.2.3.4", "Agent")
		require.NoError(t, err)

		retrieved, err := tokenRepo.GetByToken(ctx, tokenString)
		require.NoError(t, err)
		assert.Equal(t, created.Token, retrieved.Token)
		assert.Equal(t, created.UserID, retrieved.UserID)
		assert.Equal(t, "1.2.3.4", retrieved.IPAddress)
		assert.Equal(t, "Agent", retrieved.UserAgent)
	})

	t.Run("get non-existent token returns error", func(t *testing.T) {
		_, err := tokenRepo.GetByToken(ctx, "nonexistent_token")
		assert.ErrorIs(t, err, domain.ErrRefreshTokenNotFound)
	})
}

// TestRefreshTokenRepository_Revoke tests token revocation.
func TestRefreshTokenRepository_Revoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := repository.NewUserRepository(pool, getRefreshTokenTestLogger())
	tokenRepo := repository.NewRefreshTokenRepository(pool, getRefreshTokenTestLogger())
	ctx := context.Background()

	t.Run("revoke token successfully", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "Revoke", "User", "pass")
		require.NoError(t, err)

		tokenString := "revoke_token_" + uuid.New().String()
		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		_, err = tokenRepo.Create(ctx, tokenString, user.ID, expiresAt, "1.1.1.1", "UA")
		require.NoError(t, err)

		err = tokenRepo.Revoke(ctx, tokenString)
		require.NoError(t, err)

		// Verify token is revoked
		token, err := tokenRepo.GetByToken(ctx, tokenString)
		require.NoError(t, err)
		assert.True(t, token.IsRevoked())
		assert.False(t, token.IsActive())
		assert.NotNil(t, token.RevokedAt)
	})

	t.Run("revoke non-existent token returns error", func(t *testing.T) {
		err := tokenRepo.Revoke(ctx, "nonexistent_revoke_token")
		assert.ErrorIs(t, err, domain.ErrRefreshTokenNotFound)
	})
}

// TestRefreshTokenRepository_RevokeAllForUser tests revoking all user tokens.
func TestRefreshTokenRepository_RevokeAllForUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := repository.NewUserRepository(pool, getRefreshTokenTestLogger())
	tokenRepo := repository.NewRefreshTokenRepository(pool, getRefreshTokenTestLogger())
	ctx := context.Background()

	t.Run("revoke all tokens for user", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "Multi", "User", "pass")
		require.NoError(t, err)

		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		// Create 3 tokens for the user
		token1 := "multi_token_1_" + uuid.New().String()
		token2 := "multi_token_2_" + uuid.New().String()
		token3 := "multi_token_3_" + uuid.New().String()

		_, err = tokenRepo.Create(ctx, token1, user.ID, expiresAt, "1.1.1.1", "UA1")
		require.NoError(t, err)
		_, err = tokenRepo.Create(ctx, token2, user.ID, expiresAt, "2.2.2.2", "UA2")
		require.NoError(t, err)
		_, err = tokenRepo.Create(ctx, token3, user.ID, expiresAt, "3.3.3.3", "UA3")
		require.NoError(t, err)

		// Revoke all tokens
		err = tokenRepo.RevokeAllForUser(ctx, user.ID)
		require.NoError(t, err)

		// Verify all tokens are revoked
		t1, err := tokenRepo.GetByToken(ctx, token1)
		require.NoError(t, err)
		assert.True(t, t1.IsRevoked())

		t2, err := tokenRepo.GetByToken(ctx, token2)
		require.NoError(t, err)
		assert.True(t, t2.IsRevoked())

		t3, err := tokenRepo.GetByToken(ctx, token3)
		require.NoError(t, err)
		assert.True(t, t3.IsRevoked())

		// Verify no active tokens
		activeTokens, err := tokenRepo.GetActiveTokensForUser(ctx, user.ID)
		require.NoError(t, err)
		assert.Empty(t, activeTokens)
	})

	t.Run("revoke all for user with no tokens succeeds", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "No", "User", "pass")
		require.NoError(t, err)

		err = tokenRepo.RevokeAllForUser(ctx, user.ID)
		assert.NoError(t, err)
	})
}

// TestRefreshTokenRepository_GetActiveForUser tests retrieving active tokens.
func TestRefreshTokenRepository_GetActiveForUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := repository.NewUserRepository(pool, getRefreshTokenTestLogger())
	tokenRepo := repository.NewRefreshTokenRepository(pool, getRefreshTokenTestLogger())
	ctx := context.Background()

	t.Run("get active tokens for user", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "Active", "User", "pass")
		require.NoError(t, err)

		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		// Create 2 active tokens
		token1 := "active_1_" + uuid.New().String()
		token2 := "active_2_" + uuid.New().String()
		_, err = tokenRepo.Create(ctx, token1, user.ID, expiresAt, "1.1.1.1", "UA1")
		require.NoError(t, err)
		_, err = tokenRepo.Create(ctx, token2, user.ID, expiresAt, "2.2.2.2", "UA2")
		require.NoError(t, err)

		// Create 1 revoked token
		token3 := "revoked_" + uuid.New().String()
		_, err = tokenRepo.Create(ctx, token3, user.ID, expiresAt, "3.3.3.3", "UA3")
		require.NoError(t, err)
		err = tokenRepo.Revoke(ctx, token3)
		require.NoError(t, err)

		// Get active tokens
		activeTokens, err := tokenRepo.GetActiveTokensForUser(ctx, user.ID)
		require.NoError(t, err)
		assert.Len(t, activeTokens, 2)

		// Verify all returned tokens are active
		for _, token := range activeTokens {
			assert.True(t, token.IsActive())
			assert.False(t, token.IsRevoked())
		}
	})

	t.Run("get active tokens excludes expired", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "Expired", "User", "pass")
		require.NoError(t, err)

		// Create expired token
		expiredToken := "expired_" + uuid.New().String()
		expiredTime := time.Now().Add(-1 * time.Hour)
		_, err = tokenRepo.Create(ctx, expiredToken, user.ID, expiredTime, "1.1.1.1", "UA")
		require.NoError(t, err)

		// Get active tokens
		activeTokens, err := tokenRepo.GetActiveTokensForUser(ctx, user.ID)
		require.NoError(t, err)
		assert.Empty(t, activeTokens)
	})

	t.Run("get active tokens for user with no tokens", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "No", "User", "pass")
		require.NoError(t, err)

		activeTokens, err := tokenRepo.GetActiveTokensForUser(ctx, user.ID)
		require.NoError(t, err)
		assert.Empty(t, activeTokens)
	})
}

// TestRefreshTokenRepository_CountActiveForUser tests counting active tokens.
func TestRefreshTokenRepository_CountActiveForUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := repository.NewUserRepository(pool, getRefreshTokenTestLogger())
	tokenRepo := repository.NewRefreshTokenRepository(pool, getRefreshTokenTestLogger())
	ctx := context.Background()

	t.Run("count active tokens", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "Count", "User", "pass")
		require.NoError(t, err)

		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		// Initially no tokens
		count, err := tokenRepo.CountActiveForUser(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)

		// Create 3 active tokens
		for i := 0; i < 3; i++ {
			token := "count_token_" + uuid.New().String()
			_, err = tokenRepo.Create(ctx, token, user.ID, expiresAt, "1.1.1.1", "UA")
			require.NoError(t, err)
		}

		count, err = tokenRepo.CountActiveForUser(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)

		// Create 1 revoked token
		revokedToken := "revoked_count_" + uuid.New().String()
		_, err = tokenRepo.Create(ctx, revokedToken, user.ID, expiresAt, "2.2.2.2", "UA")
		require.NoError(t, err)
		err = tokenRepo.Revoke(ctx, revokedToken)
		require.NoError(t, err)

		// Count should still be 3
		count, err = tokenRepo.CountActiveForUser(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})
}

// TestRefreshTokenRepository_DeleteExpired tests deleting expired tokens.
func TestRefreshTokenRepository_DeleteExpired(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := repository.NewUserRepository(pool, getRefreshTokenTestLogger())
	tokenRepo := repository.NewRefreshTokenRepository(pool, getRefreshTokenTestLogger())
	ctx := context.Background()

	t.Run("delete expired tokens", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "Cleanup", "User", "pass")
		require.NoError(t, err)

		// Create expired token
		expiredToken := "expired_cleanup_" + uuid.New().String()
		expiredTime := time.Now().Add(-1 * time.Hour)
		_, err = tokenRepo.Create(ctx, expiredToken, user.ID, expiredTime, "1.1.1.1", "UA")
		require.NoError(t, err)

		// Create active token
		activeToken := "active_cleanup_" + uuid.New().String()
		activeTime := time.Now().Add(7 * 24 * time.Hour)
		_, err = tokenRepo.Create(ctx, activeToken, user.ID, activeTime, "2.2.2.2", "UA")
		require.NoError(t, err)

		// Delete expired tokens
		err = tokenRepo.DeleteExpired(ctx)
		require.NoError(t, err)

		// Expired token should not exist
		_, err = tokenRepo.GetByToken(ctx, expiredToken)
		assert.ErrorIs(t, err, domain.ErrRefreshTokenNotFound)

		// Active token should still exist
		token, err := tokenRepo.GetByToken(ctx, activeToken)
		require.NoError(t, err)
		assert.Equal(t, activeToken, token.Token)
	})

	t.Run("delete expired succeeds with no expired tokens", func(t *testing.T) {
		err := tokenRepo.DeleteExpired(ctx)
		assert.NoError(t, err)
	})
}

// TestRefreshTokenRepository_GetAllActiveSessions tests admin session listing.
func TestRefreshTokenRepository_GetAllActiveSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := repository.NewUserRepository(pool, getRefreshTokenTestLogger())
	tokenRepo := repository.NewRefreshTokenRepository(pool, getRefreshTokenTestLogger())
	ctx := context.Background()

	t.Run("list all active sessions with pagination", func(t *testing.T) {
		// Create multiple users with sessions
		user1, err := userRepo.Create(ctx, generateTestEmail(), "Session", "User1", "pass")
		require.NoError(t, err)
		user2, err := userRepo.Create(ctx, generateTestEmail(), "Session", "User2", "pass")
		require.NoError(t, err)

		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		// Create 3 active sessions for user1
		for i := 0; i < 3; i++ {
			token := "session1_" + uuid.New().String()
			_, err = tokenRepo.Create(ctx, token, user1.ID, expiresAt, "192.168.1."+string(rune('1'+i)), "Agent1")
			require.NoError(t, err)
		}

		// Create 2 active sessions for user2
		for i := 0; i < 2; i++ {
			token := "session2_" + uuid.New().String()
			_, err = tokenRepo.Create(ctx, token, user2.ID, expiresAt, "10.0.0."+string(rune('1'+i)), "Agent2")
			require.NoError(t, err)
		}

		// Get all sessions (should have at least 5)
		sessions, err := tokenRepo.GetAllActiveSessions(ctx, 100, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(sessions), 5)

		// Verify all sessions are active
		for _, session := range sessions {
			assert.True(t, session.IsActive())
			assert.Nil(t, session.RevokedAt)
			assert.False(t, session.IsExpired())
		}
	})

	t.Run("pagination works correctly", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "Paginate", "Sessions", "pass")
		require.NoError(t, err)

		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		// Create 5 sessions
		for i := 0; i < 5; i++ {
			token := "page_session_" + uuid.New().String()
			_, err = tokenRepo.Create(ctx, token, user.ID, expiresAt, "1.1.1.1", "Agent")
			require.NoError(t, err)
		}

		// Get first 2
		page1, err := tokenRepo.GetAllActiveSessions(ctx, 2, 0)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(page1), 2)

		// Get next 2
		page2, err := tokenRepo.GetAllActiveSessions(ctx, 2, 2)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(page2), 2)
	})

	t.Run("excludes revoked sessions", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "Revoked", "Sessions", "pass")
		require.NoError(t, err)

		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		// Create active session
		activeToken := "active_session_" + uuid.New().String()
		_, err = tokenRepo.Create(ctx, activeToken, user.ID, expiresAt, "1.1.1.1", "Active")
		require.NoError(t, err)

		// Create revoked session
		revokedToken := "revoked_session_" + uuid.New().String()
		_, err = tokenRepo.Create(ctx, revokedToken, user.ID, expiresAt, "2.2.2.2", "Revoked")
		require.NoError(t, err)
		err = tokenRepo.Revoke(ctx, revokedToken)
		require.NoError(t, err)

		// Get all sessions - should not include revoked
		sessions, err := tokenRepo.GetAllActiveSessions(ctx, 100, 0)
		require.NoError(t, err)

		for _, session := range sessions {
			assert.NotEqual(t, revokedToken, session.Token)
		}
	})

	t.Run("excludes expired sessions", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "Expired", "Sessions", "pass")
		require.NoError(t, err)

		// Create expired session
		expiredToken := "expired_session_" + uuid.New().String()
		expiredTime := time.Now().Add(-1 * time.Hour)
		_, err = tokenRepo.Create(ctx, expiredToken, user.ID, expiredTime, "1.1.1.1", "Expired")
		require.NoError(t, err)

		// Get all sessions - should not include expired
		sessions, err := tokenRepo.GetAllActiveSessions(ctx, 100, 0)
		require.NoError(t, err)

		for _, session := range sessions {
			assert.NotEqual(t, expiredToken, session.Token)
			assert.False(t, session.IsExpired())
		}
	})
}

// TestRefreshTokenRepository_CountAllActiveSessions tests counting all active sessions.
func TestRefreshTokenRepository_CountAllActiveSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := repository.NewUserRepository(pool, getRefreshTokenTestLogger())
	tokenRepo := repository.NewRefreshTokenRepository(pool, getRefreshTokenTestLogger())
	ctx := context.Background()

	t.Run("count all active sessions across users", func(t *testing.T) {
		initialCount, err := tokenRepo.CountAllActiveSessions(ctx)
		require.NoError(t, err)

		// Create multiple users with sessions
		user1, err := userRepo.Create(ctx, generateTestEmail(), "Count", "User1", "pass")
		require.NoError(t, err)
		user2, err := userRepo.Create(ctx, generateTestEmail(), "Count", "User2", "pass")
		require.NoError(t, err)

		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		// Create 3 sessions for user1
		for i := 0; i < 3; i++ {
			token := "count1_" + uuid.New().String()
			_, err = tokenRepo.Create(ctx, token, user1.ID, expiresAt, "1.1.1.1", "Agent")
			require.NoError(t, err)
		}

		// Create 2 sessions for user2
		for i := 0; i < 2; i++ {
			token := "count2_" + uuid.New().String()
			_, err = tokenRepo.Create(ctx, token, user2.ID, expiresAt, "2.2.2.2", "Agent")
			require.NoError(t, err)
		}

		// Count should increase by 5
		newCount, err := tokenRepo.CountAllActiveSessions(ctx)
		require.NoError(t, err)
		assert.Equal(t, initialCount+5, newCount)
	})

	t.Run("count excludes revoked sessions", func(t *testing.T) {
		beforeCount, err := tokenRepo.CountAllActiveSessions(ctx)
		require.NoError(t, err)

		user, err := userRepo.Create(ctx, generateTestEmail(), "Revoke", "Count", "pass")
		require.NoError(t, err)

		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		// Create session
		token := "count_revoke_" + uuid.New().String()
		_, err = tokenRepo.Create(ctx, token, user.ID, expiresAt, "1.1.1.1", "Agent")
		require.NoError(t, err)

		afterCreate, err := tokenRepo.CountAllActiveSessions(ctx)
		require.NoError(t, err)
		assert.Equal(t, beforeCount+1, afterCreate)

		// Revoke session
		err = tokenRepo.Revoke(ctx, token)
		require.NoError(t, err)

		afterRevoke, err := tokenRepo.CountAllActiveSessions(ctx)
		require.NoError(t, err)
		assert.Equal(t, beforeCount, afterRevoke)
	})

	t.Run("count excludes expired sessions", func(t *testing.T) {
		beforeCount, err := tokenRepo.CountAllActiveSessions(ctx)
		require.NoError(t, err)

		user, err := userRepo.Create(ctx, generateTestEmail(), "Expire", "Count", "pass")
		require.NoError(t, err)

		// Create expired session
		token := "count_expire_" + uuid.New().String()
		expiredTime := time.Now().Add(-1 * time.Hour)
		_, err = tokenRepo.Create(ctx, token, user.ID, expiredTime, "1.1.1.1", "Agent")
		require.NoError(t, err)

		// Count should not change
		afterCreate, err := tokenRepo.CountAllActiveSessions(ctx)
		require.NoError(t, err)
		assert.Equal(t, beforeCount, afterCreate)
	})
}

// TestRefreshTokenRepository_RevokeToken tests revoking tokens by ID for admin operations.
func TestRefreshTokenRepository_RevokeToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := repository.NewUserRepository(pool, getRefreshTokenTestLogger())
	tokenRepo := repository.NewRefreshTokenRepository(pool, getRefreshTokenTestLogger())
	ctx := context.Background()

	t.Run("revoke token by token string successfully", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "Admin", "Revoke", "pass")
		require.NoError(t, err)

		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		tokenString := "admin_revoke_" + uuid.New().String()

		created, err := tokenRepo.Create(ctx, tokenString, user.ID, expiresAt, "1.1.1.1", "Agent")
		require.NoError(t, err)
		assert.Nil(t, created.RevokedAt)

		// Admin revokes by token string
		err = tokenRepo.RevokeToken(ctx, tokenString)
		require.NoError(t, err)

		// Verify token is revoked
		token, err := tokenRepo.GetByToken(ctx, tokenString)
		require.NoError(t, err)
		assert.NotNil(t, token.RevokedAt)
		assert.True(t, token.IsRevoked())
		assert.False(t, token.IsActive())
	})

	t.Run("revoke non-existent token returns error", func(t *testing.T) {
		err := tokenRepo.RevokeToken(ctx, "nonexistent_admin_token")
		assert.ErrorIs(t, err, domain.ErrTokenNotFound)
	})

	t.Run("revoke already revoked token is idempotent", func(t *testing.T) {
		user, err := userRepo.Create(ctx, generateTestEmail(), "Double", "Revoke", "pass")
		require.NoError(t, err)

		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		tokenString := "double_revoke_" + uuid.New().String()

		_, err = tokenRepo.Create(ctx, tokenString, user.ID, expiresAt, "1.1.1.1", "Agent")
		require.NoError(t, err)

		// First revocation
		err = tokenRepo.RevokeToken(ctx, tokenString)
		require.NoError(t, err)

		// Second revocation should succeed (idempotent)
		err = tokenRepo.RevokeToken(ctx, tokenString)
		assert.ErrorIs(t, err, domain.ErrTokenNotFound)
	})
}
