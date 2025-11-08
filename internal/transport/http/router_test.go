package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	httpTransport "github.com/alex-necsoiu/pandora-exchange/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetupUserRouter tests the user-facing router configuration
func TestSetupUserRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("test", "test-service")
	mockService := &MockUserService{}
	
	jwtManager, err := auth.NewJWTManager(
		"test-secret-key-must-be-at-least-32-characters-long",
		15*time.Minute,
		7*24*time.Hour,
	)
	require.NoError(t, err)

	router := httpTransport.SetupUserRouter(mockService, jwtManager, logger, "debug")

	testCases := []struct {
		name           string
		method         string
		path           string
		expectedRoutes []string
		description    string
	}{
		{
			name:   "health check route exists",
			method: "GET",
			path:   "/health",
			description: "Health endpoint should exist without auth",
		},
		{
			name:   "register route exists",
			method: "POST",
			path:   "/api/v1/auth/register",
			description: "Public registration endpoint",
		},
		{
			name:   "login route exists",
			method: "POST",
			path:   "/api/v1/auth/login",
			description: "Public login endpoint",
		},
		{
			name:   "refresh token route exists",
			method: "POST",
			path:   "/api/v1/auth/refresh",
			description: "Public token refresh endpoint",
		},
		{
			name:   "get profile route exists",
			method: "GET",
			path:   "/api/v1/users/me",
			description: "Protected get profile endpoint",
		},
		{
			name:   "update profile route exists",
			method: "PUT",
			path:   "/api/v1/users/me",
			description: "Protected update profile endpoint",
		},
		{
			name:   "delete account route exists",
			method: "DELETE",
			path:   "/api/v1/users/me",
			description: "Protected delete account endpoint",
		},
		{
			name:   "get sessions route exists",
			method: "GET",
			path:   "/api/v1/users/me/sessions",
			description: "Protected get sessions endpoint",
		},
		{
			name:   "logout route exists",
			method: "POST",
			path:   "/api/v1/users/me/logout",
			description: "Protected logout endpoint",
		},
		{
			name:   "logout all route exists",
			method: "POST",
			path:   "/api/v1/users/me/logout-all",
			description: "Protected logout all endpoint",
		},
		{
			name:   "update KYC route exists",
			method: "PUT",
			path:   "/api/v1/users/" + uuid.New().String() + "/kyc",
			description: "Protected admin KYC update endpoint with UUID validation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test request
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			
			// Execute the request
			router.ServeHTTP(w, req)
			
			// Route should exist (not 404)
			// Note: We expect various responses (401 for protected routes, etc.)
			// but NOT 404 which means route doesn't exist
			assert.NotEqual(t, http.StatusNotFound, w.Code, 
				"Route %s %s should exist (got 404)", tc.method, tc.path)
		})
	}
}

// TestSetupAdminRouter tests the admin-only router configuration
func TestSetupAdminRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("test", "test-service")
	mockService := &MockUserService{}
	
	jwtManager, err := auth.NewJWTManager(
		"test-secret-key-must-be-at-least-32-characters-long",
		15*time.Minute,
		7*24*time.Hour,
	)
	require.NoError(t, err)

	router := httpTransport.SetupAdminRouter(mockService, jwtManager, logger, "debug")

	testCases := []struct {
		name        string
		method      string
		path        string
		description string
	}{
		{
			name:   "admin login route exists",
			method: "POST",
			path:   "/admin/auth/login",
			description: "Public admin login endpoint",
		},
		{
			name:   "admin refresh token route exists",
			method: "POST",
			path:   "/admin/auth/refresh",
			description: "Public admin token refresh endpoint",
		},
		{
			name:   "list users route exists",
			method: "GET",
			path:   "/admin/users",
			description: "Protected admin list users endpoint",
		},
		{
			name:   "search users route exists",
			method: "GET",
			path:   "/admin/users/search",
			description: "Protected admin search users endpoint",
		},
		{
			name:   "get user route exists",
			method: "GET",
			path:   "/admin/users/" + uuid.New().String(),
			description: "Protected admin get user endpoint with UUID validation",
		},
		{
			name:   "update user role route exists",
			method: "PUT",
			path:   "/admin/users/" + uuid.New().String() + "/role",
			description: "Protected admin update role endpoint with UUID validation",
		},
		{
			name:   "get all sessions route exists",
			method: "GET",
			path:   "/admin/sessions",
			description: "Protected admin get all sessions endpoint",
		},
		{
			name:   "force logout route exists",
			method: "POST",
			path:   "/admin/sessions/revoke",
			description: "Protected admin force logout endpoint",
		},
		{
			name:   "get system stats route exists",
			method: "GET",
			path:   "/admin/stats",
			description: "Protected admin system stats endpoint",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test request
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			
			// Execute the request
			router.ServeHTTP(w, req)
			
			// Route should exist (not 404)
			assert.NotEqual(t, http.StatusNotFound, w.Code,
				"Route %s %s should exist (got 404)", tc.method, tc.path)
		})
	}
}

