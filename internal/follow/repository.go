package follow

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

func (r *Repository) ArtistExists(ctx context.Context, artistID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM artists WHERE id = $1
		)
	`, artistID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *Repository) Follow(ctx context.Context, userID, artistID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO follow (user_id, artist_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, userID, artistID)
	return err
}

func (r *Repository) Unfollow(ctx context.Context, userID, artistID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM follow
		WHERE user_id = $1 AND artist_id = $2
	`, userID, artistID)
	return err
}

func (r *Repository) ListFollowed(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT artist_id
		FROM follow
		WHERE user_id = $1
		ORDER BY artist_id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *Repository) IsFollowing(ctx context.Context, userID, artistID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM follow WHERE user_id = $1 AND artist_id = $2
		)
	`, userID, artistID).Scan(&exists)
	if err != nil && err != pgx.ErrNoRows {
		return false, err
	}
	return exists, nil
}

func (r *Repository) ListFollowers(ctx context.Context, artistID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id
		FROM follow
		WHERE artist_id = $1
		ORDER BY user_id
	`, artistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
