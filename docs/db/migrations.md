# Database Migrations

> **Database migration guidelines for Pandora Exchange**  
> **Last Updated:** November 8, 2025

---

## Overview

Pandora Exchange uses [golang-migrate](https://github.com/golang-migrate/migrate) for managing database schema changes in a version-controlled, repeatable, and safe manner.

**Key Principles:**
- ✅ All schema changes via migrations (never manual ALTER TABLE)
- ✅ Migrations are immutable (never edit existing migrations)
- ✅ Migrations are reversible (always provide DOWN migration)
- ✅ Migrations are tested before production deployment
- ✅ Migrations run automatically in CI/CD pipeline

---

## Migration Files

### File Naming Convention

```
{version}_{description}.{direction}.sql
```

**Components:**
- `version`: 6-digit sequential number (000001, 000002, etc.)
- `description`: Snake_case description (e.g., `create_users_table`)
- `direction`: `up` (apply) or `down` (rollback)
- `sql`: File extension

**Examples:**
```
migrations/
├── 000001_create_users_table.up.sql
├── 000001_create_users_table.down.sql
├── 000002_create_refresh_tokens_table.up.sql
├── 000002_create_refresh_tokens_table.down.sql
├── 000003_split_full_name_to_first_last_name.up.sql
├── 000003_split_full_name_to_first_last_name.down.sql
├── 000004_add_user_role.up.sql
├── 000004_add_user_role.down.sql
├── 000005_create_audit_logs_table.up.sql
└── 000005_create_audit_logs_table.down.sql
```

---

## Creating Migrations

### Using migrate CLI

```bash
# Install migrate tool
brew install golang-migrate

# Create new migration
migrate create -ext sql -dir migrations -seq create_new_table

# This creates:
# migrations/000006_create_new_table.up.sql
# migrations/000006_create_new_table.down.sql
```

### Using Makefile

```bash
# Create migration with make
make migration name=add_email_verification
```

**Makefile target:**
```makefile
.PHONY: migration
migration:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name
```

---

## Writing Migrations

### UP Migration (000006_add_email_verification.up.sql)

```sql
-- Add email verification columns to users table
ALTER TABLE users 
    ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN email_verification_token TEXT,
    ADD COLUMN email_verification_sent_at TIMESTAMP;

-- Create index for verification token lookup
CREATE INDEX idx_users_email_verification_token 
    ON users(email_verification_token) 
    WHERE email_verification_token IS NOT NULL;

-- Add comment
COMMENT ON COLUMN users.email_verified IS 'Whether user has verified their email address';
```

### DOWN Migration (000006_add_email_verification.down.sql)

```sql
-- Remove email verification columns
DROP INDEX IF EXISTS idx_users_email_verification_token;

ALTER TABLE users 
    DROP COLUMN IF EXISTS email_verified,
    DROP COLUMN IF EXISTS email_verification_token,
    DROP COLUMN IF EXISTS email_verification_sent_at;
```

---

## Migration Best Practices

### DO ✅

**1. Make migrations atomic (single transaction)**
```sql
BEGIN;

CREATE TABLE new_table (...);
CREATE INDEX idx_new_table_id ON new_table(id);

COMMIT;
```

**2. Use IF EXISTS / IF NOT EXISTS**
```sql
-- Safe idempotent operations
CREATE TABLE IF NOT EXISTS users (...);
DROP TABLE IF EXISTS old_table;
ALTER TABLE users DROP COLUMN IF EXISTS deprecated_column;
```

**3. Add indexes concurrently (large tables)**
```sql
-- Won't block writes
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
```

**4. Set NOT NULL with defaults first**
```sql
-- Step 1: Add column with default
ALTER TABLE users ADD COLUMN role TEXT DEFAULT 'user';

-- Step 2: Backfill data (if needed)
UPDATE users SET role = 'admin' WHERE is_admin = TRUE;

-- Step 3: Make NOT NULL (next migration)
-- ALTER TABLE users ALTER COLUMN role SET NOT NULL;
```

**5. Use descriptive comments**
```sql
COMMENT ON TABLE users IS 'User accounts for Pandora Exchange platform';
COMMENT ON COLUMN users.kyc_verified IS 'KYC verification status (admin-controlled)';
```

**6. Handle foreign keys carefully**
```sql
-- Always specify ON DELETE behavior
ALTER TABLE refresh_tokens 
    ADD CONSTRAINT fk_refresh_tokens_user_id 
    FOREIGN KEY (user_id) REFERENCES users(id) 
    ON DELETE CASCADE;  -- Explicit cascade
```

### DON'T ❌

**1. Don't edit existing migrations**
```sql
❌ Never modify 000001_create_users_table.up.sql after it's been applied

✅ Create 000007_update_users_table.up.sql instead
```

**2. Don't use DROP without IF EXISTS**
```sql
❌ DROP TABLE old_table;  -- Fails if table doesn't exist

✅ DROP TABLE IF EXISTS old_table;
```

**3. Don't make irreversible changes without backups**
```sql
❌ DROP COLUMN user_data;  -- Data lost forever

✅ -- First: Rename column in migration 000008
   ALTER TABLE users RENAME COLUMN user_data TO user_data_deprecated;
   
   -- Later: Drop in migration 000012 (after verifying no usage)
   ALTER TABLE users DROP COLUMN user_data_deprecated;
```

**4. Don't add NOT NULL without defaults on large tables**
```sql
❌ ALTER TABLE users ADD COLUMN role TEXT NOT NULL;
   -- Fails on existing rows!

✅ ALTER TABLE users ADD COLUMN role TEXT DEFAULT 'user' NOT NULL;
```

**5. Don't forget indexes**
```sql
❌ ALTER TABLE users ADD COLUMN email TEXT UNIQUE;
   -- UNIQUE constraint creates index, but...
   
✅ -- Better: Explicit index with meaningful name
   ALTER TABLE users ADD COLUMN email TEXT;
   CREATE UNIQUE INDEX idx_users_email ON users(email);
```

**6. Don't run data migrations in DDL migrations**
```sql
❌ -- Bad: Mixing DDL and DML
   CREATE TABLE new_table (...);
   INSERT INTO new_table SELECT * FROM old_table;  -- Slow!
   DROP TABLE old_table;

✅ -- Good: Separate migrations
   -- 000010_create_new_table.up.sql (DDL only)
   -- 000011_migrate_data_to_new_table.up.sql (DML only)
```

---

## Running Migrations

### Locally (Development)

```bash
# Apply all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Force version (dangerous!)
make migrate-force VERSION=5

# Check current version
make migrate-version
```

**Makefile targets:**
```makefile
.PHONY: migrate-up
migrate-up:
	migrate -path migrations -database "${DATABASE_URL}" up

.PHONY: migrate-down
migrate-down:
	migrate -path migrations -database "${DATABASE_URL}" down 1

.PHONY: migrate-force
migrate-force:
	@read -p "Force version to: " version; \
	migrate -path migrations -database "${DATABASE_URL}" force $$version

.PHONY: migrate-version
migrate-version:
	migrate -path migrations -database "${DATABASE_URL}" version
```

### CI/CD Pipeline

**Automated migrations in deployment:**

```yaml
# .github/workflows/deploy.yml
- name: Run Database Migrations
  env:
    DATABASE_URL: ${{ secrets.DATABASE_URL }}
  run: |
    migrate -path migrations -database "$DATABASE_URL" up
```

**Kubernetes Init Container:**
```yaml
# deployments/k8s/base/deployment.yaml
initContainers:
  - name: migrate
    image: migrate/migrate:v4.16.2
    command:
      - migrate
      - -path=/migrations
      - -database=$(DATABASE_URL)
      - up
    env:
      - name: DATABASE_URL
        valueFrom:
          secretKeyRef:
            name: user-service-secrets
            key: database-url
    volumeMounts:
      - name: migrations
        mountPath: /migrations
volumes:
  - name: migrations
    configMap:
      name: user-service-migrations
```

---

## Migration Testing

### Test Before Production

**1. Test in local environment:**
```bash
# Apply migration
make migrate-up

# Run tests
make test

# Verify schema
psql -d pandora_db -c "\d users"

# Test rollback
make migrate-down
```

**2. Test in sandbox environment:**
```bash
# Deploy to sandbox with migrations
kubectl apply -k deployments/k8s/overlays/sandbox

# Verify deployment
kubectl logs -n pandora-exchange-sandbox -l app=user-service -c migrate

# Run integration tests
make test-integration ENV=sandbox
```

**3. Verify data integrity:**
```sql
-- Check row counts before/after
SELECT COUNT(*) FROM users;

-- Check for NULL values in new NOT NULL columns
SELECT COUNT(*) FROM users WHERE role IS NULL;

-- Verify foreign key relationships
SELECT COUNT(*) FROM refresh_tokens rt 
    LEFT JOIN users u ON rt.user_id = u.id 
    WHERE u.id IS NULL;
```

### Automated Migration Tests

```go
// internal/postgres/migrations_test.go
func TestMigrations(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()

    // Apply all migrations
    err := migrate.Up(db, "../../migrations")
    require.NoError(t, err)

    // Verify schema
    var tableExists bool
    err = db.QueryRow(`
        SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_name = 'users'
        )
    `).Scan(&tableExists)
    require.NoError(t, err)
    assert.True(t, tableExists)

    // Test rollback
    err = migrate.Down(db, "../../migrations", 1)
    require.NoError(t, err)
}
```

