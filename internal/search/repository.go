package search

import (
	"context"
	"strings"

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

// Search artists/album/track by name (case-insensitive, collapses whitespace)
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
		WITH qq AS (
		  SELECT lower(regexp_replace($1, '\s+', ' ', 'g')) AS term
		)
		SELECT kind, id, label, parent_id
		FROM (
		  -- ARTISTS
		  SELECT
		    'artist'::text AS kind,
		    a.id           AS id,
		    a.name         AS label,
		    NULL::uuid     AS parent_id
		  FROM artists a, qq
		  WHERE lower(regexp_replace(a.name, '\s+', ' ', 'g'))
		        LIKE '%' || qq.term || '%'

		  UNION ALL

		  -- ALBUMS
		  SELECT
		    'album'::text AS kind,
		    al.id         AS id,
		    (al.title) AS label,
		    al.artist_id  AS parent_id
		  FROM albums al
		  JOIN artists ar ON ar.id = al.artist_id, qq
		  WHERE lower(regexp_replace(al.title, '\s+', ' ', 'g'))
		        LIKE '%' || qq.term || '%'

		  UNION ALL

		  -- TRACKS
		  SELECT
		    'track'::text AS kind,
		    t.id          AS id,
		    t.title       AS label,
		    t.album_id    AS parent_id
		  FROM tracks t, qq
		  WHERE lower(regexp_replace(t.title, '\s+', ' ', 'g'))
		        LIKE '%' || qq.term || '%'
		) s
		ORDER BY label
		LIMIT $2
	`, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]SearchResult, 0, limit)
	for rows.Next() {
		var sr SearchResult
		var parentID *string // nullable
		if err := rows.Scan(&sr.Kind, &sr.ID, &sr.Label, &parentID); err != nil {
			return nil, err
		}
		if parentID != nil {
			sr.ParentID = *parentID
		}
		results = append(results, sr)
	}

	return results, rows.Err()
}
