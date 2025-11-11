# Prometheus Metrics

## Overview

The Pandora Exchange user service exposes comprehensive Prometheus metrics for monitoring, alerting, and performance analysis. Metrics cover HTTP/gRPC operations, business events, database queries, cache operations, and error tracking.

## Metrics Endpoint

**URL**: `/metrics`  
**Method**: `GET`  
**Authentication**: Public (configure firewall rules to restrict access)

```bash
curl http://localhost:8080/metrics
```

## Metric Categories

### 1. HTTP Metrics

#### Request Metrics
```promql
# Total HTTP requests
pandora_user_service_http_requests_total{method="POST",path="/api/v1/users",status="2xx"}

# Request duration (histogram)
pandora_user_service_http_request_duration_seconds{method="POST",path="/api/v1/users",status="2xx"}

# Request size (histogram)
pandora_user_service_http_request_size_bytes{method="POST",path="/api/v1/users"}

# Response size (histogram)
pandora_user_service_http_response_size_bytes{method="POST",path="/api/v1/users",status="2xx"}
```

**Labels**:
- `method`: HTTP method (GET, POST, PUT, DELETE)
- `path`: Route pattern (e.g., `/users/:id`)
- `status`: Status code category (2xx, 3xx, 4xx, 5xx)

#### Connection Tracking
```promql
# Active HTTP connections
pandora_user_service_http_active_connections

# Idempotency cache hits
pandora_user_service_http_idempotency_cache_hits_total{path="/api/v1/users",method="POST"}
```

### 2. gRPC Metrics

```promql
# Total gRPC requests
pandora_user_service_grpc_requests_total{method="CreateUser",status="OK"}

# gRPC request duration
pandora_user_service_grpc_request_duration_seconds{method="CreateUser",status="OK"}

# Active gRPC connections
pandora_user_service_grpc_active_connections
```

**Labels**:
- `method`: gRPC method name
- `status`: gRPC status code (OK, INVALID_ARGUMENT, etc.)

### 3. Business Metrics

#### User Operations
```promql
# User registrations
pandora_user_service_user_registrations_total{status="success|failure"}

# User logins
pandora_user_service_user_logins_total{status="success|invalid_credentials|failure"}

# User logouts
pandora_user_service_user_logouts_total{type="single|all"}

# KYC submissions
pandora_user_service_kyc_submissions_total{status="pending|approved|rejected"}
```

#### Active Entities (Gauges)
```promql
# Active user sessions
pandora_user_service_active_sessions

# Active users
pandora_user_service_active_users
```

### 4. Authentication Metrics

```promql
# Token refresh operations
pandora_user_service_token_refresh_total{status="success|failure"}

# Token validation
pandora_user_service_token_validation_total{type="access|refresh",status="success|failure"}

# Token validation errors
pandora_user_service_token_validation_errors_total{error_type="expired|invalid_signature|malformed"}

# Password hashing duration
pandora_user_service_password_hash_duration_seconds
```

### 5. Database Metrics

```promql
# Database queries
pandora_user_service_db_queries_total{operation="select|insert|update|delete",table="users|refresh_tokens",status="success|error"}

# Query duration
pandora_user_service_db_query_duration_seconds{operation="select",table="users"}

# Connection pool metrics
pandora_user_service_db_connections_active
pandora_user_service_db_connections_idle
pandora_user_service_db_connections_waiting

# Transactions
pandora_user_service_db_transactions_total{status="commit|rollback"}
```

### 6. Cache Metrics (Redis)

```promql
# Cache hits
pandora_user_service_cache_hits_total{cache_type="redis|memory",operation="get|set|delete"}

# Cache misses
pandora_user_service_cache_misses_total{cache_type="redis",operation="get"}

# Cache operation duration
pandora_user_service_cache_operation_duration_seconds{cache_type="redis",operation="get"}

# Total cache keys
pandora_user_service_cache_keys_total
```

### 7. Error Metrics

```promql
# Total errors
pandora_user_service_errors_total{error_type="validation|database|authentication",component="http|grpc|db|service"}

# Panics recovered
pandora_user_service_panics_total{component="http|grpc"}
```

### 8. Audit Metrics

```promql
# Audit logs created
pandora_user_service_audit_logs_created_total{event_type="user.login",severity="info|warning|high|critical"}

# Audit log failures
pandora_user_service_audit_log_failures_total{error_type="creation_failed"}
```

