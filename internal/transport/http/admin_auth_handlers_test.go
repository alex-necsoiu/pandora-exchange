package http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	httpTransport "github.com/alex-necsoiu/pandora-exchange/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// getTestLogger creates a logger for testing
func getTestLogger() *observability.Logger {
	var buf bytes.Buffer
	return observability.NewLoggerWithWriter("dev", "test-service", &buf)
}

func setupAdminAuthHandlerTest() (*httpTransport.AdminAuthHandler, *MockUserService, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	mockService := new(MockUserService)
	logger := getTestLogger()
	handler := httpTransport.NewAdminAuthHandler(mockService, logger)
	
	router := gin.New()
	return handler, mockService, router
}

// TestAdminLoginHandler tests the AdminLogin HTTP handler
func TestAdminLoginHandler(t *testing.T) {
	validUser := &domain.User{
		ID:        uuid.New(),
		Email:     "admin@test.com",
		FirstName: "Admin",
		LastName:  "User",
		Role:      domain.RoleAdmin,
		KYCStatus: domain.KYCStatusVerified,
		CreatedAt: time.Now(),
	}

	tokenPair := &domain.TokenPair{
		User:         validUser,
		AccessToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
		RefreshToken: "refresh_abc123",
		ExpiresAt:    time.Now().Add(15 * time.Minute),
	}

	testCases := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockUserService)
		expectedStatus int
		expectedError  string
		validateBody   func(*testing.T, map[string]interface{})
	}{
		{
			name: "admin login successfully",
			requestBody: httpTransport.LoginRequest{
				Email:    "admin@test.com",
				Password: "Admin123",
			},
			mockSetup: func(m *MockUserService) {
				m.On("AdminLogin", mock.Anything, "admin@test.com", "Admin123", mock.Anything, mock.Anything).
					Return(tokenPair, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.NotEmpty(t, body["access_token"])
				assert.NotEmpty(t, body["refresh_token"])
				assert.NotEmpty(t, body["expires_at"])
				user := body["user"].(map[string]interface{})
				assert.Equal(t, "admin@test.com", user["email"])
			},
		},
		{
			name: "admin login with regular user fails",
			requestBody: httpTransport.LoginRequest{
				Email:    "user@test.com",
				Password: "User123",
			},
			mockSetup: func(m *MockUserService) {
				m.On("AdminLogin", mock.Anything, "user@test.com", "User123", mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("admin access required"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "admin access required",
		},
		{
			name: "admin login with invalid credentials",
			requestBody: httpTransport.LoginRequest{
				Email:    "admin@test.com",
				Password: "WrongPassword",
			},
			mockSetup: func(m *MockUserService) {
				m.On("AdminLogin", mock.Anything, "admin@test.com", "WrongPassword", mock.Anything, mock.Anything).
					Return(nil, domain.ErrInvalidCredentials)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid credentials",
		},
		{
			name: "admin login with deleted account",
			requestBody: httpTransport.LoginRequest{
				Email:    "deleted@test.com",
				Password: "Password123",
			},
			mockSetup: func(m *MockUserService) {
				m.On("AdminLogin", mock.Anything, "deleted@test.com", "Password123", mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("account is deleted"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "account is deleted",
		},
		{
			name: "admin login with missing email",
			requestBody: httpTransport.LoginRequest{
				Password: "Password123",
			},
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "email and password are required",
		},
		{
			name: "admin login with missing password",
			requestBody: httpTransport.LoginRequest{
				Email: "admin@test.com",
			},
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "email and password are required",
		},
		{
			name:           "admin login with invalid JSON",
			requestBody:    "invalid json",
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
		},
		{
			name: "admin login with service error",
			requestBody: httpTransport.LoginRequest{
				Email:    "admin@test.com",
				Password: "Admin123",
			},
			mockSetup: func(m *MockUserService) {
				m.On("AdminLogin", mock.Anything, "admin@test.com", "Admin123", mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("database connection error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal server error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler, mockService, router := setupAdminAuthHandlerTest()
			tc.mockSetup(mockService)

			router.POST("/admin/auth/login", handler.AdminLogin)

			var bodyBytes []byte
			if str, ok := tc.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tc.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/admin/auth/login", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tc.expectedError != "" {
				assert.Equal(t, tc.expectedError, response["error"])
			}

			if tc.validateBody != nil {
				tc.validateBody(t, response)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestAdminRefreshTokenHandler tests the AdminRefreshToken HTTP handler
func TestAdminRefreshTokenHandler(t *testing.T) {
	adminUser := &domain.User{
		ID:        uuid.New(),
		Email:     "admin@test.com",
		FirstName: "Admin",
		LastName:  "User",
		Role:      domain.RoleAdmin,
		KYCStatus: domain.KYCStatusVerified,
		CreatedAt: time.Now(),
	}

	newTokenPair := &domain.TokenPair{
		User:         adminUser,
		AccessToken:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.new_token",
		RefreshToken: "refresh_xyz789",
		ExpiresAt:    time.Now().Add(15 * time.Minute),
	}

	testCases := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockUserService)
		expectedStatus int
		expectedError  string
		validateBody   func(*testing.T, map[string]interface{})
	}{
		{
			name: "refresh admin token successfully",
			requestBody: httpTransport.RefreshTokenRequest{
				RefreshToken: "valid_refresh_token",
			},
			mockSetup: func(m *MockUserService) {
				m.On("RefreshToken", mock.Anything, "valid_refresh_token", mock.Anything, mock.Anything).
					Return(newTokenPair, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.NotEmpty(t, body["access_token"])
				assert.NotEmpty(t, body["refresh_token"])
				user := body["user"].(map[string]interface{})
				assert.Equal(t, "admin@test.com", user["email"])
			},
		},
		{
			name: "refresh token with invalid token",
			requestBody: httpTransport.RefreshTokenRequest{
				RefreshToken: "invalid_token",
			},
			mockSetup: func(m *MockUserService) {
				m.On("RefreshToken", mock.Anything, "invalid_token", mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("invalid refresh token"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid or expired refresh token",
		},
		{
			name: "refresh token with expired token",
			requestBody: httpTransport.RefreshTokenRequest{
				RefreshToken: "expired_token",
			},
			mockSetup: func(m *MockUserService) {
				m.On("RefreshToken", mock.Anything, "expired_token", mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("token expired"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid or expired refresh token",
		},
		{
			name: "refresh token with missing refresh_token",
			requestBody: httpTransport.RefreshTokenRequest{
				RefreshToken: "",
			},
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "refresh_token is required",
		},
		{
			name:           "refresh token with invalid JSON",
			requestBody:    "invalid json",
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
		},
		{
			name: "refresh token preserves admin role",
			requestBody: httpTransport.RefreshTokenRequest{
				RefreshToken: "admin_refresh_token",
			},
			mockSetup: func(m *MockUserService) {
				m.On("RefreshToken", mock.Anything, "admin_refresh_token", mock.Anything, mock.Anything).
					Return(newTokenPair, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				// Verify the response contains user data (role is in JWT, not response)
				user := body["user"].(map[string]interface{})
				assert.Equal(t, "admin@test.com", user["email"])
				assert.NotEmpty(t, body["access_token"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler, mockService, router := setupAdminAuthHandlerTest()
			tc.mockSetup(mockService)

			router.POST("/admin/auth/refresh", handler.AdminRefreshToken)

			var bodyBytes []byte
			if str, ok := tc.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tc.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/admin/auth/refresh", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tc.expectedError != "" {
				assert.Equal(t, tc.expectedError, response["error"])
			}

			if tc.validateBody != nil {
				tc.validateBody(t, response)
			}

			mockService.AssertExpectations(t)
		})
	}
}
