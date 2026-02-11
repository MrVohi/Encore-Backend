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
	TicketType   string
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
	TicketType      string
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

type UpdateTicketTypeRequest struct {
	Name       *string `json:"name"`
	PriceCents *int64  `json:"price_cents"`
	Currency   *string `json:"currency"`
	Quantity   *int    `json:"quantity"`
	Starts     *string `json:"starts"`
	Ends       *string `json:"ends"`
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

type TicketSummary struct {
	ID        string     `json:"id"`
	ConcertID string     `json:"concert_id"`
	Artist    string     `json:"artist"`
	When      time.Time  `json:"when"`
	City      string     `json:"city"`
	Country   string     `json:"country"`
	TicketType string    `json:"ticket_type"`
	Status    string     `json:"status"`
	IssuedAt  time.Time  `json:"issued_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

type UserTicketsResponse struct {
	Active []TicketSummary `json:"active"`
	Past   []TicketSummary `json:"past"`
}

type AdminTicket struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	UserEmail  string     `json:"user_email"`
	ConcertID  string     `json:"concert_id"`
	Artist     string     `json:"artist"`
	When       time.Time  `json:"when"`
	City       string     `json:"city"`
	Country    string     `json:"country"`
	TicketType string     `json:"ticket_type"`
	Status     string     `json:"status"`
	IssuedAt   time.Time  `json:"issued_at"`
	UsedAt     *time.Time `json:"used_at,omitempty"`
	OrderID    *string    `json:"order_id,omitempty"`
}

type CreateTicketRequest struct {
	UserID     string     `json:"user_id"`
	ConcertID  string     `json:"concert_id"`
	TicketType string     `json:"ticket_type"`
	Status     string     `json:"status"`
	IssuedAt   *time.Time `json:"issued_at,omitempty"`
	UsedAt     *time.Time `json:"used_at,omitempty"`
}

type UpdateTicketRequest struct {
	TicketType *string    `json:"ticket_type"`
	Status     *string    `json:"status"`
	UsedAt     *time.Time `json:"used_at"`
}

type TicketStats struct {
	Total  int `json:"total"`
	Active int `json:"active"`
	Used   int `json:"used"`
}
