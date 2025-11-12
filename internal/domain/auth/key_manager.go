package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	// ErrKeyNotFound indicates the requested key ID was not found.
	ErrKeyNotFound = errors.New("key not found")

	// ErrNoActiveKeys indicates there are no active keys available.
	ErrNoActiveKeys = errors.New("no active keys available")

	// ErrInvalidKeyID indicates the key ID is invalid or empty.
	ErrInvalidKeyID = errors.New("invalid key ID")

	// ErrKeyAlreadyRevoked indicates the key has already been revoked.
	ErrKeyAlreadyRevoked = errors.New("key already revoked")
)

// KeyStatus represents the lifecycle state of a signing key.
type KeyStatus string

const (
	// KeyStatusActive indicates the key is currently used for signing new tokens.
	KeyStatusActive KeyStatus = "active"

	// KeyStatusGracePeriod indicates the key can validate tokens but not sign new ones.
	KeyStatusGracePeriod KeyStatus = "grace_period"

	// KeyStatusRevoked indicates the key is revoked and cannot be used.
	KeyStatusRevoked KeyStatus = "revoked"
)

// KeyMetadata contains metadata about a signing key.
type KeyMetadata struct {
	KeyID     string    `json:"key_id"`
	CreatedAt time.Time `json:"created_at"`
	RotatedAt time.Time `json:"rotated_at,omitempty"` // When it was moved to grace period
	RevokedAt time.Time `json:"revoked_at,omitempty"` // When it was revoked
	Status    KeyStatus `json:"status"`
	Algorithm string    `json:"algorithm"` // e.g., "HS256", "RS256"
	Version   int       `json:"version"`   // Incrementing version number
}

// KeyManager manages JWT signing keys with support for rotation.
// Implementations can store keys in memory, Vault, or other secure storage.
type KeyManager interface {
	// GetSigningKey retrieves the signing key bytes for the given key ID.
	// Returns ErrKeyNotFound if the key doesn't exist or is revoked.
	GetSigningKey(ctx context.Context, keyID string) ([]byte, error)

	// GetCurrentKeyID returns the ID of the current active signing key.
	// This key should be used for signing new tokens.
	GetCurrentKeyID(ctx context.Context) (string, error)

	// ListActiveKeyIDs returns all key IDs that can be used for token validation.
	// Includes keys in "active" and "grace_period" status.
	ListActiveKeyIDs(ctx context.Context) ([]string, error)

	// RotateKey generates a new signing key and sets it as active.
	// The previous active key is moved to grace period.
	// Returns the new key ID and the generated key bytes.
	RotateKey(ctx context.Context) (keyID string, key []byte, err error)

	// GetKeyMetadata returns metadata for the specified key.
	GetKeyMetadata(ctx context.Context, keyID string) (*KeyMetadata, error)

	// RevokeKey immediately revokes a key, preventing it from validating tokens.
	// This should be used when a key is compromised.
	RevokeKey(ctx context.Context, keyID string) error
}

// InMemoryKeyManager is a simple in-memory implementation of KeyManager.
// Suitable for development and testing. For production, use VaultKeyManager.
type InMemoryKeyManager struct {
	mu           sync.RWMutex
	keys         map[string][]byte       // keyID -> key bytes
	metadata     map[string]*KeyMetadata // keyID -> metadata
	currentKeyID string
	version      int
	algorithm    string
}

// NewInMemoryKeyManager creates a new in-memory key manager with an initial key.
func NewInMemoryKeyManager(algorithm string) (*InMemoryKeyManager, error) {
	if algorithm == "" {
		algorithm = "HS256"
	}

	km := &InMemoryKeyManager{
		keys:      make(map[string][]byte),
		metadata:  make(map[string]*KeyMetadata),
		version:   1,
		algorithm: algorithm,
	}

	// Generate initial key
	keyID, key, err := km.RotateKey(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to generate initial key: %w", err)
	}

	km.currentKeyID = keyID
	_ = key // key is already stored in RotateKey

	return km, nil
}

