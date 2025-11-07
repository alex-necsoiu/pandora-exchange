-- Split full_name into first_name and last_name
-- Migration: 000003

-- Add new columns
ALTER TABLE users ADD COLUMN first_name VARCHAR(100);
ALTER TABLE users ADD COLUMN last_name VARCHAR(100);

-- Migrate existing data (split full_name on first space)
UPDATE users 
SET 
    first_name = COALESCE(SPLIT_PART(full_name, ' ', 1), ''),
    last_name = CASE 
        WHEN full_name IS NOT NULL AND POSITION(' ' IN full_name) > 0 
        THEN SUBSTRING(full_name FROM POSITION(' ' IN full_name) + 1)
        ELSE ''
    END;

-- Make columns NOT NULL after data migration (with default empty string)
ALTER TABLE users ALTER COLUMN first_name SET DEFAULT '';
ALTER TABLE users ALTER COLUMN last_name SET DEFAULT '';
ALTER TABLE users ALTER COLUMN first_name SET NOT NULL;
ALTER TABLE users ALTER COLUMN last_name SET NOT NULL;

-- Drop old column
ALTER TABLE users DROP COLUMN full_name;

-- Add indexes for searching
CREATE INDEX idx_users_first_name ON users(first_name);
CREATE INDEX idx_users_last_name ON users(last_name);
