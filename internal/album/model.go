package album

import "time"

type Album struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	ReleaseDate string    `json:"release_date"`
	ArtistID    string    `json:"artist_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// For POST body
type CreateAlbumRequest struct {
	Title       string `json:"title"`
	ReleaseDate string `json:"release_date"`
}
