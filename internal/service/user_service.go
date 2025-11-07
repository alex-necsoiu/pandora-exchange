package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
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
	logger             *observability.Logger
	auditLogger        *observability.AuditLogger
}

// NewUserService creates a new UserService instance
func NewUserService(
	userRepo domain.UserRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	jwtSecret string,
	accessTokenExpiry time.Duration,
	refreshTokenExpiry time.Duration,
	logger *observability.Logger,
) (*UserService, error) {
	jwtManager, err := auth.NewJWTManager(jwtSecret, accessTokenExpiry, refreshTokenExpiry)
	if err != nil {
		logger.WithError(err).Error("failed to create JWT manager")
		return nil, fmt.Errorf("failed to create JWT manager: %w", err)
	}

	auditLogger := observability.NewAuditLogger(logger)

	logger.Info("user service initialized successfully")

	return &UserService{
		userRepo:           userRepo,
		refreshTokenRepo:   refreshTokenRepo,
		jwtManager:         jwtManager,
		accessTokenExpiry:  accessTokenExpiry,
		refreshTokenExpiry: refreshTokenExpiry,
		logger:             logger,
		auditLogger:        auditLogger,
	}, nil
}

// Register creates a new user account
func (s *UserService) Register(ctx context.Context, email, password, firstName, lastName string) (*domain.User, error) {
	s.logger.WithField("email", email).Info("user registration started")

	if email == "" {
		s.logger.Error("registration failed: email cannot be empty")
		return nil, errors.New("email cannot be empty")
	}
	if password == "" {
		s.logger.Error("registration failed: password cannot be empty")
		return nil, errors.New("password cannot be empty")
	}
	if firstName == "" {
		s.logger.Error("registration failed: first name cannot be empty")
		return nil, errors.New("first name cannot be empty")
	}
	if lastName == "" {
		s.logger.Error("registration failed: last name cannot be empty")
		return nil, errors.New("last name cannot be empty")
	}

	// Hash the password
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		s.logger.WithError(err).WithField("email", email).Error("failed to hash password")
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user in repository
	user, err := s.userRepo.Create(ctx, email, firstName, lastName, hashedPassword)
	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			s.logger.WithField("email", email).Warn("registration failed: user already exists")
		} else {
			s.logger.WithError(err).WithField("email", email).Error("failed to create user in repository")
		}
		return nil, err
	}

	// Log audit event for user registration
	s.auditLogger.LogEvent("user.registered", map[string]interface{}{
		"user_id":    user.ID.String(),
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
	})

	s.logger.WithFields(map[string]interface{}{
		"user_id": user.ID.String(),
		"email":   user.Email,
	}).Info("user registered successfully")

	return user, nil
}

