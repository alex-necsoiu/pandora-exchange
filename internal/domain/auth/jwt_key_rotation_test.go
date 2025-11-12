package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJWTManagerWithKeyManager tests JWT manager with KeyManager integration.
func TestJWTManagerWithKeyManager(t *testing.T) {
	t.Run("creates JWT manager with InMemoryKeyManager", func(t *testing.T) {
		keyManager, err := auth.NewInMemoryKeyManager("HS256")
		require.NoError(t, err)

		manager, err := auth.NewJWTManagerWithKeyManager(keyManager, 15*time.Minute, 7*24*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, manager)
	})

	t.Run("fails with nil key manager", func(t *testing.T) {
		_, err := auth.NewJWTManagerWithKeyManager(nil, 15*time.Minute, 7*24*time.Hour)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key manager cannot be nil")
	})
}

// TestJWTTokenWithKID tests that generated tokens include kid header.
func TestJWTTokenWithKID(t *testing.T) {
	keyManager, err := auth.NewInMemoryKeyManager("HS256")
	require.NoError(t, err)

	manager, err := auth.NewJWTManagerWithKeyManager(keyManager, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	userID := uuid.New()
	email := "test@example.com"

	t.Run("access token includes kid in header", func(t *testing.T) {
		token, err := manager.GenerateAccessToken(userID, email, "user")
		require.NoError(t, err)

		// Parse token to extract kid from header
		parsed, _, err := new(jwt.Parser).ParseUnverified(token, jwt.MapClaims{})
		require.NoError(t, err)

		kid, exists := parsed.Header["kid"]
		assert.True(t, exists, "kid header should exist")
		assert.NotEmpty(t, kid, "kid should not be empty")
		assert.Equal(t, "v1", kid, "first key should be v1")
	})

	t.Run("refresh token includes kid in header", func(t *testing.T) {
		token, err := manager.GenerateRefreshToken(userID)
		require.NoError(t, err)

		// Parse token to extract kid from header
		parsed, _, err := new(jwt.Parser).ParseUnverified(token, jwt.MapClaims{})
		require.NoError(t, err)

		kid, exists := parsed.Header["kid"]
		assert.True(t, exists, "kid header should exist")
		assert.NotEmpty(t, kid, "kid should not be empty")
	})
}

// TestKeyRotation tests key rotation functionality.
func TestKeyRotation(t *testing.T) {
	keyManager, err := auth.NewInMemoryKeyManager("HS256")
	require.NoError(t, err)

	manager, err := auth.NewJWTManagerWithKeyManager(keyManager, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	userID := uuid.New()
	email := "test@example.com"

	// Generate token with original key
	token1, err := manager.GenerateAccessToken(userID, email, "user")
	require.NoError(t, err)

	t.Run("tokens generated with old key can still be validated", func(t *testing.T) {
		claims, err := manager.ValidateAccessToken(token1)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
	})

	// Rotate key
	newKeyID, _, err := keyManager.RotateKey(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "v2", newKeyID)

	// Generate token with new key
	token2, err := manager.GenerateAccessToken(userID, email, "user")
	require.NoError(t, err)

	t.Run("new tokens use new key ID", func(t *testing.T) {
		parsed, _, err := new(jwt.Parser).ParseUnverified(token2, jwt.MapClaims{})
		require.NoError(t, err)

		kid := parsed.Header["kid"]
		assert.Equal(t, "v2", kid, "new tokens should use v2 key")
	})

	t.Run("old tokens still validate after rotation (grace period)", func(t *testing.T) {
		claims, err := manager.ValidateAccessToken(token1)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
	})

	t.Run("new tokens validate correctly", func(t *testing.T) {
		claims, err := manager.ValidateAccessToken(token2)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
	})
}

// TestBackwardCompatibility tests that tokens without kid still work.
func TestBackwardCompatibility(t *testing.T) {
	// Create manager with static key
	staticKey := "test-secret-key-min-32-characters-long-for-hs256"
	manager, err := auth.NewJWTManager(staticKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	userID := uuid.New()
	email := "test@example.com"

	// Generate token (will use StaticKeyManager internally)
	token, err := manager.GenerateAccessToken(userID, email, "user")
	require.NoError(t, err)

	t.Run("tokens from static key manager validate correctly", func(t *testing.T) {
		claims, err := manager.ValidateAccessToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, "user", claims.Role)
	})

	t.Run("tokens include static kid", func(t *testing.T) {
		parsed, _, err := new(jwt.Parser).ParseUnverified(token, jwt.MapClaims{})
		require.NoError(t, err)

		kid := parsed.Header["kid"]
		assert.Equal(t, "static-v1", kid)
	})
}

// TestMultipleKeyValidation tests that tokens signed with different keys can coexist.
func TestMultipleKeyValidation(t *testing.T) {
	keyManager, err := auth.NewInMemoryKeyManager("HS256")
	require.NoError(t, err)

	manager, err := auth.NewJWTManagerWithKeyManager(keyManager, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	// Generate tokens with different keys
	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()

	token1, err := manager.GenerateAccessToken(user1, "user1@example.com", "user")
	require.NoError(t, err)

	// Rotate to v2
	_, _, err = keyManager.RotateKey(context.Background())
	require.NoError(t, err)

	token2, err := manager.GenerateAccessToken(user2, "user2@example.com", "user")
	require.NoError(t, err)

	// Rotate to v3
	_, _, err = keyManager.RotateKey(context.Background())
	require.NoError(t, err)

	token3, err := manager.GenerateAccessToken(user3, "user3@example.com", "admin")
	require.NoError(t, err)

	t.Run("all tokens from active keys validate", func(t *testing.T) {
		// Token 1 (v1 - in grace period)
		claims1, err := manager.ValidateAccessToken(token1)
		require.NoError(t, err)
		assert.Equal(t, user1, claims1.UserID)

		// Token 2 (v2 - in grace period)
		claims2, err := manager.ValidateAccessToken(token2)
		require.NoError(t, err)
		assert.Equal(t, user2, claims2.UserID)

		// Token 3 (v3 - current active)
		claims3, err := manager.ValidateAccessToken(token3)
		require.NoError(t, err)
		assert.Equal(t, user3, claims3.UserID)
		assert.Equal(t, "admin", claims3.Role)
	})
}

// TestRevokedKeyRejection tests that tokens signed with revoked keys are rejected.
func TestRevokedKeyRejection(t *testing.T) {
	keyManager, err := auth.NewInMemoryKeyManager("HS256")
	require.NoError(t, err)

	manager, err := auth.NewJWTManagerWithKeyManager(keyManager, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	userID := uuid.New()

	// Generate token with v1
	token1, err := manager.GenerateAccessToken(userID, "test@example.com", "user")
	require.NoError(t, err)

	// Rotate to v2 (v1 moves to grace period)
	_, _, err = keyManager.RotateKey(context.Background())
	require.NoError(t, err)

	// Token still validates (in grace period)
	_, err = manager.ValidateAccessToken(token1)
	require.NoError(t, err)

	// Revoke v1
	err = keyManager.RevokeKey(context.Background(), "v1")
	require.NoError(t, err)

	t.Run("token signed with revoked key fails validation", func(t *testing.T) {
		_, err = manager.ValidateAccessToken(token1)
		assert.Error(t, err)
		// Token will fail validation because the key is revoked
		// The error message may vary depending on whether we check revoked status before or after signature validation
	})
}

// TestStaticKeyManager tests the StaticKeyManager implementation.
func TestStaticKeyManager(t *testing.T) {
	signingKey := []byte("test-secret-key-min-32-characters-long-for-hs256")

	t.Run("creates static key manager successfully", func(t *testing.T) {
		km, err := auth.NewStaticKeyManager(signingKey, "HS256")
		require.NoError(t, err)
		assert.NotNil(t, km)
	})

	t.Run("fails with empty key", func(t *testing.T) {
		_, err := auth.NewStaticKeyManager([]byte{}, "HS256")
		assert.Error(t, err)
	})

	t.Run("fails with short key", func(t *testing.T) {
		_, err := auth.NewStaticKeyManager([]byte("short"), "HS256")
		assert.Error(t, err)
	})

	t.Run("retrieves signing key", func(t *testing.T) {
		km, err := auth.NewStaticKeyManager(signingKey, "HS256")
		require.NoError(t, err)

		key, err := km.GetSigningKey(context.Background(), auth.StaticKeyID)
		require.NoError(t, err)
		assert.Equal(t, signingKey, key)
	})

	t.Run("retrieves key with empty key ID (backward compat)", func(t *testing.T) {
		km, err := auth.NewStaticKeyManager(signingKey, "HS256")
		require.NoError(t, err)

		key, err := km.GetSigningKey(context.Background(), "")
		require.NoError(t, err)
		assert.Equal(t, signingKey, key)
	})

	t.Run("rotation is not supported", func(t *testing.T) {
		km, err := auth.NewStaticKeyManager(signingKey, "HS256")
		require.NoError(t, err)

		_, _, err = km.RotateKey(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rotation not supported")
	})

	t.Run("revocation is not supported", func(t *testing.T) {
		km, err := auth.NewStaticKeyManager(signingKey, "HS256")
		require.NoError(t, err)

		err = km.RevokeKey(context.Background(), auth.StaticKeyID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "revocation not supported")
	})
}
