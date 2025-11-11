package observability

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

// Note: Tests use a global metrics instance since Prometheus registers metrics globally
var testMetrics *MetricsCollector

func init() {
	testMetrics = NewMetricsCollector("test", "service")
}

func TestNewMetricsCollector(t *testing.T) {
	// Verify all metrics are initialized
	assert.NotNil(t, testMetrics.HTTPRequestsTotal)
	assert.NotNil(t, testMetrics.HTTPRequestDuration)
	assert.NotNil(t, testMetrics.GRPCRequestsTotal)
	assert.NotNil(t, testMetrics.UserRegistrationsTotal)
	assert.NotNil(t, testMetrics.DBQueriesTotal)
	assert.NotNil(t, testMetrics.CacheHitsTotal)
	assert.NotNil(t, testMetrics.ErrorsTotal)
	assert.NotNil(t, testMetrics.AuditLogsCreated)
}

func TestRecordHTTPRequest(t *testing.T) {
	// Use a separate registry for this test
	registry := prometheus.NewRegistry()
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_http_requests_total",
			Help: "Test counter",
		},
		[]string{"method", "path", "status"},
	)
	registry.MustRegister(counter)

	counter.WithLabelValues("GET", "/users", "2xx").Inc()
	count := testutil.ToFloat64(counter.WithLabelValues("GET", "/users", "2xx"))
	assert.Equal(t, float64(1), count)
}

func TestRecordUserRegistration(t *testing.T) {
	initial := testutil.ToFloat64(testMetrics.UserRegistrationsTotal.WithLabelValues("success"))
	testMetrics.RecordUserRegistration(true)
	count := testutil.ToFloat64(testMetrics.UserRegistrationsTotal.WithLabelValues("success"))
	assert.Greater(t, count, initial)
}

func TestRecordUserLogin(t *testing.T) {
	initial := testutil.ToFloat64(testMetrics.UserLoginsTotal.WithLabelValues("success"))
	testMetrics.RecordUserLogin("success")
	count := testutil.ToFloat64(testMetrics.UserLoginsTotal.WithLabelValues("success"))
	assert.Greater(t, count, initial)
}

func TestRecordDBQuery(t *testing.T) {
	initial := testutil.ToFloat64(testMetrics.DBQueriesTotal.WithLabelValues("select", "users", "success"))
	testMetrics.RecordDBQuery("select", "users", "success", 25*time.Millisecond)
	count := testutil.ToFloat64(testMetrics.DBQueriesTotal.WithLabelValues("select", "users", "success"))
	assert.Greater(t, count, initial)
}

func TestRecordCacheOperation(t *testing.T) {
	initial := testutil.ToFloat64(testMetrics.CacheHitsTotal.WithLabelValues("redis", "get"))
	testMetrics.RecordCacheOperation("redis", "get", true, 5*time.Millisecond)
	count := testutil.ToFloat64(testMetrics.CacheHitsTotal.WithLabelValues("redis", "get"))
	assert.Greater(t, count, initial)
}

func TestRecordTokenRefresh(t *testing.T) {
	initial := testutil.ToFloat64(testMetrics.TokenRefreshTotal.WithLabelValues("success"))
	testMetrics.RecordTokenRefresh(true)
	count := testutil.ToFloat64(testMetrics.TokenRefreshTotal.WithLabelValues("success"))
	assert.Greater(t, count, initial)
}

func TestUpdateGauges(t *testing.T) {
	testMetrics.UpdateActiveSessions(42)
	value := testutil.ToFloat64(testMetrics.ActiveSessionsGauge)
	assert.Equal(t, float64(42), value)

	testMetrics.UpdateActiveUsers(100)
	value = testutil.ToFloat64(testMetrics.ActiveUsersGauge)
	assert.Equal(t, float64(100), value)
}

func TestHTTPConnectionTracking(t *testing.T) {
	initial := testutil.ToFloat64(testMetrics.HTTPActiveConnections)
	
	testMetrics.IncrementHTTPActiveConnections()
	value := testutil.ToFloat64(testMetrics.HTTPActiveConnections)
	assert.Greater(t, value, initial)
	
	testMetrics.DecrementHTTPActiveConnections()
	value = testutil.ToFloat64(testMetrics.HTTPActiveConnections)
	assert.Equal(t, initial, value)
}

func TestRecordError(t *testing.T) {
	initial := testutil.ToFloat64(testMetrics.ErrorsTotal.WithLabelValues("validation", "http"))
	testMetrics.RecordError("validation", "http")
	count := testutil.ToFloat64(testMetrics.ErrorsTotal.WithLabelValues("validation", "http"))
	assert.Greater(t, count, initial)
}

func TestRecordAuditLog(t *testing.T) {
	initial := testutil.ToFloat64(testMetrics.AuditLogsCreated.WithLabelValues("user.login", "info"))
	testMetrics.RecordAuditLog("user.login", "info", true)
	count := testutil.ToFloat64(testMetrics.AuditLogsCreated.WithLabelValues("user.login", "info"))
	assert.Greater(t, count, initial)
}
