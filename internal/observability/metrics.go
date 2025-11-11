package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector holds all Prometheus metrics for the application
type MetricsCollector struct {
	// HTTP Metrics
	HTTPRequestsTotal        *prometheus.CounterVec
	HTTPRequestDuration      *prometheus.HistogramVec
	HTTPRequestSize          *prometheus.HistogramVec
	HTTPResponseSize         *prometheus.HistogramVec
	HTTPActiveConnections    prometheus.Gauge
	HTTPIdempotencyCacheHits *prometheus.CounterVec

	// gRPC Metrics
	GRPCRequestsTotal    *prometheus.CounterVec
	GRPCRequestDuration  *prometheus.HistogramVec
	GRPCActiveConnections prometheus.Gauge

	// Business Metrics - User Operations
	UserRegistrationsTotal *prometheus.CounterVec
	UserLoginsTotal        *prometheus.CounterVec
	UserLogoutsTotal       *prometheus.CounterVec
	ActiveSessionsGauge    prometheus.Gauge
	ActiveUsersGauge       prometheus.Gauge
	KYCSubmissionsTotal    *prometheus.CounterVec

	// Business Metrics - Authentication
	TokenRefreshTotal      *prometheus.CounterVec
	TokenValidationTotal   *prometheus.CounterVec
	TokenValidationErrors  *prometheus.CounterVec
	PasswordHashDuration   prometheus.Histogram

	// Database Metrics
	DBQueriesTotal         *prometheus.CounterVec
	DBQueryDuration        *prometheus.HistogramVec
	DBConnectionsActive    prometheus.Gauge
	DBConnectionsIdle      prometheus.Gauge
	DBConnectionsWaiting   prometheus.Gauge
	DBTransactionsTotal    *prometheus.CounterVec

	// Cache Metrics (Redis)
	CacheHitsTotal         *prometheus.CounterVec
	CacheMissesTotal       *prometheus.CounterVec
	CacheOperationDuration *prometheus.HistogramVec
	CacheKeysTotal         prometheus.Gauge

	// Error Metrics
	ErrorsTotal            *prometheus.CounterVec
	PanicsTotal            *prometheus.CounterVec

	// Audit Metrics
	AuditLogsCreated       *prometheus.CounterVec
	AuditLogFailures       *prometheus.CounterVec
}

