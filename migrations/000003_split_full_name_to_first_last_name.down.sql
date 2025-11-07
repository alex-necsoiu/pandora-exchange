-- Revert split of full_name
-- Migration: 000003 (Down)

-- Add back full_name column
ALTER TABLE users ADD COLUMN full_name VARCHAR(255);

-- Restore data by concatenating first_name and last_name
UPDATE users 
SET full_name = CONCAT(first_name, ' ', last_name)
WHERE first_name IS NOT NULL AND last_name IS NOT NULL;

-- Make full_name NOT NULL
ALTER TABLE users ALTER COLUMN full_name SET NOT NULL;

-- Drop the split columns and their indexes
DROP INDEX IF EXISTS idx_users_first_name;
DROP INDEX IF EXISTS idx_users_last_name;
ALTER TABLE users DROP COLUMN first_name;
ALTER TABLE users DROP COLUMN last_name;
