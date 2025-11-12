# 📡 API Documentation

Complete API reference for Pandora Exchange User Service.

## 📋 Table of Contents

- [REST API](#rest-api)
- [gRPC API](#grpc-api)
- [Authentication](#authentication)
- [Rate Limiting](#rate-limiting)
- [Error Handling](#error-handling)
- [Examples](#examples)

---

## REST API

**Base URL:** `http://localhost:8080/api/v1`

**Interactive Documentation:** [Swagger UI](http://localhost:8080/swagger/index.html)

### Authentication Endpoints

#### Register New User

```http
POST /api/v1/auth/register
Content-Type: application/json
```

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "first_name": "John",
  "last_name": "Doe"
}
```

**Response:** `201 Created`
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "role": "user",
    "kyc_status": "pending",
    "created_at": "2024-11-12T10:30:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "a3bb189e-8bf9-3888-9912-ace4e6543002",
  "expires_in": 900
}
```

**Errors:**
- `400 Bad Request` - Invalid input (weak password, invalid email)
- `409 Conflict` - Email already exists
- `500 Internal Server Error` - Server error

---

#### Login

```http
POST /api/v1/auth/login
Content-Type: application/json
```

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!"
}
```

**Response:** `200 OK`
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "role": "user",
    "kyc_status": "verified"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "a3bb189e-8bf9-3888-9912-ace4e6543002",
  "expires_in": 900
}
```

**Errors:**
- `400 Bad Request` - Missing email or password
- `401 Unauthorized` - Invalid credentials
- `429 Too Many Requests` - Rate limit exceeded (5 attempts per 15 minutes)

---

#### Refresh Access Token

```http
POST /api/v1/auth/refresh
Content-Type: application/json
```

**Request Body:**
```json
{
  "refresh_token": "a3bb189e-8bf9-3888-9912-ace4e6543002"
}
```

**Response:** `200 OK`
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "b4cc290f-9cg0-4999-0023-bdf5f7654113",
  "expires_in": 900
}
```

**Errors:**
- `400 Bad Request` - Missing refresh token
- `401 Unauthorized` - Invalid or expired refresh token

---

### User Endpoints

#### Get Current User Profile

```http
GET /api/v1/users/me
Authorization: Bearer <access_token>
```

**Response:** `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "role": "user",
  "kyc_status": "verified",
  "created_at": "2024-11-12T10:30:00Z",
  "updated_at": "2024-11-12T10:30:00Z"
}
```

**Errors:**
- `401 Unauthorized` - Missing or invalid access token
- `404 Not Found` - User not found

---

#### Update User Profile

```http
PATCH /api/v1/users/me
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "first_name": "Jane",
  "last_name": "Smith"
}
```

**Response:** `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "first_name": "Jane",
  "last_name": "Smith",
  "role": "user",
  "kyc_status": "verified",
  "updated_at": "2024-11-12T11:45:00Z"
}
```

**Errors:**
- `400 Bad Request` - Invalid input
- `401 Unauthorized` - Missing or invalid access token

---

#### Logout (Current Session)

```http
POST /api/v1/users/me/logout
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "refresh_token": "a3bb189e-8bf9-3888-9912-ace4e6543002"
}
```

**Response:** `204 No Content`

---

#### Logout All Sessions

```http
POST /api/v1/users/me/logout-all
Authorization: Bearer <access_token>
```

**Response:** `204 No Content`

---

### Admin Endpoints

> ⚠️ **Requires Admin Role**

#### Get User by ID

```http
GET /api/v1/admin/users/:id
Authorization: Bearer <admin_access_token>
```

**Response:** `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "first_name": "John",
  "last_name": "Doe",
  "role": "user",
  "kyc_status": "pending",
  "created_at": "2024-11-12T10:30:00Z",
  "updated_at": "2024-11-12T10:30:00Z",
  "deleted_at": null
}
```

**Errors:**
- `401 Unauthorized` - Not authenticated
- `403 Forbidden` - Not an admin
- `404 Not Found` - User not found

---

#### Update KYC Status

```http
PATCH /api/v1/admin/users/:id/kyc
Authorization: Bearer <admin_access_token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "kyc_status": "verified"
}
```

**Valid Statuses:** `pending`, `verified`, `rejected`

**Response:** `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "kyc_status": "verified",
  "updated_at": "2024-11-12T12:00:00Z"
}
```

---

#### Soft Delete User

```http
DELETE /api/v1/admin/users/:id
Authorization: Bearer <admin_access_token>
```

**Response:** `204 No Content`

**Errors:**
- `403 Forbidden` - Not an admin
- `404 Not Found` - User not found

---

### System Endpoints

#### Health Check

```http
GET /health
```

**Response:** `200 OK`
```json
{
  "status": "ok",
  "timestamp": "2024-11-12T10:30:00Z"
}
```

---

#### Readiness Check

```http
GET /ready
```

**Response:** `200 OK`
```json
{
  "status": "ready",
  "checks": {
    "database": "ok",
    "redis": "ok"
  },
  "timestamp": "2024-11-12T10:30:00Z"
}
```

**Response if not ready:** `503 Service Unavailable`
```json
{
  "status": "not_ready",
  "checks": {
    "database": "error",
    "redis": "ok"
  },
  "errors": ["database connection failed"]
}
```

---

## gRPC API

**Address:** `localhost:9090`

**Protocol Buffers:** See [proto/user_service.proto](../proto/user_service.proto)

### Service Definition

```protobuf
service UserService {
  // User retrieval
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc GetUserByEmail(GetUserByEmailRequest) returns (GetUserResponse);
  
  // KYC management
  rpc UpdateKYCStatus(UpdateKYCRequest) returns (UpdateKYCResponse);
  
  // User validation
  rpc ValidateUser(ValidateUserRequest) returns (ValidateUserResponse);
  
  // Admin operations
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
}
```

### GetUser

**Request:**
```protobuf
message GetUserRequest {
  string user_id = 1;
}
```

**Response:**
```protobuf
message GetUserResponse {
  string id = 1;
  string email = 2;
  string first_name = 3;
  string last_name = 4;
  string role = 5;
  string kyc_status = 6;
  google.protobuf.Timestamp created_at = 7;
  google.protobuf.Timestamp updated_at = 8;
}
```

**Example (grpcurl):**
```bash
grpcurl -plaintext \
  -d '{"user_id": "550e8400-e29b-41d4-a716-446655440000"}' \
  localhost:9090 \
  user.v1.UserService/GetUser
