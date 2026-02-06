package track

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

func (r *Repository) ListAlbumTracks(ctx context.Context, id string) ([]Track, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, track_no, album_id, created_at
		FROM tracks
		WHERE album_id = $1
		ORDER BY track_no ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Track
	for rows.Next() {
		var t Track
		if err := rows.Scan(&t.ID, &t.Title, &t.TrackNo, &t.AlbumID, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (Track, error) {
	var t Track
	err := r.pool.QueryRow(ctx, `
		SELECT id, title, track_no, album_id, created_at
		FROM tracks
		WHERE id = $1
	`, id).Scan(&t.ID, &t.Title, &t.TrackNo, &t.AlbumID, &t.CreatedAt)

	return t, err
}

func (r *Repository) createAlbumTracks(ctx context.Context, req CreateTrackRequest, id string) (Track, error) {
	var t Track
	err := r.pool.QueryRow(ctx, `
		INSERT INTO tracks (title, track_no, album_id)
		VALUES ($1, $2, $3)
		RETURNING id, title, track_no, album_id, created_at
	`, req.Title, req.TrackNo, id).
		Scan(&t.ID, &t.Title, &t.TrackNo, &t.AlbumID, &t.CreatedAt)

	return t, err
}

