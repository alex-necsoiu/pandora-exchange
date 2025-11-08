package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/config"
	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/domain/auth"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
	"github.com/alex-necsoiu/pandora-exchange/internal/repository"
	"github.com/alex-necsoiu/pandora-exchange/internal/service"
	httpTransport "github.com/alex-necsoiu/pandora-exchange/internal/transport/http"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDatabaseURL = "postgres://pandora:pandora_dev_secret@localhost:5432/pandora_dev?sslmode=disable"
)

// setupIntegrationTest sets up a complete integration test environment
func setupIntegrationTest(t *testing.T) (*httptest.Server, *httptest.Server, *service.UserService, func()) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, testDatabaseURL)
	require.NoError(t, err, "failed to connect to test database")

	err = pool.Ping(ctx)
	require.NoError(t, err, "failed to ping test database")

	// Setup logger
	var buf bytes.Buffer
	logger := observability.NewLoggerWithWriter("dev", "integration-test", &buf)

	// Setup JWT manager
	jwtManager, err := auth.NewJWTManager(
		"test-secret-key-must-be-at-least-32-characters-long-for-security",
		15*time.Minute,
		7*24*time.Hour,
	)
	require.NoError(t, err)

	// Setup repositories
	userRepo := repository.NewUserRepository(pool, logger)
	refreshTokenRepo := repository.NewRefreshTokenRepository(pool, logger)
	auditRepo := repository.NewAuditRepository(pool, logger)

	// Create test config
	testCfg := &config.Config{
		Audit: config.AuditConfig{
			RetentionDays: 90,
		},
	}

	// Setup service
	userService, err := service.NewUserService(
		userRepo,
		refreshTokenRepo,
		"test-secret-key-must-be-at-least-32-characters-long-for-security",
		15*time.Minute,
		7*24*time.Hour,
		logger,
		nil, // No event publisher in integration tests
	)
	require.NoError(t, err)

	// Get JWT manager from service (we'll need it for routers)
	jwtManager, jwtErr := auth.NewJWTManager(
		"test-secret-key-must-be-at-least-32-characters-long-for-security",
		15*time.Minute,
		7*24*time.Hour,
	)
	require.NoError(t, jwtErr)

	// Setup HTTP routers
	userRouter := httpTransport.SetupUserRouter(userService, jwtManager, auditRepo, testCfg, logger, "test", false)
	adminRouter := httpTransport.SetupAdminRouter(userService, jwtManager, auditRepo, testCfg, logger, "test", false)

	// Create test servers
	userServer := httptest.NewServer(userRouter)
	adminServer := httptest.NewServer(adminRouter)

	cleanup := func() {
		userServer.Close()
		adminServer.Close()
		pool.Close()
	}

	return userServer, adminServer, userService, cleanup
}