// NewMetricsCollector creates and registers all Prometheus metrics
func NewMetricsCollector(namespace, subsystem string) *MetricsCollector {
	mc := &MetricsCollector{
		// HTTP Metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path", "status"},
		),

		HTTPRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_request_size_bytes",
				Help:      "HTTP request size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7), // 100B to 100MB
			},
			[]string{"method", "path"},
		),

		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "path", "status"},
		),

		HTTPActiveConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_active_connections",
				Help:      "Number of active HTTP connections",
			},
		),

		HTTPIdempotencyCacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "http_idempotency_cache_hits_total",
				Help:      "Total number of idempotency cache hits",
			},
			[]string{"path", "method"},
		),

		// gRPC Metrics
		GRPCRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "grpc_requests_total",
				Help:      "Total number of gRPC requests",
			},
			[]string{"method", "status"},
		),

		GRPCRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "grpc_request_duration_seconds",
				Help:      "gRPC request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			},
			[]string{"method", "status"},
		),

		GRPCActiveConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "grpc_active_connections",
				Help:      "Number of active gRPC connections",
			},
		),

		// Business Metrics - User Operations
		UserRegistrationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "user_registrations_total",
				Help:      "Total number of user registrations",
			},
			[]string{"status"}, // success, failure
		),

		UserLoginsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "user_logins_total",
				Help:      "Total number of user login attempts",
			},
			[]string{"status"}, // success, failure, invalid_credentials
		),

		UserLogoutsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "user_logouts_total",
				Help:      "Total number of user logouts",
			},
			[]string{"type"}, // single, all
		),

		ActiveSessionsGauge: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "active_sessions",
				Help:      "Current number of active user sessions",
			},
		),

		ActiveUsersGauge: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "active_users",
				Help:      "Current number of active users",
			},
		),

		KYCSubmissionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "kyc_submissions_total",
				Help:      "Total number of KYC submissions",
			},
			[]string{"status"}, // pending, approved, rejected
		),

		// Business Metrics - Authentication
		TokenRefreshTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "token_refresh_total",
				Help:      "Total number of token refresh operations",
			},
			[]string{"status"}, // success, failure
		),

		TokenValidationTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "token_validation_total",
				Help:      "Total number of token validation operations",
			},
			[]string{"type", "status"}, // type: access/refresh, status: success/failure
		),

		TokenValidationErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "token_validation_errors_total",
				Help:      "Total number of token validation errors",
			},
			[]string{"error_type"}, // expired, invalid_signature, malformed
		),

		PasswordHashDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "password_hash_duration_seconds",
				Help:      "Duration of password hashing operations",
				Buckets:   []float64{.01, .025, .05, .1, .25, .5, 1},
			},
		),

		// Database Metrics
		DBQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "db_queries_total",
				Help:      "Total number of database queries",
			},
			[]string{"operation", "table", "status"}, // operation: select/insert/update/delete
		),

		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
			},
			[]string{"operation", "table"},
		),

		DBConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "db_connections_active",
				Help:      "Number of active database connections",
			},
		),

		DBConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "db_connections_idle",
				Help:      "Number of idle database connections",
			},
		),

		DBConnectionsWaiting: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "db_connections_waiting",
				Help:      "Number of connections waiting for a database connection",
			},
		),

		DBTransactionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "db_transactions_total",
				Help:      "Total number of database transactions",
			},
			[]string{"status"}, // commit, rollback
		),

		// Cache Metrics
		CacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache_type", "operation"}, // cache_type: redis/memory, operation: get/set/delete
		),

		CacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache_type", "operation"},
		),

		CacheOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cache_operation_duration_seconds",
				Help:      "Cache operation duration in seconds",
				Buckets:   []float64{.0001, .0005, .001, .005, .01, .025, .05, .1},
			},
			[]string{"cache_type", "operation"},
		),

		CacheKeysTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "cache_keys_total",
				Help:      "Total number of keys in cache",
			},
		),

		// Error Metrics
		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "errors_total",
				Help:      "Total number of errors",
			},
			[]string{"error_type", "component"}, // component: http/grpc/db/service
		),

		PanicsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "panics_total",
				Help:      "Total number of panics recovered",
			},
			[]string{"component"},
		),

		// Audit Metrics
		AuditLogsCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "audit_logs_created_total",
				Help:      "Total number of audit logs created",
			},
			[]string{"event_type", "severity"},
		),

		AuditLogFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "audit_log_failures_total",
				Help:      "Total number of audit log creation failures",
			},
			[]string{"error_type"},
		),
	}

	return mc
}

// RecordHTTPRequest records HTTP request metrics
func (mc *MetricsCollector) RecordHTTPRequest(method, path, status string, duration time.Duration, requestSize, responseSize int) {
	mc.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	mc.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration.Seconds())
	mc.HTTPRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
	mc.HTTPResponseSize.WithLabelValues(method, path, status).Observe(float64(responseSize))
}

// RecordGRPCRequest records gRPC request metrics
func (mc *MetricsCollector) RecordGRPCRequest(method, status string, duration time.Duration) {
	mc.GRPCRequestsTotal.WithLabelValues(method, status).Inc()
	mc.GRPCRequestDuration.WithLabelValues(method, status).Observe(duration.Seconds())
}

