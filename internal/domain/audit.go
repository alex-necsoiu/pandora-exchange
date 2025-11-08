package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AuditEventCategory represents high-level categorization of audit events
type AuditEventCategory string

const (
	AuditCategoryAuthentication  AuditEventCategory = "authentication"
	AuditCategoryAuthorization   AuditEventCategory = "authorization"
	AuditCategoryDataAccess      AuditEventCategory = "data_access"
	AuditCategoryDataModification AuditEventCategory = "data_modification"
	AuditCategorySecurity        AuditEventCategory = "security"
	AuditCategoryCompliance      AuditEventCategory = "compliance"
)

// AuditSeverity represents the severity level of an audit event
type AuditSeverity string

const (
	AuditSeverityInfo     AuditSeverity = "info"
	AuditSeverityWarning  AuditSeverity = "warning"
	AuditSeverityHigh     AuditSeverity = "high"
	AuditSeverityCritical AuditSeverity = "critical"
)

// AuditActorType represents the type of actor performing an action
type AuditActorType string

const (
	AuditActorUser   AuditActorType = "user"
	AuditActorSystem AuditActorType = "system"
	AuditActorAdmin  AuditActorType = "admin"
	AuditActorAPI    AuditActorType = "api"
)

// AuditStatus represents the outcome of an audited action
type AuditStatus string

const (
	AuditStatusSuccess AuditStatus = "success"
	AuditStatusFailure AuditStatus = "failure"
	AuditStatusError   AuditStatus = "error"
)

// AuditLog represents an immutable audit trail entry
type AuditLog struct {
	ID uuid.UUID `json:"id"`

	// Event classification
	EventType     string             `json:"event_type"`
	EventCategory AuditEventCategory `json:"event_category"`
	Severity      AuditSeverity      `json:"severity"`

	// Actor information
	UserID          *uuid.UUID     `json:"user_id,omitempty"`
	ActorType       AuditActorType `json:"actor_type"`
	ActorIdentifier *string        `json:"actor_identifier,omitempty"`

	// Action details
	Action       string  `json:"action"`
	ResourceType *string `json:"resource_type,omitempty"`
	ResourceID   *string `json:"resource_id,omitempty"`

	// Request context
	IPAddress *string `json:"ip_address,omitempty"`
	UserAgent *string `json:"user_agent,omitempty"`
	RequestID *string `json:"request_id,omitempty"`
	SessionID *string `json:"session_id,omitempty"`

	// Event payload
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	PreviousState map[string]interface{} `json:"previous_state,omitempty"`
	NewState      map[string]interface{} `json:"new_state,omitempty"`

	// Security
	Status        AuditStatus `json:"status"`
	FailureReason *string     `json:"failure_reason,omitempty"`

	// Compliance
	RetentionUntil *time.Time `json:"retention_until,omitempty"`
	IsSensitive    bool       `json:"is_sensitive"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
}

// AuditLogFilter represents filter criteria for searching audit logs
type AuditLogFilter struct {
	UserID        *uuid.UUID
	EventType     *string
	EventCategory *AuditEventCategory
	Severity      *AuditSeverity
	StartDate     *time.Time
	EndDate       *time.Time
	Limit         int32
	Offset        int32
}

// AuditRepository defines the interface for audit log persistence
type AuditRepository interface {
	// Create creates a new audit log entry (immutable)
	Create(ctx context.Context, log *AuditLog) (*AuditLog, error)

	// GetByID retrieves an audit log by ID
	GetByID(ctx context.Context, id uuid.UUID) (*AuditLog, error)

	// ListByUser retrieves audit logs for a specific user
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*AuditLog, error)

	// ListByEventType retrieves audit logs by event type
	ListByEventType(ctx context.Context, eventType string, limit, offset int32) ([]*AuditLog, error)

	// ListByCategory retrieves audit logs by category
	ListByCategory(ctx context.Context, category AuditEventCategory, limit, offset int32) ([]*AuditLog, error)

	// ListByIPAddress retrieves audit logs from a specific IP
	ListByIPAddress(ctx context.Context, ipAddress string, limit, offset int32) ([]*AuditLog, error)

	// ListByResource retrieves audit logs for a specific resource
	ListByResource(ctx context.Context, resourceType, resourceID string, limit, offset int32) ([]*AuditLog, error)

	// Search performs a filtered search across audit logs
	Search(ctx context.Context, filter *AuditLogFilter) ([]*AuditLog, error)

	// CountByUser counts audit logs for a user
	CountByUser(ctx context.Context, userID uuid.UUID) (int64, error)

	// CountByEventType counts audit logs by event type
	CountByEventType(ctx context.Context, eventType string) (int64, error)

	// CountSearch counts results matching filter criteria
	CountSearch(ctx context.Context, filter *AuditLogFilter) (int64, error)

	// GetRecentSecurityEvents retrieves recent high-severity security events
	GetRecentSecurityEvents(ctx context.Context) ([]*AuditLog, error)

	// GetFailedLoginAttempts retrieves recent failed login attempts for a user
	GetFailedLoginAttempts(ctx context.Context, userID uuid.UUID) ([]*AuditLog, error)

	// DeleteExpired removes audit logs past their retention period
	DeleteExpired(ctx context.Context) error
}