// Login authenticates a user and returns access/refresh tokens
func (s *UserService) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*domain.TokenPair, error) {
	s.logger.WithFields(map[string]interface{}{
		"email":      email,
		"ip_address": ipAddress,
		"user_agent": userAgent,
	}).Info("login attempt started")

	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			s.logger.WithField("email", email).Warn("login failed: user not found")
			return nil, domain.ErrInvalidCredentials
		}
		s.logger.WithError(err).WithField("email", email).Error("failed to get user from repository")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	if err := auth.VerifyPassword(user.HashedPassword, password); err != nil {
		if errors.Is(err, auth.ErrInvalidPassword) {
			s.logger.WithFields(map[string]interface{}{
				"user_id": user.ID.String(),
				"email":   email,
			}).Warn("login failed: invalid password")
			
			// Log security event for failed login
			s.auditLogger.LogSecurityEvent("login.failed", "medium", map[string]interface{}{
				"user_id":    user.ID.String(),
				"email":      email,
				"ip_address": ipAddress,
				"reason":     "invalid_password",
			})
			
			return nil, domain.ErrInvalidCredentials
		}
		s.logger.WithError(err).WithField("user_id", user.ID.String()).Error("password verification error")
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}

	// Generate access token
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, user.Role.String())
	if err != nil {
		s.logger.WithError(err).WithField("user_id", user.ID.String()).Error("failed to generate access token")
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", user.ID.String()).Error("failed to generate refresh token")
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token in database
	expiresAt := time.Now().Add(s.refreshTokenExpiry)
	_, err = s.refreshTokenRepo.Create(ctx, refreshToken, user.ID, expiresAt, ipAddress, userAgent)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", user.ID.String()).Error("failed to store refresh token")
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Log successful login as audit event
	s.auditLogger.LogEvent("user.login", map[string]interface{}{
		"user_id":    user.ID.String(),
		"email":      user.Email,
		"ip_address": ipAddress,
		"user_agent": userAgent,
	})

	s.logger.WithFields(map[string]interface{}{
		"user_id":    user.ID.String(),
		"email":      user.Email,
		"ip_address": ipAddress,
	}).Info("user logged in successfully")

	return &domain.TokenPair{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// RefreshToken generates a new token pair from a valid refresh token
func (s *UserService) RefreshToken(ctx context.Context, refreshToken, ipAddress, userAgent string) (*domain.TokenPair, error) {
	s.logger.WithField("ip_address", ipAddress).Info("token refresh attempt")

	// Get refresh token from database
	tokenRecord, err := s.refreshTokenRepo.GetByToken(ctx, refreshToken)
	if err != nil {
		s.logger.WithError(err).Warn("failed to get refresh token from database")
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	// Check if token is revoked
	if tokenRecord.RevokedAt != nil {
		s.logger.WithFields(map[string]interface{}{
			"user_id":    tokenRecord.UserID.String(),
			"revoked_at": tokenRecord.RevokedAt,
		}).Warn("refresh token is revoked")
		
		s.auditLogger.LogSecurityEvent("token.refresh.revoked", "medium", map[string]interface{}{
			"user_id":    tokenRecord.UserID.String(),
			"ip_address": ipAddress,
		})
		
		return nil, domain.ErrRefreshTokenRevoked
	}

	// Check if token is expired
	if time.Now().After(tokenRecord.ExpiresAt) {
		s.logger.WithFields(map[string]interface{}{
			"user_id":    tokenRecord.UserID.String(),
			"expires_at": tokenRecord.ExpiresAt,
		}).Warn("refresh token is expired")
		return nil, domain.ErrRefreshTokenExpired
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, tokenRecord.UserID)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", tokenRecord.UserID.String()).Error("failed to get user")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Revoke the old refresh token
	err = s.refreshTokenRepo.Revoke(ctx, refreshToken)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", user.ID.String()).Error("failed to revoke old refresh token")
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	// Generate new access token
	newAccessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, user.Role.String())
	if err != nil {
		s.logger.WithError(err).WithField("user_id", user.ID.String()).Error("failed to generate new access token")
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate new refresh token
	newRefreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", user.ID.String()).Error("failed to generate new refresh token")
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store new refresh token
	expiresAt := time.Now().Add(s.refreshTokenExpiry)
	_, err = s.refreshTokenRepo.Create(ctx, newRefreshToken, user.ID, expiresAt, ipAddress, userAgent)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", user.ID.String()).Error("failed to store new refresh token")
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	s.auditLogger.LogEvent("token.refreshed", map[string]interface{}{
		"user_id":    user.ID.String(),
		"ip_address": ipAddress,
	})

	s.logger.WithFields(map[string]interface{}{
		"user_id":    user.ID.String(),
		"ip_address": ipAddress,
	}).Info("token refreshed successfully")

	return &domain.TokenPair{
		User:         user,
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// Logout revokes a specific refresh token
func (s *UserService) Logout(ctx context.Context, refreshToken string) error {
	s.logger.Info("logout attempt")
	
	err := s.refreshTokenRepo.Revoke(ctx, refreshToken)
	if err != nil {
		s.logger.WithError(err).Error("failed to revoke refresh token during logout")
		return err
	}

	s.auditLogger.LogEvent("user.logout", map[string]interface{}{
		"token_revoked": true,
	})

	s.logger.Info("user logged out successfully")
	return nil
}

// LogoutAll revokes all refresh tokens for a user
func (s *UserService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	s.logger.WithField("user_id", userID.String()).Info("logout all sessions attempt")
	
	err := s.refreshTokenRepo.RevokeAllForUser(ctx, userID)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", userID.String()).Error("failed to revoke all tokens")
		return err
	}

	s.auditLogger.LogEvent("user.logout_all", map[string]interface{}{
		"user_id": userID.String(),
	})

	s.logger.WithField("user_id", userID.String()).Info("all sessions logged out successfully")
	return nil
}

// GetByID retrieves a user by ID
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	s.logger.WithField("user_id", id.String()).Debug("retrieving user by ID")
	
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			s.logger.WithField("user_id", id.String()).Warn("user not found")
		} else {
			s.logger.WithError(err).WithField("user_id", id.String()).Error("failed to get user by ID")
		}
		return nil, err
	}

	s.logger.WithField("user_id", id.String()).Debug("user retrieved successfully")
	return user, nil
}

// UpdateKYC updates a user's KYC status
func (s *UserService) UpdateKYC(ctx context.Context, id uuid.UUID, status domain.KYCStatus) (*domain.User, error) {
	s.logger.WithFields(map[string]interface{}{
		"user_id":    id.String(),
		"kyc_status": status,
	}).Info("KYC status update attempt")

	user, err := s.userRepo.UpdateKYCStatus(ctx, id, status)
	if err != nil {
		s.logger.WithError(err).WithFields(map[string]interface{}{
			"user_id":    id.String(),
			"kyc_status": status,
		}).Error("failed to update KYC status")
		return nil, err
	}

	// Log audit event for KYC status change
	s.auditLogger.LogEvent("user.kyc_updated", map[string]interface{}{
		"user_id":        id.String(),
		"new_kyc_status": status,
	})

	s.logger.WithFields(map[string]interface{}{
		"user_id":    id.String(),
		"kyc_status": status,
	}).Info("KYC status updated successfully")

	return user, nil
}

// UpdateProfile updates a user's profile information
func (s *UserService) UpdateProfile(ctx context.Context, id uuid.UUID, firstName, lastName string) (*domain.User, error) {
	s.logger.WithField("user_id", id.String()).Info("profile update attempt")

	if firstName == "" {
		s.logger.WithField("user_id", id.String()).Error("profile update failed: first name cannot be empty")
		return nil, errors.New("first name cannot be empty")
	}

	if lastName == "" {
		s.logger.WithField("user_id", id.String()).Error("profile update failed: last name cannot be empty")
		return nil, errors.New("last name cannot be empty")
	}

	user, err := s.userRepo.UpdateProfile(ctx, id, firstName, lastName)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", id.String()).Error("failed to update profile")
		return nil, err
	}

	s.auditLogger.LogEvent("user.profile_updated", map[string]interface{}{
		"user_id":    id.String(),
		"first_name": firstName,
		"last_name":  lastName,
	})

	s.logger.WithField("user_id", id.String()).Info("profile updated successfully")
	return user, nil
}