// TestAdminWorkflow_CompleteLifecycle tests the complete admin user management workflow
func TestAdminWorkflow_CompleteLifecycle(t *testing.T) {
	userServer, adminServer, userService, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Step 1: Register two regular users
	user1Email := "user1_" + uuid.New().String() + "@example.com"
	user2Email := "user2_" + uuid.New().String() + "@example.com"

	user1 := registerUser(t, userServer.URL, user1Email, "John", "Doe", "password123")
	user2 := registerUser(t, userServer.URL, user2Email, "Jane", "Smith", "password456")

	assert.NotNil(t, user1)
	assert.NotNil(t, user2)
	// Note: UserDTO doesn't expose role field initially - we'll verify via admin endpoints

	// Step 2: Promote user1 to admin using service (simulating initial admin setup)
	user1ID, err := uuid.Parse(user1["id"].(string))
	require.NoError(t, err)
	_, err = userService.UpdateUserRole(ctx, user1ID, domain.RoleAdmin)
	require.NoError(t, err)

	// Step 3: User1 logs in as admin
	adminTokens := adminLogin(t, adminServer.URL, user1Email, "password123")
	assert.NotEmpty(t, adminTokens["access_token"])
	assert.NotEmpty(t, adminTokens["refresh_token"])

	// Step 4: Verify user2 cannot login to admin panel (is not admin)
	resp := attemptAdminLogin(t, adminServer.URL, user2Email, "password456")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Step 5: Admin (user1) lists all users
	accessToken := adminTokens["access_token"].(string)
	users := listUsers(t, adminServer.URL, accessToken, 10, 0)
	assert.GreaterOrEqual(t, len(users), 2, "Should have at least 2 users")

	// Step 6: Admin searches for user2 by email
	searchResults := searchUsers(t, adminServer.URL, accessToken, user2Email, 10, 0)
	assert.Len(t, searchResults, 1)
	assert.Equal(t, user2Email, searchResults[0]["email"])

	// Step 7: Admin promotes user2 to admin
	user2ID, err := uuid.Parse(user2["id"].(string))
	require.NoError(t, err)
	updateUserRole(t, adminServer.URL, accessToken, user2ID.String(), "admin")

	// Step 8: Verify user2 can now login to admin panel
	user2AdminTokens := adminLogin(t, adminServer.URL, user2Email, "password456")
	assert.NotEmpty(t, user2AdminTokens["access_token"])

	// Step 9: Admin refreshes token and verifies admin role persists
	refreshedTokens := adminRefreshToken(t, adminServer.URL, adminTokens["refresh_token"].(string))
	assert.NotEmpty(t, refreshedTokens["access_token"])

	// Step 10: Admin demotes user2 back to regular user
	updateUserRole(t, adminServer.URL, accessToken, user2ID.String(), "user")

	// Step 11: Verify user2 can no longer login to admin panel
	resp = attemptAdminLogin(t, adminServer.URL, user2Email, "password456")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Step 12: Admin views system stats
	stats := getSystemStats(t, adminServer.URL, accessToken)
	assert.Greater(t, stats["total_users"].(float64), 0.0)
	assert.GreaterOrEqual(t, stats["active_sessions"].(float64), 0.0)
}

// TestAdminWorkflow_SessionManagement tests admin session management capabilities
func TestAdminWorkflow_SessionManagement(t *testing.T) {
	userServer, adminServer, userService, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create admin user
	adminEmail := "admin_" + uuid.New().String() + "@example.com"
	adminUser := registerUser(t, userServer.URL, adminEmail, "Admin", "User", "adminpass")
	adminID, err := uuid.Parse(adminUser["id"].(string))
	require.NoError(t, err)
	_, err = userService.UpdateUserRole(ctx, adminID, domain.RoleAdmin)
	require.NoError(t, err)

	// Admin logs in
	adminTokens := adminLogin(t, adminServer.URL, adminEmail, "adminpass")
	accessToken := adminTokens["access_token"].(string)

	// Create multiple users with sessions
	user1Email := "sessuser1_" + uuid.New().String() + "@example.com"
	user2Email := "sessuser2_" + uuid.New().String() + "@example.com"

	registerUser(t, userServer.URL, user1Email, "User", "One", "password1")
	registerUser(t, userServer.URL, user2Email, "User", "Two", "password2")

	// Users log in to create sessions
	user1Login := userLogin(t, userServer.URL, user1Email, "password1")
	user2Login := userLogin(t, userServer.URL, user2Email, "password2")

	// Admin lists all active sessions
	sessions := getAllSessions(t, adminServer.URL, accessToken, 20, 0)
	initialSessionCount := len(sessions)
	assert.Greater(t, initialSessionCount, 0)

	// Admin force logs out user1
	user1RefreshToken := user1Login["refresh_token"].(string)
	forceLogout(t, adminServer.URL, accessToken, user1RefreshToken)

	// Verify user1's session is revoked (refresh should fail)
	resp := attemptUserRefreshToken(t, userServer.URL, user1RefreshToken)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Verify user2's session still works
	user2RefreshToken := user2Login["refresh_token"].(string)
	user2Refreshed := userRefreshToken(t, userServer.URL, user2RefreshToken)
	assert.NotEmpty(t, user2Refreshed["access_token"])

	// Admin views updated stats
	stats := getSystemStats(t, adminServer.URL, accessToken)
	assert.Greater(t, stats["active_sessions"].(float64), 0.0)
}

