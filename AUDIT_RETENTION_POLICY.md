# Audit Log Retention Policy

## Overview

The audit logging system includes automated retention management to prevent unbounded table growth while maintaining compliance with regulatory requirements.

## Retention Periods by Environment

| Environment | Retention Period | Rationale |
|-------------|------------------|-----------|
| **dev** | 30 days | Short retention for development/testing |
| **sandbox** | 90 days | Moderate retention for QA and testing cycles |
| **audit** | 7 years (2555 days) | Regulatory compliance (SOX, GDPR, etc.) |
| **prod** | 7 years (2555 days) | Regulatory compliance (SOX, GDPR, etc.) |

## Configuration

Configure retention in environment variables:

```bash
# How many days to retain audit logs
AUDIT_RETENTION_DAYS=90

# How often to run cleanup job (supports: 1h, 24h, 168h, etc.)
AUDIT_CLEANUP_INTERVAL=24h
```

### Recommended Settings

**Development:**
```bash
AUDIT_RETENTION_DAYS=30
AUDIT_CLEANUP_INTERVAL=24h
```

**Production:**
```bash
AUDIT_RETENTION_DAYS=2555  # 7 years
AUDIT_CLEANUP_INTERVAL=24h
```

**Compliance (Audit Environment):**
```bash
AUDIT_RETENTION_DAYS=2555  # 7 years minimum
AUDIT_CLEANUP_INTERVAL=24h
```

## How It Works

### 1. Retention Date Calculation

When an audit log is created, the system calculates a `retention_until` timestamp:

```go
retentionDate := time.Now().Add(time.Duration(retentionDays) * 24 * time.Hour)
```

This field is stored with each audit log entry and is **immutable**.

### 2. Automated Cleanup Job

The `AuditCleanupJob` runs periodically to delete expired audit logs:

- **Initial cleanup**: Runs immediately on service startup
- **Periodic cleanup**: Runs on schedule (default: every 24 hours)
- **Query used**: `DELETE FROM audit_logs WHERE retention_until < NOW()`

### 3. Safe Deletion

The cleanup process:
- Uses database-level WHERE clause (no application logic bugs)
- Executes within a 5-minute timeout context
- Logs all cleanup operations for monitoring
- Continues running even if individual cleanups fail

## Integration in Application

### Starting the Cleanup Job

In `cmd/user-service/main.go`:

```go
// Initialize audit repository
auditRepo := repository.NewAuditRepository(pool, logger)

// Create and start cleanup job
cleanupJob := service.NewAuditCleanupJob(
    auditRepo,
    logger,
    cfg.Audit.CleanupInterval,
)
cleanupJob.Start(ctx)

// Ensure graceful shutdown
defer cleanupJob.Stop()
```

### Manual Cleanup (Admin Operation)

For one-time cleanup operations:

```go
cleanupJob := service.NewAuditCleanupJob(auditRepo, logger, 24*time.Hour)
err := cleanupJob.RunOnce(ctx)
if err != nil {
    log.Error("Cleanup failed:", err)
}
```

## Monitoring

### Key Metrics to Monitor

1. **Cleanup Execution**: Check logs for `"Audit log cleanup completed successfully"`
2. **Cleanup Duration**: Monitor `duration_ms` field in cleanup logs
3. **Cleanup Errors**: Alert on `"audit log cleanup failed"` errors
4. **Table Growth**: Monitor `audit_logs` table size over time

### Sample Log Output

**Successful Cleanup:**
```json
{
  "level": "info",
  "service": "user-service",
  "message": "Audit log cleanup completed successfully",
  "duration_ms": 142
}
```

**Failed Cleanup:**
```json
{
  "level": "error",
  "service": "user-service",
  "error": "failed to delete expired audit logs: connection timeout",
  "message": "Scheduled audit log cleanup failed"
}
```

## Database Queries

### Check Current Retention Status

```sql
-- Count logs by retention status
SELECT 
    CASE 
        WHEN retention_until < NOW() THEN 'expired'
        WHEN retention_until < NOW() + INTERVAL '30 days' THEN 'expiring_soon'
        ELSE 'active'
    END AS status,
    COUNT(*) as count,
    MIN(retention_until) as earliest_expiry,
    MAX(retention_until) as latest_expiry
FROM audit_logs
GROUP BY status;
```

### View Oldest Audit Logs

