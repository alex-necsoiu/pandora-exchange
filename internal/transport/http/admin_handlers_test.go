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
	httpTransport "github.com/alex-necsoiu/pandora-exchange/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestListUsers tests the ListUsers HTTP handler
func TestListUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user1 := &domain.User{
		ID:        uuid.New(),
		Email:     "user1@test.com",
		FirstName: "User",
		LastName:  "One",
		Role:      domain.RoleUser,
		KYCStatus: domain.KYCStatusVerified,
		CreatedAt: time.Now(),
	}

	user2 := &domain.User{
		ID:        uuid.New(),
		Email:     "user2@test.com",
		FirstName: "User",
		LastName:  "Two",
		Role:      domain.RoleAdmin,
		KYCStatus: domain.KYCStatusPending,
		CreatedAt: time.Now(),
	}

	testCases := []struct {
		name           string
		queryParams    string
		mockSetup      func(m *MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:        "list users successfully with defaults",
			queryParams: "",
			mockSetup: func(m *MockUserService) {
				m.On("ListUsers", mock.Anything, 20, 0).
					Return([]*domain.User{user1, user2}, int64(2), nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(2), body["total"])
				assert.Equal(t, float64(20), body["limit"])
				assert.Equal(t, float64(0), body["offset"])
				users := body["users"].([]interface{})
				assert.Len(t, users, 2)
			},
		},
		{
			name:        "list users with custom pagination",
			queryParams: "?limit=10&offset=5",
			mockSetup: func(m *MockUserService) {
				m.On("ListUsers", mock.Anything, 10, 5).
					Return([]*domain.User{user1}, int64(10), nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(10), body["total"])
				assert.Equal(t, float64(10), body["limit"])
				assert.Equal(t, float64(5), body["offset"])
			},
		},
		{
			name:        "list users with service error",
			queryParams: "",
			mockSetup: func(m *MockUserService) {
				m.On("ListUsers", mock.Anything, 20, 0).
					Return([]*domain.User(nil), int64(0), fmt.Errorf("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "internal_error", body["error"])
			},
		},
		{
			name:        "list users with invalid limit (too large)",
			queryParams: "?limit=101",
			mockSetup:   func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_request", body["error"])
			},
		},
		{
			name:        "list users with invalid limit (negative)",
			queryParams: "?limit=-1",
			mockSetup:   func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_request", body["error"])
			},
		},
		{
			name:        "list users with invalid offset (negative)",
			queryParams: "?offset=-1",
			mockSetup:   func(m *MockUserService) {},
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

			logger := getTestLogger()
			handler := httpTransport.NewAdminHandler(mockService, logger)

			router := gin.New()
			router.GET("/admin/users", handler.ListUsers)

			req := httptest.NewRequest(http.MethodGet, "/admin/users"+tc.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tc.validateBody != nil {
				tc.validateBody(t, response)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestSearchUsers tests the SearchUsers HTTP handler
func TestSearchUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	user1 := &domain.User{
		ID:        uuid.New(),
		Email:     "john@test.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      domain.RoleUser,
		KYCStatus: domain.KYCStatusVerified,
		CreatedAt: time.Now(),
	}

	testCases := []struct {
		name           string
		queryParams    string
		mockSetup      func(m *MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:        "search users successfully",
			queryParams: "?query=john",
			mockSetup: func(m *MockUserService) {
				m.On("SearchUsers", mock.Anything, "john", 20, 0).
					Return([]*domain.User{user1}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				users := body["users"].([]interface{})
				assert.Len(t, users, 1)
				assert.Equal(t, float64(1), body["total"])
			},
		},
		{
			name:        "search users with no results",
			queryParams: "?query=nonexistent",
			mockSetup: func(m *MockUserService) {
				m.On("SearchUsers", mock.Anything, "nonexistent", 20, 0).
					Return([]*domain.User{}, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				users := body["users"].([]interface{})
				assert.Len(t, users, 0)
			},
		},
		{
			name:        "search users with service error",
			queryParams: "?query=error",
			mockSetup: func(m *MockUserService) {
				m.On("SearchUsers", mock.Anything, "error", 20, 0).
					Return([]*domain.User(nil), fmt.Errorf("search failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "internal_error", body["error"])
			},
		},
		{
			name:        "search users with missing query",
			queryParams: "",
			mockSetup:   func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_request", body["error"])
			},
		},
		{
			name:        "search users with invalid limit",
			queryParams: "?query=test&limit=200",
			mockSetup:   func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_request", body["error"])
			},
		},
		{
			name:        "search users with invalid offset",
			queryParams: "?query=test&offset=-5",
			mockSetup:   func(m *MockUserService) {},
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

			logger := getTestLogger()
			handler := httpTransport.NewAdminHandler(mockService, logger)

			router := gin.New()
			router.GET("/admin/users/search", handler.SearchUsers)

			req := httptest.NewRequest(http.MethodGet, "/admin/users/search"+tc.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tc.validateBody != nil {
				tc.validateBody(t, response)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestGetUser tests the GetUser HTTP handler
func TestGetUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	user := &domain.User{
		ID:        userID,
		Email:     "test@test.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      domain.RoleUser,
		KYCStatus: domain.KYCStatusVerified,
		CreatedAt: time.Now(),
	}

	testCases := []struct {
		name           string
		userID         string
		mockSetup      func(m *MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:   "get user successfully",
			userID: userID.String(),
			mockSetup: func(m *MockUserService) {
				m.On("GetUserByIDAdmin", mock.Anything, userID).
					Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "test@test.com", body["email"])
			},
		},
		{
			name:   "get user with invalid ID",
			userID: "invalid-uuid",
			mockSetup: func(m *MockUserService) {
				// No service call expected
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_user_id", body["error"])
			},
		},
		{
			name:   "get user not found",
			userID: userID.String(),
			mockSetup: func(m *MockUserService) {
				m.On("GetUserByIDAdmin", mock.Anything, userID).
					Return((*domain.User)(nil), domain.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "user_not_found", body["error"])
			},
		},
		{
			name:   "get user with service error",
			userID: userID.String(),
			mockSetup: func(m *MockUserService) {
				m.On("GetUserByIDAdmin", mock.Anything, userID).
					Return((*domain.User)(nil), fmt.Errorf("database error"))
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

			logger := getTestLogger()
			handler := httpTransport.NewAdminHandler(mockService, logger)

			router := gin.New()
			router.GET("/admin/users/:id", handler.GetUser)

			req := httptest.NewRequest(http.MethodGet, "/admin/users/"+tc.userID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tc.validateBody != nil {
				tc.validateBody(t, response)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestUpdateUserRole tests the UpdateUserRole HTTP handler
func TestUpdateUserRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	updatedUser := &domain.User{
		ID:        userID,
		Email:     "test@test.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      domain.RoleAdmin,
		KYCStatus: domain.KYCStatusVerified,
		CreatedAt: time.Now(),
	}

	testCases := []struct {
		name           string
		userID         string
		requestBody    interface{}
		mockSetup      func(m *MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:   "update role successfully",
			userID: userID.String(),
			requestBody: httpTransport.AdminUpdateRoleRequest{
				Role: string(domain.RoleAdmin),
			},
			mockSetup: func(m *MockUserService) {
				m.On("UpdateUserRole", mock.Anything, userID, domain.RoleAdmin).
					Return(updatedUser, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, string(domain.RoleAdmin), body["role"])
			},
		},
		{
			name:   "update role with invalid user ID",
			userID: "invalid-uuid",
			requestBody: httpTransport.AdminUpdateRoleRequest{
				Role: string(domain.RoleAdmin),
			},
			mockSetup: func(m *MockUserService) {
				// No service call expected
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_user_id", body["error"])
			},
		},
		{
			name:           "update role with invalid JSON",
			userID:         userID.String(),
			requestBody:    "invalid json",
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_request", body["error"])
			},
		},
		{
			name:   "update role user not found",
			userID: userID.String(),
			requestBody: httpTransport.AdminUpdateRoleRequest{
				Role: string(domain.RoleAdmin),
			},
			mockSetup: func(m *MockUserService) {
				m.On("UpdateUserRole", mock.Anything, userID, domain.RoleAdmin).
					Return((*domain.User)(nil), domain.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "user_not_found", body["error"])
			},
		},
		{
			name:   "update role with invalid role",
			userID: userID.String(),
			requestBody: httpTransport.AdminUpdateRoleRequest{
				Role: "invalid_role",
			},
			mockSetup: func(m *MockUserService) {
				m.On("UpdateUserRole", mock.Anything, userID, domain.Role("invalid_role")).
					Return((*domain.User)(nil), domain.ErrInvalidRole)
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_role", body["error"])
			},
		},
		{
			name:   "update role with service error",
			userID: userID.String(),
			requestBody: httpTransport.AdminUpdateRoleRequest{
				Role: string(domain.RoleAdmin),
			},
			mockSetup: func(m *MockUserService) {
				m.On("UpdateUserRole", mock.Anything, userID, domain.RoleAdmin).
					Return((*domain.User)(nil), fmt.Errorf("database error"))
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

			logger := getTestLogger()
			handler := httpTransport.NewAdminHandler(mockService, logger)

			router := gin.New()
			router.PUT("/admin/users/:id/role", handler.UpdateUserRole)

			var bodyBytes []byte
			if str, ok := tc.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tc.requestBody)
			}

			req := httptest.NewRequest(http.MethodPut, "/admin/users/"+tc.userID+"/role", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tc.validateBody != nil {
				tc.validateBody(t, response)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestGetAllSessions tests the GetAllSessions HTTP handler
func TestGetAllSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID1 := uuid.New()
	userID2 := uuid.New()

	session1 := &domain.RefreshToken{
		Token:     "token1",
		UserID:    userID1,
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	session2 := &domain.RefreshToken{
		Token:     "token2",
		UserID:    userID2,
		IPAddress: "192.168.1.2",
		UserAgent: "Chrome/91.0",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	testCases := []struct {
		name           string
		queryParams    string
		mockSetup      func(m *MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:        "get all sessions successfully",
			queryParams: "",
			mockSetup: func(m *MockUserService) {
				m.On("GetAllActiveSessions", mock.Anything, 50, 0).
					Return([]*domain.RefreshToken{session1, session2}, int64(2), nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				sessions := body["sessions"].([]interface{})
				assert.Len(t, sessions, 2)
				assert.Equal(t, float64(2), body["total"])
			},
		},
		{
			name:        "get all sessions with pagination",
			queryParams: "?limit=10&offset=5",
			mockSetup: func(m *MockUserService) {
				m.On("GetAllActiveSessions", mock.Anything, 10, 5).
					Return([]*domain.RefreshToken{session1}, int64(20), nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(20), body["total"])
				assert.Equal(t, float64(10), body["limit"])
			},
		},
		{
			name:        "get all sessions with service error",
			queryParams: "",
			mockSetup: func(m *MockUserService) {
				m.On("GetAllActiveSessions", mock.Anything, 50, 0).
					Return([]*domain.RefreshToken(nil), int64(0), fmt.Errorf("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "internal_error", body["error"])
			},
		},
		{
			name:        "get all sessions with invalid limit",
			queryParams: "?limit=150",
			mockSetup:   func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_request", body["error"])
			},
		},
		{
			name:        "get all sessions with negative offset",
			queryParams: "?offset=-10",
			mockSetup:   func(m *MockUserService) {},
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

			logger := getTestLogger()
			handler := httpTransport.NewAdminHandler(mockService, logger)

			router := gin.New()
			router.GET("/admin/sessions", handler.GetAllSessions)

			req := httptest.NewRequest(http.MethodGet, "/admin/sessions"+tc.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tc.validateBody != nil {
				tc.validateBody(t, response)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestForceLogout tests the ForceLogout HTTP handler
func TestForceLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(m *MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "force logout successfully",
			requestBody: httpTransport.AdminForceLogoutRequest{
				Token: "valid_token",
			},
			mockSetup: func(m *MockUserService) {
				m.On("ForceLogout", mock.Anything, "valid_token").
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "Session revoked successfully", body["message"])
			},
		},
		{
			name:           "force logout with invalid JSON",
			requestBody:    "invalid json",
			mockSetup:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "invalid_request", body["error"])
			},
		},
		{
			name: "force logout token not found",
			requestBody: httpTransport.AdminForceLogoutRequest{
				Token: "nonexistent_token",
			},
			mockSetup: func(m *MockUserService) {
				m.On("ForceLogout", mock.Anything, "nonexistent_token").
					Return(domain.ErrTokenNotFound)
			},
			expectedStatus: http.StatusNotFound,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "token_not_found", body["error"])
			},
		},
		{
			name: "force logout with service error",
			requestBody: httpTransport.AdminForceLogoutRequest{
				Token: "error_token",
			},
			mockSetup: func(m *MockUserService) {
				m.On("ForceLogout", mock.Anything, "error_token").
					Return(fmt.Errorf("database error"))
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

			logger := getTestLogger()
			handler := httpTransport.NewAdminHandler(mockService, logger)

			router := gin.New()
			router.POST("/admin/sessions/revoke", handler.ForceLogout)

			var bodyBytes []byte
			if str, ok := tc.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tc.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/admin/sessions/revoke", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tc.validateBody != nil {
				tc.validateBody(t, response)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestGetSystemStats tests the GetSystemStats HTTP handler
func TestGetSystemStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stats := map[string]interface{}{
		"total_users":     100,
		"active_users":    80,
		"deleted_users":   20,
		"pending_kyc":     15,
		"verified_kyc":    65,
		"rejected_kyc":    0,
		"active_sessions": 50,
		"total_admins":    5,
	}

	testCases := []struct {
		name           string
		mockSetup      func(m *MockUserService)
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "get system stats successfully",
			mockSetup: func(m *MockUserService) {
				m.On("GetSystemStats", mock.Anything).
					Return(stats, nil)
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, float64(100), body["total_users"])
				assert.Equal(t, float64(80), body["active_users"])
				assert.Equal(t, float64(5), body["total_admins"])
			},
		},
		{
			name: "get system stats with service error",
			mockSetup: func(m *MockUserService) {
				m.On("GetSystemStats", mock.Anything).
					Return(map[string]interface{}(nil), fmt.Errorf("database error"))
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

			logger := getTestLogger()
			handler := httpTransport.NewAdminHandler(mockService, logger)

			router := gin.New()
			router.GET("/admin/stats", handler.GetSystemStats)

			req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tc.validateBody != nil {
				tc.validateBody(t, response)
			}

			mockService.AssertExpectations(t)
		})
	}
}