---

## Migration Strategies

### Zero-Downtime Migrations

**Challenge:** Migrations that require application downtime

**Solution:** Multi-phase migrations

**Example: Renaming a column**

**Phase 1: Add new column (000010_add_new_column.up.sql)**
```sql
-- Add new column with same data
ALTER TABLE users ADD COLUMN full_name TEXT;
UPDATE users SET full_name = first_name || ' ' || last_name;
```
- Deploy: Application still uses `first_name`, `last_name`
- Downtime: None

**Phase 2: Update application code**
- Read from `full_name` if not null, else `first_name + last_name`
- Write to both `full_name` AND `first_name`, `last_name`
- Deploy: Gradual rollout (blue-green or canary)
- Downtime: None

**Phase 3: Backfill and enforce (000011_backfill_full_name.up.sql)**
```sql
-- Backfill any missing values
UPDATE users SET full_name = first_name || ' ' || last_name WHERE full_name IS NULL;

-- Make NOT NULL
ALTER TABLE users ALTER COLUMN full_name SET NOT NULL;
```
- Deploy: Application reads from `full_name` only
- Downtime: None

**Phase 4: Remove old columns (000012_remove_old_columns.up.sql)**
```sql
-- Safe to drop now
ALTER TABLE users 
    DROP COLUMN first_name,
    DROP COLUMN last_name;
```
- Deploy: Clean up
- Downtime: None

