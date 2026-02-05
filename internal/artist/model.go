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
}

// For POST body (so clients can't set ID/CreatedAt)
type CreateArtistRequest struct {
	Name       string  `json:"name"`
	Genre      string  `json:"genre"`
	ImageURL   *string `json:"image_url"`
	PreviewURL *string `json:"preview_url"`
}

func (a *Artist) ResolveURLs(baseURL string) {
	base := strings.TrimRight(baseURL, "/")

	if a.ArtworkObjectKey != nil && *a.ArtworkObjectKey != "" {
		a.ArtworkURL = base + "/uploads/" + *a.ArtworkObjectKey
	} else if a.LegacyImageURL != nil {
		a.ArtworkURL = *a.LegacyImageURL
	}

	if a.PreviewObjectKey != nil && *a.PreviewObjectKey != "" {
		a.PreviewURL = base + "/uploads/" + *a.PreviewObjectKey
	} else if a.LegacyPreviewURL != nil {
		a.PreviewURL = *a.LegacyPreviewURL
	}
}
