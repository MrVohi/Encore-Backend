package album

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

func (r *Repository) List(ctx context.Context) ([]Album, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, title, release_date, artist_id, created_at
		FROM albums
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Album
	for rows.Next() {
		var a Album
		var rd time.Time
		if err := rows.Scan(&a.ID, &a.Title, &rd, &a.ArtistID, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.ReleaseDate = rd.Format("2006-01-02")
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) listArtistAlbums(ctx context.Context, id string) ([]Album, error) {
	var out []Album

	rows, err := r.pool.Query(ctx, `
		SELECT id, title, release_date, artist_id, created_at
		FROM albums
		WHERE artist_id = $1
		ORDER BY release_date DESC NULLS LAST, created_at DESC;

	`, id)

	defer rows.Close()

	for rows.Next() {
		var a Album
		var rd time.Time
		if err := rows.Scan(&a.ID, &a.Title, &rd, &a.ArtistID, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.ReleaseDate = rd.Format("2006-01-02")
		out = append(out, a)
	}

	return out, err
}

func (r *Repository) GetByID(ctx context.Context, id string) (Album, error) {
	var a Album
	var rd time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT id, title, release_date, artist_id, created_at
		FROM albums
		WHERE id = $1
	`, id).Scan(&a.ID, &a.Title, &rd, &a.ArtistID, &a.CreatedAt)

	a.ReleaseDate = rd.Format("2006-01-02")

	return a, err
}

func (r *Repository) Create(ctx context.Context, req CreateAlbumRequest, id string) (Album, error) {
	rd, errTime := time.Parse("2006-01-02", req.ReleaseDate)

	if errTime != nil {
		return Album{}, errTime
	}

	var a Album
	var rdOut time.Time

	err := r.pool.QueryRow(ctx, `
		INSERT INTO albums (title, release_date, artist_id)
		VALUES ($1, $2, $3)
		RETURNING id, title, release_date, artist_id, created_at
	`, req.Title, rd, id).
		Scan(&a.ID, &a.Title, &rdOut, &a.ArtistID, &a.CreatedAt)

	a.ReleaseDate = rdOut.Format("2006-01-02")

	return a, err
}
