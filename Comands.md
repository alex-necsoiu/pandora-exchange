

## Connect to PostgreSQL
docker exec -it pandora-postgres psql -U pandora -d pandora_dev

docker exec -it pandora-postgres psql -U pandora -d pandora_dev -c "SELECT id, email, full_name, kyc_status, is_active, created_at FROM users;"

## Update Role of User to Admin
docker exec -it pandora-postgres psql -U pandora -d pandora_dev -c "UPDATE users SET role = 'admin' WHERE email = 'admin@pandora.com'; SELECT id, email, first_name, last_name, role FROM users WHERE email = 'admin@pandora.com';"