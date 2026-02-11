package concert

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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

func (r *Repository) ListByArtist(ctx context.Context, id string) ([]Concert, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, artist_id, "when", city, country, capacity, status, created_at, lat, lng
		FROM concerts
		WHERE artist_id = $1
		ORDER BY "when" DESC NULLS LAST, created_at DESC;
	`, id)
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
	c.CreatedAt = createdAt

	return c, err
}

func (r *Repository) DeleteByID(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM concerts
		WHERE id = $1
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, id string, req UpdateConcertRequest) (Concert, error) {
	set := make([]string, 0, 7)
	args := make([]any, 0, 8)

	if req.When != nil {
		when, err := time.Parse("2006-01-02 15:04:05", *req.When)
		if err != nil {
			return Concert{}, err
		}
		args = append(args, when)
		set = append(set, "\"when\" = $"+strconv.Itoa(len(args)))
	}
	if req.City != nil {
		args = append(args, *req.City)
		set = append(set, "city = $"+strconv.Itoa(len(args)))
	}
	if req.Country != nil {
		args = append(args, *req.Country)
		set = append(set, "country = $"+strconv.Itoa(len(args)))
	}
	if req.Capacity != nil {
		args = append(args, *req.Capacity)
		set = append(set, "capacity = $"+strconv.Itoa(len(args)))
	}
	if req.Status != nil {
		args = append(args, *req.Status)
		set = append(set, "status = $"+strconv.Itoa(len(args)))
	}
	if req.Lat != nil {
		args = append(args, *req.Lat)
		set = append(set, "lat = $"+strconv.Itoa(len(args)))
	}
	if req.Lng != nil {
		args = append(args, *req.Lng)
		set = append(set, "lng = $"+strconv.Itoa(len(args)))
	}

	if len(set) == 0 {
		return Concert{}, fmt.Errorf("no fields to update")
	}

	args = append(args, id)
	var c Concert
	var when time.Time
	err := r.pool.QueryRow(ctx, `
		UPDATE concerts
		SET `+strings.Join(set, ", ")+`
		WHERE id = $`+strconv.Itoa(len(args))+`
		RETURNING id, artist_id, "when", city, country, capacity, status, created_at, lat, lng
	`, args...).Scan(&c.ID, &c.ArtistID, &when, &c.City, &c.Country, &c.Capacity, &c.Status, &c.CreatedAt, &c.Lat, &c.Lng)
	if err != nil {
		return Concert{}, err
	}
	c.When = when.Format("2006-01-02 15:04:05")
	return c, nil
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
