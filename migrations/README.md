# Migrations

Place versioned SQL files here (e.g., `0002_add_follow_table.sql`).

Baseline generation (from the *real* dev DB, not `schema.sql`):

```bash
# Ensure docker-compose is running and .env provides POSTGRES_USER/POSTGRES_DB
# Service name from docker-compose.yml is: db

docker compose exec db pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --schema-only --no-owner --no-privileges > migrations/0001_baseline.sql
```

If you are unsure about service/user/db names:
- Service: check `docker-compose.yml` under `services:`
- DB/User: check your `.env` values for `POSTGRES_DB` and `POSTGRES_USER`