// GetSigningKey retrieves the signing key for the given key ID.
func (km *InMemoryKeyManager) GetSigningKey(ctx context.Context, keyID string) ([]byte, error) {
	if keyID == "" {
		return nil, ErrInvalidKeyID
	}

	km.mu.RLock()
	defer km.mu.RUnlock()

	key, exists := km.keys[keyID]
	if !exists {
		return nil, ErrKeyNotFound
	}

	// Check if key is revoked
	meta, exists := km.metadata[keyID]
	if !exists || meta.Status == KeyStatusRevoked {
		return nil, ErrKeyNotFound
	}

	return key, nil
}

// GetCurrentKeyID returns the ID of the current active signing key.
func (km *InMemoryKeyManager) GetCurrentKeyID(ctx context.Context) (string, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if km.currentKeyID == "" {
		return "", ErrNoActiveKeys
	}

	return km.currentKeyID, nil
}

// ListActiveKeyIDs returns all key IDs that can be used for validation.
func (km *InMemoryKeyManager) ListActiveKeyIDs(ctx context.Context) ([]string, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	var activeKeys []string
	for keyID, meta := range km.metadata {
		if meta.Status == KeyStatusActive || meta.Status == KeyStatusGracePeriod {
			activeKeys = append(activeKeys, keyID)
		}
	}

	if len(activeKeys) == 0 {
		return nil, ErrNoActiveKeys
	}

	return activeKeys, nil
}

// RotateKey generates a new signing key and sets it as active.
func (km *InMemoryKeyManager) RotateKey(ctx context.Context) (string, []byte, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Generate new key
	key, err := generateSecureKey(MinSigningKeyLength)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Create key ID (version-based)
	keyID := fmt.Sprintf("v%d", km.version)
	now := time.Now()

	// Move current active key to grace period
	if km.currentKeyID != "" {
		if meta, exists := km.metadata[km.currentKeyID]; exists {
			meta.Status = KeyStatusGracePeriod
			meta.RotatedAt = now
		}
	}

	// Store new key
	km.keys[keyID] = key
	km.metadata[keyID] = &KeyMetadata{
		KeyID:     keyID,
		CreatedAt: now,
		Status:    KeyStatusActive,
		Algorithm: km.algorithm,
		Version:   km.version,
	}

	km.currentKeyID = keyID
	km.version++

	return keyID, key, nil
}

// GetKeyMetadata returns metadata for the specified key.
func (km *InMemoryKeyManager) GetKeyMetadata(ctx context.Context, keyID string) (*KeyMetadata, error) {
	if keyID == "" {
		return nil, ErrInvalidKeyID
	}

	km.mu.RLock()
	defer km.mu.RUnlock()

	meta, exists := km.metadata[keyID]
	if !exists {
		return nil, ErrKeyNotFound
	}

	// Return a copy to prevent external modification
	metaCopy := *meta
	return &metaCopy, nil
}

// RevokeKey immediately revokes a key.
func (km *InMemoryKeyManager) RevokeKey(ctx context.Context, keyID string) error {
	if keyID == "" {
		return ErrInvalidKeyID
	}

	km.mu.Lock()
	defer km.mu.Unlock()

	meta, exists := km.metadata[keyID]
	if !exists {
		return ErrKeyNotFound
	}

	if meta.Status == KeyStatusRevoked {
		return ErrKeyAlreadyRevoked
	}

	// Don't allow revoking the current active key without rotation
	if keyID == km.currentKeyID {
		return fmt.Errorf("cannot revoke current active key %s: rotate first", keyID)
	}

	meta.Status = KeyStatusRevoked
	meta.RevokedAt = time.Now()

	return nil
}

// generateSecureKey generates a cryptographically secure random key.
func generateSecureKey(length int) ([]byte, error) {
	key := make([]byte, length)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	return key, nil
}

// FormatKeyID creates a human-readable key ID from a timestamp and version.
func FormatKeyID(timestamp time.Time, version int) string {
	return fmt.Sprintf("%s-v%d", timestamp.Format("20060102"), version)
}

// GenerateKeyID generates a unique key ID using base64 encoding of random bytes.
func GenerateKeyID() (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate key ID: %w", err)
	}
	return base64.URLEncoding.EncodeToString(randomBytes), nil
}
