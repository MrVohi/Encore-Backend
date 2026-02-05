package media

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

func (r *Repository) Create(ctx context.Context, asset Asset) (Asset, error) {
	var out Asset
	err := r.pool.QueryRow(ctx, `
		INSERT INTO media_assets (kind, storage, object_key, mime_type, size_bytes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, kind, storage, object_key, mime_type, size_bytes, created_at
	`, asset.Kind, asset.Storage, asset.ObjectKey, asset.MimeType, asset.SizeBytes).
		Scan(
			&out.ID,
			&out.Kind,
			&out.Storage,
			&out.ObjectKey,
			&out.MimeType,
			&out.SizeBytes,
			&out.CreatedAt,
		)

	return out, err
}

func (r *Repository) DeleteByID(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM media_assets
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