## Common Queries

### Request Rate

```promql
# Overall request rate (requests/second)
rate(pandora_user_service_http_requests_total[5m])

# Request rate by endpoint
sum(rate(pandora_user_service_http_requests_total[5m])) by (path)

# Error rate
sum(rate(pandora_user_service_http_requests_total{status=~"4xx|5xx"}[5m])) by (path, status)
```

### Latency

```promql
# 95th percentile latency
histogram_quantile(0.95, rate(pandora_user_service_http_request_duration_seconds_bucket[5m]))

# 99th percentile latency by endpoint
histogram_quantile(0.99, sum(rate(pandora_user_service_http_request_duration_seconds_bucket[5m])) by (path, le))

# Average latency
rate(pandora_user_service_http_request_duration_seconds_sum[5m]) / rate(pandora_user_service_http_request_duration_seconds_count[5m])
```

### Error Tracking

```promql
# Error rate percentage
(sum(rate(pandora_user_service_http_requests_total{status=~"5xx"}[5m])) / sum(rate(pandora_user_service_http_requests_total[5m]))) * 100

# Failed login attempts
rate(pandora_user_service_user_logins_total{status="invalid_credentials"}[5m])

# Database errors
rate(pandora_user_service_db_queries_total{status="error"}[5m])
```

### Business Metrics

```promql
# Registration rate
rate(pandora_user_service_user_registrations_total{status="success"}[1h])

# Login success rate
(sum(rate(pandora_user_service_user_logins_total{status="success"}[5m])) / sum(rate(pandora_user_service_user_logins_total[5m]))) * 100

# Active sessions trend
pandora_user_service_active_sessions

# KYC approval rate
rate(pandora_user_service_kyc_submissions_total{status="approved"}[1d]) / rate(pandora_user_service_kyc_submissions_total[1d])
```

### Database Performance

```promql
# Slow queries (> 100ms)
histogram_quantile(0.95, rate(pandora_user_service_db_query_duration_seconds_bucket[5m])) > 0.1

# Query rate by operation
sum(rate(pandora_user_service_db_queries_total[5m])) by (operation)

# Connection pool saturation
pandora_user_service_db_connections_waiting > 0
```

### Cache Performance

```promql
# Cache hit rate
(sum(rate(pandora_user_service_cache_hits_total[5m])) / (sum(rate(pandora_user_service_cache_hits_total[5m])) + sum(rate(pandora_user_service_cache_misses_total[5m])))) * 100

# Cache miss rate
rate(pandora_user_service_cache_misses_total[5m])
```

## Alerting Rules

### Critical Alerts

```yaml
groups:
  - name: user_service_critical
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: (sum(rate(pandora_user_service_http_requests_total{status="5xx"}[5m])) / sum(rate(pandora_user_service_http_requests_total[5m]))) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value | humanizePercentage }} (threshold: 5%)"

      # Service down
      - alert: ServiceDown
        expr: up{job="user-service"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "User service is down"

      # High latency
      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(pandora_user_service_http_request_duration_seconds_bucket[5m])) > 1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High latency detected"
          description: "95th percentile latency is {{ $value }}s (threshold: 1s)"
```

### Warning Alerts

```yaml
  - name: user_service_warnings
    rules:
      # Database connection pool near saturation
      - alert: DatabaseConnectionPoolSaturation
        expr: pandora_user_service_db_connections_waiting > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Database connection pool near saturation"
          description: "{{ $value }} connections waiting"

      # High failed login rate
      - alert: HighFailedLoginRate
        expr: rate(pandora_user_service_user_logins_total{status="invalid_credentials"}[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High failed login rate detected"
          description: "{{ $value }} failed logins per second (possible brute force attack)"

      # Low cache hit rate
      - alert: LowCacheHitRate
        expr: (sum(rate(pandora_user_service_cache_hits_total[5m])) / (sum(rate(pandora_user_service_cache_hits_total[5m])) + sum(rate(pandora_user_service_cache_misses_total[5m])))) < 0.7
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "Low cache hit rate"
          description: "Cache hit rate is {{ $value | humanizePercentage }} (threshold: 70%)"
```

## Grafana Dashboards

### Dashboard Panels

