

## Connect to PostgreSQL
docker exec -it pandora-postgres psql -U pandora -d pandora_dev

docker exec -it pandora-postgres psql -U pandora -d pandora_dev -c "SELECT id, email, full_name, kyc_status, is_active, created_at FROM users;"