// DeleteAccount soft deletes a user account and revokes all tokens
func (s *UserService) DeleteAccount(ctx context.Context, id uuid.UUID) error {
	s.logger.WithField("user_id", id.String()).Info("account deletion attempt")

	// Revoke all refresh tokens first
	err := s.refreshTokenRepo.RevokeAllForUser(ctx, id)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", id.String()).Error("failed to revoke tokens during account deletion")
		return fmt.Errorf("failed to revoke tokens: %w", err)
	}

	// Soft delete the user
	err = s.userRepo.SoftDelete(ctx, id)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", id.String()).Error("failed to soft delete user account")
		return err
	}

	s.auditLogger.LogEvent("user.account_deleted", map[string]interface{}{
		"user_id": id.String(),
	})

	s.logger.WithField("user_id", id.String()).Info("account deleted successfully")
	return nil
}

// GetActiveSessions retrieves all active sessions (refresh tokens) for a user
func (s *UserService) GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]*domain.RefreshToken, error) {
	s.logger.WithField("user_id", userID.String()).Debug("retrieving active sessions")
	
	sessions, err := s.refreshTokenRepo.GetActiveTokensForUser(ctx, userID)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", userID.String()).Error("failed to get active sessions")
		return nil, err
	}

	s.logger.WithFields(map[string]interface{}{
		"user_id":       userID.String(),
		"session_count": len(sessions),
	}).Debug("active sessions retrieved")

	return sessions, nil
}

