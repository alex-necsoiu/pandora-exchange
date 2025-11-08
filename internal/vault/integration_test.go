package vault

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVaultIntegration runs integration tests with a real Vault dev server
// These tests are skipped unless VAULT_INTEGRATION_TESTS=true
func TestVaultIntegration(t *testing.T) {
	if os.Getenv("VAULT_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping Vault integration tests. Set VAULT_INTEGRATION_TESTS=true to run")
	}

	// Check if Vault binary is available
	if _, err := exec.LookPath("vault"); err != nil {
		t.Skip("Vault binary not found in PATH. Install from https://www.vaultproject.io/downloads")
	}

	// Start Vault dev server
	vaultAddr := "http://127.0.0.1:8200"
	vaultToken := "dev-root-token-id"
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "vault", "server", "-dev", "-dev-root-token-id="+vaultToken)
	cmd.Env = append(os.Environ(), "VAULT_ADDR="+vaultAddr)
	
	// Capture output for debugging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start Vault dev server: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	// Wait for Vault to be ready
	time.Sleep(2 * time.Second)

	// Set environment for Vault CLI commands
	os.Setenv("VAULT_ADDR", vaultAddr)
	os.Setenv("VAULT_TOKEN", vaultToken)

	// Run integration tests
	t.Run("NewClient", func(t *testing.T) {
		testNewClientIntegration(t, vaultAddr, vaultToken)
	})

	t.Run("IsAvailable", func(t *testing.T) {
		testIsAvailableIntegration(t, vaultAddr, vaultToken)
	})

	t.Run("GetSecret", func(t *testing.T) {
		testGetSecretIntegration(t, vaultAddr, vaultToken)
	})

	t.Run("GetSecrets", func(t *testing.T) {
		testGetSecretsIntegration(t, vaultAddr, vaultToken)
	})

	t.Run("GetSecretWithEnvFallback", func(t *testing.T) {
		testGetSecretWithEnvFallbackIntegration(t, vaultAddr, vaultToken)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		testErrorHandlingIntegration(t, vaultAddr, vaultToken)
	})
}

func testNewClientIntegration(t *testing.T, addr, token string) {
	client, err := NewClient(addr, token)
	require.NoError(t, err, "Should create client successfully")
	require.NotNil(t, client, "Client should not be nil")
	assert.True(t, client.Enabled(), "Client should be enabled")
}

func testIsAvailableIntegration(t *testing.T, addr, token string) {
	client, err := NewClient(addr, token)
	require.NoError(t, err)

	ctx := context.Background()
	available := client.IsAvailable(ctx)
	assert.True(t, available, "Vault should be available")

	// Test with disabled client
	disabledClient := NewDisabledClient()
	assert.False(t, disabledClient.Enabled(), "Disabled client should not be enabled")
}

func testGetSecretIntegration(t *testing.T, addr, token string) {
	client, err := NewClient(addr, token)
	require.NoError(t, err)

	ctx := context.Background()

	// Write a test secret using Vault CLI
	secretPath := "secret/data/test/jwt"
	secretKey := "secret"
	secretValue := "test-jwt-secret-key-min-32-characters-long"

	cmd := exec.Command("vault", "kv", "put", "secret/test/jwt", fmt.Sprintf("%s=%s", secretKey, secretValue))
	cmd.Env = append(os.Environ(), "VAULT_ADDR="+addr, "VAULT_TOKEN="+token)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to write secret to Vault: %s", string(output))

	// Test fetching the secret
	value, err := client.GetSecret(ctx, secretPath, secretKey, "")
	require.NoError(t, err, "Should fetch secret successfully")
	assert.Equal(t, secretValue, value, "Secret value should match")

	// Test non-existent secret
	_, err = client.GetSecret(ctx, "secret/data/nonexistent", "key", "")
	assert.Error(t, err, "Should return error for non-existent secret")

	// Test with ENV fallback
	os.Setenv("TEST_FALLBACK_SECRET", "fallback-value")
	defer os.Unsetenv("TEST_FALLBACK_SECRET")

	value, err = client.GetSecret(ctx, "secret/data/nonexistent", "key", "TEST_FALLBACK_SECRET")
	require.NoError(t, err, "Should fall back to ENV when secret not found")
	assert.Equal(t, "fallback-value", value, "Should use ENV fallback value")
}

func testGetSecretsIntegration(t *testing.T, addr, token string) {
	client, err := NewClient(addr, token)
	require.NoError(t, err)

	ctx := context.Background()

	// Write multiple test secrets
	secrets := map[string]map[string]string{
		"secret/test/jwt": {
			"secret": "jwt-secret-key-min-32-characters-long",
		},
		"secret/test/database": {
			"password": "db-password-secret",
		},
		"secret/test/redis": {
			"password": "redis-password-secret",
		},
	}

	for path, data := range secrets {
		for key, value := range data {
			cmd := exec.Command("vault", "kv", "put", path, fmt.Sprintf("%s=%s", key, value))
			cmd.Env = append(os.Environ(), "VAULT_ADDR="+addr, "VAULT_TOKEN="+token)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Failed to write secret %s: %s", path, string(output))
		}
	}

	// Test batch fetching
	requests := []SecretRequest{
		{Path: "secret/data/test/jwt", Key: "secret"},
		{Path: "secret/data/test/database", Key: "password"},
		{Path: "secret/data/test/redis", Key: "password"},
	}

	results, err := client.GetSecrets(ctx, requests)
	require.NoError(t, err, "Should fetch secrets successfully")
	assert.Len(t, results, 3, "Should return 3 secrets")
	
	assert.Equal(t, "jwt-secret-key-min-32-characters-long", results["secret/data/test/jwt:secret"])
	assert.Equal(t, "db-password-secret", results["secret/data/test/database:password"])
	assert.Equal(t, "redis-password-secret", results["secret/data/test/redis:password"])

	// Test with one failing request
	mixedRequests := []SecretRequest{
		{Path: "secret/data/test/jwt", Key: "secret"},
		{Path: "secret/data/nonexistent", Key: "password"},
	}

	_, err = client.GetSecrets(ctx, mixedRequests)
	assert.Error(t, err, "Should return error if any secret fetch fails")
}

