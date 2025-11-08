package http_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	httpTransport "github.com/alex-necsoiu/pandora-exchange/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestRegister tests the Register handler
func TestRegister(t *testing.T) {
	testCases := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful registration",
			requestBody: map[string]interface{}{
				"email":      "newuser@test.com",
				"password":   "SecurePass123!",
				"first_name": "John",
				"last_name":  "Doe",
			},
			mockSetup: func(m *MockUserService) {
				user := &domain.User{
					ID:        uuid.New(),
					Email:     "newuser@test.com",
					FirstName: "John",
					LastName:  "Doe",
					Role:      domain.RoleUser,
					KYCStatus: domain.KYCStatusPending,
					CreatedAt: time.Now(),
				}
				m.On("Register", mock.Anything, "newuser@test.com", "SecurePass123!", "John", "Doe").
					Return(user, nil)
				
				tokenPair := &domain.TokenPair{
					AccessToken:  "test_access_token",
					RefreshToken: "test_refresh_token",
					ExpiresAt:    time.Now().Add(15 * time.Minute),
					User:         user,
				}
				m.On("Login", mock.Anything, "newuser@test.com", "SecurePass123!", mock.Anything, mock.Anything).
					Return(tokenPair, nil)
			},
			expectedStatus: http.StatusCreated,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.NotEmpty(t, body["access_token"])
				assert.NotEmpty(t, body["refresh_token"])
				assert.NotNil(t, body["user"])
			},
		},
		{
			name: "user already exists",
			requestBody: map[string]interface{}{
				"email":      "existing@test.com",
				"password":   "password123",
				"first_name": "Jane",
				"last_name":  "Doe",
			},
			mockSetup: func(m *MockUserService) {
				m.On("Register", mock.Anything, "existing@test.com", "password123", "Jane", "Doe").
					Return(nil, domain.ErrUserAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "user_already_exists", body["error"])
			},
		},
		{
			name: "weak password",
			requestBody: map[string]interface{}{
				"email":      "test@test.com",
				"password":   "weakpass",
				"first_name": "John",
				"last_name":  "Doe",
			},
			mockSetup: func(m *MockUserService) {
				m.On("Register", mock.Anything, "test@test.com", "weakpass", "John", "Doe").
					Return(nil, domain.ErrWeakPassword)
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "weak_password", body["error"])
			},
		},
		{
			name: "invalid email",
			requestBody: map[string]interface{}{
				"email":      "invalid@test.com",
				"password":   "password123",
				"first_name": "John",
				"last_name":  "Doe",
			},
			mockSetup: func(m *MockUserService) {
				m.On("Register", mock.Anything, "invalid@test.com", "password123", "John", "Doe").
					Return(nil, domain.ErrInvalidEmail)
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_email", body["error"])
			},
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json string",
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_request", body["error"])
			},
		},
		{
			name: "registration succeeds but auto-login fails",
			requestBody: map[string]interface{}{
				"email":      "test@test.com",
				"password":   "password123",
				"first_name": "John",
				"last_name":  "Doe",
			},
			mockSetup: func(m *MockUserService) {
				user := &domain.User{
					ID:        uuid.New(),
					Email:     "test@test.com",
					FirstName: "John",
					LastName:  "Doe",
				}
				m.On("Register", mock.Anything, "test@test.com", "password123", "John", "Doe").
					Return(user, nil)
				m.On("Login", mock.Anything, "test@test.com", "password123", mock.Anything, mock.Anything).
					Return(nil, errors.New("internal error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "internal_error", body["error"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tc.mockSetup(mockService)
			handler := httpTransport.NewHandler(mockService, getTestLogger())

			var body []byte
			if str, ok := tc.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tc.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := gin.New()
			router.POST("/api/v1/auth/register", handler.Register)
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tc.validateBody(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestLogin tests the Login handler
func TestLogin(t *testing.T) {
	testCases := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful login",
			requestBody: map[string]interface{}{
				"email":    "user@test.com",
				"password": "correctPassword",
			},
			mockSetup: func(m *MockUserService) {
				user := &domain.User{
					ID:        uuid.New(),
					Email:     "user@test.com",
					FirstName: "John",
					LastName:  "Doe",
					Role:      domain.RoleUser,
				}
				tokenPair := &domain.TokenPair{
					AccessToken:  "access_token",
					RefreshToken: "refresh_token",
					ExpiresAt:    time.Now().Add(15 * time.Minute),
					User:         user,
				}
				m.On("Login", mock.Anything, "user@test.com", "correctPassword", mock.Anything, mock.Anything).
					Return(tokenPair, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.NotEmpty(t, body["access_token"])
				assert.NotEmpty(t, body["refresh_token"])
			},
		},
		{
			name: "invalid credentials",
			requestBody: map[string]interface{}{
				"email":    "user@test.com",
				"password": "wrongPassword",
			},
			mockSetup: func(m *MockUserService) {
				m.On("Login", mock.Anything, "user@test.com", "wrongPassword", mock.Anything, mock.Anything).
					Return(nil, domain.ErrInvalidCredentials)
			},
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_credentials", body["error"])
			},
		},
		{
			name: "user not found",
			requestBody: map[string]interface{}{
				"email":    "nonexistent@test.com",
				"password": "password123",
			},
			mockSetup: func(m *MockUserService) {
				m.On("Login", mock.Anything, "nonexistent@test.com", "password123", mock.Anything, mock.Anything).
					Return(nil, domain.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "user_not_found", body["error"])
			},
		},
		{
			name:           "invalid JSON",
			requestBody:    "not json",
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_request", body["error"])
			},
		},
		{
			name: "service error",
			requestBody: map[string]interface{}{
				"email":    "user@test.com",
				"password": "password123",
			},
			mockSetup: func(m *MockUserService) {
				m.On("Login", mock.Anything, "user@test.com", "password123", mock.Anything, mock.Anything).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "internal_error", body["error"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tc.mockSetup(mockService)
			handler := httpTransport.NewHandler(mockService, getTestLogger())

			var body []byte
			if str, ok := tc.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tc.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := gin.New()
			router.POST("/api/v1/auth/login", handler.Login)
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tc.validateBody(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestRefreshTokenHandler tests the RefreshToken handler
func TestRefreshTokenHandler(t *testing.T) {
	testCases := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful token refresh",
			requestBody: map[string]interface{}{
				"refresh_token": "valid_refresh_token",
			},
			mockSetup: func(m *MockUserService) {
				user := &domain.User{
					ID:        uuid.New(),
					Email:     "user@test.com",
					FirstName: "John",
					LastName:  "Doe",
				}
				tokenPair := &domain.TokenPair{
					AccessToken:  "new_access_token",
					RefreshToken: "new_refresh_token",
					ExpiresAt:    time.Now().Add(15 * time.Minute),
					User:         user,
				}
				m.On("RefreshToken", mock.Anything, "valid_refresh_token", mock.Anything, mock.Anything).
					Return(tokenPair, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.NotEmpty(t, body["access_token"])
				assert.NotEmpty(t, body["refresh_token"])
			},
		},
		{
			name: "invalid refresh token",
			requestBody: map[string]interface{}{
				"refresh_token": "invalid_token",
			},
			mockSetup: func(m *MockUserService) {
				m.On("RefreshToken", mock.Anything, "invalid_token", mock.Anything, mock.Anything).
					Return(nil, domain.ErrRefreshTokenNotFound)
			},
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_refresh_token", body["error"])
			},
		},
		{
			name: "expired refresh token",
			requestBody: map[string]interface{}{
				"refresh_token": "expired_token",
			},
			mockSetup: func(m *MockUserService) {
				m.On("RefreshToken", mock.Anything, "expired_token", mock.Anything, mock.Anything).
					Return(nil, domain.ErrRefreshTokenExpired)
			},
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_refresh_token", body["error"])
			},
		},
		{
			name: "revoked refresh token",
			requestBody: map[string]interface{}{
				"refresh_token": "revoked_token",
			},
			mockSetup: func(m *MockUserService) {
				m.On("RefreshToken", mock.Anything, "revoked_token", mock.Anything, mock.Anything).
					Return(nil, domain.ErrRefreshTokenRevoked)
			},
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_refresh_token", body["error"])
			},
		},
		{
			name:           "invalid JSON",
			requestBody:    "not json",
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_request", body["error"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tc.mockSetup(mockService)
			handler := httpTransport.NewHandler(mockService, getTestLogger())

			var body []byte
			if str, ok := tc.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tc.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := gin.New()
			router.POST("/api/v1/auth/refresh", handler.RefreshToken)
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tc.validateBody(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestLogout tests the Logout handler
func TestLogout(t *testing.T) {
	userID := uuid.New()

	testCases := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful logout",
			requestBody: map[string]interface{}{
				"refresh_token": "valid_refresh_token",
			},
			mockSetup: func(m *MockUserService) {
				m.On("Logout", mock.Anything, "valid_refresh_token").
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "logged out successfully", body["message"])
			},
		},
		{
			name: "token not found",
			requestBody: map[string]interface{}{
				"refresh_token": "nonexistent_token",
			},
			mockSetup: func(m *MockUserService) {
				m.On("Logout", mock.Anything, "nonexistent_token").
					Return(domain.ErrRefreshTokenNotFound)
			},
			expectedStatus: http.StatusUnauthorized,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_refresh_token", body["error"])
			},
		},
		{
			name:           "invalid JSON",
			requestBody:    "not json",
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_request", body["error"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tc.mockSetup(mockService)
			handler := httpTransport.NewHandler(mockService, getTestLogger())

			var body []byte
			if str, ok := tc.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tc.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := gin.New()
			router.POST("/api/v1/auth/logout", func(c *gin.Context) {
				c.Set("user_id", userID)
				handler.Logout(c)
			})
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tc.validateBody(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestLogoutAll tests the LogoutAll handler
func TestLogoutAll(t *testing.T) {
	userID := uuid.New()

	testCases := []struct {
		name           string
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "successful logout all",
			mockSetup: func(m *MockUserService) {
				m.On("LogoutAll", mock.Anything, userID).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "logged out from all devices successfully", body["message"])
			},
		},
		{
			name: "service error",
			mockSetup: func(m *MockUserService) {
				m.On("LogoutAll", mock.Anything, userID).
					Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "internal_error", body["error"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tc.mockSetup(mockService)
			handler := httpTransport.NewHandler(mockService, getTestLogger())

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout-all", nil)
			w := httptest.NewRecorder()

			router := gin.New()
			router.POST("/api/v1/auth/logout-all", func(c *gin.Context) {
				c.Set("user_id", userID)
				handler.LogoutAll(c)
			})
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tc.validateBody(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestGetProfile tests the GetProfile handler
func TestGetProfile(t *testing.T) {
	userID := uuid.New()

	testCases := []struct {
		name           string
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "get profile successfully",
			mockSetup: func(m *MockUserService) {
				user := &domain.User{
					ID:        userID,
					Email:     "user@test.com",
					FirstName: "John",
					LastName:  "Doe",
					Role:      domain.RoleUser,
					KYCStatus: domain.KYCStatusPending,
				}
				m.On("GetByID", mock.Anything, userID).
					Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "user@test.com", body["email"])
				assert.Equal(t, "John", body["first_name"])
			},
		},
		{
			name: "user not found",
			mockSetup: func(m *MockUserService) {
				m.On("GetByID", mock.Anything, userID).
					Return(nil, domain.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "user_not_found", body["error"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tc.mockSetup(mockService)
			handler := httpTransport.NewHandler(mockService, getTestLogger())

			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
			w := httptest.NewRecorder()

			router := gin.New()
			router.GET("/api/v1/users/me", func(c *gin.Context) {
				c.Set("user_id", userID)
				handler.GetProfile(c)
			})
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tc.validateBody(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestUpdateProfile tests the UpdateProfile handler
func TestUpdateProfile(t *testing.T) {
	userID := uuid.New()

	testCases := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "update profile successfully",
			requestBody: map[string]interface{}{
				"first_name": "Jane",
				"last_name":  "Smith",
			},
			mockSetup: func(m *MockUserService) {
				user := &domain.User{
					ID:        userID,
					Email:     "user@test.com",
					FirstName: "Jane",
					LastName:  "Smith",
					Role:      domain.RoleUser,
				}
				m.On("UpdateProfile", mock.Anything, userID, "Jane", "Smith").
					Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "Jane", body["first_name"])
				assert.Equal(t, "Smith", body["last_name"])
			},
		},
		{
			name: "user not found",
			requestBody: map[string]interface{}{
				"first_name": "Jane",
				"last_name":  "Smith",
			},
			mockSetup: func(m *MockUserService) {
				m.On("UpdateProfile", mock.Anything, userID, "Jane", "Smith").
					Return(nil, domain.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "user_not_found", body["error"])
			},
		},
		{
			name:           "invalid JSON",
			requestBody:    "not json",
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_request", body["error"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tc.mockSetup(mockService)
			handler := httpTransport.NewHandler(mockService, getTestLogger())

			var body []byte
			if str, ok := tc.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tc.requestBody)
			}

			req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := gin.New()
			router.PUT("/api/v1/users/me", func(c *gin.Context) {
				c.Set("user_id", userID)
				handler.UpdateProfile(c)
			})
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tc.validateBody(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestDeleteAccount tests the DeleteAccount handler
func TestDeleteAccount(t *testing.T) {
	userID := uuid.New()

	testCases := []struct {
		name           string
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "delete account successfully",
			mockSetup: func(m *MockUserService) {
				m.On("DeleteAccount", mock.Anything, userID).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "account deleted successfully", body["message"])
			},
		},
		{
			name: "user not found",
			mockSetup: func(m *MockUserService) {
				m.On("DeleteAccount", mock.Anything, userID).
					Return(domain.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "user_not_found", body["error"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tc.mockSetup(mockService)
			handler := httpTransport.NewHandler(mockService, getTestLogger())

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", nil)
			w := httptest.NewRecorder()

			router := gin.New()
			router.DELETE("/api/v1/users/me", func(c *gin.Context) {
				c.Set("user_id", userID)
				handler.DeleteAccount(c)
			})
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tc.validateBody(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestGetActiveSessions tests the GetActiveSessions handler
func TestGetActiveSessions(t *testing.T) {
	userID := uuid.New()

	testCases := []struct {
		name           string
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "get active sessions successfully",
			mockSetup: func(m *MockUserService) {
				sessions := []*domain.RefreshToken{
					{
						Token:     "token1",
						UserID:    userID,
						ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
						CreatedAt: time.Now(),
						IPAddress: "192.168.1.1",
						UserAgent: "Mozilla/5.0",
					},
					{
						Token:     "token2",
						UserID:    userID,
						ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
						CreatedAt: time.Now(),
						IPAddress: "192.168.1.2",
						UserAgent: "Chrome",
					},
				}
				m.On("GetActiveSessions", mock.Anything, userID).
					Return(sessions, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				sessions := body["sessions"].([]interface{})
				assert.Len(t, sessions, 2)
			},
		},
		{
			name: "service error",
			mockSetup: func(m *MockUserService) {
				m.On("GetActiveSessions", mock.Anything, userID).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "internal_error", body["error"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tc.mockSetup(mockService)
			handler := httpTransport.NewHandler(mockService, getTestLogger())

			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/sessions", nil)
			w := httptest.NewRecorder()

			router := gin.New()
			router.GET("/api/v1/users/me/sessions", func(c *gin.Context) {
				c.Set("user_id", userID)
				handler.GetActiveSessions(c)
			})
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tc.validateBody(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestUpdateKYC tests the UpdateKYC handler
func TestUpdateKYC(t *testing.T) {
	userID := uuid.New()

	testCases := []struct {
		name           string
		userID         string
		requestBody    interface{}
		mockSetup      func(*MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:   "update KYC status successfully",
			userID: userID.String(),
			requestBody: map[string]interface{}{
				"status": "approved",
			},
			mockSetup: func(m *MockUserService) {
				user := &domain.User{
					ID:        userID,
					Email:     "user@test.com",
					FirstName: "John",
					LastName:  "Doe",
					KYCStatus: domain.KYCStatus("approved"),
				}
				m.On("UpdateKYC", mock.Anything, userID, domain.KYCStatus("approved")).
					Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "approved", body["kyc_status"])
			},
		},
		{
			name:   "invalid user ID",
			userID: "invalid-uuid",
			requestBody: map[string]interface{}{
				"status": "verified",
			},
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_user_id", body["error"])
			},
		},
		{
			name:   "invalid KYC status",
			userID: userID.String(),
			requestBody: map[string]interface{}{
				"status": "invalid_status",
			},
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_request", body["error"])
			},
		},
		{
			name:           "invalid JSON",
			userID:         userID.String(),
			requestBody:    "not json",
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_request", body["error"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := new(MockUserService)
			tc.mockSetup(mockService)
			handler := httpTransport.NewHandler(mockService, getTestLogger())

			var body []byte
			if str, ok := tc.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tc.requestBody)
			}

			req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+tc.userID+"/kyc", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := gin.New()
			router.PUT("/api/v1/users/:id/kyc", handler.UpdateKYC)
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			tc.validateBody(t, response)

			mockService.AssertExpectations(t)
		})
	}
}

// TestHealthCheck tests the HealthCheck handler
func TestHealthCheck(t *testing.T) {
	mockService := new(MockUserService)
	handler := httpTransport.NewHandler(mockService, getTestLogger())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router := gin.New()
	router.GET("/health", handler.HealthCheck)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "healthy", response["status"])
}
