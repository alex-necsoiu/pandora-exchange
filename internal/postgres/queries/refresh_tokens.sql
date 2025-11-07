-- name: CreateRefreshToken :one
-- CreateRefreshToken stores a new refresh token for a user.
-- Includes audit information (IP address and user agent).
INSERT INTO refresh_tokens (
    token,
    user_id,
    expires_at,
    ip_address,
    user_agent
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetRefreshToken :one
-- GetRefreshToken retrieves a refresh token by its value.
-- Returns the token regardless of revoked status (caller should check IsRevoked).
SELECT * FROM refresh_tokens
WHERE token = $1;

-- name: RevokeRefreshToken :execrows
-- RevokeRefreshToken marks a refresh token as revoked.
-- Sets revoked_at timestamp to current time.
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE token = $1 AND revoked_at IS NULL;

-- name: RevokeAllUserTokens :exec
-- RevokeAllUserTokens revokes all active refresh tokens for a user.
-- Used when user logs out from all devices or password changes.
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: DeleteExpiredTokens :exec
-- DeleteExpiredTokens removes expired refresh tokens from the database.
-- Should be run periodically as a cleanup job.
DELETE FROM refresh_tokens
WHERE expires_at < NOW();

-- name: GetUserActiveTokens :many
-- GetUserActiveTokens retrieves all active (non-expired, non-revoked) tokens for a user.
-- Useful for session management and "active devices" feature.
SELECT * FROM refresh_tokens
WHERE user_id = $1 
  AND revoked_at IS NULL 
  AND expires_at > NOW()
ORDER BY created_at DESC;

-- name: CountUserActiveTokens :one
-- CountUserActiveTokens returns the number of active sessions for a user.
SELECT COUNT(*) FROM refresh_tokens
WHERE user_id = $1 
  AND revoked_at IS NULL 
  AND expires_at > NOW();

-- name: GetAllActiveSessions :many
-- GetAllActiveSessions retrieves all active sessions across all users (admin only).
SELECT rt.*, u.email, u.first_name, u.last_name
FROM refresh_tokens rt
INNER JOIN users u ON rt.user_id = u.id
WHERE rt.revoked_at IS NULL 
  AND rt.expires_at > NOW()
  AND u.deleted_at IS NULL
ORDER BY rt.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAllActiveSessions :one
-- CountAllActiveSessions returns the total count of active sessions across all users (admin only).
SELECT COUNT(*) FROM refresh_tokens
WHERE revoked_at IS NULL 
  AND expires_at > NOW();

-- name: RevokeTokenByID :execrows
-- RevokeTokenByID revokes a specific refresh token by token value (admin only).
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE token = $1 AND revoked_at IS NULL;
