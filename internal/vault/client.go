// Package vault provides HashiCorp Vault integration for secure secret management.
// It implements the secret fetching interface with fallback to environment variables
// for local development. In production, secrets are fetched from Vault KV v2 store.
package vault

import (
	"context"
	"fmt"
	"os"
	"strings"

	vault "github.com/hashicorp/vault/api"
)

// Client wraps the Vault API client with convenience methods for secret management
type Client struct {
	client *vault.Client
	enabled bool
}

// SecretRequest represents a request to fetch a secret from Vault
type SecretRequest struct {
	Path string // Vault path (e.g., "secret/data/app/jwt")
	Key  string // Key within the secret (e.g., "secret")
}

// SecretResponse contains the fetched secret value
type SecretResponse struct {
	Path  string
	Key   string
	Value string
}

// NewClient creates a new Vault client instance
//
// Parameters:
//   - addr: Vault server address (e.g., "http://localhost:8200" or "https://vault.example.com")
//   - token: Vault authentication token
//
// Returns:
//   - *Client: Configured Vault client
//   - error: Returns error if addr or token is empty, or if client creation fails
//
// Security: Token should be provided via environment variable, not hardcoded
func NewClient(addr, token string) (*Client, error) {
	if addr == "" {
		return nil, fmt.Errorf("vault address cannot be empty")
	}
	if token == "" {
		return nil, fmt.Errorf("vault token cannot be empty")
	}

	config := vault.DefaultConfig()
	config.Address = addr

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	client.SetToken(token)

	return &Client{
		client:  client,
		enabled: true,
	}, nil
}

// NewDisabledClient creates a disabled client for local development
// When disabled, all Get* methods will fall back to environment variables
//
// Returns:
//   - *Client: Client with enabled=false
func NewDisabledClient() *Client {
	return &Client{
		client:  nil,
		enabled: false,
	}
}

// IsAvailable checks if Vault server is reachable and client is authenticated
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//
// Returns:
//   - bool: true if Vault is available and client can authenticate
//
// Use this for health checks before attempting to fetch secrets
func (c *Client) IsAvailable(ctx context.Context) bool {
	if !c.enabled || c.client == nil {
		return false
	}

	// Try to read sys/health endpoint
	req := c.client.NewRequest("GET", "/v1/sys/health")
	resp, err := c.client.RawRequestWithContext(ctx, req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200 || resp.StatusCode == 429 // 429 = sealed but available
}

// GetSecret fetches a single secret from Vault KV v2
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - path: Full Vault path including mount point (e.g., "secret/data/app/jwt")
//   - key: Key within the secret data (e.g., "secret")
//   - envFallback: Environment variable name to use if Vault unavailable
//
// Returns:
//   - string: Secret value
//   - error: Returns error if path/key empty, secret not found, or Vault unavailable with no fallback
//
// Example:
//   secret, err := client.GetSecret(ctx, "secret/data/app/jwt", "secret", "JWT_SECRET")
//
// Security: Falls back to ENV only if Vault is disabled (dev mode)
func (c *Client) GetSecret(ctx context.Context, path, key, envFallback string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("vault path cannot be empty")
	}
	if key == "" {
		return "", fmt.Errorf("secret key cannot be empty")
	}

	// If Vault is disabled (dev mode), use environment variable
	if !c.enabled {
		if envFallback == "" {
			return "", fmt.Errorf("vault disabled and no environment fallback provided")
		}
		value := os.Getenv(envFallback)
		if value == "" {
			return "", fmt.Errorf("environment variable %s not set", envFallback)
		}
		return value, nil
	}

	// Fetch from Vault
	secret, err := c.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		// If Vault fetch fails and ENV fallback provided, try ENV
		if envFallback != "" {
			if value := os.Getenv(envFallback); value != "" {
				return value, nil
			}
		}
		return "", fmt.Errorf("failed to read secret from vault: %w", err)
	}

	if secret == nil {
		return "", fmt.Errorf("secret not found at path: %s", path)
	}

	// KV v2 stores data under "data" key
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected secret format at path: %s", path)
	}

	value, ok := data[key].(string)
	if !ok {
		return "", fmt.Errorf("key '%s' not found or not a string in secret at path: %s", key, path)
	}

	return value, nil
}

// GetSecrets fetches multiple secrets in batch
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - requests: Slice of secret requests to fetch
//
// Returns:
//   - map[string]string: Map of "path:key" to secret value
//   - error: Returns error if any secret fetch fails
//
// Example:
//   requests := []SecretRequest{
//     {Path: "secret/data/app/jwt", Key: "secret"},
//     {Path: "secret/data/app/db", Key: "password"},
//   }
//   secrets, err := client.GetSecrets(ctx, requests)
//
// Note: If any secret fails, the entire operation fails (fail-fast)
func (c *Client) GetSecrets(ctx context.Context, requests []SecretRequest) (map[string]string, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("no secret requests provided")
	}

	secrets := make(map[string]string, len(requests))

	for _, req := range requests {
		// For batch operations, we don't provide ENV fallback
		value, err := c.GetSecret(ctx, req.Path, req.Key, "")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch secret %s:%s: %w", req.Path, req.Key, err)
		}

		// Use "path:key" as map key for uniqueness
		mapKey := fmt.Sprintf("%s:%s", req.Path, req.Key)
		secrets[mapKey] = value
	}

	return secrets, nil
}

// GetSecretWithEnvFallback is a convenience method that always tries ENV if Vault fails
// This is useful for local development where Vault might not be available
//
// Parameters:
//   - ctx: Context for cancellation
//   - path: Vault secret path
//   - key: Key within secret
//   - envVar: Environment variable name
//
// Returns:
//   - string: Secret value from Vault or ENV
//   - error: Only if both Vault and ENV fail
func (c *Client) GetSecretWithEnvFallback(ctx context.Context, path, key, envVar string) (string, error) {
	return c.GetSecret(ctx, path, key, envVar)
}

// Enabled returns whether Vault integration is enabled
func (c *Client) Enabled() bool {
	return c.enabled
}

// parseVaultPath splits a Vault path into mount and secret path
// For KV v2: "secret/data/app/jwt" -> ("secret", "app/jwt")
func parseVaultPath(fullPath string) (mount string, secretPath string, err error) {
	parts := strings.Split(fullPath, "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid vault path format: %s", fullPath)
	}

	// KV v2 paths include "/data/"
	if len(parts) >= 3 && parts[1] == "data" {
		mount = parts[0]
		secretPath = strings.Join(parts[2:], "/")
		return mount, secretPath, nil
	}

	return "", "", fmt.Errorf("unsupported vault path format (expected KV v2): %s", fullPath)
}