```

---

### GetUserByEmail

**Request:**
```protobuf
message GetUserByEmailRequest {
  string email = 1;
}
```

**Example:**
```bash
grpcurl -plaintext \
  -d '{"email": "user@example.com"}' \
  localhost:9090 \
  user.v1.UserService/GetUserByEmail
```

---

### UpdateKYCStatus

**Request:**
```protobuf
message UpdateKYCRequest {
  string user_id = 1;
  string kyc_status = 2;
}
```

**Response:**
```protobuf
message UpdateKYCResponse {
  bool success = 1;
  string message = 2;
}
```

**Example:**
```bash
grpcurl -plaintext \
  -d '{"user_id": "550e8400-e29b-41d4-a716-446655440000", "kyc_status": "verified"}' \
  localhost:9090 \
  user.v1.UserService/UpdateKYCStatus
```

---

### ValidateUser

**Request:**
```protobuf
message ValidateUserRequest {
  string email = 1;
  string password = 2;
}
```

**Response:**
```protobuf
message ValidateUserResponse {
  bool valid = 1;
  string user_id = 2;
}
```

---

### ListUsers

**Request:**
```protobuf
message ListUsersRequest {
  int32 page = 1;
  int32 page_size = 2;
  string filter = 3;
}
```

**Response:**
```protobuf
message ListUsersResponse {
  repeated GetUserResponse users = 1;
  int32 total_count = 2;
  int32 page = 3;
  int32 page_size = 4;
}
```

---

## Authentication

### JWT Token Structure

**Header:**
```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

