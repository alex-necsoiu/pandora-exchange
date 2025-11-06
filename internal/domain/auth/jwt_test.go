package auth_test

import (
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSigningKey = "test-secret-key-min-32-characters-long-for-hs256"

// TestNewJWTManager tests JWT manager creation.
func TestNewJWTManager(t *testing.T) {
	t.Run("create JWT manager successfully", func(t *testing.T) {
		manager, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 7*24*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, manager)
	})

	t.Run("create with empty signing key fails", func(t *testing.T) {
		_, err := auth.NewJWTManager("", 15*time.Minute, 7*24*time.Hour)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signing key cannot be empty")
	})

	t.Run("create with short signing key fails", func(t *testing.T) {
		_, err := auth.NewJWTManager("short", 15*time.Minute, 7*24*time.Hour)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signing key too short")
	})

	t.Run("create with zero access token duration fails", func(t *testing.T) {
		_, err := auth.NewJWTManager(testSigningKey, 0, 7*24*time.Hour)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access token duration must be positive")
	})

	t.Run("create with zero refresh token duration fails", func(t *testing.T) {
		_, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "refresh token duration must be positive")
	})
}

// TestGenerateAccessToken tests access token generation.
func TestGenerateAccessToken(t *testing.T) {
	manager, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("generate access token successfully", func(t *testing.T) {
		userID := uuid.New()
		email := "test@example.com"

		token, err := manager.GenerateAccessToken(userID, email)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Token should have 3 parts (header.payload.signature)
		parts := len(token)
		assert.Greater(t, parts, 100, "token should be reasonably long")
	})

	t.Run("generate different tokens for different users", func(t *testing.T) {
		userID1 := uuid.New()
		userID2 := uuid.New()

		token1, err := manager.GenerateAccessToken(userID1, "user1@example.com")
		require.NoError(t, err)

		token2, err := manager.GenerateAccessToken(userID2, "user2@example.com")
		require.NoError(t, err)

		assert.NotEqual(t, token1, token2)
	})

	t.Run("generate with nil user ID fails", func(t *testing.T) {
		_, err := manager.GenerateAccessToken(uuid.Nil, "test@example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user ID cannot be nil")
	})

	t.Run("generate with empty email fails", func(t *testing.T) {
		userID := uuid.New()
		_, err := manager.GenerateAccessToken(userID, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email cannot be empty")
	})
}

// TestGenerateRefreshToken tests refresh token generation.
func TestGenerateRefreshToken(t *testing.T) {
	manager, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("generate refresh token successfully", func(t *testing.T) {
		userID := uuid.New()

		token, err := manager.GenerateRefreshToken(userID)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Greater(t, len(token), 50, "refresh token should be long enough")
	})

	t.Run("generate different tokens for same user", func(t *testing.T) {
		userID := uuid.New()

		token1, err := manager.GenerateRefreshToken(userID)
		require.NoError(t, err)

		token2, err := manager.GenerateRefreshToken(userID)
		require.NoError(t, err)

		// Different tokens even for same user (different jti)
		assert.NotEqual(t, token1, token2)
	})

	t.Run("generate with nil user ID fails", func(t *testing.T) {
		_, err := manager.GenerateRefreshToken(uuid.Nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user ID cannot be nil")
	})
}

// TestValidateAccessToken tests access token validation.
func TestValidateAccessToken(t *testing.T) {
	manager, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("validate correct access token", func(t *testing.T) {
		userID := uuid.New()
		email := "valid@example.com"

		token, err := manager.GenerateAccessToken(userID, email)
		require.NoError(t, err)

		claims, err := manager.ValidateAccessToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, "access", claims.TokenType)
	})

	t.Run("validate with empty token fails", func(t *testing.T) {
		_, err := manager.ValidateAccessToken("")
		assert.Error(t, err)
	})

	t.Run("validate with malformed token fails", func(t *testing.T) {
		_, err := manager.ValidateAccessToken("not.a.valid.token")
		assert.Error(t, err)
	})

	t.Run("validate with wrong signature fails", func(t *testing.T) {
		// Create token with different key
		otherManager, err := auth.NewJWTManager("different-secret-key-min-32-chars", 15*time.Minute, 7*24*time.Hour)
		require.NoError(t, err)

		userID := uuid.New()
		token, err := otherManager.GenerateAccessToken(userID, "test@example.com")
		require.NoError(t, err)

		// Validate with original manager
		_, err = manager.ValidateAccessToken(token)
		assert.Error(t, err)
	})

	t.Run("validate expired token fails", func(t *testing.T) {
		// Create manager with very short expiration
		shortManager, err := auth.NewJWTManager(testSigningKey, 1*time.Millisecond, 7*24*time.Hour)
		require.NoError(t, err)

		userID := uuid.New()
		token, err := shortManager.GenerateAccessToken(userID, "test@example.com")
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		_, err = shortManager.ValidateAccessToken(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("validate refresh token as access token fails", func(t *testing.T) {
		userID := uuid.New()
		refreshToken, err := manager.GenerateRefreshToken(userID)
		require.NoError(t, err)

		_, err = manager.ValidateAccessToken(refreshToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid token type")
	})
}

// TestValidateRefreshToken tests refresh token validation.
func TestValidateRefreshToken(t *testing.T) {
	manager, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("validate correct refresh token", func(t *testing.T) {
		userID := uuid.New()

		token, err := manager.GenerateRefreshToken(userID)
		require.NoError(t, err)

		claims, err := manager.ValidateRefreshToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, "refresh", claims.TokenType)
		assert.NotEmpty(t, claims.TokenID)
	})

	t.Run("validate with empty token fails", func(t *testing.T) {
		_, err := manager.ValidateRefreshToken("")
		assert.Error(t, err)
	})

	t.Run("validate with wrong signature fails", func(t *testing.T) {
		otherManager, err := auth.NewJWTManager("another-secret-key-min-32-chars-", 15*time.Minute, 7*24*time.Hour)
		require.NoError(t, err)

		userID := uuid.New()
		token, err := otherManager.GenerateRefreshToken(userID)
		require.NoError(t, err)

		_, err = manager.ValidateRefreshToken(token)
		assert.Error(t, err)
	})

	t.Run("validate expired refresh token fails", func(t *testing.T) {
		shortManager, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 1*time.Millisecond)
		require.NoError(t, err)

		userID := uuid.New()
		token, err := shortManager.GenerateRefreshToken(userID)
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		_, err = shortManager.ValidateRefreshToken(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("validate access token as refresh token fails", func(t *testing.T) {
		userID := uuid.New()
		accessToken, err := manager.GenerateAccessToken(userID, "test@example.com")
		require.NoError(t, err)

		_, err = manager.ValidateRefreshToken(accessToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid token type")
	})
}

// TestTokenClaims tests token claims structure.
func TestTokenClaims(t *testing.T) {
	manager, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("access token has correct claims", func(t *testing.T) {
		userID := uuid.New()
		email := "claims@example.com"

		token, err := manager.GenerateAccessToken(userID, email)
		require.NoError(t, err)

		claims, err := manager.ValidateAccessToken(token)
		require.NoError(t, err)

		// Check standard claims
		assert.NotZero(t, claims.ExpiresAt)
		assert.NotZero(t, claims.IssuedAt)
		assert.Equal(t, "pandora-exchange", claims.Issuer)

		// Check custom claims
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, "access", claims.TokenType)

		// Access token should have a JTI
		assert.NotEmpty(t, claims.TokenID)
	})

	t.Run("refresh token has correct claims", func(t *testing.T) {
		userID := uuid.New()

		token, err := manager.GenerateRefreshToken(userID)
		require.NoError(t, err)

		claims, err := manager.ValidateRefreshToken(token)
		require.NoError(t, err)

		assert.NotZero(t, claims.ExpiresAt)
		assert.NotZero(t, claims.IssuedAt)
		assert.Equal(t, "pandora-exchange", claims.Issuer)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, "refresh", claims.TokenType)
		assert.NotEmpty(t, claims.TokenID)
	})

	t.Run("token expiration is correctly set", func(t *testing.T) {
		accessDuration := 15 * time.Minute
		manager, err := auth.NewJWTManager(testSigningKey, accessDuration, 7*24*time.Hour)
		require.NoError(t, err)

		userID := uuid.New()
		token, err := manager.GenerateAccessToken(userID, "test@example.com")
		require.NoError(t, err)

		claims, err := manager.ValidateAccessToken(token)
		require.NoError(t, err)

		expectedExpiry := time.Now().Add(accessDuration)
		actualExpiry := claims.ExpiresAt.Time

		// Allow 5 second tolerance for test execution time
		diff := actualExpiry.Sub(expectedExpiry)
		assert.Less(t, diff, 5*time.Second)
		assert.Greater(t, diff, -5*time.Second)
	})
}

// TestGetTokenExpiration tests extracting expiration from token.
func TestGetTokenExpiration(t *testing.T) {
	manager, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("get expiration for refresh token", func(t *testing.T) {
		userID := uuid.New()
		token, err := manager.GenerateRefreshToken(userID)
		require.NoError(t, err)

		expiration, err := manager.GetTokenExpiration(token)
		require.NoError(t, err)

		expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
		diff := expiration.Sub(expectedExpiry)

		// Allow 5 second tolerance
		assert.Less(t, diff, 5*time.Second)
		assert.Greater(t, diff, -5*time.Second)
	})

	t.Run("get expiration fails for invalid token", func(t *testing.T) {
		_, err := manager.GetTokenExpiration("invalid.token.here")
		assert.Error(t, err)
	})
}

// TestVaultPlaceholder tests Vault integration placeholder.
func TestVaultPlaceholder(t *testing.T) {
	t.Run("JWT manager supports future Vault integration", func(t *testing.T) {
		// This test documents that the current implementation uses
		// a static signing key, but is designed to support Vault integration
		// in the future (Task #22: Vault Integration)

		manager, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 7*24*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, manager)

		// TODO: In Task #22, replace static key with Vault-sourced key
		// The JWTManager interface should remain the same
		t.Log("Current: Static signing key")
		t.Log("Future: Vault-sourced signing key with rotation support")
	})
}
