-- Add role column to users table
-- Migration: 000004_add_user_role
-- Description: Add role column for role-based access control (RBAC)

-- Add role column with default 'user' value
ALTER TABLE users
ADD COLUMN role TEXT NOT NULL DEFAULT 'user';

-- Add check constraint to ensure role is valid
ALTER TABLE users
ADD CONSTRAINT users_role_check CHECK (role IN ('user', 'admin'));

-- Create index on role for efficient filtering
CREATE INDEX idx_users_role ON users(role) WHERE deleted_at IS NULL;

-- Add comment for documentation
COMMENT ON COLUMN users.role IS 'User role for authorization: user (default) or admin';
