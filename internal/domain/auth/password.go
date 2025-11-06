// Package auth provides authentication utilities including password hashing and JWT tokens.
// This package is part of the domain layer and contains no infrastructure dependencies.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	// Argon2id parameters (OWASP recommended for 2024)
	argon2Time      = 1        // Number of iterations
	argon2Memory    = 64 * 1024 // Memory in KiB (64MB)
	argon2Threads   = 4        // Number of threads
	argon2KeyLength = 32       // Length of derived key in bytes
	argon2SaltSize  = 16       // Salt size in bytes
)

var (
	// ErrInvalidPassword indicates password verification failed.
	ErrInvalidPassword = errors.New("invalid password")
	
	// ErrInvalidHash indicates the hash format is invalid.
	ErrInvalidHash = errors.New("invalid hash format")
	
	// ErrEmptyPassword indicates an empty password was provided.
	ErrEmptyPassword = errors.New("password cannot be empty")
)

// HashPassword generates an Argon2id hash of the provided password.
// Returns a PHC string format hash that includes algorithm parameters and salt.
// Format: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
//
// Security parameters:
// - Time cost: 1 iteration (balanced for production)
// - Memory: 64MB (prevents GPU attacks)
// - Parallelism: 4 threads
// - Salt: 16 random bytes (cryptographically secure)
// - Output: 32-byte key
//
// The function uses constant-time comparison to prevent timing attacks.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	// Generate cryptographically secure random salt
	salt := make([]byte, argon2SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Generate hash using Argon2id
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLength,
	)

	// Encode to PHC string format
	// $argon2id$v=19$m=65536,t=1,p=4$base64(salt)$base64(hash)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Memory,
		argon2Time,
		argon2Threads,
		b64Salt,
		b64Hash,
	)

	return encodedHash, nil
}

// VerifyPassword verifies a password against an Argon2id hash.
// Uses constant-time comparison to prevent timing attacks.
//
// Returns nil if password matches, ErrInvalidPassword if it doesn't match,
// or another error if the hash format is invalid.
func VerifyPassword(encodedHash, password string) error {
	if encodedHash == "" {
		return ErrInvalidHash
	}
	if password == "" {
		return ErrEmptyPassword
	}

	// Parse the hash
	salt, hash, params, err := decodeHash(encodedHash)
	if err != nil {
		return err
	}

	// Generate hash with same parameters
	computedHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.time,
		params.memory,
		params.threads,
		params.keyLength,
	)

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(hash, computedHash) == 1 {
		return nil
	}

	return ErrInvalidPassword
}

// hashParams holds Argon2id parameters extracted from encoded hash.
type hashParams struct {
	memory    uint32
	time      uint32
	threads   uint8
	keyLength uint32
}

// decodeHash parses an Argon2id PHC string format hash.
// Returns salt, hash, and parameters.
func decodeHash(encodedHash string) (salt, hash []byte, params *hashParams, err error) {
	// Split hash into components
	// Expected format: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, fmt.Errorf("%w: expected 6 parts, got %d", ErrInvalidHash, len(parts))
	}

	// Verify algorithm
	if parts[1] != "argon2id" {
		return nil, nil, nil, fmt.Errorf("%w: not an argon2id hash", ErrInvalidHash)
	}

	// Parse version
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, nil, fmt.Errorf("%w: invalid version: %v", ErrInvalidHash, err)
	}
	if version != argon2.Version {
		return nil, nil, nil, fmt.Errorf("%w: unsupported version %d", ErrInvalidHash, version)
	}

	// Parse parameters
	params = &hashParams{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&params.memory, &params.time, &params.threads); err != nil {
		return nil, nil, nil, fmt.Errorf("%w: invalid parameters: %v", ErrInvalidHash, err)
	}

	// Decode salt
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: invalid salt encoding: %v", ErrInvalidHash, err)
	}

	// Decode hash
	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: invalid hash encoding: %v", ErrInvalidHash, err)
	}

	params.keyLength = uint32(len(hash))

	return salt, hash, params, nil
}
