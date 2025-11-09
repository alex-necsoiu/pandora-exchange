package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/alex-necsoiu/pandora-exchange/internal/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RefreshTokenRepository implements domain.RefreshTokenRepository using sqlc-generated queries.
type RefreshTokenRepository struct {
	queries *postgres.Queries
	logger  *observability.Logger
}

// NewRefreshTokenRepository creates a new RefreshTokenRepository instance.
func NewRefreshTokenRepository(db postgres.DBTX, logger *observability.Logger) *RefreshTokenRepository {
	logger.Info("RefreshTokenRepository initialized")
	return &RefreshTokenRepository{
		queries: postgres.New(db),
		logger:  logger,
	}
}

// Create creates a new refresh token with audit information.
func (r *RefreshTokenRepository) Create(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time, ipAddress, userAgent string) (*domain.RefreshToken, error) {
	r.logger.WithFields(map[string]interface{}{
		"user_id":    userID,
		"ip_address": ipAddress,
	}).Debug("Creating refresh token")
	
	var ipPtr, uaPtr *string
	if ipAddress != "" {
		ipPtr = &ipAddress
	}
	if userAgent != "" {
		uaPtr = &userAgent
	}

	dbToken, err := r.queries.CreateRefreshToken(ctx, postgres.CreateRefreshTokenParams{
		Token:     token,
		UserID:    userID,
		ExpiresAt: timeToPgTimestamp(expiresAt),
		IpAddress: ipPtr,
		UserAgent: uaPtr,
	})
	if err != nil {
		r.logger.WithFields(map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		}).Error("Failed to create refresh token")
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	r.logger.WithField("user_id", userID).Info("Refresh token created successfully")
	return dbRefreshTokenToDomain(&dbToken), nil
}

// GetByToken retrieves a refresh token by its token value.
// Returns domain.ErrRefreshTokenNotFound if token doesn't exist.
func (r *RefreshTokenRepository) GetByToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	r.logger.Debug("Getting refresh token")
	
	dbToken, err := r.queries.GetRefreshToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Debug("Refresh token not found")
			return nil, domain.ErrRefreshTokenNotFound
		}
		r.logger.WithField("error", err.Error()).Error("Failed to get refresh token")
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return dbRefreshTokenToDomain(&dbToken), nil
}

// Revoke marks a refresh token as revoked.
// Returns domain.ErrRefreshTokenNotFound if token doesn't exist.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, token string) error {
	r.logger.Debug("Revoking refresh token")
	
	rowsAffected, err := r.queries.RevokeRefreshToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			r.logger.Debug("Refresh token not found for revocation")
			return domain.ErrRefreshTokenNotFound
		}
		r.logger.WithField("error", err.Error()).Error("Failed to revoke refresh token")
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.Debug("Refresh token not found for revocation (no rows affected)")
		return domain.ErrRefreshTokenNotFound
	}

	r.logger.Info("Refresh token revoked successfully")
	return nil
}

// RevokeAllForUser revokes all active refresh tokens for a specific user.
// Used when user logs out from all devices or changes password.
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	r.logger.WithField("user_id", userID).Debug("Revoking all user tokens")
	
	err := r.queries.RevokeAllUserTokens(ctx, userID)
	if err != nil {
		r.logger.WithFields(map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		}).Error("Failed to revoke all user tokens")
		return fmt.Errorf("failed to revoke all user tokens: %w", err)
	}

	r.logger.WithField("user_id", userID).Info("All user tokens revoked successfully")
	return nil
}

// GetActiveTokensForUser retrieves all active (non-revoked, non-expired) tokens for a user.
func (r *RefreshTokenRepository) GetActiveTokensForUser(ctx context.Context, userID uuid.UUID) ([]*domain.RefreshToken, error) {
	r.logger.WithField("user_id", userID).Debug("Getting active tokens for user")
	
	dbTokens, err := r.queries.GetUserActiveTokens(ctx, userID)
	if err != nil {
		r.logger.WithFields(map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		}).Error("Failed to get user active tokens")
		return nil, fmt.Errorf("failed to get user active tokens: %w", err)
	}

	tokens := make([]*domain.RefreshToken, len(dbTokens))
	for i, dbToken := range dbTokens {
		tokens[i] = dbRefreshTokenToDomain(&dbToken)
	}

	r.logger.WithFields(map[string]interface{}{
		"user_id": userID,
		"count":   len(tokens),
	}).Debug("Active tokens retrieved successfully")
	return tokens, nil
}

// CountActiveForUser returns the count of active tokens for a user.
// Used to enforce device/session limits.
func (r *RefreshTokenRepository) CountActiveForUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	r.logger.WithField("user_id", userID).Debug("Counting active tokens for user")
	
	count, err := r.queries.CountUserActiveTokens(ctx, userID)
	if err != nil {
		r.logger.WithFields(map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		}).Error("Failed to count user active tokens")
		return 0, fmt.Errorf("failed to count user active tokens: %w", err)
	}

	r.logger.WithFields(map[string]interface{}{
		"user_id": userID,
		"count":   count,
	}).Debug("Active tokens counted successfully")
	return count, nil
}