```sql
SELECT 
    id,
    event_type,
    created_at,
    retention_until,
    AGE(retention_until, NOW()) as time_until_deletion
FROM audit_logs
ORDER BY created_at ASC
LIMIT 100;
```

### Manually Delete Expired Logs (Emergency)

```sql
-- Same query used by automated cleanup
DELETE FROM audit_logs 
WHERE retention_until < NOW();
```

## Compliance Considerations

### Regulatory Requirements

Different regulations have different retention requirements:

| Regulation | Minimum Retention | Applies To |
|------------|------------------|------------|
| **SOX** | 7 years | Financial records, access logs |
| **GDPR** | As needed, then delete | Personal data access logs |
| **HIPAA** | 6 years | Healthcare data access |
| **PCI DSS** | 1 year (3 months online) | Payment card data logs |

**Recommendation**: Use **7 years (2555 days)** in production to satisfy strictest requirements.

### Right to Erasure (GDPR)

Even with 7-year retention:
- Audit logs for deleted users are preserved (user_id becomes NULL via FK cascade)
- IP addresses and user agents remain for security analysis
- Pseudonymization ensures compliance with "right to be forgotten"

### Audit Log Export

Before deletion, consider archiving expired logs:

```bash
# Export expired logs before cleanup
psql -U pandora -d pandora_prod -c "
COPY (
    SELECT * FROM audit_logs 
    WHERE retention_until < NOW()
) TO '/backup/audit_logs_$(date +%Y%m%d).csv' CSV HEADER;
"
```

## Performance Considerations

### Index Optimization

The `idx_audit_logs_retention` index ensures fast cleanup:

```sql
CREATE INDEX idx_audit_logs_retention ON audit_logs(retention_until);
```

### Cleanup During Off-Hours

For large-scale production systems, consider running cleanup during low-traffic periods:

```bash
# Run cleanup at 3 AM daily (configure in cron or Kubernetes CronJob)
AUDIT_CLEANUP_INTERVAL=24h
```

### Batch Deletion

For very large tables (>100M rows), consider batch deletion:

```sql
-- Delete in batches of 10,000
DELETE FROM audit_logs 
WHERE id IN (
    SELECT id FROM audit_logs 
    WHERE retention_until < NOW() 
    LIMIT 10000
);
```

## Troubleshooting

### Cleanup Job Not Running

**Check logs for:**
- `"Starting audit log cleanup job"` - Job initialization
- `"Audit cleanup job stopped"` - Premature shutdown

**Common causes:**
- Context cancelled before job starts
- Application shutdown during cleanup
- Database connection issues

**Solution:**
```bash
# Verify configuration
echo $AUDIT_CLEANUP_INTERVAL

# Manually trigger cleanup
curl -X POST http://localhost:8081/admin/audit/cleanup
```

### Table Still Growing

**Verify retention dates:**
```sql
SELECT 
    MIN(retention_until) as earliest_expiry,
    COUNT(*) FILTER (WHERE retention_until < NOW()) as expired_count
FROM audit_logs;
```

**Check cleanup execution:**
```bash
# Search logs for cleanup operations
grep "Audit log cleanup" /var/log/user-service.log
```

### Performance Degradation

If cleanup takes too long:

1. **Check table size:**
```sql
SELECT pg_size_pretty(pg_total_relation_size('audit_logs'));
```

2. **Vacuum table:**
```sql
VACUUM ANALYZE audit_logs;
```

3. **Reindex retention index:**
```sql
REINDEX INDEX idx_audit_logs_retention;
```

## Testing

Run unit tests:
```bash
go test -v ./internal/service -run TestAuditCleanup
```

Run integration test with real database:
```bash
go test -v ./internal/repository -run TestAuditRepository_DeleteExpired
```

## Security

### Preventing Unauthorized Deletion

- Cleanup job runs with application service account
- No external API to trigger arbitrary deletions
- Retention dates are immutable once set
- All cleanup operations are logged in application logs

### Audit Trail of Cleanup

Every cleanup operation logs:
- Timestamp of cleanup
- Duration of operation
- Success/failure status
- Error details (if failed)

---

**Last Updated:** November 8, 2025  
**Related Files:**
- `internal/service/audit_cleanup_job.go`
- `internal/repository/audit_repository.go`
- `internal/config/config.go`
- `migrations/000005_create_audit_logs_table.up.sql`
