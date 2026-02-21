package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/config"
	"github.com/alex-necsoiu/pandora-exchange/internal/domain/audit"
	"github.com/alex-necsoiu/pandora-exchange/internal/mocks"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAuditMiddleware_AuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("dev", "test-service")
	mockAuditRepo := new(mocks.MockAuditRepository)
	
	cfg := &config.Config{
		Audit: config.AuditConfig{
			RetentionDays: 30,
		},
	}

	// Mock expects audit log creation
	mockAuditRepo.On("Create", mock.Anything, mock.MatchedBy(func(log *audit.Log) bool {
		// Verify audit log has correct fields
		return log.EventType == "user.view" &&
			log.EventCategory == audit.CategoryDataAccess &&
			log.ActorType == audit.ActorUser &&
			log.Status == audit.StatusSuccess &&
			log.UserID != nil
	})).Return(&audit.Log{}, nil).Once()

	router := gin.New()
	router.Use(AuditMiddleware(mockAuditRepo, cfg, logger))
	
	userID := uuid.New()
	router.GET("/users/:id", func(c *gin.Context) {
		// Simulate AuthMiddleware setting user context
		c.Set("user_id", userID)
		c.Set("email", "test@example.com")
		c.Set("user_role", "user")
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/users/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	// Wait for async audit log creation
	time.Sleep(100 * time.Millisecond)
	mockAuditRepo.AssertExpectations(t)
}

func TestAuditMiddleware_AdminUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("dev", "test-service")
	mockAuditRepo := new(mocks.MockAuditRepository)
	
	cfg := &config.Config{
		Audit: config.AuditConfig{
			RetentionDays: 90,
		},
	}

	// Mock expects admin audit log with critical severity
	mockAuditRepo.On("Create", mock.Anything, mock.MatchedBy(func(log *audit.Log) bool {
		return log.ActorType == audit.ActorAdmin &&
			log.Severity == audit.SeverityCritical &&
			log.EventCategory == audit.CategorySecurity
	})).Return(&audit.Log{}, nil).Once()

	router := gin.New()
	router.Use(AuditMiddleware(mockAuditRepo, cfg, logger))
	
	adminID := uuid.New()
	router.GET("/admin/users", func(c *gin.Context) {
		c.Set("user_id", adminID)
		c.Set("email", "admin@example.com")
		c.Set("user_role", "admin")
		c.JSON(http.StatusOK, gin.H{"users": []string{}})
	})

	req := httptest.NewRequest("GET", "/admin/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	time.Sleep(100 * time.Millisecond)
	mockAuditRepo.AssertExpectations(t)
}

func TestAuditMiddleware_AnonymousRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("dev", "test-service")
	mockAuditRepo := new(mocks.MockAuditRepository)
	
	cfg := &config.Config{
		Audit: config.AuditConfig{
			RetentionDays: 30,
		},
	}

	// Mock expects anonymous audit log
	mockAuditRepo.On("Create", mock.Anything, mock.MatchedBy(func(log *audit.Log) bool {
		return log.ActorType == audit.ActorAPI &&
			log.UserID == nil &&
			log.ActorIdentifier != nil &&
			*log.ActorIdentifier == "anonymous"
	})).Return(&audit.Log{}, nil).Once()

	router := gin.New()
	router.Use(AuditMiddleware(mockAuditRepo, cfg, logger))
	
	router.POST("/auth/login", func(c *gin.Context) {
		// No user context set - anonymous request
		c.JSON(http.StatusOK, gin.H{"token": "fake-token"})
	})

	req := httptest.NewRequest("POST", "/auth/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	time.Sleep(100 * time.Millisecond)
	mockAuditRepo.AssertExpectations(t)
}

func TestAuditMiddleware_SkipHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("dev", "test-service")
	mockAuditRepo := new(mocks.MockAuditRepository)
	
	cfg := &config.Config{
		Audit: config.AuditConfig{
			RetentionDays: 30,
		},
	}

	// Should NOT be called for health check
	mockAuditRepo.On("Create", mock.Anything, mock.Anything).Return(&audit.Log{}, nil).Maybe()

	router := gin.New()
	router.Use(AuditMiddleware(mockAuditRepo, cfg, logger))
	
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	time.Sleep(100 * time.Millisecond)
	// Verify Create was NOT called
	mockAuditRepo.AssertNotCalled(t, "Create")
}

