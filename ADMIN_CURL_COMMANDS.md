# Admin API - cURL Commands Reference

## Important: Two-Server Architecture

**User Server (Port 8080):** Public-facing API for regular users  
**Admin Server (Port 8081):** Admin-only API (separate network/port for security)

- ✅ Admin login/refresh: `http://localhost:8081/admin/auth/*`
- ✅ Admin endpoints: `http://localhost:8081/admin/*`
- ❌ Admins **cannot** login via port 8080 (security separation)

---

## Quick Start - Create First Admin User & Login

```bash
# 1. Start the complete dev environment (both servers: 8080 + 8081)
make dev

# 2. In another terminal, register a new user on USER server (port 8080)
curl -X POST "http://localhost:8080/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@pandora.com",
    "password": "Admin123",
    "first_name": "Admin",
    "last_name": "User"
  }'

# 3. Promote user to admin role in database
docker exec -it pandora-postgres psql -U pandora -d pandora_dev -c \
  "UPDATE users SET role = 'admin' WHERE email = 'admin@pandora.com';"

# 4. Login via ADMIN server (port 8081) - this is the KEY difference!
curl -X POST "http://localhost:8081/admin/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@pandora.com",
    "password": "Admin123"
  }'

# 5. Extract and export the access token from the response
export ADMIN_TOKEN="eyJhbGc..."
```

---

## Prerequisites

**Before using admin endpoints, you need an admin access token from the admin server (port 8081).**

```bash
# 1. Login via ADMIN server to get access token
curl -X POST "http://localhost:8081/admin/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "your_password"
  }'

# 2. Export the access token
export ADMIN_TOKEN="eyJhbGc..."
```

---

## Admin Authentication Endpoints (Port 8081)

### Admin Login

**POST** `http://localhost:8081/admin/auth/login`

```bash
curl -X POST "http://localhost:8081/admin/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@pandora.com",
    "password": "Admin123"
  }'
```

**Response:**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "admin@pandora.com",
    "first_name": "Admin",
    "last_name": "User",
    "kyc_status": "verified",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "refresh_abc123...",
  "expires_at": "2024-01-01T01:00:00Z"
}
```

### Admin Refresh Token

**POST** `http://localhost:8081/admin/auth/refresh`

```bash
curl -X POST "http://localhost:8081/admin/auth/refresh" \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "refresh_abc123..."
  }'
```

---

## Admin Endpoints (Port 8081)

### 1. List All Users

**All admin endpoints use port 8081**

```bash
curl -X GET "http://localhost:8081/admin/users?limit=20&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json"
```

### 2. Search Users

```bash
curl -X GET "http://localhost:8081/admin/users/search?query=john&limit=20&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json"
```

### 3. Get User by ID

```bash
curl -X GET "http://localhost:8081/admin/users/550e8400-e29b-41d4-a716-446655440000" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json"
```

### 4. Update User Role (Promote to Admin)

```bash
curl -X PUT "http://localhost:8081/admin/users/550e8400-e29b-41d4-a716-446655440000/role" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role": "admin"
  }'
```

### 5. Update User Role (Demote to User)

```bash
curl -X PUT "http://localhost:8081/admin/users/550e8400-e29b-41d4-a716-446655440000/role" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role": "user"
  }'
```

### 6. Get All Active Sessions

```bash
curl -X GET "http://localhost:8081/admin/sessions?limit=50&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json"
```

### 7. Force Logout (Revoke Session)

```bash
curl -X POST "http://localhost:8081/admin/sessions/revoke" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "token": "refresh_token_string_to_revoke"
  }'
```

### 8. Get System Statistics

```bash
curl -X GET "http://localhost:8081/admin/stats" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json"
```

---

## Pretty Print Output (with jq)

```bash
# Install jq if needed
brew install jq

# Use with any admin command (port 8081)
curl -X GET "http://localhost:8081/admin/users?limit=5&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" | jq '.'
```

---

## Quick Test Script

```bash
#!/bin/bash

# Login via ADMIN server and get token
RESPONSE=$(curl -s -X POST "http://localhost:8081/admin/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@pandora.com",
    "password": "Admin123"
  }')

# Extract access token
export ADMIN_TOKEN=$(echo $RESPONSE | jq -r '.access_token')

echo "Logged in! Token: ${ADMIN_TOKEN:0:20}..."

# Test admin endpoints (all on port 8081)
echo "\n📋 Listing users..."
curl -s -X GET "http://localhost:8081/admin/users?limit=5&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.users[] | {email, role}'

echo "\n📊 System stats..."
curl -s -X GET "http://localhost:8081/admin/stats" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.'

echo "\n🔐 Active sessions..."
curl -s -X GET "http://localhost:8081/admin/sessions?limit=5&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.sessions[] | {user_id, ip_address}'
```

---

## Security Notes

**Why Two Servers?**

1. **Network Isolation**: Admin server (8081) can be firewalled/VPN-only in production
2. **Attack Surface**: Public users never touch admin endpoints (different process)
3. **Audit Trail**: Clear separation between user and admin actions in logs
4. **Defense in Depth**: Even if user server is compromised, admin server remains isolated

**Production Deployment:**
- User server (8080): Public internet
- Admin server (8081): Internal network / VPN / Bastion host only
- Different JWT secrets (optional)
- Mutual TLS for admin server (optional)
