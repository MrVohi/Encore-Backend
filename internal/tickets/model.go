package tickets

import "time"

type ConcertForSale struct {
	ID         string    `json:"id"`
	ArtistID   string    `json:"artist_id"`
	When       time.Time `json:"when"`
	City       string    `json:"city"`
	Country    string    `json:"country"`
	Capacity   int       `json:"capacity"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	Title      string    `json:"title"`
	Venue      string    `json:"venue"`
	PriceCents int64     `json:"price_cents"`
	Currency   string    `json:"currency"`
}

type CheckoutRequest struct {
	ConcertID string `json:"concert_id"`
	Quantity  int    `json:"quantity"`
}

type CheckoutResponse struct {
	URL       string `json:"url"`
	SessionID string `json:"session_id,omitempty"`
}

type CheckoutInfo struct {
	ConcertID    string
	TicketTypeID string
	Title        string
	City         string
	Country      string
	When         time.Time
	PriceCents   int64
	Currency     string
	Available    int
}

type FulfillRequest struct {
	UserID          string
	ConcertID       string
	TicketTypeID    string
	SessionID       string
	PaymentIntentID string
	Quantity        int
}

type CreateTicketTypeRequest struct {
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"`
	Currency   string `json:"currency"`
	Quantity   int    `json:"quantity"`
	Starts     string `json:"starts"`
	Ends       string `json:"ends"`
}

type TicketType struct {
	ID         string    `json:"id"`
	ConcertID  string    `json:"concert_id"`
	Name       string    `json:"name"`
	PriceCents int64     `json:"price_cents"`
	Currency   string    `json:"currency"`
	Quantity   int       `json:"quantity"`
	Starts     time.Time `json:"starts"`
	Ends       time.Time `json:"ends"`
}