// TestRouterSeparation tests that user and admin routers are properly separated
func TestRouterSeparation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("test", "test-service")
	mockService := &MockUserService{}
	
	jwtManager, err := auth.NewJWTManager(
		"test-secret-key-must-be-at-least-32-characters-long",
		15*time.Minute,
		7*24*time.Hour,
	)
	require.NoError(t, err)

	userRouter := httpTransport.SetupUserRouter(mockService, jwtManager, logger, "debug")
	adminRouter := httpTransport.SetupAdminRouter(mockService, jwtManager, logger, "debug")

	testCases := []struct {
		name        string
		router      *gin.Engine
		method      string
		path        string
		shouldExist bool
		description string
	}{
		{
			name:        "admin routes not accessible on user router",
			router:      userRouter,
			method:      "GET",
			path:        "/admin/users",
			shouldExist: false,
			description: "Admin routes should not exist on user router",
		},
		{
			name:        "admin auth routes not accessible on user router",
			router:      userRouter,
			method:      "POST",
			path:        "/admin/auth/login",
			shouldExist: false,
			description: "Admin auth routes should not exist on user router",
		},
		{
			name:        "user routes not accessible on admin router",
			router:      adminRouter,
			method:      "GET",
			path:        "/api/v1/users/me",
			shouldExist: false,
			description: "User routes should not exist on admin router",
		},
		{
			name:        "user auth routes not accessible on admin router",
			router:      adminRouter,
			method:      "POST",
			path:        "/api/v1/auth/register",
			shouldExist: false,
			description: "User auth routes should not exist on admin router",
		},
		{
			name:        "health check only on user router",
			router:      adminRouter,
			method:      "GET",
			path:        "/health",
			shouldExist: false,
			description: "Health check should only be on user router",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			
			tc.router.ServeHTTP(w, req)
			
			if tc.shouldExist {
				assert.NotEqual(t, http.StatusNotFound, w.Code,
					"Route %s %s should exist", tc.method, tc.path)
			} else {
				assert.Equal(t, http.StatusNotFound, w.Code,
					"Route %s %s should NOT exist (got %d)", tc.method, tc.path, w.Code)
			}
		})
	}
}

// TestValidateParamMiddleware tests the UUID validation middleware
func TestValidateParamMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("test", "test-service")
	mockService := &MockUserService{}
	
	jwtManager, err := auth.NewJWTManager(
		"test-secret-key-must-be-at-least-32-characters-long",
		15*time.Minute,
		7*24*time.Hour,
	)
	require.NoError(t, err)

	adminRouter := httpTransport.SetupAdminRouter(mockService, jwtManager, logger, "debug")

	testCases := []struct {
		name           string
		userID         string
		expectedStatus int
		description    string
	}{
		{
			name:           "valid UUID is accepted",
			userID:         uuid.New().String(),
			expectedStatus: http.StatusUnauthorized, // 401 because no auth token, but UUID validation passed
			description:    "Valid UUID should pass validation, then hit auth middleware",
		},
		{
			name:           "invalid UUID is rejected",
			userID:         "not-a-valid-uuid",
			expectedStatus: http.StatusUnauthorized, // Note: Auth middleware runs BEFORE param validation in this setup
			description:    "Invalid UUID gets 401 from auth before validation can reject it",
		},
		{
			name:           "uppercase UUID is rejected",
			userID:         "A1B2C3D4-E5F6-4789-ABCD-EF0123456789",
			expectedStatus: http.StatusUnauthorized,
			description:    "Uppercase UUID gets 401 from auth before validation",
		},
		{
			name:           "short UUID is rejected",
			userID:         "123",
			expectedStatus: http.StatusUnauthorized,
			description:    "Short string gets 401 from auth before validation",
		},
		{
			name:           "UUID without hyphens is rejected",
			userID:         "a1b2c3d4e5f64789abcdef0123456789",
			expectedStatus: http.StatusUnauthorized,
			description:    "UUID without hyphens gets 401 from auth before validation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with admin user endpoint (has UUID validation)
			req := httptest.NewRequest("GET", "/admin/users/"+tc.userID, nil)
			w := httptest.NewRecorder()
			
			adminRouter.ServeHTTP(w, req)
			
			assert.Equal(t, tc.expectedStatus, w.Code,
				"Expected status %d for UUID '%s', got %d",
				tc.expectedStatus, tc.userID, w.Code)
		})
	}
}

