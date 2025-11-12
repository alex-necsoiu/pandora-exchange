package vault

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClient tests the Vault client initialization
func TestNewClient(t *testing.T) {
	testCases := []struct {
		name        string
		addr        string
		token       string
		expectError bool
		errorMsg    string
		description string
	}{
		{
			name:        "valid configuration creates client",
			addr:        "http://localhost:8200",
			token:       "dev-token",
			expectError: false,
			description: "Should create client with valid addr and token",
		},
		{
			name:        "empty address fails",
			addr:        "",
			token:       "dev-token",
			expectError: true,
			errorMsg:    "vault address cannot be empty",
			description: "Empty Vault address should return error",
		},
		{
			name:        "empty token fails",
			addr:        "http://localhost:8200",
			token:       "",
			expectError: true,
			errorMsg:    "vault token cannot be empty",
			description: "Empty Vault token should return error",
		},
		{
			name:        "https address works",
			addr:        "https://vault.example.com:8200",
			token:       "prod-token",
			expectError: false,
			description: "Should support HTTPS addresses",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClient(tc.addr, tc.token)

			if tc.expectError {
				assert.Error(t, err, tc.description)
				assert.Nil(t, client)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err, tc.description)
				assert.NotNil(t, client)
				assert.True(t, client.Enabled())
			}
		})
	}
}

// TestNewDisabledClient tests disabled client creation
func TestNewDisabledClient(t *testing.T) {
	client := NewDisabledClient()

	assert.NotNil(t, client, "Should create disabled client")
	assert.False(t, client.Enabled(), "Disabled client should not be enabled")
	assert.Nil(t, client.client, "Disabled client should have nil internal client")
}

// TestIsAvailable tests Vault health check
func TestIsAvailable(t *testing.T) {
	t.Run("returns false when Vault unreachable", func(t *testing.T) {
		client, err := NewClient("http://localhost:9999", "fake-token")
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		available := client.IsAvailable(ctx)
		assert.False(t, available, "Should return false when Vault unreachable")
	})

	t.Run("returns false for disabled client", func(t *testing.T) {
		client := NewDisabledClient()

		available := client.IsAvailable(context.Background())
		assert.False(t, available, "Disabled client should not be available")
	})

	t.Run("returns false for nil internal client", func(t *testing.T) {
		client := &Client{
			client:  nil,
			enabled: true,
		}

		available := client.IsAvailable(context.Background())
		assert.False(t, available, "Should return false with nil client")
	})

	t.Run("respects context timeout", func(t *testing.T) {
		client, err := NewClient("http://localhost:9999", "fake-token")
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond) // Ensure timeout

		available := client.IsAvailable(ctx)
		assert.False(t, available, "Should return false on context timeout")
	})
}

