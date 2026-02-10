package concerts

import "time"

type Concert struct {
	ID        string    `json:"id"`
	ArtistID  string    `json:"artist_id"`
	When      time.Time `json:"when"`
	City      string    `json:"city"`
	Country   string    `json:"country"`
	Capacity  int       `json:"capacity"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
}

type CreateConcertRequest struct {
	When     string  `json:"when"`
	City     string  `json:"city"`
	Country  string  `json:"country"`
	Capacity int     `json:"capacity"`
	Status   string  `json:"status"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

type UpdateConcertRequest struct {
	When     string  `json:"when"`
	City     string  `json:"city"`
	Country  string  `json:"country"`
	Capacity int     `json:"capacity"`
	Status   string  `json:"status"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}
