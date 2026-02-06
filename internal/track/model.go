package track

import "time"

type Track struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	TrackNo     int       `json:"track_no"`
	AlbumID     string    `json:"album_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// For POST body
type CreateTrackRequest struct {
	Title       string `json:"title"`
	TrackNo     int    `json:"track_no"`

}
