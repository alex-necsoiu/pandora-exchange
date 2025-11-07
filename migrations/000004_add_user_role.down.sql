-- Rollback: Remove role column from users table
-- Migration: 000004_add_user_role (down)

-- Drop index
DROP INDEX IF EXISTS idx_users_role;

-- Drop check constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;

-- Drop role column
ALTER TABLE users DROP COLUMN IF EXISTS role;
