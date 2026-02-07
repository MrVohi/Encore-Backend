package concert

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ArtistName(ctx context.Context, artistID string) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx, `
		SELECT name FROM artists WHERE id = $1
	`, artistID).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

func (r *Repository) Create(ctx context.Context, artistID string, req CreateConcertRequest) (Concert, error) {
	var c Concert
	err := r.pool.QueryRow(ctx, `
		INSERT INTO concerts (artist_id, "when", country, city, capacity, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, artist_id, "when", country, city, capacity, status
	`, artistID, req.When, req.Country, req.City, req.Capacity, req.Status).Scan(
		&c.ID,
		&c.ArtistID,
		&c.When,
		&c.Country,
		&c.City,
		&c.Capacity,
		&c.Status,
	)
	return c, err
}

func (r *Repository) ListByArtist(ctx context.Context, artistID string) ([]Concert, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, artist_id, "when", country, city, capacity, status
		FROM concerts
		WHERE artist_id = $1
		ORDER BY "when" DESC
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
			&c.Country,
			&c.City,
			&c.Capacity,
			&c.Status,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) ArtistExists(ctx context.Context, artistID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM artists WHERE id = $1
		)
	`, artistID).Scan(&exists)
	if err != nil && err != pgx.ErrNoRows {
		return false, err
	}
	return exists, nil
}
