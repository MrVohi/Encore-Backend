### Launch the database (docker)

1. You will need the `.env` file that you won't find here obviously, put it on the same folder as the ```docker-compose.yml```

2. From the folder with both files, execute `docker compose up -d`

3. Initialize the db with ```docker compose exec db psql -U Groupie -d Encore_DB < db/schema.sql```

4. If you need to update the db to the lastest data, have the `backup.sql` ready and ```docker compose exec -T db psql -U Groupie -d Encore_DB < backup.sql ```

### Local dev

1. Copy `.env.example` to `.env` and fill in values (names only are provided).

2. Start Postgres:
   `docker compose up -d`

3. Initialize the database if needed:
   `docker compose exec db psql -U Groupie -d Encore_DB < db/schema.sql`

4. Run the API:
   `go run cmd/api/main.go`

### Object storage (R2)

For production, set `STORAGE_DRIVER=r2` and configure the R2 env vars in `.env.example`.
Bucket CORS should allow:
- `GET`, `HEAD` (and optionally `OPTIONS`)
- `Range` header if audio previews need streaming/range requests
- Origins matching your frontend domain

### Migrations

Run migrations:
`go run cmd/migrate/main.go`

See `migrations/README.md` for how to generate a baseline from a real DB schema.
