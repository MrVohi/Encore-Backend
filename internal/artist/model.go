package artist

import "time"

type Artist struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Genre      string    `json:"genre"`
	ImageURL   string    `json:"image_url"`
	PreviewURL string    `json:"preview_url"`
	CreatedAt  time.Time `json:"created_at"`
}

// For POST body (so clients can't set ID/CreatedAt)
type CreateArtistRequest struct {
	Name       string `json:"name"`
	Genre      string `json:"genre"`
	ImageURL   string `json:"image_url"`
	PreviewURL string `json:"preview_url"`
}
