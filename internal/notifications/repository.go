package notifications

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

type Follower struct {
	UserID        string
	Email         string
	EmailVerified bool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListFollowersForArtist(ctx context.Context, artistID string) ([]Follower, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.email, u.is_email_verified
		FROM follow f
		JOIN users u ON u.id = f.user_id
		WHERE f.artist_id = $1
		ORDER BY u.id
	`, artistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Follower
	for rows.Next() {
		var f Follower
		if err := rows.Scan(&f.UserID, &f.Email, &f.EmailVerified); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repository) TryMarkNotified(ctx context.Context, concertID, userID string) (bool, error) {
	var inserted bool
	err := r.pool.QueryRow(ctx, `
		INSERT INTO concert_notifications (concert_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
		RETURNING true
	`, concertID, userID).Scan(&inserted)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return inserted, nil
}