func TestAuditMiddleware_FailedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("dev", "test-service")
	mockAuditRepo := new(mocks.MockAuditRepository)
	
	cfg := &config.Config{
		Audit: config.AuditConfig{
			RetentionDays: 30,
		},
	}

	// Mock expects failure status
	mockAuditRepo.On("Create", mock.Anything, mock.MatchedBy(func(log *audit.Log) bool {
		return log.Status == audit.StatusFailure &&
			log.Severity == audit.SeverityWarning
	})).Return(&audit.Log{}, nil).Once()

	router := gin.New()
	router.Use(AuditMiddleware(mockAuditRepo, cfg, logger))
	
	router.GET("/users/:id", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
	})

	req := httptest.NewRequest("GET", "/users/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	
	time.Sleep(100 * time.Millisecond)
	mockAuditRepo.AssertExpectations(t)
}

func TestAuditMiddleware_ServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("dev", "test-service")
	mockAuditRepo := new(mocks.MockAuditRepository)
	
	cfg := &config.Config{
		Audit: config.AuditConfig{
			RetentionDays: 30,
		},
	}

	// Mock expects error status
	mockAuditRepo.On("Create", mock.Anything, mock.MatchedBy(func(log *audit.Log) bool {
		return log.Status == audit.StatusError
	})).Return(&audit.Log{}, nil).Once()

	router := gin.New()
	router.Use(AuditMiddleware(mockAuditRepo, cfg, logger))
	
	router.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
	})

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	time.Sleep(100 * time.Millisecond)
	mockAuditRepo.AssertExpectations(t)
}

func TestCategorizeRequest(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		method        string
		expectedType  string
		expectedCat   audit.EventCategory
	}{
		{
			name:         "user registration",
			path:         "/auth/register",
			method:       "POST",
			expectedType: "user.register",
			expectedCat:  audit.CategoryAuthentication,
		},
		{
			name:         "user login",
			path:         "/auth/login",
			method:       "POST",
			expectedType: "user.login",
			expectedCat:  audit.CategoryAuthentication,
		},
		{
			name:         "token refresh",
			path:         "/auth/refresh",
			method:       "POST",
			expectedType: "token.refresh",
			expectedCat:  audit.CategoryAuthentication,
		},
		{
			name:         "user logout",
			path:         "/auth/logout",
			method:       "POST",
			expectedType: "user.logout",
			expectedCat:  audit.CategoryAuthentication,
		},
		{
			name:         "KYC update",
			path:         "/users/123/kyc",
			method:       "PUT",
			expectedType: "user.kyc_update",
			expectedCat:  audit.CategoryCompliance,
		},
		{
			name:         "KYC view",
			path:         "/users/123/kyc",
			method:       "GET",
			expectedType: "user.kyc_view",
			expectedCat:  audit.CategoryDataAccess,
		},
		{
			name:         "user view",
			path:         "/users/123",
			method:       "GET",
			expectedType: "user.view",
			expectedCat:  audit.CategoryDataAccess,
		},
		{
			name:         "user update",
			path:         "/users/123",
			method:       "PUT",
			expectedType: "user.update",
			expectedCat:  audit.CategoryDataModification,
		},
		{
			name:         "user delete",
			path:         "/users/123",
			method:       "DELETE",
			expectedType: "user.delete",
			expectedCat:  audit.CategoryDataModification,
		},
		{
			name:         "admin endpoint",
			path:         "/admin/users",
			method:       "GET",
			expectedType: "admin.get",
			expectedCat:  audit.CategorySecurity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(tt.method, tt.path, nil)

			eventType, category := categorizeRequest(c)
			assert.Equal(t, tt.expectedType, eventType)
			assert.Equal(t, tt.expectedCat, category)
		})
	}
}

func TestDetermineSeverity(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		method           string
		statusCode       int
		expectedSeverity audit.Severity
	}{
		{
			name:             "admin action - critical",
			path:             "/admin/users",
			method:           "GET",
			statusCode:       200,
			expectedSeverity: audit.SeverityCritical,
		},
		{
			name:             "auth failure - high",
			path:             "/auth/login",
			method:           "POST",
			statusCode:       401,
			expectedSeverity: audit.SeverityHigh,
		},
		{
			name:             "KYC update - high",
			path:             "/users/123/kyc",
			method:           "PUT",
			statusCode:       200,
			expectedSeverity: audit.SeverityHigh,
		},
		{
			name:             "user deletion - high",
			path:             "/users/123",
			method:           "DELETE",
			statusCode:       200,
			expectedSeverity: audit.SeverityHigh,
		},
		{
			name:             "client error - warning",
			path:             "/users/123",
			method:           "GET",
			statusCode:       404,
			expectedSeverity: audit.SeverityWarning,
		},
		{
			name:             "successful operation - info",
			path:             "/users/123",
			method:           "GET",
			statusCode:       200,
			expectedSeverity: audit.SeverityInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(tt.method, tt.path, nil)
			
			// Set status code
			c.Writer.WriteHeader(tt.statusCode)

			severity := determineSeverity(c)
			assert.Equal(t, tt.expectedSeverity, severity)
		})
	}
}

