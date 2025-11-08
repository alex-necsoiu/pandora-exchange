-- name: CreateAuditLog :one
INSERT INTO audit_logs (
    event_type,
    event_category,
    severity,
    user_id,
    actor_type,
    actor_identifier,
    action,
    resource_type,
    resource_id,
    ip_address,
    user_agent,
    request_id,
    session_id,
    metadata,
    previous_state,
    new_state,
    status,
    failure_reason,
    retention_until,
    is_sensitive
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
) RETURNING *;

-- name: GetAuditLogByID :one
SELECT * FROM audit_logs
WHERE id = $1;

-- name: ListAuditLogsByUser :many
SELECT * FROM audit_logs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByEventType :many
SELECT * FROM audit_logs
WHERE event_type = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByCategory :many
SELECT * FROM audit_logs
WHERE event_category = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByIPAddress :many
SELECT * FROM audit_logs
WHERE ip_address = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByResource :many
SELECT * FROM audit_logs
WHERE resource_type = $1 AND resource_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListAuditLogsByDateRange :many
SELECT * FROM audit_logs
WHERE created_at BETWEEN $1 AND $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListAuditLogsBySeverity :many
SELECT * FROM audit_logs
WHERE severity = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAuditLogsByUser :one
SELECT COUNT(*) FROM audit_logs
WHERE user_id = $1;

-- name: CountAuditLogsByEventType :one
SELECT COUNT(*) FROM audit_logs
WHERE event_type = $1;

-- name: CountAuditLogsByCategory :one
SELECT COUNT(*) FROM audit_logs
WHERE event_category = $1;

-- name: SearchAuditLogs :many
SELECT * FROM audit_logs
WHERE 
    ($1::uuid IS NULL OR user_id = $1) AND
    ($2::varchar IS NULL OR event_type = $2) AND
    ($3::varchar IS NULL OR event_category = $3) AND
    ($4::varchar IS NULL OR severity = $4) AND
    ($5::timestamptz IS NULL OR created_at >= $5) AND
    ($6::timestamptz IS NULL OR created_at <= $6)
ORDER BY created_at DESC
LIMIT $7 OFFSET $8;

-- name: CountSearchAuditLogs :one
SELECT COUNT(*) FROM audit_logs
WHERE 
    ($1::uuid IS NULL OR user_id = $1) AND
    ($2::varchar IS NULL OR event_type = $2) AND
    ($3::varchar IS NULL OR event_category = $3) AND
    ($4::varchar IS NULL OR severity = $4) AND
    ($5::timestamptz IS NULL OR created_at >= $5) AND
    ($6::timestamptz IS NULL OR created_at <= $6);

-- name: GetRecentSecurityEvents :many
SELECT * FROM audit_logs
WHERE event_category = 'security'
  AND severity IN ('high', 'critical')
  AND created_at >= NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC;

-- name: GetFailedLoginAttempts :many
SELECT * FROM audit_logs
WHERE event_type = 'user.login.failed'
  AND user_id = $1
  AND created_at >= NOW() - INTERVAL '1 hour'
ORDER BY created_at DESC;

-- name: DeleteExpiredAuditLogs :exec
DELETE FROM audit_logs
WHERE retention_until IS NOT NULL
  AND retention_until < NOW();
