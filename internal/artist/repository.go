package artist

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
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

func (r *Repository) List(ctx context.Context, userID *string, genre *string) ([]Artist, error) {
	var genreFilter *string
	if genre != nil {
		value := strings.TrimSpace(*genre)
		if value != "" {
			normalized := normalizeQuery(value)
			like := "%" + normalized + "%"
			genreFilter = &like
		}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.name, a.genre, a.image_url, a.preview_url,
			a.artwork_asset_id, a.preview_asset_id, a.created_at,
			art.object_key, prev.object_key,
			art.storage, prev.storage,
			(SELECT COUNT(*) FROM follow f WHERE f.artist_id = a.id) AS followers_count,
			CASE WHEN $1::uuid IS NULL THEN NULL
				ELSE EXISTS(
					SELECT 1 FROM follow f WHERE f.artist_id = a.id AND f.user_id = $1
				)
			END AS is_followed
		FROM artists a
		LEFT JOIN media_assets art ON art.id = a.artwork_asset_id
		LEFT JOIN media_assets prev ON prev.id = a.preview_asset_id
		WHERE ($2::text IS NULL OR a.genre ILIKE $2)
		ORDER BY a.created_at DESC
	`, userID, genreFilter)
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
			&a.ArtworkStorage,
			&a.PreviewStorage,
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
		SELECT a.id, a.name, a.genre, a.image_url, a.preview_url,
			a.artwork_asset_id, a.preview_asset_id, a.created_at,
			art.object_key, prev.object_key,
			art.storage, prev.storage,
			(SELECT COUNT(*) FROM follow f WHERE f.artist_id = a.id) AS followers_count,
			CASE WHEN $2::uuid IS NULL THEN NULL
				ELSE EXISTS(
					SELECT 1 FROM follow f WHERE f.artist_id = a.id AND f.user_id = $2
				)
			END AS is_followed
		FROM artists a
		LEFT JOIN media_assets art ON art.id = a.artwork_asset_id
		LEFT JOIN media_assets prev ON prev.id = a.preview_asset_id
		WHERE a.id = $1
	`, id, userID).Scan(
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
		&a.ArtworkStorage,
		&a.PreviewStorage,
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

func (r *Repository) Update(ctx context.Context, id string, req UpdateArtistRequest) (Artist, error) {
	set := make([]string, 0, 4)
	args := make([]any, 0, 5)

	if req.Name != nil {
		args = append(args, *req.Name)
		set = append(set, "name = $"+strconv.Itoa(len(args)))
	}
	if req.Genre != nil {
		args = append(args, *req.Genre)
		set = append(set, "genre = $"+strconv.Itoa(len(args)))
	}
	if req.ImageURL != nil {
		args = append(args, *req.ImageURL)
		set = append(set, "image_url = $"+strconv.Itoa(len(args)))
	}
	if req.PreviewURL != nil {
		args = append(args, *req.PreviewURL)
		set = append(set, "preview_url = $"+strconv.Itoa(len(args)))
	}

	if len(set) == 0 {
		return Artist{}, pgx.ErrNoRows
	}

	args = append(args, id)
	sql := fmt.Sprintf(`
		UPDATE artists
		SET %s
		WHERE id = $%d
		RETURNING id, name, genre, image_url, preview_url,
			artwork_asset_id, preview_asset_id, created_at
	`, strings.Join(set, ", "), len(args))

	var a Artist
	err := r.pool.QueryRow(ctx, sql, args...).Scan(
		&a.ID,
		&a.Name,
		&a.Genre,
		&a.LegacyImageURL,
		&a.LegacyPreviewURL,
		&a.ArtworkAssetID,
		&a.PreviewAssetID,
		&a.CreatedAt,
	)
	if err != nil {
		return Artist{}, err
	}

	return r.GetByID(ctx, id, nil)
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
		WHERE lower(regexp_replace(name, '\\s+', ' ', 'g'))
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
