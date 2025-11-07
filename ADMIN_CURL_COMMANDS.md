# Admin API - cURL Commands Reference

## Prerequisites

```bash
# 1. Login to get access token
curl -X POST "http://localhost:8080/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "your_password"
  }'

# 2. Export the access token
export ADMIN_TOKEN="eyJhbGc..."
```

---

## Admin Endpoints

### 1. List All Users

```bash
curl -X GET "http://localhost:8080/api/v1/admin/users?limit=20&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json"
```

### 2. Search Users

```bash
curl -X GET "http://localhost:8080/api/v1/admin/users/search?query=john&limit=20&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json"
```

### 3. Get User by ID

```bash
curl -X GET "http://localhost:8080/api/v1/admin/users/550e8400-e29b-41d4-a716-446655440000" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json"
```

### 4. Update User Role (Promote to Admin)

```bash
curl -X PUT "http://localhost:8080/api/v1/admin/users/550e8400-e29b-41d4-a716-446655440000/role" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role": "admin"
  }'
```

### 5. Update User Role (Demote to User)

```bash
curl -X PUT "http://localhost:8080/api/v1/admin/users/550e8400-e29b-41d4-a716-446655440000/role" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role": "user"
  }'
```

### 6. Get All Active Sessions

```bash
curl -X GET "http://localhost:8080/api/v1/admin/sessions?limit=50&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json"
```

### 7. Force Logout (Revoke Session)

```bash
curl -X POST "http://localhost:8080/api/v1/admin/sessions/revoke" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "token": "refresh_token_string_to_revoke"
  }'
```

### 8. Get System Statistics

```bash
curl -X GET "http://localhost:8080/api/v1/admin/stats" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json"
```

---

## Pretty Print Output (with jq)

```bash
# Install jq if needed
brew install jq

# Use with any command
curl -X GET "http://localhost:8080/api/v1/admin/users?limit=5&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" | jq '.'
```

---

## Quick Test Script

```bash
#!/bin/bash

# Login and get token
RESPONSE=$(curl -s -X POST "http://localhost:8080/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "your_password"
  }')

# Extract access token
export ADMIN_TOKEN=$(echo $RESPONSE | jq -r '.access_token')

echo "Logged in! Token: ${ADMIN_TOKEN:0:20}..."

# Test endpoints
echo "\n📋 Listing users..."
curl -s -X GET "http://localhost:8080/api/v1/admin/users?limit=5&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.users[] | {email, role}'

echo "\n📊 System stats..."
curl -s -X GET "http://localhost:8080/api/v1/admin/stats" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.'

echo "\n🔐 Active sessions..."
curl -s -X GET "http://localhost:8080/api/v1/admin/sessions?limit=5&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.sessions[] | {user_id, ip_address}'
```
