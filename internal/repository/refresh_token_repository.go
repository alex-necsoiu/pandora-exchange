package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RefreshTokenRepository implements domain.RefreshTokenRepository using sqlc-generated queries.
type RefreshTokenRepository struct {
	queries *postgres.Queries
}

// NewRefreshTokenRepository creates a new RefreshTokenRepository instance.
func NewRefreshTokenRepository(db postgres.DBTX) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		queries: postgres.New(db),
	}
}

// Create creates a new refresh token with audit information.
func (r *RefreshTokenRepository) Create(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time, ipAddress, userAgent string) (*domain.RefreshToken, error) {
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
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return dbRefreshTokenToDomain(&dbToken), nil
}

// Get retrieves a refresh token by its token value.
// Returns domain.ErrRefreshTokenNotFound if token doesn't exist.
func (r *RefreshTokenRepository) Get(ctx context.Context, token string) (*domain.RefreshToken, error) {
	dbToken, err := r.queries.GetRefreshToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return dbRefreshTokenToDomain(&dbToken), nil
}

// Revoke marks a refresh token as revoked.
// Returns domain.ErrRefreshTokenNotFound if token doesn't exist.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, token string) error {
	rowsAffected, err := r.queries.RevokeRefreshToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return domain.ErrRefreshTokenNotFound
		}
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrRefreshTokenNotFound
	}

	return nil
}

// RevokeAllForUser revokes all active refresh tokens for a specific user.
// Used when user logs out from all devices or changes password.
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	err := r.queries.RevokeAllUserTokens(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all user tokens: %w", err)
	}

	return nil
}

// GetActiveForUser retrieves all active (non-revoked, non-expired) tokens for a user.
func (r *RefreshTokenRepository) GetActiveForUser(ctx context.Context, userID uuid.UUID) ([]*domain.RefreshToken, error) {
	dbTokens, err := r.queries.GetUserActiveTokens(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user active tokens: %w", err)
	}

	tokens := make([]*domain.RefreshToken, len(dbTokens))
	for i, dbToken := range dbTokens {
		tokens[i] = dbRefreshTokenToDomain(&dbToken)
	}

	return tokens, nil
}

// CountActiveForUser returns the count of active tokens for a user.
// Used to enforce device/session limits.
func (r *RefreshTokenRepository) CountActiveForUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := r.queries.CountUserActiveTokens(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to count user active tokens: %w", err)
	}

	return count, nil
}

// DeleteExpired removes all expired refresh tokens from the database.
// Should be called periodically as a cleanup job.
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	err := r.queries.DeleteExpiredTokens(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete expired tokens: %w", err)
	}

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
