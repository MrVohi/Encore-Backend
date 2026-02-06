package concert

import (
	"context"
	"time"

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
		SELECT id, artist_id, "when", city, country, capacity, status, created_at, lat, lng
		FROM concerts
		ORDER BY created_at DESC;	
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Concert
	for rows.Next() {
		var c Concert
		var when time.Time
		if err := rows.Scan(&c.ID, &c.ArtistID, &when, &c.City, &c.Country, &c.Capacity, &c.Status, &c.CreatedAt, &c.Lat, &c.Lng); err != nil {
			return nil, err
		}
		c.When = when.Format("2006-01-02 15:04:05")
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) listArtistConcerts(ctx context.Context, id string) ([]Concert, error) {
	var out []Concert

	rows, err := r.pool.Query(ctx, `
		SELECT id, artist_id, "when", city, country, capacity, status, created_at, lat, lng
		FROM concerts
		WHERE artist_id = $1
		ORDER BY "when" DESC NULLS LAST, created_at DESC;

	`, id)

	defer rows.Close()

	for rows.Next() {
		var c Concert
		var when time.Time
		if err := rows.Scan(&c.ID, &c.ArtistID, &when, &c.City, &c.Country, &c.Capacity, &c.Status, &c.CreatedAt, &c.Lat, &c.Lng); err != nil {
			return nil, err
		}
		c.When = when.Format("2006-01-02 15:04:05")
		out = append(out, c)
	}

	return out, err
}

func (r *Repository) GetByID(ctx context.Context, id string) (Concert, error) {
	var c Concert
	var when time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT id, artist_id, "when", city, country, capacity, status, created_at, lat, lng
		FROM concerts
		WHERE id = $1
	`, id).Scan(&c.ID, &c.ArtistID, &when, &c.City, &c.Country, &c.Capacity, &c.Status, &c.CreatedAt, &c.Lat, &c.Lng)

	c.When = when.Format("2006-01-02 15:04:05")

	return c, err
}

func (r *Repository) Create(ctx context.Context, req CreateConcertRequest, id string) (Concert, error) {
	when, errTime := time.Parse("2006-01-02 15:04:05", req.When)

	if errTime != nil {
		return Concert{}, errTime
	}

	var c Concert
	var createdAt time.Time

	err := r.pool.QueryRow(ctx, `
		INSERT INTO concerts (artist_id, "when", country, city, capacity, status, lat, lng)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, artist_id, "when", city, country, capacity, status, created_at, lat, lng
	`, id, when, req.Country, req.City, req.Capacity, req.Status, req.Lat, req.Lng).
		Scan(&c.ID, &c.ArtistID, &when, &c.City, &c.Country, &c.Capacity, &c.Status, &createdAt, &c.Lat, &c.Lng)

	c.When = when.Format("2006-01-02 15:04:05")

	return c, err
}