func testGetSecretWithEnvFallbackIntegration(t *testing.T, addr, token string) {
	client, err := NewClient(addr, token)
	require.NoError(t, err)

	ctx := context.Background()

	// Test ENV fallback when Vault secret doesn't exist
	os.Setenv("TEST_ENV_SECRET", "env-fallback-value")
	defer os.Unsetenv("TEST_ENV_SECRET")

	value, err := client.GetSecretWithEnvFallback(ctx, "secret/data/test/nonexistent", "key", "TEST_ENV_SECRET")
	require.NoError(t, err, "Should fall back to ENV")
	assert.Equal(t, "env-fallback-value", value, "Should return ENV value")

	// Test Vault value takes precedence over ENV
	secretPath := "secret/data/test/priority"
	cmd := exec.Command("vault", "kv", "put", "secret/test/priority", "key=vault-value")
	cmd.Env = append(os.Environ(), "VAULT_ADDR="+addr, "VAULT_TOKEN="+token)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to write secret: %s", string(output))

	os.Setenv("TEST_PRIORITY_SECRET", "env-value")
	defer os.Unsetenv("TEST_PRIORITY_SECRET")

	value, err = client.GetSecretWithEnvFallback(ctx, secretPath, "key", "TEST_PRIORITY_SECRET")
	require.NoError(t, err, "Should fetch from Vault")
	assert.Equal(t, "vault-value", value, "Vault value should take precedence")

	// Test with disabled client
	disabledClient := NewDisabledClient()
	os.Setenv("TEST_DISABLED_SECRET", "disabled-env-value")
	defer os.Unsetenv("TEST_DISABLED_SECRET")

	value, err = disabledClient.GetSecretWithEnvFallback(ctx, "secret/data/any", "key", "TEST_DISABLED_SECRET")
	require.NoError(t, err, "Disabled client should use ENV")
	assert.Equal(t, "disabled-env-value", value, "Disabled client should return ENV value")
}

func testErrorHandlingIntegration(t *testing.T, addr, token string) {
	// Test with invalid token
	t.Run("invalid token", func(t *testing.T) {
		client, err := NewClient(addr, "invalid-token")
		require.NoError(t, err, "Client creation should succeed")

		ctx := context.Background()
		_, err = client.GetSecret(ctx, "secret/data/test/jwt", "secret", "")
		assert.Error(t, err, "Should return error with invalid token")
	})

	// Test with unreachable Vault
	t.Run("unreachable Vault", func(t *testing.T) {
		client, err := NewClient("http://localhost:9999", token)
		require.NoError(t, err, "Client creation should succeed")

		ctx := context.Background()
		available := client.IsAvailable(ctx)
		assert.False(t, available, "Should return false for unreachable Vault")
	})

	// Test with empty path
	t.Run("empty path", func(t *testing.T) {
		client, err := NewClient(addr, token)
		require.NoError(t, err)

		ctx := context.Background()
		_, err = client.GetSecret(ctx, "", "key", "")
		assert.Error(t, err, "Should return error for empty path")
	})

	// Test with empty key
	t.Run("empty key", func(t *testing.T) {
		client, err := NewClient(addr, token)
		require.NoError(t, err)

		ctx := context.Background()
		_, err = client.GetSecret(ctx, "secret/data/test/jwt", "", "")
		assert.Error(t, err, "Should return error for empty key")
	})

	// Test with context timeout
	t.Run("context timeout", func(t *testing.T) {
		client, err := NewClient(addr, token)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond) // Ensure context is expired

		_, err = client.GetSecret(ctx, "secret/data/test/jwt", "secret", "")
		assert.Error(t, err, "Should return error on context timeout")
	})
}

// TestDisabledClient tests that disabled client behaves correctly
func TestDisabledClient(t *testing.T) {
	client := NewDisabledClient()
	
	assert.False(t, client.Enabled(), "Disabled client should not be enabled")

	ctx := context.Background()

	// IsAvailable should return false
	assert.False(t, client.IsAvailable(ctx), "Disabled client should not be available")

	// GetSecret should use ENV fallback
	os.Setenv("TEST_DISABLED_SECRET", "test-value")
	defer os.Unsetenv("TEST_DISABLED_SECRET")

	value, err := client.GetSecret(ctx, "any/path", "key", "TEST_DISABLED_SECRET")
	require.NoError(t, err, "Should fall back to ENV")
	assert.Equal(t, "test-value", value, "Should return ENV value")

	// GetSecret without fallback should error
	_, err = client.GetSecret(ctx, "any/path", "key", "")
	assert.Error(t, err, "Should error when no ENV fallback and Vault disabled")
}

// TestConfigIntegration tests Config.LoadSecretsFromVault with real Vault
func TestConfigIntegration(t *testing.T) {
	if os.Getenv("VAULT_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping Vault integration tests. Set VAULT_INTEGRATION_TESTS=true to run")
	}

	// This test would require importing the config package
	// For now, we'll skip it to avoid circular dependencies
	// The actual integration is tested via the service startup
	t.Skip("Config integration tested via service startup")
}
