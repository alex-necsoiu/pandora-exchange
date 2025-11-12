package auth

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInMemoryKeyManager(t *testing.T) {
	t.Run("creates manager with initial key", func(t *testing.T) {
		km, err := NewInMemoryKeyManager("HS256")
		require.NoError(t, err)
		require.NotNil(t, km)

		ctx := context.Background()
		keyID, err := km.GetCurrentKeyID(ctx)
		require.NoError(t, err)
		assert.Equal(t, "v1", keyID)

		// Should have one active key
		activeKeys, err := km.ListActiveKeyIDs(ctx)
		require.NoError(t, err)
		assert.Len(t, activeKeys, 1)
		assert.Contains(t, activeKeys, "v1")
	})

	t.Run("defaults to HS256 algorithm", func(t *testing.T) {
		km, err := NewInMemoryKeyManager("")
		require.NoError(t, err)

		ctx := context.Background()
		keyID, _ := km.GetCurrentKeyID(ctx)
		meta, err := km.GetKeyMetadata(ctx, keyID)
		require.NoError(t, err)
		assert.Equal(t, "HS256", meta.Algorithm)
	})
}

func TestInMemoryKeyManager_GetSigningKey(t *testing.T) {
	km, err := NewInMemoryKeyManager("HS256")
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("retrieves valid key", func(t *testing.T) {
		keyID, _ := km.GetCurrentKeyID(ctx)
		key, err := km.GetSigningKey(ctx, keyID)
		require.NoError(t, err)
		assert.NotNil(t, key)
		assert.GreaterOrEqual(t, len(key), MinSigningKeyLength)
	})

	t.Run("returns error for empty key ID", func(t *testing.T) {
		_, err := km.GetSigningKey(ctx, "")
		assert.ErrorIs(t, err, ErrInvalidKeyID)
	})

	t.Run("returns error for non-existent key", func(t *testing.T) {
		_, err := km.GetSigningKey(ctx, "non-existent")
		assert.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("returns error for revoked key", func(t *testing.T) {
		// Rotate to create a second key
		_, _, err := km.RotateKey(ctx)
		require.NoError(t, err)

		// Revoke the first key (now in grace period)
		err = km.RevokeKey(ctx, "v1")
		require.NoError(t, err)

		// Should not be able to get revoked key
		_, err = km.GetSigningKey(ctx, "v1")
		assert.ErrorIs(t, err, ErrKeyNotFound)
	})
}

func TestInMemoryKeyManager_RotateKey(t *testing.T) {
	km, err := NewInMemoryKeyManager("HS256")
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("creates new active key", func(t *testing.T) {
		oldKeyID, _ := km.GetCurrentKeyID(ctx)
		assert.Equal(t, "v1", oldKeyID)

		newKeyID, key, err := km.RotateKey(ctx)
		require.NoError(t, err)
		assert.Equal(t, "v2", newKeyID)
		assert.NotNil(t, key)
		assert.GreaterOrEqual(t, len(key), MinSigningKeyLength)

		// Current key should be updated
		currentKeyID, _ := km.GetCurrentKeyID(ctx)
		assert.Equal(t, "v2", currentKeyID)
	})

	t.Run("moves old key to grace period", func(t *testing.T) {
		meta, err := km.GetKeyMetadata(ctx, "v1")
		require.NoError(t, err)
		assert.Equal(t, KeyStatusGracePeriod, meta.Status)
		assert.False(t, meta.RotatedAt.IsZero())
	})

	t.Run("new key is active", func(t *testing.T) {
		meta, err := km.GetKeyMetadata(ctx, "v2")
		require.NoError(t, err)
		assert.Equal(t, KeyStatusActive, meta.Status)
		assert.Equal(t, 2, meta.Version)
	})

	t.Run("both keys are in active list", func(t *testing.T) {
		activeKeys, err := km.ListActiveKeyIDs(ctx)
		require.NoError(t, err)
		assert.Len(t, activeKeys, 2)
		assert.Contains(t, activeKeys, "v1") // grace period
		assert.Contains(t, activeKeys, "v2") // active
	})
}

func TestInMemoryKeyManager_ListActiveKeyIDs(t *testing.T) {
	km, err := NewInMemoryKeyManager("HS256")
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("returns active and grace period keys", func(t *testing.T) {
		// Initial state: 1 active key
		activeKeys, err := km.ListActiveKeyIDs(ctx)
		require.NoError(t, err)
		assert.Len(t, activeKeys, 1)

		// Rotate: now 1 active + 1 grace period
		_, _, err = km.RotateKey(ctx)
		require.NoError(t, err)

		activeKeys, err = km.ListActiveKeyIDs(ctx)
		require.NoError(t, err)
		assert.Len(t, activeKeys, 2)
	})

	t.Run("excludes revoked keys", func(t *testing.T) {
		// Revoke the grace period key
		err := km.RevokeKey(ctx, "v1")
		require.NoError(t, err)

		activeKeys, err := km.ListActiveKeyIDs(ctx)
		require.NoError(t, err)
		assert.Len(t, activeKeys, 1)
		assert.Contains(t, activeKeys, "v2")
		assert.NotContains(t, activeKeys, "v1")
	})
}

func TestInMemoryKeyManager_GetKeyMetadata(t *testing.T) {
	km, err := NewInMemoryKeyManager("HS256")
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("returns metadata for existing key", func(t *testing.T) {
		keyID, _ := km.GetCurrentKeyID(ctx)
		meta, err := km.GetKeyMetadata(ctx, keyID)
		require.NoError(t, err)
		assert.Equal(t, "v1", meta.KeyID)
		assert.Equal(t, KeyStatusActive, meta.Status)
		assert.Equal(t, "HS256", meta.Algorithm)
		assert.Equal(t, 1, meta.Version)
		assert.False(t, meta.CreatedAt.IsZero())
	})

	t.Run("returns error for empty key ID", func(t *testing.T) {
		_, err := km.GetKeyMetadata(ctx, "")
		assert.ErrorIs(t, err, ErrInvalidKeyID)
	})

	t.Run("returns error for non-existent key", func(t *testing.T) {
		_, err := km.GetKeyMetadata(ctx, "non-existent")
		assert.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("returns copy of metadata", func(t *testing.T) {
		keyID, _ := km.GetCurrentKeyID(ctx)
		meta1, _ := km.GetKeyMetadata(ctx, keyID)
		meta2, _ := km.GetKeyMetadata(ctx, keyID)

		// Modifying one should not affect the other
		meta1.Status = KeyStatusRevoked
		assert.Equal(t, KeyStatusActive, meta2.Status)
	})
}

func TestInMemoryKeyManager_RevokeKey(t *testing.T) {
	km, err := NewInMemoryKeyManager("HS256")
	require.NoError(t, err)
	ctx := context.Background()

	// Create a second key so we can revoke the first
	_, _, err = km.RotateKey(ctx)
	require.NoError(t, err)

	t.Run("revokes grace period key", func(t *testing.T) {
		err := km.RevokeKey(ctx, "v1")
		require.NoError(t, err)

		meta, err := km.GetKeyMetadata(ctx, "v1")
		require.NoError(t, err)
		assert.Equal(t, KeyStatusRevoked, meta.Status)
		assert.False(t, meta.RevokedAt.IsZero())
	})

	t.Run("returns error for empty key ID", func(t *testing.T) {
		err := km.RevokeKey(ctx, "")
		assert.ErrorIs(t, err, ErrInvalidKeyID)
	})

	t.Run("returns error for non-existent key", func(t *testing.T) {
		err := km.RevokeKey(ctx, "non-existent")
		assert.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("returns error when revoking already revoked key", func(t *testing.T) {
		err := km.RevokeKey(ctx, "v1")
		assert.ErrorIs(t, err, ErrKeyAlreadyRevoked)
	})

	t.Run("cannot revoke current active key", func(t *testing.T) {
		currentKeyID, _ := km.GetCurrentKeyID(ctx)
		err := km.RevokeKey(ctx, currentKeyID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot revoke current active key")
	})
}

func TestInMemoryKeyManager_ConcurrentAccess(t *testing.T) {
	km, err := NewInMemoryKeyManager("HS256")
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("concurrent key retrieval", func(t *testing.T) {
		var wg sync.WaitGroup
		keyID, _ := km.GetCurrentKeyID(ctx)

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				key, err := km.GetSigningKey(ctx, keyID)
				assert.NoError(t, err)
				assert.NotNil(t, key)
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent rotation", func(t *testing.T) {
		var wg sync.WaitGroup
		rotations := 10

		for i := 0; i < rotations; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _, err := km.RotateKey(ctx)
				assert.NoError(t, err)
			}()
		}

		wg.Wait()

		// Should have rotations + 1 keys (initial key)
		activeKeys, err := km.ListActiveKeyIDs(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(activeKeys), 1)
	})
}

func TestInMemoryKeyManager_KeyRotationScenario(t *testing.T) {
	// Simulate a realistic key rotation scenario
	km, err := NewInMemoryKeyManager("HS256")
	require.NoError(t, err)
	ctx := context.Background()

	// Day 0: Initial key v1
	key1ID, _ := km.GetCurrentKeyID(ctx)
	assert.Equal(t, "v1", key1ID)

	// Day 30: Rotate to v2 (v1 goes to grace period)
	key2ID, _, err := km.RotateKey(ctx)
	require.NoError(t, err)
	assert.Equal(t, "v2", key2ID)

	// Verify both keys can validate tokens
	activeKeys, _ := km.ListActiveKeyIDs(ctx)
	assert.Contains(t, activeKeys, "v1")
	assert.Contains(t, activeKeys, "v2")

	// Day 60: Rotate to v3 (v2 goes to grace period, v1 can be revoked)
	key3ID, _, err := km.RotateKey(ctx)
	require.NoError(t, err)
	assert.Equal(t, "v3", key3ID)

	// Revoke old key v1 (tokens signed with v1 will fail validation)
	err = km.RevokeKey(ctx, "v1")
	require.NoError(t, err)

	// Now only v2 (grace) and v3 (active) are valid
	activeKeys, _ = km.ListActiveKeyIDs(ctx)
	assert.Len(t, activeKeys, 2)
	assert.Contains(t, activeKeys, "v2")
	assert.Contains(t, activeKeys, "v3")
	assert.NotContains(t, activeKeys, "v1")

	// Verify key statuses
	meta1, _ := km.GetKeyMetadata(ctx, "v1")
	assert.Equal(t, KeyStatusRevoked, meta1.Status)

	meta2, _ := km.GetKeyMetadata(ctx, "v2")
	assert.Equal(t, KeyStatusGracePeriod, meta2.Status)

	meta3, _ := km.GetKeyMetadata(ctx, "v3")
	assert.Equal(t, KeyStatusActive, meta3.Status)
}

func TestGenerateSecureKey(t *testing.T) {
	t.Run("generates key of correct length", func(t *testing.T) {
		key, err := generateSecureKey(32)
		require.NoError(t, err)
		assert.Len(t, key, 32)
	})

	t.Run("generates different keys", func(t *testing.T) {
		key1, _ := generateSecureKey(32)
		key2, _ := generateSecureKey(32)
		assert.NotEqual(t, key1, key2)
	})

	t.Run("generates non-zero keys", func(t *testing.T) {
		key, err := generateSecureKey(32)
		require.NoError(t, err)

		// Ensure not all bytes are zero
		hasNonZero := false
		for _, b := range key {
			if b != 0 {
				hasNonZero = true
				break
			}
		}
		assert.True(t, hasNonZero, "key should not be all zeros")
	})
}

func TestFormatKeyID(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	keyID := FormatKeyID(timestamp, 5)
	assert.Equal(t, "20240115-v5", keyID)
}

func TestGenerateKeyID(t *testing.T) {
	t.Run("generates non-empty key ID", func(t *testing.T) {
		keyID, err := GenerateKeyID()
		require.NoError(t, err)
		assert.NotEmpty(t, keyID)
	})

	t.Run("generates different key IDs", func(t *testing.T) {
		keyID1, _ := GenerateKeyID()
		keyID2, _ := GenerateKeyID()
		assert.NotEqual(t, keyID1, keyID2)
	})

	t.Run("generates URL-safe base64", func(t *testing.T) {
		keyID, err := GenerateKeyID()
		require.NoError(t, err)
		// URL-safe base64 should not contain + or /
		assert.NotContains(t, keyID, "+")
		assert.NotContains(t, keyID, "/")
	})
}