// TestAdminWorkflow_AuthorizationEnforcement tests that non-admin users are properly blocked
func TestAdminWorkflow_AuthorizationEnforcement(t *testing.T) {
	userServer, adminServer, _, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Create regular user
	userEmail := "regular_" + uuid.New().String() + "@example.com"
	registerUser(t, userServer.URL, userEmail, "Regular", "User", "password123")

	// Regular user logs in to user service
	userTokens := userLogin(t, userServer.URL, userEmail, "password123")
	userAccessToken := userTokens["access_token"].(string)

	// Attempt to access admin endpoints with user token
	testCases := []struct {
		name       string
		endpoint   string
		method     string
		body       map[string]interface{}
		shouldFail bool
	}{
		{
			name:       "list users",
			endpoint:   "/admin/users",
			method:     "GET",
			shouldFail: true,
		},
		{
			name:       "search users",
			endpoint:   "/admin/users/search?query=test",
			method:     "GET",
			shouldFail: true,
		},
		{
			name:       "get user",
			endpoint:   "/admin/users/" + uuid.New().String(),
			method:     "GET",
			shouldFail: true,
		},
		{
			name:     "update user role",
			endpoint: "/admin/users/" + uuid.New().String() + "/role",
			method:   "PUT",
			body:     map[string]interface{}{"role": "admin"},
			shouldFail: true,
		},
		{
			name:       "get all sessions",
			endpoint:   "/admin/sessions",
			method:     "GET",
			shouldFail: true,
		},
		{
			name:     "force logout",
			endpoint: "/admin/sessions/revoke",
			method:   "POST",
			body:     map[string]interface{}{"refresh_token": "fake-token"},
			shouldFail: true,
		},
		{
			name:       "get system stats",
			endpoint:   "/admin/stats",
			method:     "GET",
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader *bytes.Reader
			if tc.body != nil {
				bodyBytes, _ := json.Marshal(tc.body)
				bodyReader = bytes.NewReader(bodyBytes)
			} else {
				bodyReader = bytes.NewReader([]byte{})
			}

			req, err := http.NewRequest(tc.method, adminServer.URL+tc.endpoint, bodyReader)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+userAccessToken)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should get 403 Forbidden (user is authenticated but not admin)
			assert.Equal(t, http.StatusForbidden, resp.StatusCode, 
				"Regular user should be forbidden from "+tc.name)
		})
	}
}

// TestAdminWorkflow_TokenRefreshPreservesRole tests that token refresh maintains admin role
func TestAdminWorkflow_TokenRefreshPreservesRole(t *testing.T) {
	userServer, adminServer, userService, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create and promote admin
	adminEmail := "admin_" + uuid.New().String() + "@example.com"
	adminUser := registerUser(t, userServer.URL, adminEmail, "Admin", "Test", "adminpass")
	adminID, err := uuid.Parse(adminUser["id"].(string))
	require.NoError(t, err)
	_, err = userService.UpdateUserRole(ctx, adminID, domain.RoleAdmin)
	require.NoError(t, err)

	// Admin logs in
	tokens1 := adminLogin(t, adminServer.URL, adminEmail, "adminpass")

	// Refresh token multiple times
	tokens2 := adminRefreshToken(t, adminServer.URL, tokens1["refresh_token"].(string))
	tokens3 := adminRefreshToken(t, adminServer.URL, tokens2["refresh_token"].(string))

	// Verify all tokens work for admin operations
	accessToken3 := tokens3["access_token"].(string)
	users := listUsers(t, adminServer.URL, accessToken3, 10, 0)
	assert.NotEmpty(t, users, "Admin should be able to list users after multiple token refreshes")

	stats := getSystemStats(t, adminServer.URL, accessToken3)
	assert.NotNil(t, stats, "Admin should be able to get stats after token refresh")
}

