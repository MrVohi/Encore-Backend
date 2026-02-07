package artist

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context, userID *string) ([]Artist, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.name, a.genre, a.image_url, a.preview_url, a.created_at,
			(SELECT COUNT(*) FROM follow f WHERE f.artist_id = a.id) AS followers_count,
			CASE WHEN $1::uuid IS NULL THEN NULL
				ELSE EXISTS(
					SELECT 1 FROM follow f WHERE f.artist_id = a.id AND f.user_id = $1
				)
			END AS is_followed
		FROM artists a
		ORDER BY a.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Artist
	for rows.Next() {
		var a Artist
		if err := rows.Scan(
			&a.ID,
			&a.Name,
			&a.Genre,
			&a.ImageURL,
			&a.PreviewURL,
			&a.CreatedAt,
			&a.FollowersCount,
			&a.IsFollowed,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string, userID *string) (Artist, error) {
	var a Artist
	err := r.pool.QueryRow(ctx, `
		SELECT a.id, a.name, a.genre, a.image_url, a.preview_url, a.created_at,
			(SELECT COUNT(*) FROM follow f WHERE f.artist_id = a.id) AS followers_count,
			CASE WHEN $2::uuid IS NULL THEN NULL
				ELSE EXISTS(
					SELECT 1 FROM follow f WHERE f.artist_id = a.id AND f.user_id = $2
				)
			END AS is_followed
		FROM artists a
		WHERE a.id = $1
	`, id, userID).Scan(
		&a.ID,
		&a.Name,
		&a.Genre,
		&a.ImageURL,
		&a.PreviewURL,
		&a.CreatedAt,
		&a.FollowersCount,
		&a.IsFollowed,
	)

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