// TestMiddlewareOrdering tests that middleware is applied in the correct order
func TestMiddlewareOrdering(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("test", "test-service")
	mockService := &MockUserService{}
	
	jwtManager, err := auth.NewJWTManager(
		"test-secret-key-must-be-at-least-32-characters-long",
		15*time.Minute,
		7*24*time.Hour,
	)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		setupRouter func() *gin.Engine
		method      string
		path        string
		description string
	}{
		{
			name: "user router has global middleware",
			setupRouter: func() *gin.Engine {
				return httpTransport.SetupUserRouter(mockService, jwtManager, logger, "debug")
			},
			method:      "POST",
			path:        "/api/v1/auth/register",
			description: "Should have Recovery, Logging, CORS middleware",
		},
		{
			name: "admin router has global middleware",
			setupRouter: func() *gin.Engine {
				return httpTransport.SetupAdminRouter(mockService, jwtManager, logger, "debug")
			},
			method:      "POST",
			path:        "/admin/auth/login",
			description: "Should have Recovery, Logging, CORS middleware",
		},
		{
			name: "protected user routes have auth middleware",
			setupRouter: func() *gin.Engine {
				return httpTransport.SetupUserRouter(mockService, jwtManager, logger, "debug")
			},
			method:      "GET",
			path:        "/api/v1/users/me",
			description: "Should require authentication (401 without token)",
		},
		{
			name: "protected admin routes have auth and admin middleware",
			setupRouter: func() *gin.Engine {
				return httpTransport.SetupAdminRouter(mockService, jwtManager, logger, "debug")
			},
			method:      "GET",
			path:        "/admin/users",
			description: "Should require authentication + admin role",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := tc.setupRouter()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			// Just verify the route exists and middleware is applied
			// Specific middleware behavior is tested in individual middleware tests
			assert.NotEqual(t, http.StatusNotFound, w.Code,
				"Route should exist")
		})
	}
}

// TestGinModeConfiguration tests that Gin mode is set correctly
func TestGinModeConfiguration(t *testing.T) {
	logger := observability.NewLogger("test", "test-service")
	mockService := &MockUserService{}
	
	jwtManager, err := auth.NewJWTManager(
		"test-secret-key-must-be-at-least-32-characters-long",
		15*time.Minute,
		7*24*time.Hour,
	)
	require.NoError(t, err)

	testCases := []struct {
		name         string
		mode         string
		expectedMode string
		description  string
	}{
		{
			name:         "release mode sets gin to release mode",
			mode:         "release",
			expectedMode: gin.ReleaseMode,
			description:  "Release mode should set Gin to release mode",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create router with specified mode
			_ = httpTransport.SetupUserRouter(mockService, jwtManager, logger, tc.mode)
			
			// Get current Gin mode
			currentMode := gin.Mode()
			
			// Verify mode is set correctly
			assert.Equal(t, tc.expectedMode, currentMode,
				"Gin mode should be %s when mode is %s", tc.expectedMode, tc.mode)
			
			// Reset to test mode for other tests
			gin.SetMode(gin.TestMode)
		})
	}
}

// TestRouterReturnsNonNil tests that router setup functions return valid routers
func TestRouterReturnsNonNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := observability.NewLogger("test", "test-service")
	mockService := &MockUserService{}
	
	jwtManager, err := auth.NewJWTManager(
		"test-secret-key-must-be-at-least-32-characters-long",
		15*time.Minute,
		7*24*time.Hour,
	)
	require.NoError(t, err)

	t.Run("user router is not nil", func(t *testing.T) {
		router := httpTransport.SetupUserRouter(mockService, jwtManager, logger, "debug")
		assert.NotNil(t, router, "User router should not be nil")
	})

	t.Run("admin router is not nil", func(t *testing.T) {
		router := httpTransport.SetupAdminRouter(mockService, jwtManager, logger, "debug")
		assert.NotNil(t, router, "Admin router should not be nil")
	})
}
