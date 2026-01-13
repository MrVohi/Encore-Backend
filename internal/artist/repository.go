package artist

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

func (r *Repository) List(ctx context.Context, name, genre, order string) ([]Artist, error) {
	var (
		args  []any
		where []string
	)

	// Filters with dynamic placeholders
	if name != "" {
		args = append(args, "%"+name+"%")
		where = append(where, fmt.Sprintf("name ILIKE $%d", len(args)))
	}

	if genre != "" {
		args = append(args, strings.ReplaceAll(genre, "+", " "))
		where = append(where, fmt.Sprintf("genre = $%d", len(args)))
	}

	whereSQL := "1=1"
	if len(where) > 0 {
		whereSQL = strings.Join(where, " AND ")
	}

	// ORDER BY whitelist (avoid SQL injection)
	orderBy := "created_at DESC"
	switch order {
	case "name_asc":
		orderBy = "name ASC"
	case "name_desc":
		orderBy = "name DESC"
	}

	sql := fmt.Sprintf(`
		SELECT id, name, genre, image_url, preview_url, created_at
		FROM artists
		WHERE %s
		ORDER BY %s
	`, whereSQL, orderBy)

	rows, err := r.pool.Query(ctx, sql, args...)
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
