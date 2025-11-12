package auth

import (
	"context"
	"errors"
	"time"
)

const (
	// StaticKeyID is the key ID used for static keys (backward compatibility).
	StaticKeyID = "static-v1"
)

// StaticKeyManager implements KeyManager with a single static key.
// This is used for backward compatibility during migration from static keys
// to dynamic key rotation. It allows existing JWTManager code to work with
// the KeyManager interface without requiring immediate key rotation infrastructure.
//
// For production use with key rotation, use InMemoryKeyManager or VaultKeyManager.
type StaticKeyManager struct {
	key       []byte
	algorithm string
	metadata  *KeyMetadata
}

// NewStaticKeyManager creates a new static key manager from a signing key.
// The key is permanently active and cannot be rotated.
//
// Parameters:
//   - signingKey: The static signing key bytes (minimum 32 bytes for HS256)
//   - algorithm: The signing algorithm (e.g., "HS256")
//
// Returns:
//   - *StaticKeyManager: A new static key manager instance
//   - error: ErrSigningKeyTooShort if key is too short, ErrSigningKeyEmpty if empty
func NewStaticKeyManager(signingKey []byte, algorithm string) (*StaticKeyManager, error) {
	if len(signingKey) == 0 {
		return nil, ErrSigningKeyEmpty
	}

	if len(signingKey) < MinSigningKeyLength {
		return nil, ErrSigningKeyTooShort
	}

	if algorithm == "" {
		algorithm = "HS256"
	}

	now := time.Now()
	metadata := &KeyMetadata{
		KeyID:     StaticKeyID,
		CreatedAt: now,
		Status:    KeyStatusActive,
		Algorithm: algorithm,
		Version:   1,
	}

	return &StaticKeyManager{
		key:       signingKey,
		algorithm: algorithm,
		metadata:  metadata,
	}, nil
}

// GetSigningKey retrieves the static signing key.
// Always returns the same key for StaticKeyID.
func (sm *StaticKeyManager) GetSigningKey(ctx context.Context, keyID string) ([]byte, error) {
	// Support both explicit static key ID and empty key ID (for backward compatibility)
	if keyID != "" && keyID != StaticKeyID {
		return nil, ErrKeyNotFound
	}

	return sm.key, nil
}

// GetCurrentKeyID returns the static key ID.
func (sm *StaticKeyManager) GetCurrentKeyID(ctx context.Context) (string, error) {
	return StaticKeyID, nil
}

// ListActiveKeyIDs returns only the static key ID.
func (sm *StaticKeyManager) ListActiveKeyIDs(ctx context.Context) ([]string, error) {
	return []string{StaticKeyID}, nil
}

// RotateKey is not supported for static keys.
// Returns an error indicating rotation is not available.
func (sm *StaticKeyManager) RotateKey(ctx context.Context) (string, []byte, error) {
	return "", nil, ErrKeyRotationNotSupported
}

// GetKeyMetadata returns metadata for the static key.
func (sm *StaticKeyManager) GetKeyMetadata(ctx context.Context, keyID string) (*KeyMetadata, error) {
	if keyID != "" && keyID != StaticKeyID {
		return nil, ErrKeyNotFound
	}

	// Return a copy
	metaCopy := *sm.metadata
	return &metaCopy, nil
}

// RevokeKey is not supported for static keys.
// The static key cannot be revoked.
func (sm *StaticKeyManager) RevokeKey(ctx context.Context, keyID string) error {
	return ErrKeyRevocationNotSupported
}

var (
	// ErrKeyRotationNotSupported indicates key rotation is not available for static keys.
	ErrKeyRotationNotSupported = errors.New("key rotation not supported for static key manager")

	// ErrKeyRevocationNotSupported indicates key revocation is not available for static keys.
	ErrKeyRevocationNotSupported = errors.New("key revocation not supported for static key manager")
)
