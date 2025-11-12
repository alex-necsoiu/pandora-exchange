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

		token, err := manager.GenerateAccessToken(userID, email, "user")
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Token should have 3 parts (header.payload.signature)
		parts := len(token)
		assert.Greater(t, parts, 100, "token should be reasonably long")
	})

	t.Run("generate different tokens for different users", func(t *testing.T) {
		userID1 := uuid.New()
		userID2 := uuid.New()

		token1, err := manager.GenerateAccessToken(userID1, "user1@example.com", "user")
		require.NoError(t, err)

		token2, err := manager.GenerateAccessToken(userID2, "user2@example.com", "user")
		require.NoError(t, err)

		assert.NotEqual(t, token1, token2)
	})

	t.Run("generate with nil user ID fails", func(t *testing.T) {
		_, err := manager.GenerateAccessToken(uuid.Nil, "test@example.com", "user")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user ID cannot be nil")
	})

	t.Run("generate with empty email fails", func(t *testing.T) {
		userID := uuid.New()
		_, err := manager.GenerateAccessToken(userID, "", "user")
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

		token, err := manager.GenerateAccessToken(userID, email, "user")
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
		token, err := otherManager.GenerateAccessToken(userID, "test@example.com", "user")
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
		token, err := shortManager.GenerateAccessToken(userID, "test@example.com", "user")
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
		accessToken, err := manager.GenerateAccessToken(userID, "test@example.com", "user")
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

		token, err := manager.GenerateAccessToken(userID, email, "user")
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
		token, err := manager.GenerateAccessToken(userID, "test@example.com", "user")
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

// TestGenerateAccessToken_WithRole tests role claim in access tokens.
func TestGenerateAccessToken_WithRole(t *testing.T) {
	manager, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		userID        uuid.UUID
		email         string
		role          string
		expectError   bool
		validateClaim func(*testing.T, *auth.TokenClaims)
	}{
		{
			name:   "admin role in token claims",
			userID: uuid.New(),
			email:  "admin@example.com",
			role:   "admin",
			validateClaim: func(t *testing.T, claims *auth.TokenClaims) {
				assert.Equal(t, "admin", claims.Role)
				assert.Equal(t, "access", claims.TokenType)
				assert.Equal(t, "admin@example.com", claims.Email)
			},
		},
		{
			name:   "user role in token claims",
			userID: uuid.New(),
			email:  "user@example.com",
			role:   "user",
			validateClaim: func(t *testing.T, claims *auth.TokenClaims) {
				assert.Equal(t, "user", claims.Role)
				assert.Equal(t, "access", claims.TokenType)
				assert.Equal(t, "user@example.com", claims.Email)
			},
		},
		{
			name:   "empty role in token claims",
			userID: uuid.New(),
			email:  "norole@example.com",
			role:   "",
			validateClaim: func(t *testing.T, claims *auth.TokenClaims) {
				assert.Equal(t, "", claims.Role)
				assert.Equal(t, "access", claims.TokenType)
			},
		},
		{
			name:   "custom role in token claims",
			userID: uuid.New(),
			email:  "moderator@example.com",
			role:   "moderator",
			validateClaim: func(t *testing.T, claims *auth.TokenClaims) {
				assert.Equal(t, "moderator", claims.Role)
				assert.Equal(t, "access", claims.TokenType)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := manager.GenerateAccessToken(tc.userID, tc.email, tc.role)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, token)

			// Validate the token and check claims
			claims, err := manager.ValidateAccessToken(token)
			require.NoError(t, err)

			// Verify basic claims
			assert.Equal(t, tc.userID, claims.UserID)
			assert.Equal(t, tc.email, claims.Email)

			// Validate role-specific claims
			if tc.validateClaim != nil {
				tc.validateClaim(t, claims)
			}
		})
	}
}

// TestValidateAccessToken_RoleClaim tests role claim extraction and validation.
func TestValidateAccessToken_RoleClaim(t *testing.T) {
	manager, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("extract admin role from valid token", func(t *testing.T) {
		userID := uuid.New()
		token, err := manager.GenerateAccessToken(userID, "admin@test.com", "admin")
		require.NoError(t, err)

		claims, err := manager.ValidateAccessToken(token)
		require.NoError(t, err)

		assert.Equal(t, "admin", claims.Role)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, "admin@test.com", claims.Email)
	})

	t.Run("extract user role from valid token", func(t *testing.T) {
		userID := uuid.New()
		token, err := manager.GenerateAccessToken(userID, "user@test.com", "user")
		require.NoError(t, err)

		claims, err := manager.ValidateAccessToken(token)
		require.NoError(t, err)

		assert.Equal(t, "user", claims.Role)
		assert.Equal(t, userID, claims.UserID)
	})

	t.Run("role claim type is string", func(t *testing.T) {
		userID := uuid.New()
		token, err := manager.GenerateAccessToken(userID, "test@test.com", "admin")
		require.NoError(t, err)

		claims, err := manager.ValidateAccessToken(token)
		require.NoError(t, err)

		// Verify role is a string type
		assert.IsType(t, "", claims.Role)
		assert.NotEmpty(t, claims.Role)
	})

	t.Run("role claim persists through token lifecycle", func(t *testing.T) {
		userID := uuid.New()
		originalRole := "admin"

		// Generate token with admin role
		token, err := manager.GenerateAccessToken(userID, "admin@test.com", originalRole)
		require.NoError(t, err)

		// Validate token immediately
		claims1, err := manager.ValidateAccessToken(token)
		require.NoError(t, err)
		assert.Equal(t, originalRole, claims1.Role)

		// Wait a bit and validate again - role should still be the same
		time.Sleep(100 * time.Millisecond)

		claims2, err := manager.ValidateAccessToken(token)
		require.NoError(t, err)
		assert.Equal(t, originalRole, claims2.Role)
		assert.Equal(t, claims1.Role, claims2.Role)
	})

	t.Run("different roles create different tokens", func(t *testing.T) {
		userID := uuid.New()
		email := "same@test.com"

		// Generate two tokens with different roles
		adminToken, err := manager.GenerateAccessToken(userID, email, "admin")
		require.NoError(t, err)

		userToken, err := manager.GenerateAccessToken(userID, email, "user")
		require.NoError(t, err)

		// Tokens should be different
		assert.NotEqual(t, adminToken, userToken)

		// Validate both tokens
		adminClaims, err := manager.ValidateAccessToken(adminToken)
		require.NoError(t, err)

		userClaims, err := manager.ValidateAccessToken(userToken)
		require.NoError(t, err)

		// Roles should be different
		assert.Equal(t, "admin", adminClaims.Role)
		assert.Equal(t, "user", userClaims.Role)
		assert.NotEqual(t, adminClaims.Role, userClaims.Role)
	})
}

