# Developer Operations

Internal development commands and operational procedures.

## 🔧 Quick Reference

Common commands for local development and debugging.

### Database Operations

```bash
# Connect to development database
docker exec -it pandora-postgres psql -U pandora -d pandora_dev

# View all users
docker exec -it pandora-postgres psql -U pandora -d pandora_dev -c \
  "SELECT id, email, first_name, last_name, role, kyc_status FROM users;"

# Promote user to admin
docker exec -it pandora-postgres psql -U pandora -d pandora_dev -c \
  "UPDATE users SET role = 'admin' WHERE email = 'your@email.com';"
```

### Development Workflow

```bash
# Start local environment
make dev-up

# Run migrations
make migrate

# Generate sqlc code
make sqlc

# Run tests with coverage
make test-coverage

# Start service in development mode
make dev
```

### Debugging

```bash
# View service logs
docker logs -f pandora-user-service

# View database logs
docker logs -f pandora-postgres

# View Redis logs
docker logs -f pandora-redis

# Check health endpoints
curl http://localhost:8080/health
curl http://localhost:8081/admin/health
```

## 📚 Related Documentation

- [Quick Start Guide](../QUICK_START.md) - Getting started
- [Docker Guide](../DOCKER.md) - Container setup
- [Database Migrations](../db/migrations.md) - Schema management
- [Makefile](../../Makefile) - All available commands
