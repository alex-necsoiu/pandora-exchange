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
-- Returns error if token not found or revoked.
SELECT * FROM refresh_tokens
WHERE token = $1 AND revoked_at IS NULL;

-- name: RevokeRefreshToken :exec
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