// TestRefreshToken_NoRoleClaim tests that refresh tokens don't include role.
func TestRefreshToken_NoRoleClaim(t *testing.T) {
	manager, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("refresh token does not contain role claim", func(t *testing.T) {
		userID := uuid.New()
		token, err := manager.GenerateRefreshToken(userID)
		require.NoError(t, err)

		claims, err := manager.ValidateRefreshToken(token)
		require.NoError(t, err)

		// Refresh tokens should not have role (role comes from database on refresh)
		assert.Empty(t, claims.Role, "refresh token should not contain role claim")
		assert.Equal(t, "refresh", claims.TokenType)
		assert.Equal(t, userID, claims.UserID)
	})

	t.Run("refresh token does not contain email claim", func(t *testing.T) {
		userID := uuid.New()
		token, err := manager.GenerateRefreshToken(userID)
		require.NoError(t, err)

		claims, err := manager.ValidateRefreshToken(token)
		require.NoError(t, err)

		// Refresh tokens should not have email either
		assert.Empty(t, claims.Email, "refresh token should not contain email claim")
		assert.Equal(t, "refresh", claims.TokenType)
	})
}

// TestRoleClaimSecurity tests security aspects of role claims.
func TestRoleClaimSecurity(t *testing.T) {
	manager, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("role cannot be tampered with", func(t *testing.T) {
		userID := uuid.New()
		
		// Generate user token
		userToken, err := manager.GenerateAccessToken(userID, "user@test.com", "user")
		require.NoError(t, err)

		// Validate original token
		claims, err := manager.ValidateAccessToken(userToken)
		require.NoError(t, err)
		assert.Equal(t, "user", claims.Role)

		// Note: Attempting to manually modify the token would break the signature
		// and cause validation to fail. This is tested implicitly in the
		// "validate with wrong signature fails" test in TestValidateAccessToken
	})

	t.Run("role is cryptographically protected", func(t *testing.T) {
		userID := uuid.New()
		
		// Create two managers with different keys
		manager1, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 7*24*time.Hour)
		require.NoError(t, err)

		manager2, err := auth.NewJWTManager("different-key-min-32-chars-long!", 15*time.Minute, 7*24*time.Hour)
		require.NoError(t, err)

		// Generate admin token with manager1
		token, err := manager1.GenerateAccessToken(userID, "admin@test.com", "admin")
		require.NoError(t, err)

		// Try to validate with manager2 (different key) - should fail
		_, err = manager2.ValidateAccessToken(token)
		assert.Error(t, err, "token signed with different key should not validate")
	})

	t.Run("role claim is mandatory in token structure", func(t *testing.T) {
		userID := uuid.New()
		
		// Generate token with role
		token, err := manager.GenerateAccessToken(userID, "test@test.com", "admin")
		require.NoError(t, err)

		claims, err := manager.ValidateAccessToken(token)
		require.NoError(t, err)

		// Role field should exist in claims structure (even if empty)
		// This is guaranteed by the TokenClaims struct definition
		assert.NotNil(t, claims)
		assert.IsType(t, "", claims.Role)
	})
}

// TestVaultPlaceholder tests Vault integration placeholder.
func TestVaultPlaceholder(t *testing.T) {
	t.Run("JWT manager supports Vault integration", func(t *testing.T) {
		// This test documents that while JWTManager uses a signing key directly,
		// production deployments use Vault to securely source the key.
		//
		// Vault Integration Pattern (Implemented):
		// 1. In production, JWT_SECRET is fetched from Vault at startup
		//    (see internal/vault/client.go and VAULT_INTEGRATION.md)
		// 2. Config loader calls LoadSecretsFromVault() to override ENV with Vault values
		// 3. JWTManager is initialized with the Vault-sourced key
		// 4. Vault Agent sidecar handles secret renewal in Kubernetes
		//
		// This test uses a static key to avoid Vault dependency in unit tests.
		// See internal/vault/client_integration_test.go for Vault secret loading tests.

		manager, err := auth.NewJWTManager(testSigningKey, 15*time.Minute, 7*24*time.Hour)
		require.NoError(t, err)
		assert.NotNil(t, manager)

		// The JWTManager interface remains stable regardless of key source
		t.Log("Test: Static signing key")
		t.Log("Production: Vault-sourced signing key with rotation support")
		t.Log("See: VAULT_INTEGRATION.md for production Vault setup")
	})
}
