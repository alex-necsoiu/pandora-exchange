package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/google/uuid"
)

// Compile-time check to ensure UserService implements domain.UserService
var _ domain.UserService = (*UserService)(nil)

// UserService implements domain.UserService
type UserService struct {
	userRepo           domain.UserRepository
	refreshTokenRepo   domain.RefreshTokenRepository
	jwtManager         *auth.JWTManager
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

// NewUserService creates a new UserService instance
func NewUserService(
	userRepo domain.UserRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	jwtSecret string,
	accessTokenExpiry time.Duration,
	refreshTokenExpiry time.Duration,
) (*UserService, error) {
	jwtManager, err := auth.NewJWTManager(jwtSecret, accessTokenExpiry, refreshTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT manager: %w", err)
	}

	return &UserService{
		userRepo:           userRepo,
		refreshTokenRepo:   refreshTokenRepo,
		jwtManager:         jwtManager,
		accessTokenExpiry:  accessTokenExpiry,
		refreshTokenExpiry: refreshTokenExpiry,
	}, nil
}

// Register creates a new user account
func (s *UserService) Register(ctx context.Context, email, password, fullName string) (*domain.User, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}
	if password == "" {
		return nil, errors.New("password cannot be empty")
	}
	if fullName == "" {
		return nil, errors.New("full name cannot be empty")
	}

	// Hash the password
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user in repository
	user, err := s.userRepo.Create(ctx, email, fullName, hashedPassword)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Login authenticates a user and returns access/refresh tokens
func (s *UserService) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*domain.TokenPair, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	if err := auth.VerifyPassword(user.HashedPassword, password); err != nil {
		if errors.Is(err, auth.ErrInvalidPassword) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}

	// Generate access token
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token in database
	expiresAt := time.Now().Add(s.refreshTokenExpiry)
	_, err = s.refreshTokenRepo.Create(ctx, refreshToken, user.ID, expiresAt, ipAddress, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// RefreshToken generates a new token pair from a valid refresh token
func (s *UserService) RefreshToken(ctx context.Context, refreshToken, ipAddress, userAgent string) (*domain.TokenPair, error) {
	// Get refresh token from database
	tokenRecord, err := s.refreshTokenRepo.GetByToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	// Check if token is revoked
	if tokenRecord.RevokedAt != nil {
		return nil, domain.ErrRefreshTokenRevoked
	}

	// Check if token is expired
	if time.Now().After(tokenRecord.ExpiresAt) {
		return nil, domain.ErrRefreshTokenExpired
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, tokenRecord.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Revoke the old refresh token
	err = s.refreshTokenRepo.Revoke(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	// Generate new access token
	newAccessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate new refresh token
	newRefreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store new refresh token
	expiresAt := time.Now().Add(s.refreshTokenExpiry)
	_, err = s.refreshTokenRepo.Create(ctx, newRefreshToken, user.ID, expiresAt, ipAddress, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// Logout revokes a specific refresh token
func (s *UserService) Logout(ctx context.Context, refreshToken string) error {
	return s.refreshTokenRepo.Revoke(ctx, refreshToken)
}

// LogoutAll revokes all refresh tokens for a user
func (s *UserService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.refreshTokenRepo.RevokeAllForUser(ctx, userID)
}

// GetByID retrieves a user by ID
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// UpdateKYC updates a user's KYC status
func (s *UserService) UpdateKYC(ctx context.Context, id uuid.UUID, status domain.KYCStatus) (*domain.User, error) {
	return s.userRepo.UpdateKYCStatus(ctx, id, status)
}

// UpdateProfile updates a user's profile information
func (s *UserService) UpdateProfile(ctx context.Context, id uuid.UUID, fullName string) (*domain.User, error) {
	if fullName == "" {
		return nil, errors.New("full name cannot be empty")
	}
	return s.userRepo.UpdateProfile(ctx, id, fullName)
}

// DeleteAccount soft deletes a user account and revokes all tokens
func (s *UserService) DeleteAccount(ctx context.Context, id uuid.UUID) error {
	// Revoke all refresh tokens first
	err := s.refreshTokenRepo.RevokeAllForUser(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to revoke tokens: %w", err)
	}

	// Soft delete the user
	err = s.userRepo.SoftDelete(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

// GetActiveSessions retrieves all active sessions (refresh tokens) for a user
func (s *UserService) GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]*domain.RefreshToken, error) {
	return s.refreshTokenRepo.GetActiveTokensForUser(ctx, userID)
}