// RecordDBQuery records database query metrics
func (mc *MetricsCollector) RecordDBQuery(operation, table, status string, duration time.Duration) {
	mc.DBQueriesTotal.WithLabelValues(operation, table, status).Inc()
	mc.DBQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordCacheOperation records cache operation metrics
func (mc *MetricsCollector) RecordCacheOperation(cacheType, operation string, hit bool, duration time.Duration) {
	mc.CacheOperationDuration.WithLabelValues(cacheType, operation).Observe(duration.Seconds())
	
	if hit {
		mc.CacheHitsTotal.WithLabelValues(cacheType, operation).Inc()
	} else {
		mc.CacheMissesTotal.WithLabelValues(cacheType, operation).Inc()
	}
}

// RecordUserRegistration records user registration metrics
func (mc *MetricsCollector) RecordUserRegistration(success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	mc.UserRegistrationsTotal.WithLabelValues(status).Inc()
}

// RecordUserLogin records user login metrics
func (mc *MetricsCollector) RecordUserLogin(status string) {
	mc.UserLoginsTotal.WithLabelValues(status).Inc()
}

// RecordUserLogout records user logout metrics
func (mc *MetricsCollector) RecordUserLogout(logoutType string) {
	mc.UserLogoutsTotal.WithLabelValues(logoutType).Inc()
}

// RecordKYCSubmission records KYC submission metrics
func (mc *MetricsCollector) RecordKYCSubmission(status string) {
	mc.KYCSubmissionsTotal.WithLabelValues(status).Inc()
}

// RecordTokenRefresh records token refresh metrics
func (mc *MetricsCollector) RecordTokenRefresh(success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	mc.TokenRefreshTotal.WithLabelValues(status).Inc()
}

// RecordTokenValidation records token validation metrics
func (mc *MetricsCollector) RecordTokenValidation(tokenType, status string) {
	mc.TokenValidationTotal.WithLabelValues(tokenType, status).Inc()
}

// RecordTokenValidationError records token validation error metrics
func (mc *MetricsCollector) RecordTokenValidationError(errorType string) {
	mc.TokenValidationErrors.WithLabelValues(errorType).Inc()
}

// RecordPasswordHash records password hashing duration
func (mc *MetricsCollector) RecordPasswordHash(duration time.Duration) {
	mc.PasswordHashDuration.Observe(duration.Seconds())
}

// RecordError records error metrics
func (mc *MetricsCollector) RecordError(errorType, component string) {
	mc.ErrorsTotal.WithLabelValues(errorType, component).Inc()
}

// RecordPanic records panic metrics
func (mc *MetricsCollector) RecordPanic(component string) {
	mc.PanicsTotal.WithLabelValues(component).Inc()
}

// RecordAuditLog records audit log creation metrics
func (mc *MetricsCollector) RecordAuditLog(eventType, severity string, success bool) {
	if success {
		mc.AuditLogsCreated.WithLabelValues(eventType, severity).Inc()
	} else {
		mc.AuditLogFailures.WithLabelValues("creation_failed").Inc()
	}
}

// UpdateActiveSessions updates the active sessions gauge
func (mc *MetricsCollector) UpdateActiveSessions(count int) {
	mc.ActiveSessionsGauge.Set(float64(count))
}

// UpdateActiveUsers updates the active users gauge
func (mc *MetricsCollector) UpdateActiveUsers(count int) {
	mc.ActiveUsersGauge.Set(float64(count))
}

// UpdateDBConnections updates database connection pool metrics
func (mc *MetricsCollector) UpdateDBConnections(active, idle, waiting int) {
	mc.DBConnectionsActive.Set(float64(active))
	mc.DBConnectionsIdle.Set(float64(idle))
	mc.DBConnectionsWaiting.Set(float64(waiting))
}

// UpdateCacheKeys updates the cache keys gauge
func (mc *MetricsCollector) UpdateCacheKeys(count int) {
	mc.CacheKeysTotal.Set(float64(count))
}

// IncrementHTTPActiveConnections increments HTTP active connections
func (mc *MetricsCollector) IncrementHTTPActiveConnections() {
	mc.HTTPActiveConnections.Inc()
}

// DecrementHTTPActiveConnections decrements HTTP active connections
func (mc *MetricsCollector) DecrementHTTPActiveConnections() {
	mc.HTTPActiveConnections.Dec()
}

// IncrementGRPCActiveConnections increments gRPC active connections
func (mc *MetricsCollector) IncrementGRPCActiveConnections() {
	mc.GRPCActiveConnections.Inc()
}

// DecrementGRPCActiveConnections decrements gRPC active connections
func (mc *MetricsCollector) DecrementGRPCActiveConnections() {
	mc.GRPCActiveConnections.Dec()
}

// RecordDBTransaction records database transaction metrics
func (mc *MetricsCollector) RecordDBTransaction(committed bool) {
	status := "commit"
	if !committed {
		status = "rollback"
	}
	mc.DBTransactionsTotal.WithLabelValues(status).Inc()
}

// RecordIdempotencyCacheHit records idempotency cache hit
func (mc *MetricsCollector) RecordIdempotencyCacheHit(path, method string) {
	mc.HTTPIdempotencyCacheHits.WithLabelValues(path, method).Inc()
}