// ListUsers retrieves a paginated list of all users (admin only).
func (s *UserService) ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, int64, error) {
	s.logger.WithFields(map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}).Debug("Admin: listing users")

	users, err := s.userRepo.List(ctx, limit, offset)
	if err != nil {
		s.logger.WithError(err).Error("Failed to list users")
		return nil, 0, err
	}

	total, err := s.userRepo.Count(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to count users")
		return nil, 0, err
	}

	s.logger.WithFields(map[string]interface{}{
		"count": len(users),
		"total": total,
	}).Info("Admin: users listed successfully")

	return users, total, nil
}

// SearchUsers searches for users by email, first name, or last name (admin only).
func (s *UserService) SearchUsers(ctx context.Context, query string, limit, offset int) ([]*domain.User, error) {
	s.logger.WithFields(map[string]interface{}{
		"query":  query,
		"limit":  limit,
		"offset": offset,
	}).Debug("Admin: searching users")

	users, err := s.userRepo.SearchUsers(ctx, query, limit, offset)
	if err != nil {
		s.logger.WithError(err).Error("Failed to search users")
		return nil, err
	}

	s.logger.WithField("count", len(users)).Info("Admin: users search completed")
	return users, nil
}

// GetUserByIDAdmin retrieves a user by ID including deleted users (admin only).
func (s *UserService) GetUserByIDAdmin(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	s.logger.WithField("user_id", id).Debug("Admin: getting user by ID (include deleted)")

	user, err := s.userRepo.GetByIDIncludeDeleted(ctx, id)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", id).Error("Failed to get user")
		return nil, err
	}

	s.logger.WithField("user_id", id).Debug("Admin: user retrieved successfully")
	return user, nil
}

// UpdateUserRole updates a user's role (admin only).
func (s *UserService) UpdateUserRole(ctx context.Context, id uuid.UUID, role domain.Role) (*domain.User, error) {
	s.logger.WithFields(map[string]interface{}{
		"user_id": id,
		"role":    role,
	}).Info("Admin: updating user role")

	// Validate role
	if !role.IsValid() {
		return nil, domain.ErrInvalidRole
	}

	user, err := s.userRepo.UpdateRole(ctx, id, role)
	if err != nil {
		s.logger.WithError(err).Error("Failed to update user role")
		return nil, err
	}

	s.auditLogger.LogEvent("admin.user_role_updated", map[string]interface{}{
		"user_id": id.String(),
		"role":    role.String(),
	})

	s.logger.WithField("user_id", id).Info("Admin: user role updated successfully")
	return user, nil
}

// GetAllActiveSessions retrieves all active sessions across all users (admin only).
func (s *UserService) GetAllActiveSessions(ctx context.Context, limit, offset int) ([]*domain.RefreshToken, int64, error) {
	s.logger.WithFields(map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}).Debug("Admin: getting all active sessions")

	sessions, err := s.refreshTokenRepo.GetAllActiveSessions(ctx, limit, offset)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get all active sessions")
		return nil, 0, err
	}

	total, err := s.refreshTokenRepo.CountAllActiveSessions(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to count all active sessions")
		return nil, 0, err
	}

	s.logger.WithFields(map[string]interface{}{
		"count": len(sessions),
		"total": total,
	}).Info("Admin: all active sessions retrieved")

	return sessions, total, nil
}

// ForceLogout revokes a specific refresh token (admin only).
func (s *UserService) ForceLogout(ctx context.Context, token string) error {
	s.logger.Debug("Admin: forcing logout for token")

	err := s.refreshTokenRepo.RevokeToken(ctx, token)
	if err != nil {
		s.logger.WithError(err).Error("Failed to revoke token")
		return err
	}

	s.auditLogger.LogEvent("admin.force_logout", map[string]interface{}{
		"token": "[REDACTED]",
	})

	s.logger.Info("Admin: token revoked successfully (force logout)")
	return nil
}

// GetSystemStats retrieves system statistics for admin dashboard (admin only).
func (s *UserService) GetSystemStats(ctx context.Context) (map[string]interface{}, error) {
	s.logger.Debug("Admin: getting system statistics")

	totalUsers, err := s.userRepo.Count(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to count users")
		return nil, err
	}

	activeSessions, err := s.refreshTokenRepo.CountAllActiveSessions(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to count active sessions")
		return nil, err
	}

	stats := map[string]interface{}{
		"total_users":     totalUsers,
		"active_sessions": activeSessions,
	}

	s.logger.WithFields(stats).Info("Admin: system statistics retrieved")
	return stats, nil
}
