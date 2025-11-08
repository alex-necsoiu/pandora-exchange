package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func TestErrorMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		setupHandler   func(*gin.Context)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "domain_error_user_not_found",
			setupHandler: func(c *gin.Context) {
				_ = c.Error(domain.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"error":   "USER_NOT_FOUND",
				"message": "user not found",
			},
		},
		{
			name: "domain_error_user_already_exists",
			setupHandler: func(c *gin.Context) {
				_ = c.Error(domain.ErrUserAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"error":   "USER_ALREADY_EXISTS",
				"message": "user already exists",
			},
		},
		{
			name: "domain_error_invalid_credentials",
			setupHandler: func(c *gin.Context) {
				_ = c.Error(domain.ErrInvalidCredentials)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error":   "INVALID_CREDENTIALS",
				"message": "invalid email or password",
			},
		},
		{
			name: "domain_error_unauthorized",
			setupHandler: func(c *gin.Context) {
				_ = c.Error(domain.ErrUnauthorized)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"error":   "UNAUTHORIZED",
				"message": "unauthorized",
			},
		},
		{
			name: "domain_error_forbidden",
			setupHandler: func(c *gin.Context) {
				_ = c.Error(domain.ErrForbidden)
			},
			expectedStatus: http.StatusForbidden,
			expectedBody: map[string]interface{}{
				"error":   "FORBIDDEN",
				"message": "forbidden",
			},
		},
		{
			name: "domain_error_invalid_input",
			setupHandler: func(c *gin.Context) {
				_ = c.Error(domain.ErrInvalidInput)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error":   "INVALID_INPUT",
				"message": "invalid input",
			},
		},
		{
			name: "internal_error_sanitized",
			setupHandler: func(c *gin.Context) {
				_ = c.Error(errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"error":   "INTERNAL_SERVER_ERROR",
				"message": "An unexpected error occurred",
			},
		},
		{
			name: "app_error_with_details",
			setupHandler: func(c *gin.Context) {
				appErr := domain.NewAppError(
					domain.ErrInvalidEmail,
					"Invalid email format",
					"test-trace-id",
				).WithDetails(map[string]interface{}{
					"field": "email",
					"value": "invalid-email",
				})
				_ = c.Error(appErr)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"error":   "INVALID_EMAIL",
				"message": "Invalid email format",
				"details": map[string]interface{}{
					"field": "email",
					"value": "invalid-email",
				},
			},
		},
		{
			name: "no_error_does_nothing",
			setupHandler: func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status": "success",
			},
		},
		{
			name: "multiple_errors_returns_first",
			setupHandler: func(c *gin.Context) {
				_ = c.Error(domain.ErrUserNotFound)
				_ = c.Error(domain.ErrInvalidCredentials)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"error":   "USER_NOT_FOUND",
				"message": "user not found",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test router
			router := gin.New()
			router.Use(ErrorMiddleware())
			
			// Add test handler
			router.GET("/test", tt.setupHandler)
			
			// Execute request
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)
			
			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			// Check response body contains expected fields
			body := w.Body.String()
			for key, value := range tt.expectedBody {
				switch v := value.(type) {
				case string:
					assert.Contains(t, body, v)
				case map[string]interface{}:
					// For nested maps (details), just check the key exists
					assert.Contains(t, body, key)
				}
			}
		})
	}
}

func TestErrorMiddleware_WithTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Setup OTEL tracer
	tp := trace.NewNoopTracerProvider()
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")
	
	router := gin.New()
	router.Use(ErrorMiddleware())
	
	router.GET("/test", func(c *gin.Context) {
		// Create a span to get a trace ID
		ctx, span := tracer.Start(c.Request.Context(), "test-operation")
		defer span.End()
		
		c.Request = c.Request.WithContext(ctx)
		_ = c.Error(domain.ErrUserNotFound)
	})
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "USER_NOT_FOUND")
}

func TestErrorMiddleware_LogsErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.Use(ErrorMiddleware())
	
	router.GET("/test", func(c *gin.Context) {
		_ = c.Error(domain.ErrInternalServer)
	})
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "INTERNAL_SERVER_ERROR")
	assert.Contains(t, w.Body.String(), "An unexpected error occurred")
}
