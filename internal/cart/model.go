package cart

import "time"

type CartItem struct {
	ID           string    `json:"id"`
	ConcertID    string    `json:"concert_id"`
	TicketTypeID string    `json:"ticket_type_id"`
	Quantity     int       `json:"quantity"`
	PriceCents   int64     `json:"price_cents"`
	Currency     string    `json:"currency"`
	Title        string    `json:"title"`
	City         string    `json:"city"`
	Country      string    `json:"country"`
	When         time.Time `json:"when"`
}

type CartResponse struct {
	Items      []CartItem `json:"items"`
	TotalCents int64      `json:"total_cents"`
	Currency   string     `json:"currency,omitempty"`
}

type AddItemRequest struct {
	TicketTypeID string `json:"ticket_type_id"`
	Quantity     int    `json:"quantity"`
}

type UpdateItemRequest struct {
	Quantity int `json:"quantity"`
}

type CheckoutResponse struct {
	URL       string `json:"url"`
	SessionID string `json:"session_id,omitempty"`
}

type CheckoutItem struct {
	ID           string
	TicketTypeID string
	Quantity     int
	PriceCents   int64
	Currency     string
	Title        string
	City         string
	Country      string
	When         time.Time
	ConcertID    string
}