// Helper functions

func registerUser(t *testing.T, baseURL, email, firstName, lastName, password string) map[string]interface{} {
	t.Helper()

	payload := map[string]interface{}{
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
		"password":   password,
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		t.Fatalf("Expected 201, got %d: %v", resp.StatusCode, errResp)
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	return result["user"].(map[string]interface{})
}

func userLogin(t *testing.T, baseURL, email, password string) map[string]interface{} {
	t.Helper()

	payload := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	return result
}

func adminLogin(t *testing.T, baseURL, email, password string) map[string]interface{} {
	t.Helper()

	payload := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/admin/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	return result
}

func attemptAdminLogin(t *testing.T, baseURL, email, password string) *http.Response {
	t.Helper()

	payload := map[string]interface{}{
		"email":    email,
		"password": password,
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/admin/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)

	return resp
}

func adminRefreshToken(t *testing.T, baseURL, refreshToken string) map[string]interface{} {
	t.Helper()

	payload := map[string]interface{}{
		"refresh_token": refreshToken,
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/admin/auth/refresh", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	return result
}

func userRefreshToken(t *testing.T, baseURL, refreshToken string) map[string]interface{} {
	t.Helper()

	payload := map[string]interface{}{
		"refresh_token": refreshToken,
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	return result
}

func attemptUserRefreshToken(t *testing.T, baseURL, refreshToken string) *http.Response {
	t.Helper()

	payload := map[string]interface{}{
		"refresh_token": refreshToken,
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(baseURL+"/api/v1/auth/refresh", "application/json", bytes.NewReader(body))
	require.NoError(t, err)

	return resp
}

func listUsers(t *testing.T, baseURL, accessToken string, limit, offset int) []map[string]interface{} {
	t.Helper()

	url := fmt.Sprintf("%s/admin/users?limit=%d&offset=%d", baseURL, limit, offset)
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	users := result["users"].([]interface{})
	userList := make([]map[string]interface{}, len(users))
	for i, u := range users {
		userList[i] = u.(map[string]interface{})
	}

	return userList
}

func searchUsers(t *testing.T, baseURL, accessToken, query string, limit, offset int) []map[string]interface{} {
	t.Helper()

	url := fmt.Sprintf("%s/admin/users/search?query=%s&limit=%d&offset=%d", baseURL, query, limit, offset)
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	users := result["users"].([]interface{})
	userList := make([]map[string]interface{}, len(users))
	for i, u := range users {
		userList[i] = u.(map[string]interface{})
	}

	return userList
}

func updateUserRole(t *testing.T, baseURL, accessToken, userID, role string) {
	t.Helper()

	payload := map[string]interface{}{
		"role": role,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("PUT", baseURL+"/admin/users/"+userID+"/role", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func getAllSessions(t *testing.T, baseURL, accessToken string, limit, offset int) []map[string]interface{} {
	t.Helper()

	url := fmt.Sprintf("%s/admin/sessions?limit=%d&offset=%d", baseURL, limit, offset)
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	sessions := result["sessions"].([]interface{})
	sessionList := make([]map[string]interface{}, len(sessions))
	for i, s := range sessions {
		sessionList[i] = s.(map[string]interface{})
	}

	return sessionList
}

func forceLogout(t *testing.T, baseURL, accessToken, refreshToken string) {
	t.Helper()

	payload := map[string]interface{}{
		"token": refreshToken,  // Note: field name is 'token' in AdminForceLogoutRequest
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", baseURL+"/admin/sessions/revoke", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		t.Fatalf("Force logout failed. Expected 200, got %d: %v", resp.StatusCode, errResp)
	}
}

func getSystemStats(t *testing.T, baseURL, accessToken string) map[string]interface{} {
	t.Helper()

	req, err := http.NewRequest("GET", baseURL+"/admin/stats", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	return result
}