### Large Table Migrations

**Problem:** Adding index to table with 10M+ rows (can take hours, blocks writes)

**Solution: Use CONCURRENTLY**

```sql
-- Create index without blocking writes
CREATE INDEX CONCURRENTLY idx_users_created_at ON users(created_at);

-- If fails partway, clean up invalid index
DROP INDEX CONCURRENTLY IF EXISTS idx_users_created_at;
```

**Limitations:**
- Cannot run in transaction (remove BEGIN/COMMIT)
- Cannot be rolled back automatically
- Must check for success manually

### Data Migrations

**Approach 1: In-migration (small datasets < 10k rows)**
```sql
-- 000013_update_user_roles.up.sql
UPDATE users SET role = 'admin' WHERE email LIKE '%@pandora.exchange';
```

**Approach 2: Background job (large datasets)**
```sql
-- 000014_add_user_tier.up.sql
ALTER TABLE users ADD COLUMN tier TEXT DEFAULT 'bronze';

-- Then run background job to populate based on trading volume
-- (handled by application code, not migration)
```

**Approach 3: Batch processing**
```sql
-- Process in batches to avoid locking table
DO $$
DECLARE
    batch_size INT := 1000;
    processed INT := 0;
BEGIN
    LOOP
        WITH batch AS (
            SELECT id FROM users 
            WHERE tier IS NULL 
            LIMIT batch_size
        )
        UPDATE users u
        SET tier = 'silver'
        FROM batch
        WHERE u.id = batch.id;
        
        GET DIAGNOSTICS processed = ROW_COUNT;
        EXIT WHEN processed = 0;
        
        PERFORM pg_sleep(0.1); -- Throttle to reduce load
    END LOOP;
END $$;
```

---

## Rollback Procedures

### Standard Rollback

```bash
# Rollback last migration
make migrate-down

# Rollback to specific version
migrate -path migrations -database "$DATABASE_URL" force 5
```

### Emergency Rollback

**Scenario:** Migration breaks production

**Steps:**
1. **Stop deployments** (prevent new pods from running broken migration)
   ```bash
   kubectl scale deployment -n pandora-exchange user-service --replicas=0
   ```

2. **Rollback migration**
   ```bash
   # Get current version
   migrate -path migrations -database "$DATABASE_URL" version
   # Output: 6
   
   # Rollback to previous version
   migrate -path migrations -database "$DATABASE_URL" down 1
   # Now at version 5
   ```

3. **Verify database state**
   ```sql
   -- Check schema
   \d users
   
   -- Verify data integrity
   SELECT COUNT(*) FROM users;
   ```

4. **Revert application code** (if deployed)
   ```bash
   # Rollback Kubernetes deployment to previous version
   kubectl rollout undo deployment -n pandora-exchange user-service
   ```

5. **Restore service**
   ```bash
   kubectl scale deployment -n pandora-exchange user-service --replicas=3
   ```

6. **Monitor recovery**
   ```bash
   kubectl logs -n pandora-exchange -l app=user-service -f
   ```

### Point-in-Time Recovery (Worst Case)

**Scenario:** Migration caused data loss or corruption

**Steps:**
1. **Stop all writes** (read-only mode)
2. **Restore from backup** (see [Incident Response](../runbooks/incident-response.md))
3. **Replay transactions** (if using PITR)
4. **Fix migration** (create corrective migration)
5. **Test thoroughly** (in sandbox environment)
6. **Redeploy**

---

## Migration Version Control

### schema_migrations Table

**Automatically created by golang-migrate:**
```sql
CREATE TABLE schema_migrations (
    version BIGINT PRIMARY KEY,
    dirty BOOLEAN NOT NULL
);
```

**Check current version:**
```sql
SELECT version, dirty FROM schema_migrations;
```

**Dirty flag:**
- `dirty = false`: Clean state
- `dirty = true`: Migration failed partway (manual intervention required)

**Fixing dirty state:**
```bash
# Check what failed
migrate -path migrations -database "$DATABASE_URL" version
# Output: 6/d (dirty)

# Manually fix database, then force version
migrate -path migrations -database "$DATABASE_URL" force 6
```

---

## References

- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Database Reliability Engineering](https://www.oreilly.com/library/view/database-reliability-engineering/9781491925935/)

---

**Last Updated:** November 8, 2025  
**Owner:** Database Team
