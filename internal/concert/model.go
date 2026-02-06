package concert

import "time"

type Concert struct {
	ID        string    `json:"id"`
	ArtistID  string    `json:"artist_id"`
	When      string    `json:"when"`
	City      string    `json:"city"`
	Country   string    `json:"country"`
	Capacity  int       `json:"capacity"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`

	Lat *float64 `json:"lat"`
	Lng *float64 `json:"lng"`
}

// For POST body
type CreateConcertRequest struct {
	ArtistID string `json:"artist_id"`
	When     string `json:"when"`
	City     string `json:"city"`
	Country  string `json:"country"`
	Capacity int    `json:"capacity"`
	Status   string `json:"status"`

	Lat *float64 `json:"lat"`
	Lng *float64 `json:"lng"`
}
