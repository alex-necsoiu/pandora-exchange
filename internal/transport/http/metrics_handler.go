package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler returns a Gin handler for the Prometheus metrics endpoint
func MetricsHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// HealthCheckMetricsResponse is the response structure for health check with metrics
type HealthCheckMetricsResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
	Metrics string `json:"metrics"` // Link to metrics endpoint
}

// HealthCheckWithMetrics returns a health check handler that includes metrics link
func HealthCheckWithMetrics(version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, HealthCheckMetricsResponse{
			Status:  "healthy",
			Version: version,
			Metrics: "/metrics",
		})
	}
}
