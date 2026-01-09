package artist

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func normalizeQuery(q string) string {
	return strings.Join(strings.Fields(q), " ")
}

func (r *Repository) List(ctx context.Context) ([]Artist, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, genre, image_url, preview_url, created_at
		FROM artists
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Artist
	for rows.Next() {
		var a Artist
		if err := rows.Scan(&a.ID, &a.Name, &a.Genre, &a.ImageURL, &a.PreviewURL, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (Artist, error) {
	var a Artist
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, genre, image_url, preview_url, created_at
		FROM artists
		WHERE id = $1
	`, id).Scan(&a.ID, &a.Name, &a.Genre, &a.ImageURL, &a.PreviewURL, &a.CreatedAt)

	return a, err
}

func (r *Repository) Create(ctx context.Context, req CreateArtistRequest) (Artist, error) {
	var a Artist
	err := r.pool.QueryRow(ctx, `
		INSERT INTO artists (name, genre, image_url, preview_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, genre, image_url, preview_url, created_at
	`, req.Name, req.Genre, req.ImageURL, req.PreviewURL).
		Scan(&a.ID, &a.Name, &a.Genre, &a.ImageURL, &a.PreviewURL, &a.CreatedAt)

	return a, err
}

// Search artists by name (case-insensitive, collapses whitespace)
func (r *Repository) Search(ctx context.Context, q string, limit int) ([]SearchResult, error) {
	q = normalizeQuery(q)
	if len(q) < 2 {
		return []SearchResult{}, nil
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, name
		FROM artists
		WHERE lower(regexp_replace(name, '\s+', ' ', 'g'))
		      LIKE '%' || lower($1) || '%'
		ORDER BY name
		LIMIT $2
	`, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]SearchResult, 0, limit)
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, rows.Err()
}
