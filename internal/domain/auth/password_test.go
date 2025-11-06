package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHashPassword tests password hashing functionality.
func TestHashPassword(t *testing.T) {
	t.Run("hash password successfully", func(t *testing.T) {
		password := "MySecureP@ssw0rd123"
		
		hash, err := auth.HashPassword(password)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash)
		
		// Hash should contain argon2id identifier
		assert.True(t, strings.HasPrefix(hash, "$argon2id$"))
	})

	t.Run("hash same password produces different hashes", func(t *testing.T) {
		password := "SamePassword123!"
		
		hash1, err := auth.HashPassword(password)
		require.NoError(t, err)
		
		hash2, err := auth.HashPassword(password)
		require.NoError(t, err)
		
		// Hashes should be different due to random salt
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("hash empty password returns error", func(t *testing.T) {
		_, err := auth.HashPassword("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password cannot be empty")
	})

	t.Run("hash very long password succeeds", func(t *testing.T) {
		// Test with 100-character password
		longPassword := strings.Repeat("a", 100)
		
		hash, err := auth.HashPassword(longPassword)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("hash contains proper format", func(t *testing.T) {
		password := "TestPassword123"
		
		hash, err := auth.HashPassword(password)
		require.NoError(t, err)
		
		// Argon2id hash format: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
		parts := strings.Split(hash, "$")
		assert.GreaterOrEqual(t, len(parts), 6, "hash should have at least 6 parts")
		assert.Equal(t, "argon2id", parts[1])
		assert.Equal(t, "v=19", parts[2])
		assert.Contains(t, parts[3], "m=65536") // 64MB memory
		assert.Contains(t, parts[3], "t=1")     // 1 iteration
		assert.Contains(t, parts[3], "p=4")     // 4 threads
	})
}

// TestVerifyPassword tests password verification.
func TestVerifyPassword(t *testing.T) {
	t.Run("verify correct password", func(t *testing.T) {
		password := "CorrectPassword123!"
		
		hash, err := auth.HashPassword(password)
		require.NoError(t, err)
		
		err = auth.VerifyPassword(hash, password)
		assert.NoError(t, err)
	})

	t.Run("verify incorrect password fails", func(t *testing.T) {
		password := "CorrectPassword123!"
		wrongPassword := "WrongPassword456!"
		
		hash, err := auth.HashPassword(password)
		require.NoError(t, err)
		
		err = auth.VerifyPassword(hash, wrongPassword)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid password")
	})

	t.Run("verify with empty password fails", func(t *testing.T) {
		hash, err := auth.HashPassword("ValidPassword123")
		require.NoError(t, err)
		
		err = auth.VerifyPassword(hash, "")
		assert.Error(t, err)
	})

	t.Run("verify with empty hash fails", func(t *testing.T) {
		err := auth.VerifyPassword("", "SomePassword123")
		assert.Error(t, err)
	})

	t.Run("verify with malformed hash fails", func(t *testing.T) {
		malformedHash := "not-a-valid-hash"
		
		err := auth.VerifyPassword(malformedHash, "Password123")
		assert.Error(t, err)
	})

	t.Run("verify with wrong algorithm hash fails", func(t *testing.T) {
		// BCrypt hash (different algorithm)
		bcryptHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
		
		err := auth.VerifyPassword(bcryptHash, "Password123")
		assert.Error(t, err)
	})

	t.Run("verify multiple times with same credentials succeeds", func(t *testing.T) {
		password := "ConsistentPassword123"
		hash, err := auth.HashPassword(password)
		require.NoError(t, err)
		
		// Verify multiple times
		for i := 0; i < 5; i++ {
			err = auth.VerifyPassword(hash, password)
			assert.NoError(t, err, "verification %d should succeed", i+1)
		}
	})
}

// TestPasswordHashingTiming tests for timing attack resistance.
func TestPasswordHashingTiming(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing test in short mode")
	}

	t.Run("verify has consistent timing regardless of result", func(t *testing.T) {
		password := "TimingTestPassword123"
		hash, err := auth.HashPassword(password)
		require.NoError(t, err)
		
		const iterations = 10
		
		// Time correct password verifications
		correctTimes := make([]time.Duration, iterations)
		for i := 0; i < iterations; i++ {
			start := time.Now()
			_ = auth.VerifyPassword(hash, password)
			correctTimes[i] = time.Since(start)
		}
		
		// Time incorrect password verifications
		incorrectTimes := make([]time.Duration, iterations)
		for i := 0; i < iterations; i++ {
			start := time.Now()
			_ = auth.VerifyPassword(hash, "WrongPassword"+string(rune(i)))
			incorrectTimes[i] = time.Since(start)
		}
		
		// Calculate average times
		var correctAvg, incorrectAvg time.Duration
		for i := 0; i < iterations; i++ {
			correctAvg += correctTimes[i]
			incorrectAvg += incorrectTimes[i]
		}
		correctAvg /= iterations
		incorrectAvg /= iterations
		
		// Timing difference should be minimal (within 20%)
		// This protects against timing attacks
		diff := float64(correctAvg - incorrectAvg)
		if diff < 0 {
			diff = -diff
		}
		percentDiff := (diff / float64(correctAvg)) * 100
		
		t.Logf("Correct password avg: %v, Incorrect password avg: %v, Diff: %.2f%%", 
			correctAvg, incorrectAvg, percentDiff)
		
		// Allow up to 20% difference due to system variance
		assert.Less(t, percentDiff, 20.0, 
			"timing difference should be minimal to prevent timing attacks")
	})
}

// TestPasswordHashingPerformance tests hashing performance.
func TestPasswordHashingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	t.Run("hashing takes reasonable time", func(t *testing.T) {
		password := "PerformanceTestPassword123"
		
		start := time.Now()
		_, err := auth.HashPassword(password)
		duration := time.Since(start)
		
		require.NoError(t, err)
		
		// Hashing should take between 50ms and 500ms
		// (too fast = weak security, too slow = DoS vector)
		assert.Greater(t, duration, 10*time.Millisecond, 
			"hashing should take at least 10ms for security")
		assert.Less(t, duration, 1*time.Second, 
			"hashing should complete within 1 second to prevent DoS")
		
		t.Logf("Hashing took: %v", duration)
	})

	t.Run("verification takes reasonable time", func(t *testing.T) {
		password := "VerificationTestPassword123"
		hash, err := auth.HashPassword(password)
		require.NoError(t, err)
		
		start := time.Now()
		err = auth.VerifyPassword(hash, password)
		duration := time.Since(start)
		
		require.NoError(t, err)
		
		// Verification should be similar to hashing time
		assert.Greater(t, duration, 10*time.Millisecond)
		assert.Less(t, duration, 1*time.Second)
		
		t.Logf("Verification took: %v", duration)
	})
}

// BenchmarkHashPassword benchmarks password hashing.
func BenchmarkHashPassword(b *testing.B) {
	password := "BenchmarkPassword123!"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = auth.HashPassword(password)
	}
}

// BenchmarkVerifyPassword benchmarks password verification.
func BenchmarkVerifyPassword(b *testing.B) {
	password := "BenchmarkPassword123!"
	hash, _ := auth.HashPassword(password)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = auth.VerifyPassword(hash, password)
	}
}