// DeleteExpired removes all expired refresh tokens from the database.
// Should be called periodically as a cleanup job.
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	r.logger.Debug("Deleting expired tokens")
	
	err := r.queries.DeleteExpiredTokens(ctx)
	if err != nil {
		r.logger.WithField("error", err.Error()).Error("Failed to delete expired tokens")
		return fmt.Errorf("failed to delete expired tokens: %w", err)
	}

	r.logger.Info("Expired tokens deleted successfully")
	return nil
}

// dbRefreshTokenToDomain converts a database RefreshToken model to a domain RefreshToken model.
func dbRefreshTokenToDomain(dbToken *postgres.RefreshToken) *domain.RefreshToken {
	token := &domain.RefreshToken{
		Token:     dbToken.Token,
		UserID:    dbToken.UserID,
		ExpiresAt: pgTimestampToTime(dbToken.ExpiresAt),
		CreatedAt: pgTimestampToTime(dbToken.CreatedAt),
	}

	// Handle optional revoked_at
	if dbToken.RevokedAt.Valid {
		revokedAt := pgTimestampToTime(dbToken.RevokedAt)
		token.RevokedAt = &revokedAt
	}

	// Handle optional ip_address
	if dbToken.IpAddress != nil {
		token.IPAddress = *dbToken.IpAddress
	}

	// Handle optional user_agent
	if dbToken.UserAgent != nil {
		token.UserAgent = *dbToken.UserAgent
	}

	return token
}

// timeToPgTimestamp converts time.Time to pgtype.Timestamptz.
func timeToPgTimestamp(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  t,
		Valid: !t.IsZero(),
	}
}

// GetAllActiveSessions retrieves all active sessions across all users with pagination.
// Admin-only operation for monitoring and audit purposes.
func (r *RefreshTokenRepository) GetAllActiveSessions(ctx context.Context, limit, offset int) ([]*domain.RefreshToken, error) {
	r.logger.WithFields(map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}).Debug("Getting all active sessions")

	// Validate pagination parameters to prevent integer overflow
	if limit < 0 || limit > 1000 {
		return nil, fmt.Errorf("invalid limit: must be between 0 and 1000")
	}
	if offset < 0 {
		return nil, fmt.Errorf("invalid offset: must be non-negative")
	}

	dbTokens, err := r.queries.GetAllActiveSessions(ctx, postgres.GetAllActiveSessionsParams{
		Limit:  int32(limit),  // #nosec G115 -- validated above
		Offset: int32(offset), // #nosec G115 -- validated above
	})
	if err != nil {
		r.logger.WithError(err).Error("Failed to get all active sessions")
		return nil, fmt.Errorf("failed to get all active sessions: %w", err)
	}

	tokens := make([]*domain.RefreshToken, len(dbTokens))
	for i, dbToken := range dbTokens {
		tokens[i] = &domain.RefreshToken{
			Token:     dbToken.Token,
			UserID:    dbToken.UserID,
			ExpiresAt: dbToken.ExpiresAt.Time,
			CreatedAt: dbToken.CreatedAt.Time,
			IPAddress: getStringValue(dbToken.IpAddress),
			UserAgent: getStringValue(dbToken.UserAgent),
		}
		if dbToken.RevokedAt.Valid {
			revokedAt := dbToken.RevokedAt.Time
			tokens[i].RevokedAt = &revokedAt
		}
	}

	r.logger.WithField("count", len(tokens)).Debug("All active sessions retrieved")
	return tokens, nil
}

// CountAllActiveSessions returns the total count of active sessions across all users.
func (r *RefreshTokenRepository) CountAllActiveSessions(ctx context.Context) (int64, error) {
	r.logger.Debug("Counting all active sessions")

	count, err := r.queries.CountAllActiveSessions(ctx)
	if err != nil {
		r.logger.WithError(err).Error("Failed to count all active sessions")
		return 0, fmt.Errorf("failed to count all active sessions: %w", err)
	}

	r.logger.WithField("count", count).Debug("All active sessions counted")
	return count, nil
}

// RevokeToken revokes a specific token by its value.
// Admin-only operation for force logout.
func (r *RefreshTokenRepository) RevokeToken(ctx context.Context, token string) error {
	r.logger.WithField("token", "[REDACTED]").Debug("Revoking specific token")

	rows, err := r.queries.RevokeTokenByID(ctx, token)
	if err != nil {
		r.logger.WithError(err).Error("Failed to revoke token")
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	if rows == 0 {
		r.logger.Debug("Token not found or already revoked")
		return domain.ErrTokenNotFound
	}

	r.logger.Info("Token revoked successfully")
	return nil
}

// getStringValue safely extracts string value from pointer.
func getStringValue(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
