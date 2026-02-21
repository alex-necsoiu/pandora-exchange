package audit

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents audit event types
type EventType string

const (
	// Audit-specific events
	EventTypeAuditLogged EventType = "audit.logged"
)

// Event represents an audit domain event
type Event struct {
	ID        string                 `json:"id"`        // Unique event ID
	Type      EventType              `json:"type"`      // Event type
	Timestamp time.Time              `json:"timestamp"` // When the event occurred
	AuditID   uuid.UUID              `json:"audit_id"`  // Audit log ID
	Payload   map[string]interface{} `json:"payload"`   // Event-specific data
	Metadata  map[string]string      `json:"metadata"`  // Additional metadata
}

// NewEvent creates a new audit domain event
func NewEvent(eventType EventType, auditID uuid.UUID, payload map[string]interface{}) *Event {
	return &Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		AuditID:   auditID,
		Payload:   payload,
		Metadata:  make(map[string]string),
	}
}

// WithMetadata adds metadata to the event
func (e *Event) WithMetadata(key, value string) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}
