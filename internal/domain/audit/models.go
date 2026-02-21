// Package audit contains the audit logging domain model and related types.
// This package follows Clean Architecture principles, remaining independent
// of infrastructure and transport concerns.
package audit

import (
	"time"

	"github.com/google/uuid"
)

// EventCategory represents high-level categorization of audit events
type EventCategory string

const (
	CategoryAuthentication   EventCategory = "authentication"
	CategoryAuthorization    EventCategory = "authorization"
	CategoryDataAccess       EventCategory = "data_access"
	CategoryDataModification EventCategory = "data_modification"
	CategorySecurity         EventCategory = "security"
	CategoryCompliance       EventCategory = "compliance"
)

// Severity represents the severity level of an audit event
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// ActorType represents the type of actor performing an action
type ActorType string

const (
	ActorUser   ActorType = "user"
	ActorSystem ActorType = "system"
	ActorAdmin  ActorType = "admin"
	ActorAPI    ActorType = "api"
)

// Status represents the outcome of an audited action
type Status string

const (
	StatusSuccess Status = "success"
	StatusFailure Status = "failure"
	StatusError   Status = "error"
)

// Log represents an immutable audit trail entry
type Log struct {
	ID uuid.UUID `json:"id"`

	// Event classification
	EventType     string        `json:"event_type"`
	EventCategory EventCategory `json:"event_category"`
	Severity      Severity      `json:"severity"`

	// Actor information
	UserID          *uuid.UUID `json:"user_id,omitempty"`
	ActorType       ActorType  `json:"actor_type"`
	ActorIdentifier *string    `json:"actor_identifier,omitempty"`

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
	Status        Status  `json:"status"`
	FailureReason *string `json:"failure_reason,omitempty"`

	// Compliance
	RetentionUntil *time.Time `json:"retention_until,omitempty"`
	IsSensitive    bool       `json:"is_sensitive"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
}

// Filter represents filter criteria for searching audit logs
type Filter struct {
	UserID        *uuid.UUID
	EventType     *string
	EventCategory *EventCategory
	Severity      *Severity
	StartDate     *time.Time
	EndDate       *time.Time
	Limit         int32
	Offset        int32
}
