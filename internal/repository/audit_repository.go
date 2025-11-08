package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/alex-necsoiu/pandora-exchange/internal/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditRepository implements domain.AuditRepository using sqlc
type AuditRepository struct {
	pool    *pgxpool.Pool
	queries *postgres.Queries
	logger  *observability.Logger
}

// NewAuditRepository creates a new AuditRepository instance
func NewAuditRepository(pool *pgxpool.Pool, logger *observability.Logger) *AuditRepository {
	return &AuditRepository{
		pool:    pool,
		queries: postgres.New(pool),
		logger:  logger,
	}
}

// Create creates a new immutable audit log entry
func (r *AuditRepository) Create(ctx context.Context, log *domain.AuditLog) (*domain.AuditLog, error) {
	// Marshal JSONB fields
	metadataJSON, err := marshalJSON(log.Metadata)
	if err != nil {
		r.logger.WithError(err).Error("failed to marshal audit log metadata")
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	previousStateJSON, err := marshalJSON(log.PreviousState)
	if err != nil {
		r.logger.WithError(err).Error("failed to marshal previous state")
		return nil, fmt.Errorf("failed to marshal previous state: %w", err)
	}

	newStateJSON, err := marshalJSON(log.NewState)
	if err != nil {
		r.logger.WithError(err).Error("failed to marshal new state")
		return nil, fmt.Errorf("failed to marshal new state: %w", err)
	}

	// Build parameters
	params := postgres.CreateAuditLogParams{
		EventType:     log.EventType,
		EventCategory: string(log.EventCategory),
		Severity:      string(log.Severity),
		ActorType:     string(log.ActorType),
		Action:        log.Action,
		Status:        string(log.Status),
		Metadata:      metadataJSON,
		PreviousState: previousStateJSON,
		NewState:      newStateJSON,
	}

	// Handle optional UUID
	if log.UserID != nil {
		params.UserID = pgtype.UUID{Bytes: *log.UserID, Valid: true}
	}

	// Handle optional strings (pointers)
	params.ActorIdentifier = log.ActorIdentifier
	params.ResourceType = log.ResourceType
	params.ResourceID = log.ResourceID
	params.UserAgent = log.UserAgent
	params.RequestID = log.RequestID
	params.SessionID = log.SessionID
	params.FailureReason = log.FailureReason
	params.IsSensitive = &log.IsSensitive

	// Handle IP address conversion
	if log.IPAddress != nil {
		addr, err := netip.ParseAddr(*log.IPAddress)
		if err != nil {
			r.logger.WithError(err).WithField("ip", *log.IPAddress).Warn("invalid IP address format")
		} else {
			params.IpAddress = &addr
		}
	}

	// Handle retention timestamp
	if log.RetentionUntil != nil {
		params.RetentionUntil = pgtype.Timestamptz{Time: *log.RetentionUntil, Valid: true}
	}

	created, err := r.queries.CreateAuditLog(ctx, params)
	if err != nil {
		r.logger.WithError(err).WithField("event_type", log.EventType).Error("failed to create audit log")
		return nil, fmt.Errorf("failed to create audit log: %w", err)
	}

	return r.toDomainAuditLog(&created)
}

// GetByID retrieves an audit log by ID
func (r *AuditRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	log, err := r.queries.GetAuditLogByID(ctx, id)
	if err != nil {
		r.logger.WithError(err).WithField("id", id.String()).Error("failed to get audit log")
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	return r.toDomainAuditLog(&log)
}

// ListByUser retrieves audit logs for a specific user
func (r *AuditRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*domain.AuditLog, error) {
	logs, err := r.queries.ListAuditLogsByUser(ctx, postgres.ListAuditLogsByUserParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		r.logger.WithError(err).WithField("user_id", userID.String()).Error("failed to list audit logs by user")
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}

	return r.toDomainAuditLogs(logs)
}

// ListByEventType retrieves audit logs by event type
func (r *AuditRepository) ListByEventType(ctx context.Context, eventType string, limit, offset int32) ([]*domain.AuditLog, error) {
	logs, err := r.queries.ListAuditLogsByEventType(ctx, postgres.ListAuditLogsByEventTypeParams{
		EventType: eventType,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		r.logger.WithError(err).WithField("event_type", eventType).Error("failed to list audit logs by event type")
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}

	return r.toDomainAuditLogs(logs)
}

// ListByCategory retrieves audit logs by category
func (r *AuditRepository) ListByCategory(ctx context.Context, category domain.AuditEventCategory, limit, offset int32) ([]*domain.AuditLog, error) {
	logs, err := r.queries.ListAuditLogsByCategory(ctx, postgres.ListAuditLogsByCategoryParams{
		EventCategory: string(category),
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		r.logger.WithError(err).WithField("category", category).Error("failed to list audit logs by category")
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}

	return r.toDomainAuditLogs(logs)
}

// ListByIPAddress retrieves audit logs from a specific IP
func (r *AuditRepository) ListByIPAddress(ctx context.Context, ipAddress string, limit, offset int32) ([]*domain.AuditLog, error) {
	addr, err := netip.ParseAddr(ipAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid IP address: %w", err)
	}

	logs, err := r.queries.ListAuditLogsByIPAddress(ctx, postgres.ListAuditLogsByIPAddressParams{
		IpAddress: &addr,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		r.logger.WithError(err).WithField("ip_address", ipAddress).Error("failed to list audit logs by IP")
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}

	return r.toDomainAuditLogs(logs)
}

// ListByResource retrieves audit logs for a specific resource
func (r *AuditRepository) ListByResource(ctx context.Context, resourceType, resourceID string, limit, offset int32) ([]*domain.AuditLog, error) {
	logs, err := r.queries.ListAuditLogsByResource(ctx, postgres.ListAuditLogsByResourceParams{
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		r.logger.WithError(err).WithFields(map[string]interface{}{
			"resource_type": resourceType,
			"resource_id":   resourceID,
		}).Error("failed to list audit logs by resource")
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}

	return r.toDomainAuditLogs(logs)
}

// Search performs a filtered search across audit logs
func (r *AuditRepository) Search(ctx context.Context, filter *domain.AuditLogFilter) ([]*domain.AuditLog, error) {
	params := postgres.SearchAuditLogsParams{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	// The SearchAuditLogs query uses positional parameters with nullable types
	// Column1 = user_id, Column2 = event_type, Column3 = event_category, 
	// Column4 = severity, Column5 = start_date, Column6 = end_date
	if filter.UserID != nil {
		params.Column1 = *filter.UserID
	}
	if filter.EventType != nil {
		params.Column2 = *filter.EventType
	}
	if filter.EventCategory != nil {
		params.Column3 = string(*filter.EventCategory)
	}
	if filter.Severity != nil {
		params.Column4 = string(*filter.Severity)
	}
	if filter.StartDate != nil {
		params.Column5 = *filter.StartDate
	}
	if filter.EndDate != nil {
		params.Column6 = *filter.EndDate
	}

	logs, err := r.queries.SearchAuditLogs(ctx, params)
	if err != nil {
		r.logger.WithError(err).Error("failed to search audit logs")
		return nil, fmt.Errorf("failed to search audit logs: %w", err)
	}

	return r.toDomainAuditLogs(logs)
}

// CountByUser counts audit logs for a user
func (r *AuditRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := r.queries.CountAuditLogsByUser(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		r.logger.WithError(err).WithField("user_id", userID.String()).Error("failed to count audit logs")
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}
	return count, nil
}

// CountByEventType counts audit logs by event type
func (r *AuditRepository) CountByEventType(ctx context.Context, eventType string) (int64, error) {
	count, err := r.queries.CountAuditLogsByEventType(ctx, eventType)
	if err != nil {
		r.logger.WithError(err).WithField("event_type", eventType).Error("failed to count audit logs")
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}
	return count, nil
}

// CountSearch counts results matching filter criteria
func (r *AuditRepository) CountSearch(ctx context.Context, filter *domain.AuditLogFilter) (int64, error) {
	params := postgres.CountSearchAuditLogsParams{}

	if filter.UserID != nil {
		params.Column1 = *filter.UserID
	}
	if filter.EventType != nil {
		params.Column2 = *filter.EventType
	}
	if filter.EventCategory != nil {
		params.Column3 = string(*filter.EventCategory)
	}
	if filter.Severity != nil {
		params.Column4 = string(*filter.Severity)
	}
	if filter.StartDate != nil {
		params.Column5 = *filter.StartDate
	}
	if filter.EndDate != nil {
		params.Column6 = *filter.EndDate
	}

	count, err := r.queries.CountSearchAuditLogs(ctx, params)
	if err != nil {
		r.logger.WithError(err).Error("failed to count search audit logs")
		return 0, fmt.Errorf("failed to count search audit logs: %w", err)
	}
	return count, nil
}

// GetRecentSecurityEvents retrieves recent high-severity security events
func (r *AuditRepository) GetRecentSecurityEvents(ctx context.Context) ([]*domain.AuditLog, error) {
	logs, err := r.queries.GetRecentSecurityEvents(ctx)
	if err != nil {
		r.logger.WithError(err).Error("failed to get recent security events")
		return nil, fmt.Errorf("failed to get security events: %w", err)
	}

	return r.toDomainAuditLogs(logs)
}

// GetFailedLoginAttempts retrieves recent failed login attempts for a user
func (r *AuditRepository) GetFailedLoginAttempts(ctx context.Context, userID uuid.UUID) ([]*domain.AuditLog, error) {
	logs, err := r.queries.GetFailedLoginAttempts(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		r.logger.WithError(err).WithField("user_id", userID.String()).Error("failed to get failed login attempts")
		return nil, fmt.Errorf("failed to get failed login attempts: %w", err)
	}

	return r.toDomainAuditLogs(logs)
}

// DeleteExpired removes audit logs past their retention period
func (r *AuditRepository) DeleteExpired(ctx context.Context) error {
	err := r.queries.DeleteExpiredAuditLogs(ctx)
	if err != nil {
		r.logger.WithError(err).Error("failed to delete expired audit logs")
		return fmt.Errorf("failed to delete expired audit logs: %w", err)
	}
	return nil
}

// toDomainAuditLog converts sqlc AuditLog to domain.AuditLog
func (r *AuditRepository) toDomainAuditLog(log *postgres.AuditLog) (*domain.AuditLog, error) {
	domainLog := &domain.AuditLog{
		ID:              log.ID,
		EventType:       log.EventType,
		EventCategory:   domain.AuditEventCategory(log.EventCategory),
		Severity:        domain.AuditSeverity(log.Severity),
		ActorType:       domain.AuditActorType(log.ActorType),
		Action:          log.Action,
		Status:          domain.AuditStatus(log.Status),
		ActorIdentifier: log.ActorIdentifier,
		ResourceType:    log.ResourceType,
		ResourceID:      log.ResourceID,
		UserAgent:       log.UserAgent,
		RequestID:       log.RequestID,
		SessionID:       log.SessionID,
		FailureReason:   log.FailureReason,
		CreatedAt:       log.CreatedAt.Time,
	}

	// Handle optional UUID
	if log.UserID.Valid {
		userID := uuid.UUID(log.UserID.Bytes)
		domainLog.UserID = &userID
	}

	// Handle IP address
	if log.IpAddress != nil {
		ipStr := log.IpAddress.String()
		domainLog.IPAddress = &ipStr
	}

	// Handle retention
	if log.RetentionUntil.Valid {
		domainLog.RetentionUntil = &log.RetentionUntil.Time
	}

	// Handle sensitive flag
	if log.IsSensitive != nil {
		domainLog.IsSensitive = *log.IsSensitive
	}

	// Unmarshal JSONB fields
	if len(log.Metadata) > 0 && string(log.Metadata) != "null" {
		if err := json.Unmarshal(log.Metadata, &domainLog.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	if len(log.PreviousState) > 0 && string(log.PreviousState) != "null" {
		if err := json.Unmarshal(log.PreviousState, &domainLog.PreviousState); err != nil {
			return nil, fmt.Errorf("failed to unmarshal previous state: %w", err)
		}
	}
	if len(log.NewState) > 0 && string(log.NewState) != "null" {
		if err := json.Unmarshal(log.NewState, &domainLog.NewState); err != nil {
			return nil, fmt.Errorf("failed to unmarshal new state: %w", err)
		}
	}

	return domainLog, nil
}

// toDomainAuditLogs converts a slice of sqlc AuditLogs to domain.AuditLogs
func (r *AuditRepository) toDomainAuditLogs(logs []postgres.AuditLog) ([]*domain.AuditLog, error) {
	domainLogs := make([]*domain.AuditLog, 0, len(logs))
	for i := range logs {
		domainLog, err := r.toDomainAuditLog(&logs[i])
		if err != nil {
			return nil, err
		}
		domainLogs = append(domainLogs, domainLog)
	}
	return domainLogs, nil
}

// marshalJSON marshals data to JSON, returning empty JSON object for nil
func marshalJSON(data map[string]interface{}) ([]byte, error) {
	if data == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(data)
}