**Payload:**
```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "role": "user",
  "iat": 1699785600,
  "exp": 1699786500,
  "iss": "pandora-exchange"
}
```

### Using JWT in Requests

**Header:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Example (curl):**
```bash
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/api/v1/users/me
```

### Token Expiration

| Token Type | Expiry | Renewable |
|------------|--------|-----------|
| Access Token | 15 minutes | Via refresh token |
| Refresh Token | 7 days | No (must re-login) |

---

## Rate Limiting

### Limits

| Endpoint Pattern | Limit | Window |
|-----------------|-------|--------|
| **Global** (per IP) | 100 requests | 1 minute |
| **Authenticated** (per user) | 60 requests | 1 minute |
| **Login** (`/auth/login`) | 5 attempts | 15 minutes |
| **Register** (`/auth/register`) | 10 attempts | 1 hour |

### Rate Limit Headers

**Response Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1699785660
```

### Rate Limit Exceeded Response

**Status:** `429 Too Many Requests`

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded. Try again in 45 seconds.",
    "details": {
      "limit": 100,
      "window": "1m",
      "retry_after": 45
    }
  }
}
```

---

## Error Handling

### Error Response Format

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "trace_id": "550e8400-e29b-41d4-a716-446655440000",
    "details": {}
  }
}
```

### HTTP Status Codes

| Code | Meaning | Common Scenarios |
|------|---------|------------------|
| `200` | OK | Successful GET, PATCH |
| `201` | Created | Successful POST (register, create) |
| `204` | No Content | Successful DELETE, logout |
| `400` | Bad Request | Invalid input, validation errors |
| `401` | Unauthorized | Missing/invalid token, wrong credentials |
| `403` | Forbidden | Insufficient permissions (not admin) |
| `404` | Not Found | User not found, endpoint not found |
| `409` | Conflict | Email already exists, duplicate resource |
| `429` | Too Many Requests | Rate limit exceeded |
| `500` | Internal Server Error | Unexpected server error |
| `503` | Service Unavailable | Service not ready (DB down) |

### Error Codes

| Code | Description |
|------|-------------|
| `VALIDATION_ERROR` | Input validation failed |
| `USER_NOT_FOUND` | User does not exist |
| `USER_EXISTS` | Email already registered |
| `INVALID_CREDENTIALS` | Wrong email or password |
| `UNAUTHORIZED` | Authentication required |
| `FORBIDDEN` | Insufficient permissions |
| `RATE_LIMIT_EXCEEDED` | Too many requests |
| `INTERNAL_ERROR` | Unexpected server error |

---

## Examples

### Complete Registration Flow

```bash
# 1. Register new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "password": "SecurePass123!",
    "first_name": "Alice",
    "last_name": "Johnson"
  }'

# Response:
# {
#   "user": { "id": "...", "email": "newuser@example.com", ... },
#   "access_token": "eyJ...",
#   "refresh_token": "a3b...",
#   "expires_in": 900
# }

# 2. Get user profile
ACCESS_TOKEN="eyJ..."
curl -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://localhost:8080/api/v1/users/me

# 3. Update profile
curl -X PATCH http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Alicia"
  }'

# 4. Logout
curl -X POST http://localhost:8080/api/v1/users/me/logout \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "a3b..."
  }'
```

### Admin Operations

```bash
# Get admin token (must login as admin)
ADMIN_TOKEN="eyJ..."

# Get user by ID
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:8080/api/v1/admin/users/550e8400-e29b-41d4-a716-446655440000

# Update KYC status
curl -X PATCH http://localhost:8080/api/v1/admin/users/550e8400-e29b-41d4-a716-446655440000/kyc \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "kyc_status": "verified"
  }'

# Delete user
curl -X DELETE http://localhost:8080/api/v1/admin/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

## Related Documentation

- 🏗️ [Architecture Overview](../ARCHITECTURE.md)
- 🔐 [Security Guidelines](./SECURITY.md)
- 🧪 [Testing Guide](./TESTING.md)
- 🚀 [Quick Start](./QUICK_START.md)

---

**Last Updated:** November 12, 2025