func TestDetermineStatus(t *testing.T) {
	tests := []struct {
		statusCode     int
		expectedStatus audit.Status
	}{
		{200, audit.StatusSuccess},
		{201, audit.StatusSuccess},
		{204, audit.StatusSuccess},
		{400, audit.StatusFailure},
		{401, audit.StatusFailure},
		{404, audit.StatusFailure},
		{500, audit.StatusError},
		{502, audit.StatusError},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.statusCode)), func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Writer.WriteHeader(tt.statusCode)

			status := determineStatus(c)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestIsSensitiveEndpoint(t *testing.T) {
	tests := []struct {
		path        string
		isSensitive bool
	}{
		{"/auth/login", true},
		{"/auth/register", true},
		{"/users/123/kyc", true},
		{"/admin/users", true},
		{"/users/123", false},
		{"/health", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isSensitiveEndpoint(tt.path)
			assert.Equal(t, tt.isSensitive, result)
		})
	}
}

func TestShouldSkipAudit(t *testing.T) {
	tests := []struct {
		path       string
		shouldSkip bool
	}{
		{"/health", true},
		{"/metrics", true},
		{"/ready", true},
		{"/live", true},
		{"/users", false},
		{"/auth/login", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := shouldSkipAudit(tt.path)
			assert.Equal(t, tt.shouldSkip, result)
		})
	}
}

func TestAuditMiddleware_WithRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("dev", "test-service")
	mockAuditRepo := new(mocks.MockAuditRepository)
	
	cfg := &config.Config{
		Audit: config.AuditConfig{
			RetentionDays: 30,
		},
	}

	requestID := "test-request-123"

	// Mock expects audit log with request ID
	mockAuditRepo.On("Create", mock.Anything, mock.MatchedBy(func(log *audit.Log) bool {
		return log.RequestID != nil && *log.RequestID == requestID
	})).Return(&audit.Log{}, nil).Once()

	router := gin.New()
	router.Use(AuditMiddleware(mockAuditRepo, cfg, logger))
	
	router.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"users": []string{}})
	})

	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set("X-Request-ID", requestID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	time.Sleep(100 * time.Millisecond)
	mockAuditRepo.AssertExpectations(t)
}

func TestAuditMiddleware_CapturesIPAndUserAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("dev", "test-service")
	mockAuditRepo := new(mocks.MockAuditRepository)
	
	cfg := &config.Config{
		Audit: config.AuditConfig{
			RetentionDays: 30,
		},
	}

	userAgent := "Mozilla/5.0 Test Browser"

	// Mock expects audit log with IP and user agent
	mockAuditRepo.On("Create", mock.Anything, mock.MatchedBy(func(log *audit.Log) bool {
		return log.IPAddress != nil &&
			log.UserAgent != nil &&
			*log.UserAgent == userAgent
	})).Return(&audit.Log{}, nil).Once()

	router := gin.New()
	router.Use(AuditMiddleware(mockAuditRepo, cfg, logger))
	
	router.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"users": []string{}})
	})

	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set("User-Agent", userAgent)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	
	time.Sleep(100 * time.Millisecond)
	mockAuditRepo.AssertExpectations(t)
}

func TestAuditMiddleware_RetentionPeriod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("dev", "test-service")
	mockAuditRepo := new(mocks.MockAuditRepository)
	
	retentionDays := 365
	cfg := &config.Config{
		Audit: config.AuditConfig{
			RetentionDays: retentionDays,
		},
	}

	// Mock expects audit log with correct retention period
	mockAuditRepo.On("Create", mock.Anything, mock.MatchedBy(func(log *audit.Log) bool {
		if log.RetentionUntil == nil {
			return false
		}
		
		// Check retention is approximately retentionDays in the future
		expectedRetention := time.Now().Add(time.Duration(retentionDays) * 24 * time.Hour)
		diff := log.RetentionUntil.Sub(expectedRetention)
		
		// Allow 1 minute tolerance for test execution time
		return diff < time.Minute && diff > -time.Minute
	})).Return(&audit.Log{}, nil).Once()

	router := gin.New()
	router.Use(AuditMiddleware(mockAuditRepo, cfg, logger))
	
	router.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"users": []string{}})
	})

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	
	time.Sleep(100 * time.Millisecond)
	mockAuditRepo.AssertExpectations(t)
}
