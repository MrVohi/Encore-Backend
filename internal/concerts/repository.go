package concerts

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context) ([]Concert, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			artist_id,
			"when",
			city,
			country,
			capacity,
			status,
			"when" AS created_at,
			COALESCE(lat, 0)::double precision AS lat,
			COALESCE(lng, 0)::double precision AS lng
		FROM concerts
		ORDER BY "when" ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Concert
	for rows.Next() {
		var c Concert
		if err := rows.Scan(
			&c.ID,
			&c.ArtistID,
			&c.When,
			&c.City,
			&c.Country,
			&c.Capacity,
			&c.Status,
			&c.CreatedAt,
			&c.Lat,
			&c.Lng,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) ListByArtist(ctx context.Context, artistID string) ([]Concert, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id,
			artist_id,
			"when",
			city,
			country,
			capacity,
			status,
			"when" AS created_at,
			COALESCE(lat, 0)::double precision AS lat,
			COALESCE(lng, 0)::double precision AS lng
		FROM concerts
		WHERE artist_id = $1
		ORDER BY "when" ASC
	`, artistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Concert
	for rows.Next() {
		var c Concert
		if err := rows.Scan(
			&c.ID,
			&c.ArtistID,
			&c.When,
			&c.City,
			&c.Country,
			&c.Capacity,
			&c.Status,
			&c.CreatedAt,
			&c.Lat,
			&c.Lng,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (Concert, error) {
	var c Concert
	err := r.pool.QueryRow(ctx, `
		SELECT
			id,
			artist_id,
			"when",
			city,
			country,
			capacity,
			status,
			"when" AS created_at,
			COALESCE(lat, 0)::double precision AS lat,
			COALESCE(lng, 0)::double precision AS lng
		FROM concerts
		WHERE id = $1
	`, id).Scan(
		&c.ID,
		&c.ArtistID,
		&c.When,
		&c.City,
		&c.Country,
		&c.Capacity,
		&c.Status,
		&c.CreatedAt,
		&c.Lat,
		&c.Lng,
	)

	return c, err
}

func (r *Repository) Create(ctx context.Context, artistID string, req CreateConcertRequest) (Concert, error) {
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "scheduled"
	}

	var c Concert
	err := r.pool.QueryRow(ctx, `
		INSERT INTO concerts (
			artist_id,
			"when",
			city,
			country,
			capacity,
			status,
			external_id,
			lat,
			lng
		)
		VALUES ($1, $2, $3, $4, $5, $6, CONCAT('manual-', gen_random_uuid()), $7, $8)
		RETURNING
			id,
			artist_id,
			"when",
			city,
			country,
			capacity,
			status,
			"when" AS created_at,
			COALESCE(lat, 0)::double precision AS lat,
			COALESCE(lng, 0)::double precision AS lng
	`, artistID, req.When, req.City, req.Country, req.Capacity, status, req.Lat, req.Lng).Scan(
		&c.ID,
		&c.ArtistID,
		&c.When,
		&c.City,
		&c.Country,
		&c.Capacity,
		&c.Status,
		&c.CreatedAt,
		&c.Lat,
		&c.Lng,
	)

	return c, err
}

func (r *Repository) Update(ctx context.Context, id string, req UpdateConcertRequest) (Concert, error) {
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "scheduled"
	}

	var c Concert
	err := r.pool.QueryRow(ctx, `
		UPDATE concerts
		SET "when" = $1,
			city = $2,
			country = $3,
			capacity = $4,
			status = $5,
			lat = $6,
			lng = $7
		WHERE id = $8
		RETURNING
			id,
			artist_id,
			"when",
			city,
			country,
			capacity,
			status,
			"when" AS created_at,
			COALESCE(lat, 0)::double precision AS lat,
			COALESCE(lng, 0)::double precision AS lng
	`, req.When, req.City, req.Country, req.Capacity, status, req.Lat, req.Lng, id).Scan(
		&c.ID,
		&c.ArtistID,
		&c.When,
		&c.City,
		&c.Country,
		&c.Capacity,
		&c.Status,
		&c.CreatedAt,
		&c.Lat,
		&c.Lng,
	)

	return c, err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM concerts WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("concert not found")
	}
	return nil
}
