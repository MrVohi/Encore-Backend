package geo

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

// Get returns (lat,lng,true,nil) if found, (0,0,false,nil) if not found.
func (r *Repository) Get(ctx context.Context, city, country string) (float64, float64, bool, error) {
	var lat, lng float64
	err := r.pool.QueryRow(ctx, `
		SELECT lat, lng
		FROM geo_cache
		WHERE city = $1 AND country = $2
	`, city, country).Scan(&lat, &lng)

	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, 0, false, nil
		}
		return 0, 0, false, err
	}

	return lat, lng, true, nil
}

// Upsert inserts or updates the cache.
func (r *Repository) Upsert(ctx context.Context, city, country string, lat, lng float64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO geo_cache (city, country, lat, lng)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (city, country)
		DO UPDATE SET lat = EXCLUDED.lat, lng = EXCLUDED.lng
	`, city, country, lat, lng)
	return err
}