// TestGetSecret tests secret fetching with various scenarios
func TestGetSecret(t *testing.T) {
	t.Run("empty path returns error", func(t *testing.T) {
		client, err := NewClient("http://localhost:8200", "token")
		require.NoError(t, err)

		_, err = client.GetSecret(context.Background(), "", "key", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "vault path cannot be empty")
	})

	t.Run("empty key returns error", func(t *testing.T) {
		client, err := NewClient("http://localhost:8200", "token")
		require.NoError(t, err)

		_, err = client.GetSecret(context.Background(), "secret/data/app", "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret key cannot be empty")
	})

	t.Run("disabled client uses env fallback", func(t *testing.T) {
		client := NewDisabledClient()

		os.Setenv("TEST_ENV_VAR", "env-value")
		defer os.Unsetenv("TEST_ENV_VAR")

		value, err := client.GetSecret(context.Background(), "any/path", "key", "TEST_ENV_VAR")
		require.NoError(t, err)
		assert.Equal(t, "env-value", value)
	})

	t.Run("disabled client without fallback returns error", func(t *testing.T) {
		client := NewDisabledClient()

		_, err := client.GetSecret(context.Background(), "any/path", "key", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "vault disabled and no environment fallback provided")
	})

	t.Run("disabled client with empty env var returns error", func(t *testing.T) {
		client := NewDisabledClient()

		os.Unsetenv("NONEXISTENT_VAR")

		_, err := client.GetSecret(context.Background(), "any/path", "key", "NONEXISTENT_VAR")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environment variable NONEXISTENT_VAR not set")
	})

	t.Run("unreachable vault with fallback uses env", func(t *testing.T) {
		client, err := NewClient("http://localhost:9999", "token")
		require.NoError(t, err)

		os.Setenv("TEST_FALLBACK_VAR", "fallback-value")
		defer os.Unsetenv("TEST_FALLBACK_VAR")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		value, err := client.GetSecret(ctx, "secret/data/app", "key", "TEST_FALLBACK_VAR")
		require.NoError(t, err)
		assert.Equal(t, "fallback-value", value)
	})

	t.Run("unreachable vault without fallback returns error", func(t *testing.T) {
		client, err := NewClient("http://localhost:9999", "token")
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		_, err = client.GetSecret(ctx, "secret/data/app", "key", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read secret from vault")
	})
}

// TestGetSecrets tests batch secret fetching
func TestGetSecrets(t *testing.T) {
	t.Run("empty requests returns error", func(t *testing.T) {
		client, err := NewClient("http://localhost:8200", "token")
		require.NoError(t, err)

		_, err = client.GetSecrets(context.Background(), []SecretRequest{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no secret requests provided")
	})

	t.Run("nil requests returns error", func(t *testing.T) {
		client, err := NewClient("http://localhost:8200", "token")
		require.NoError(t, err)

		_, err = client.GetSecrets(context.Background(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no secret requests provided")
	})

	t.Run("disabled client with env vars works", func(t *testing.T) {
		client := NewDisabledClient()

		os.Setenv("JWT_SECRET", "jwt-value")
		os.Setenv("DB_PASSWORD", "db-value")
		defer func() {
			os.Unsetenv("JWT_SECRET")
			os.Unsetenv("DB_PASSWORD")
		}()

		requests := []SecretRequest{
			{Path: "secret/data/app/jwt", Key: "secret"},
			{Path: "secret/data/app/db", Key: "password"},
		}

		// Note: GetSecrets doesn't support fallback, so this will fail
		_, err := client.GetSecrets(context.Background(), requests)
		assert.Error(t, err, "GetSecrets should fail without fallback support")
	})

	t.Run("invalid request fails entire batch", func(t *testing.T) {
		client, err := NewClient("http://localhost:9999", "token")
		require.NoError(t, err)

		requests := []SecretRequest{
			{Path: "secret/data/app/jwt", Key: "secret"},
			{Path: "", Key: "password"}, // Invalid: empty path
		}

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		_, err = client.GetSecrets(ctx, requests)
		assert.Error(t, err)
	})
}

// TestGetSecretWithEnvFallback tests convenience method
func TestGetSecretWithEnvFallback(t *testing.T) {
	t.Run("disabled client uses env", func(t *testing.T) {
		client := NewDisabledClient()

		os.Setenv("TEST_SECRET", "test-value")
		defer os.Unsetenv("TEST_SECRET")

		value, err := client.GetSecretWithEnvFallback(context.Background(), "any/path", "key", "TEST_SECRET")
		require.NoError(t, err)
		assert.Equal(t, "test-value", value)
	})

	t.Run("unreachable vault falls back to env", func(t *testing.T) {
		client, err := NewClient("http://localhost:9999", "token")
		require.NoError(t, err)

		os.Setenv("FALLBACK_SECRET", "fallback-value")
		defer os.Unsetenv("FALLBACK_SECRET")

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		value, err := client.GetSecretWithEnvFallback(ctx, "secret/data/app", "key", "FALLBACK_SECRET")
		require.NoError(t, err)
		assert.Equal(t, "fallback-value", value)
	})
}

// TestEnabled tests the Enabled method
func TestEnabled(t *testing.T) {
	t.Run("enabled client returns true", func(t *testing.T) {
		client, err := NewClient("http://localhost:8200", "token")
		require.NoError(t, err)

		assert.True(t, client.Enabled())
	})

	t.Run("disabled client returns false", func(t *testing.T) {
		client := NewDisabledClient()

		assert.False(t, client.Enabled())
	})
}

// TestParseVaultPath tests internal path parsing
func TestParseVaultPath(t *testing.T) {
	testCases := []struct {
		name          string
		path          string
		expectMount   string
		expectSecret  string
		expectError   bool
		description   string
	}{
		{
			name:         "valid KV v2 path",
			path:         "secret/data/app/jwt",
			expectMount:  "secret",
			expectSecret: "app/jwt",
			expectError:  false,
			description:  "Should parse standard KV v2 path",
		},
		{
			name:         "nested secret path",
			path:         "secret/data/prod/service/database",
			expectMount:  "secret",
			expectSecret: "prod/service/database",
			expectError:  false,
			description:  "Should handle nested paths",
		},
		{
			name:        "invalid short path",
			path:        "secret/data",
			expectError: true,
			description: "Should reject too short paths",
		},
		{
			name:        "missing data segment",
			path:        "secret/app/jwt",
			expectError: true,
			description: "Should reject non-KV v2 paths",
		},
		{
			name:        "empty path",
			path:        "",
			expectError: true,
			description: "Should reject empty path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mount, secretPath, err := parseVaultPath(tc.path)

			if tc.expectError {
				assert.Error(t, err, tc.description)
			} else {
				assert.NoError(t, err, tc.description)
				assert.Equal(t, tc.expectMount, mount)
				assert.Equal(t, tc.expectSecret, secretPath)
			}
		})
	}
}

// TestContextCancellation tests proper handling of context cancellation
func TestContextCancellation(t *testing.T) {
	t.Run("IsAvailable respects cancelled context", func(t *testing.T) {
		client, err := NewClient("http://localhost:9999", "token")
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		available := client.IsAvailable(ctx)
		assert.False(t, available, "Should return false with cancelled context")
	})

	t.Run("GetSecret respects cancelled context", func(t *testing.T) {
		client, err := NewClient("http://localhost:9999", "token")
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = client.GetSecret(ctx, "secret/data/app", "key", "")
		assert.Error(t, err, "Should return error with cancelled context")
	})
}

// TestNetworkFailureScenarios tests various network failure modes
func TestNetworkFailureScenarios(t *testing.T) {
	t.Run("connection refused with fallback", func(t *testing.T) {
		client, err := NewClient("http://localhost:9999", "token")
		require.NoError(t, err)

		os.Setenv("NETWORK_FALLBACK", "fallback-value")
		defer os.Unsetenv("NETWORK_FALLBACK")

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		value, err := client.GetSecret(ctx, "secret/data/app", "key", "NETWORK_FALLBACK")
		require.NoError(t, err)
		assert.Equal(t, "fallback-value", value)
	})

	t.Run("connection timeout without fallback", func(t *testing.T) {
		client, err := NewClient("http://10.255.255.1:8200", "token") // Non-routable IP
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err = client.GetSecret(ctx, "secret/data/app", "key", "")
		assert.Error(t, err)
	})

	t.Run("IsAvailable handles network errors gracefully", func(t *testing.T) {
		client, err := NewClient("http://invalid-hostname-that-does-not-exist:8200", "token")
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		available := client.IsAvailable(ctx)
		assert.False(t, available, "Should return false on network error")
	})
}

// TestTimeoutScenarios tests timeout handling
func TestTimeoutScenarios(t *testing.T) {
	t.Run("very short timeout fails", func(t *testing.T) {
		client, err := NewClient("http://localhost:9999", "token")
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(1 * time.Millisecond) // Ensure timeout

		_, err = client.GetSecret(ctx, "secret/data/app", "key", "")
		assert.Error(t, err)
	})

	t.Run("reasonable timeout with fallback succeeds", func(t *testing.T) {
		client, err := NewClient("http://localhost:9999", "token")
		require.NoError(t, err)

		os.Setenv("TIMEOUT_FALLBACK", "timeout-value")
		defer os.Unsetenv("TIMEOUT_FALLBACK")

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		value, err := client.GetSecret(ctx, "secret/data/app", "key", "TIMEOUT_FALLBACK")
		require.NoError(t, err)
		assert.Equal(t, "timeout-value", value)
	})
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("GetSecrets with single request", func(t *testing.T) {
		client := NewDisabledClient()

		os.Setenv("SINGLE_SECRET", "single-value")
		defer os.Unsetenv("SINGLE_SECRET")

		requests := []SecretRequest{
			{Path: "secret/data/app", Key: "key"},
		}

		// Will fail because GetSecrets doesn't support fallback
		_, err := client.GetSecrets(context.Background(), requests)
		assert.Error(t, err)
	})

	t.Run("multiple GetSecret calls are independent", func(t *testing.T) {
		client := NewDisabledClient()

		os.Setenv("SECRET_1", "value-1")
		os.Setenv("SECRET_2", "value-2")
		defer func() {
			os.Unsetenv("SECRET_1")
			os.Unsetenv("SECRET_2")
		}()

		value1, err1 := client.GetSecret(context.Background(), "path1", "key1", "SECRET_1")
		value2, err2 := client.GetSecret(context.Background(), "path2", "key2", "SECRET_2")

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, "value-1", value1)
		assert.Equal(t, "value-2", value2)
	})

	t.Run("special characters in env var name", func(t *testing.T) {
		client := NewDisabledClient()

		os.Setenv("MY_SECRET_KEY", "special-value")
		defer os.Unsetenv("MY_SECRET_KEY")

		value, err := client.GetSecret(context.Background(), "any", "key", "MY_SECRET_KEY")
		require.NoError(t, err)
		assert.Equal(t, "special-value", value)
	})

	t.Run("GetSecrets with multiple empty path requests", func(t *testing.T) {
		client, err := NewClient("http://localhost:8200", "token")
		require.NoError(t, err)

		requests := []SecretRequest{
			{Path: "", Key: "key1"},
			{Path: "", Key: "key2"},
		}

		_, err = client.GetSecrets(context.Background(), requests)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "vault path cannot be empty")
	})

	t.Run("GetSecrets with empty key in batch", func(t *testing.T) {
		client, err := NewClient("http://localhost:8200", "token")
		require.NoError(t, err)

		requests := []SecretRequest{
			{Path: "secret/data/app", Key: ""},
		}

		_, err = client.GetSecrets(context.Background(), requests)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret key cannot be empty")
	})
}

// TestSecretRequestValidation tests SecretRequest validation
func TestSecretRequestValidation(t *testing.T) {
	client, err := NewClient("http://localhost:8200", "token")
	require.NoError(t, err)

	testCases := []struct {
		name    string
		request SecretRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid request",
			request: SecretRequest{Path: "secret/data/app", Key: "key"},
			wantErr: true, // Will fail because Vault is unreachable, but validates inputs
		},
		{
			name:    "empty path",
			request: SecretRequest{Path: "", Key: "key"},
			wantErr: true,
			errMsg:  "vault path cannot be empty",
		},
		{
			name:    "empty key",
			request: SecretRequest{Path: "secret/data/app", Key: ""},
			wantErr: true,
			errMsg:  "secret key cannot be empty",
		},
		{
			name:    "both empty",
			request: SecretRequest{Path: "", Key: ""},
			wantErr: true,
			errMsg:  "vault path cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			_, err := client.GetSecret(ctx, tc.request.Path, tc.request.Key, "")
			
			if tc.wantErr {
				assert.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestClientStateConsistency tests that client state remains consistent
func TestClientStateConsistency(t *testing.T) {
	t.Run("enabled client stays enabled", func(t *testing.T) {
		client, err := NewClient("http://localhost:8200", "token")
		require.NoError(t, err)

		// Call multiple operations
		assert.True(t, client.Enabled())
		client.IsAvailable(context.Background())
		assert.True(t, client.Enabled())
		_, _ = client.GetSecret(context.Background(), "path", "key", "FALLBACK")
		assert.True(t, client.Enabled())
	})

	t.Run("disabled client stays disabled", func(t *testing.T) {
		client := NewDisabledClient()

		assert.False(t, client.Enabled())
		client.IsAvailable(context.Background())
		assert.False(t, client.Enabled())

		os.Setenv("TEST_VAR", "value")
		defer os.Unsetenv("TEST_VAR")

		_, _ = client.GetSecret(context.Background(), "path", "key", "TEST_VAR")
		assert.False(t, client.Enabled())
	})
}

// TestConcurrentAccess tests thread-safety
func TestConcurrentAccess(t *testing.T) {
	client := NewDisabledClient()

	os.Setenv("CONCURRENT_SECRET", "concurrent-value")
	defer os.Unsetenv("CONCURRENT_SECRET")

	// Run multiple goroutines accessing the client concurrently
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			value, err := client.GetSecret(context.Background(), "path", "key", "CONCURRENT_SECRET")
			assert.NoError(t, err)
			assert.Equal(t, "concurrent-value", value)
			assert.False(t, client.Enabled())
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

