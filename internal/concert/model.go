package concert

import "time"

type Concert struct {
	ID       string    `json:"id"`
	ArtistID string    `json:"artist_id"`
	When     time.Time `json:"when"`
	Country  string    `json:"country"`
	City     string    `json:"city"`
	Capacity int       `json:"capacity"`
	Status   string    `json:"status"`
}

type CreateConcertRequest struct {
	When     time.Time `json:"when" binding:"required"`
	Country  string    `json:"country" binding:"required"`
	City     string    `json:"city" binding:"required"`
	Capacity int       `json:"capacity" binding:"required"`
	Status   string    `json:"status" binding:"required"`
}
