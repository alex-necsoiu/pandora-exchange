-- name: CreateUser :one
-- CreateUser creates a new user with the provided email, full name, and hashed password.
-- Returns the created user record.
INSERT INTO users (
    email,
    full_name,
    hashed_password,
    kyc_status
) VALUES (
    $1, $2, $3, 'pending'
) RETURNING *;

-- name: GetUserByID :one
-- GetUserByID retrieves a user by their unique ID.
-- Returns error if user not found or soft-deleted.
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
-- GetUserByEmail retrieves a user by their email address.
-- Returns error if user not found or soft-deleted.
SELECT * FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: UpdateUserKYCStatus :one
-- UpdateUserKYCStatus updates the KYC verification status for a user.
-- Valid statuses: pending, verified, rejected
UPDATE users
SET kyc_status = $2
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteUser :execrows
-- SoftDeleteUser marks a user as deleted without removing the record.
-- Sets deleted_at timestamp to current time.
UPDATE users
SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListUsers :many
-- ListUsers retrieves paginated list of active users.
-- Supports filtering and pagination.
SELECT * FROM users
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
-- CountUsers returns the total count of active users.
SELECT COUNT(*) FROM users
WHERE deleted_at IS NULL;

-- name: UpdateUserProfile :one
-- UpdateUserProfile updates user's profile information (full_name).
UPDATE users
SET full_name = $2
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;
