package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// MinSigningKeyLength is the minimum length for HMAC signing keys (256 bits).
	MinSigningKeyLength = 32

	// TokenIssuer identifies tokens issued by this service.
	TokenIssuer = "pandora-exchange"
)

var (
	// ErrInvalidToken indicates the token is invalid or malformed.
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenExpired indicates the token has expired.
	ErrTokenExpired = errors.New("token expired")

	// ErrInvalidTokenType indicates the token type doesn't match expected type.
	ErrInvalidTokenType = errors.New("invalid token type")

	// ErrSigningKeyTooShort indicates the signing key is too short for security.
	ErrSigningKeyTooShort = errors.New("signing key too short")

	// ErrSigningKeyEmpty indicates no signing key was provided.
	ErrSigningKeyEmpty = errors.New("signing key cannot be empty")

	// ErrInvalidDuration indicates an invalid token duration.
	ErrInvalidDuration = errors.New("token duration must be positive")

	// ErrNilUserID indicates a nil user ID was provided.
	ErrNilUserID = errors.New("user ID cannot be nil")

	// ErrEmptyEmail indicates an empty email was provided.
	ErrEmptyEmail = errors.New("email cannot be empty")
)

// TokenClaims represents the JWT claims for both access and refresh tokens.
type TokenClaims struct {
	jwt.RegisteredClaims
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email,omitempty"`     // Only in access tokens
	Role      string    `json:"role,omitempty"`      // User role for authorization
	TokenType string    `json:"token_type"`          // "access" or "refresh"
	TokenID   string    `json:"jti"`                 // Unique token identifier
}

// JWTManager handles JWT token generation and validation.
// Uses HMAC-SHA256 for signing (HS256).
// Future: Will integrate with HashiCorp Vault for key management (Task #22).
type JWTManager struct {
	signingKey           []byte
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
}

// NewJWTManager creates a new JWT manager with the provided signing key and durations.
// The signing key should be at least 32 bytes (256 bits) for HS256.
// 
// TODO (Task #22): Replace static signing key with Vault-sourced key
// This will enable:
// - Automatic key rotation
// - Centralized secret management
// - Audit trail for key access
func NewJWTManager(signingKey string, accessTokenDuration, refreshTokenDuration time.Duration) (*JWTManager, error) {
	if signingKey == "" {
		return nil, ErrSigningKeyEmpty
	}

	if len(signingKey) < MinSigningKeyLength {
		return nil, fmt.Errorf("%w: minimum %d bytes required", ErrSigningKeyTooShort, MinSigningKeyLength)
	}

	if accessTokenDuration <= 0 {
		return nil, fmt.Errorf("%w: access token duration must be positive", ErrInvalidDuration)
	}

	if refreshTokenDuration <= 0 {
		return nil, fmt.Errorf("%w: refresh token duration must be positive", ErrInvalidDuration)
	}

	return &JWTManager{
		signingKey:           []byte(signingKey),
		accessTokenDuration:  accessTokenDuration,
		refreshTokenDuration: refreshTokenDuration,
	}, nil
}

// GenerateAccessToken generates a short-lived JWT access token.
// Access tokens include the user's email, role, and are used for API authorization.
// Default duration: 15 minutes.
func (m *JWTManager) GenerateAccessToken(userID uuid.UUID, email, role string) (string, error) {
	if userID == uuid.Nil {
		return "", ErrNilUserID
	}

	if email == "" {
		return "", ErrEmptyEmail
	}

	now := time.Now()
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    TokenIssuer,
			ID:        uuid.New().String(), // Unique token ID (jti)
		},
		UserID:    userID,
		Email:     email,
		Role:      role,
		TokenType: "access",
		TokenID:   uuid.New().String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return signedToken, nil
}

// GenerateRefreshToken generates a long-lived JWT refresh token.
// Refresh tokens don't include email and are used to obtain new access tokens.
// Default duration: 7 days.
// The jti (JWT ID) is stored in the database to enable revocation.
func (m *JWTManager) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	if userID == uuid.Nil {
		return "", ErrNilUserID
	}

	now := time.Now()
	tokenID := uuid.New().String()

	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    TokenIssuer,
			ID:        tokenID, // This will be stored in database
		},
		UserID:    userID,
		TokenType: "refresh",
		TokenID:   tokenID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return signedToken, nil
}

// ValidateAccessToken validates and parses an access token.
// Returns the token claims if valid, or an error if invalid/expired/wrong type.
func (m *JWTManager) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	claims, err := m.parseToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "access" {
		return nil, fmt.Errorf("%w: expected 'access', got '%s'", ErrInvalidTokenType, claims.TokenType)
	}

	return claims, nil
}

// ValidateRefreshToken validates and parses a refresh token.
// Returns the token claims if valid, or an error if invalid/expired/wrong type.
func (m *JWTManager) ValidateRefreshToken(tokenString string) (*TokenClaims, error) {
	claims, err := m.parseToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("%w: expected 'refresh', got '%s'", ErrInvalidTokenType, claims.TokenType)
	}

	return claims, nil
}

// GetTokenExpiration extracts the expiration time from a token without full validation.
// Useful for determining when to store refresh token expiration in database.
func (m *JWTManager) GetTokenExpiration(tokenString string) (time.Time, error) {
	claims, err := m.parseToken(tokenString)
	if err != nil {
		return time.Time{}, err
	}

	return claims.ExpiresAt.Time, nil
}

// parseToken is a helper that parses and validates a JWT token.
func (m *JWTManager) parseToken(tokenString string) (*TokenClaims, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method %v", ErrInvalidToken, token.Header["alg"])
		}
		return m.signingKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
