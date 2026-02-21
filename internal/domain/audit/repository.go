package audit

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for audit log persistence
type Repository interface {
	// Create creates a new audit log entry (immutable)
	Create(ctx context.Context, log *Log) (*Log, error)

	// GetByID retrieves an audit log by ID
	GetByID(ctx context.Context, id uuid.UUID) (*Log, error)

	// ListByUser retrieves audit logs for a specific user
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*Log, error)

	// ListByEventType retrieves audit logs by event type
	ListByEventType(ctx context.Context, eventType string, limit, offset int32) ([]*Log, error)

	// ListByCategory retrieves audit logs by category
	ListByCategory(ctx context.Context, category EventCategory, limit, offset int32) ([]*Log, error)

	// ListByIPAddress retrieves audit logs from a specific IP
	ListByIPAddress(ctx context.Context, ipAddress string, limit, offset int32) ([]*Log, error)

	// ListByResource retrieves audit logs for a specific resource
	ListByResource(ctx context.Context, resourceType, resourceID string, limit, offset int32) ([]*Log, error)

	// Search performs a filtered search across audit logs
	Search(ctx context.Context, filter *Filter) ([]*Log, error)

	// CountByUser counts audit logs for a user
	CountByUser(ctx context.Context, userID uuid.UUID) (int64, error)

	// CountByEventType counts audit logs by event type
	CountByEventType(ctx context.Context, eventType string) (int64, error)

	// CountSearch counts results matching filter criteria
	CountSearch(ctx context.Context, filter *Filter) (int64, error)

	// GetRecentSecurityEvents retrieves recent high-severity security events
	GetRecentSecurityEvents(ctx context.Context) ([]*Log, error)

	// GetFailedLoginAttempts retrieves recent failed login attempts for a user
	GetFailedLoginAttempts(ctx context.Context, userID uuid.UUID) ([]*Log, error)

	// DeleteExpired removes audit logs past their retention period
	DeleteExpired(ctx context.Context) error
}
