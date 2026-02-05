package artist

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

func (r *Repository) List(ctx context.Context) ([]Artist, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.name, a.genre, a.image_url, a.preview_url,
			a.artwork_asset_id, a.preview_asset_id, a.created_at,
			art.object_key, prev.object_key
		FROM artists a
		LEFT JOIN media_assets art ON art.id = a.artwork_asset_id
		LEFT JOIN media_assets prev ON prev.id = a.preview_asset_id
		ORDER BY a.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Artist, 0)
	for rows.Next() {
		var a Artist
		if err := rows.Scan(
			&a.ID,
			&a.Name,
			&a.Genre,
			&a.LegacyImageURL,
			&a.LegacyPreviewURL,
			&a.ArtworkAssetID,
			&a.PreviewAssetID,
			&a.CreatedAt,
			&a.ArtworkObjectKey,
			&a.PreviewObjectKey,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (Artist, error) {
	var a Artist
	err := r.pool.QueryRow(ctx, `
		SELECT a.id, a.name, a.genre, a.image_url, a.preview_url,
			a.artwork_asset_id, a.preview_asset_id, a.created_at,
			art.object_key, prev.object_key
		FROM artists a
		LEFT JOIN media_assets art ON art.id = a.artwork_asset_id
		LEFT JOIN media_assets prev ON prev.id = a.preview_asset_id
		WHERE a.id = $1
	`, id).Scan(
		&a.ID,
		&a.Name,
		&a.Genre,
		&a.LegacyImageURL,
		&a.LegacyPreviewURL,
		&a.ArtworkAssetID,
		&a.PreviewAssetID,
		&a.CreatedAt,
		&a.ArtworkObjectKey,
		&a.PreviewObjectKey,
	)

	return a, err
}

func (r *Repository) Create(ctx context.Context, req CreateArtistRequest) (Artist, error) {
	var a Artist
	err := r.pool.QueryRow(ctx, `
		INSERT INTO artists (name, genre, image_url, preview_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, genre, image_url, preview_url,
			artwork_asset_id, preview_asset_id, created_at
	`, req.Name, req.Genre, req.ImageURL, req.PreviewURL).
		Scan(
			&a.ID,
			&a.Name,
			&a.Genre,
			&a.LegacyImageURL,
			&a.LegacyPreviewURL,
			&a.ArtworkAssetID,
			&a.PreviewAssetID,
			&a.CreatedAt,
		)

	return a, err
}

func (r *Repository) UpdateArtworkAsset(ctx context.Context, artistID string, assetID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE artists
		SET artwork_asset_id = $1
		WHERE id = $2
	`, assetID, artistID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) UpdatePreviewAsset(ctx context.Context, artistID string, assetID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE artists
		SET preview_asset_id = $1
		WHERE id = $2
	`, assetID, artistID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteByID(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM artists
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