#### 1. Overview Dashboard

**Metrics**:
- Total requests/sec
- Error rate
- Average latency
- Active users
- Active sessions

**Queries**:
```promql
# Total requests
sum(rate(pandora_user_service_http_requests_total[5m]))

# Error rate
(sum(rate(pandora_user_service_http_requests_total{status=~"5xx"}[5m])) / sum(rate(pandora_user_service_http_requests_total[5m]))) * 100

# Latency (95th percentile)
histogram_quantile(0.95, rate(pandora_user_service_http_request_duration_seconds_bucket[5m]))
```

#### 2. Business Metrics Dashboard

**Panels**:
- User registrations (last 24h)
- Login success rate
- KYC submissions by status
- Active users trend

#### 3. Database Dashboard

**Panels**:
- Query rate by operation
- Slow queries (p95 > 100ms)
- Connection pool status
- Transaction rate (commit/rollback)

#### 4. Cache Dashboard

**Panels**:
- Cache hit/miss rate
- Cache operation latency
- Total keys in cache

## Integration Examples

### HTTP Router Integration

```go
import (
	"github.com/alex-necsoiu/pandora-exchange/internal/middleware"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/alex-necsoiu/pandora-exchange/internal/transport/http"
)

// Create metrics collector
metrics := observability.NewMetricsCollector("pandora", "user_service")

// Setup router
router := gin.New()

// Add Prometheus middleware
router.Use(middleware.PrometheusMiddleware(metrics))

// Expose metrics endpoint
router.GET("/metrics", http.MetricsHandler())

// Your application routes
router.POST("/users", handler.CreateUser)
```

### Manual Instrumentation

```go
// Record business metric
metrics.RecordUserRegistration(true)

// Record database query
start := time.Now()
// ... execute query
metrics.RecordDBQuery("select", "users", "success", time.Since(start))

// Track active sessions
sessionCount := repo.CountActiveSessions(ctx)
metrics.UpdateActiveSessions(sessionCount)
```

## Best Practices

### 1. Label Cardinality

**Good** (low cardinality):
```promql
pandora_user_service_http_requests_total{method="POST",path="/users/:id",status="2xx"}
```

**Bad** (high cardinality):
```promql
# Don't use actual UUIDs as labels!
pandora_user_service_http_requests_total{user_id="550e8400-e29b-41d4-a716-446655440000"}
```

### 2. Metric Naming

- Use `_total` suffix for counters
- Use `_seconds` suffix for durations
- Use `_bytes` suffix for sizes
- No suffix for gauges

### 3. Recording Rules

For expensive queries, use recording rules:

```yaml
groups:
  - name: user_service_recording_rules
    interval: 30s
    rules:
      - record: job:pandora_user_service_http_request_duration_seconds:p95
        expr: histogram_quantile(0.95, rate(pandora_user_service_http_request_duration_seconds_bucket[5m]))
      
      - record: job:pandora_user_service_http_requests:rate5m
        expr: sum(rate(pandora_user_service_http_requests_total[5m])) by (path, method)
```

### 4. Retention

- Raw metrics: 15 days
- Recording rules (5m): 90 days
- Recording rules (1h): 1 year

## Troubleshooting

### High Cardinality Issues

**Symptom**: Prometheus storage growing rapidly

**Solution**: Check for labels with too many unique values
```promql
# Find metrics with most time series
topk(10, count by (__name__)({__name__=~".+"}))
```

### Missing Metrics

**Symptom**: Metrics not appearing in Prometheus

**Checks**:
1. Verify `/metrics` endpoint is accessible
2. Check Prometheus scrape config
3. Verify service discovery
4. Check Prometheus logs

```bash
# Test metrics endpoint
curl http://localhost:8080/metrics | grep pandora_user_service

# Check Prometheus targets
curl http://localhost:9090/api/v1/targets
```

### Memory Issues

**Symptom**: Service using excessive memory

**Solution**: Limit histogram buckets and reduce label cardinality

```go
// Use fewer, more focused buckets
Buckets: []float64{.001, .01, .1, 1, 10} // Instead of many buckets
```

## References

- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [Grafana Dashboard Examples](https://grafana.com/grafana/dashboards/)
- [PromQL Cheat Sheet](https://promlabs.com/promql-cheat-sheet/)
