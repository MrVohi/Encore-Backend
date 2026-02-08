package artist

import (
	"strings"
	"time"
)

type Artist struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Genre      string    `json:"genre"`
	ArtworkURL string    `json:"artwork_url"`
	PreviewURL string    `json:"preview_url"`
	CreatedAt  time.Time `json:"created_at"`

	LegacyImageURL   *string `json:"-"`
	LegacyPreviewURL *string `json:"-"`
	ArtworkAssetID   *string `json:"-"`
	PreviewAssetID   *string `json:"-"`
	ArtworkObjectKey *string `json:"-"`
	PreviewObjectKey *string `json:"-"`
	ArtworkStorage   *string `json:"-"`
	PreviewStorage   *string `json:"-"`

	FollowersCount int   `json:"followers_count"`
	IsFollowed     *bool `json:"is_followed,omitempty"`
}

// For POST body (so clients can't set ID/CreatedAt)
type CreateArtistRequest struct {
	Name       string  `json:"name"`
	Genre      string  `json:"genre"`
	ImageURL   *string `json:"image_url"`
	PreviewURL *string `json:"preview_url"`
}

// For PUT body (partial updates allowed)
type UpdateArtistRequest struct {
	Name       *string `json:"name"`
	Genre      *string `json:"genre"`
	ImageURL   *string `json:"image_url"`
	PreviewURL *string `json:"preview_url"`
}

func (a *Artist) ResolveURLs(baseURL string, r2PublicBase string) {
	base := strings.TrimRight(baseURL, "/")
	r2Base := strings.TrimRight(r2PublicBase, "/")

	if a.ArtworkObjectKey != nil && *a.ArtworkObjectKey != "" {
		storage := ""
		if a.ArtworkStorage != nil {
			storage = *a.ArtworkStorage
		}
		a.ArtworkURL = resolveAssetURL(storage, *a.ArtworkObjectKey, base, r2Base)
	} else if a.LegacyImageURL != nil && *a.LegacyImageURL != "" {
		a.ArtworkURL = resolveLegacyURL(*a.LegacyImageURL, base, r2Base)
	}

	if a.PreviewObjectKey != nil && *a.PreviewObjectKey != "" {
		storage := ""
		if a.PreviewStorage != nil {
			storage = *a.PreviewStorage
		}
		a.PreviewURL = resolveAssetURL(storage, *a.PreviewObjectKey, base, r2Base)
	} else if a.LegacyPreviewURL != nil && *a.LegacyPreviewURL != "" {
		a.PreviewURL = resolveLegacyURL(*a.LegacyPreviewURL, base, r2Base)
	}
}

func resolveAssetURL(storage, objectKey, baseURL, r2PublicBase string) string {
	cleanKey := strings.TrimLeft(objectKey, "/")
	if storage == "r2" && r2PublicBase != "" {
		return strings.TrimRight(r2PublicBase, "/") + "/" + cleanKey
	}
	return strings.TrimRight(baseURL, "/") + "/uploads/" + cleanKey
}

func resolveLegacyURL(value, baseURL, r2PublicBase string) string {
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if r2PublicBase != "" {
		return strings.TrimRight(r2PublicBase, "/") + "/" + strings.TrimLeft(value, "/")
	}
	return strings.TrimRight(baseURL, "/") + "/uploads/" + strings.TrimLeft(value, "/")
}

type SearchResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
