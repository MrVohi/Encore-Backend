package tickets

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListUpcoming(ctx context.Context) ([]ConcertForSale, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			c.id,
			c.artist_id,
			c."when",
			c.city,
			c.country,
			c.capacity,
			c.status,
			c."when" AS created_at,
			COALESCE(c.lat, 0)::double precision AS lat,
			COALESCE(c.lng, 0)::double precision AS lng,
			a.name AS title,
			c.city AS venue,
			round(tt.price * 100)::bigint AS price_cents,
			tt.currency
		FROM concerts c
		JOIN artists a ON a.id = c.artist_id
		JOIN LATERAL (
			SELECT price, currency, quantity
			FROM ticket_types
			WHERE concert_id = c.id
				AND quantity > 0
				AND now() BETWEEN starts AND ends
			ORDER BY price ASC
			LIMIT 1
		) tt ON true
		WHERE c."when" >= now()
		ORDER BY c."when" ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ConcertForSale
	for rows.Next() {
		var c ConcertForSale
		if err := rows.Scan(
			&c.ID,
			&c.ArtistID,
			&c.When,
			&c.City,
			&c.Country,
			&c.Capacity,
			&c.Status,
			&c.CreatedAt,
			&c.Lat,
			&c.Lng,
			&c.Title,
			&c.Venue,
			&c.PriceCents,
			&c.Currency,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) GetCheckoutInfo(ctx context.Context, concertID string) (CheckoutInfo, error) {
	var info CheckoutInfo
	err := r.pool.QueryRow(ctx, `
		SELECT
			c.id,
			c."when",
			c.city,
			c.country,
			a.name AS title,
			tt.id AS ticket_type_id,
			round(tt.price * 100)::bigint AS price_cents,
			tt.currency,
			tt.quantity
		FROM concerts c
		JOIN artists a ON a.id = c.artist_id
		JOIN LATERAL (
			SELECT id, price, currency, quantity
			FROM ticket_types
			WHERE concert_id = c.id
				AND quantity > 0
				AND now() BETWEEN starts AND ends
			ORDER BY price ASC
			LIMIT 1
		) tt ON true
		WHERE c.id = $1
	`, concertID).Scan(
		&info.ConcertID,
		&info.When,
		&info.City,
		&info.Country,
		&info.Title,
		&info.TicketTypeID,
		&info.PriceCents,
		&info.Currency,
		&info.Available,
	)

	return info, err
}

func (r *Repository) FulfillCheckout(ctx context.Context, req FulfillRequest) error {
	if req.Quantity < 1 {
		return fmt.Errorf("invalid quantity")
	}
	if req.UserID == "" || req.TicketTypeID == "" || req.SessionID == "" {
		return fmt.Errorf("missing checkout data")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM payments WHERE stripe_checkout_session_id = $1
		)
	`, req.SessionID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return tx.Commit(ctx)
	}

	var concertID string
	var priceCents int64
	var currency string
	var available int
	if err := tx.QueryRow(ctx, `
		SELECT concert_id, round(price * 100)::bigint, currency, quantity
		FROM ticket_types
		WHERE id = $1
		FOR UPDATE
	`, req.TicketTypeID).Scan(&concertID, &priceCents, &currency, &available); err != nil {
		return err
	}

	if req.ConcertID != "" && concertID != req.ConcertID {
		return fmt.Errorf("ticket type does not match concert")
	}
	if available < req.Quantity {
		return fmt.Errorf("not enough tickets available")
	}

	if _, err := tx.Exec(ctx, `
		UPDATE ticket_types
		SET quantity = quantity - $1
		WHERE id = $2
	`, req.Quantity, req.TicketTypeID); err != nil {
		return err
	}

	totalCents := priceCents * int64(req.Quantity)

	var orderID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO orders (user_id, status, total_amount, currency)
		VALUES ($1, 'paid', $2::numeric / 100, $3)
		RETURNING id
	`, req.UserID, totalCents, currency).Scan(&orderID); err != nil {
		return err
	}

	var paymentIntent interface{} = nil
	if req.PaymentIntentID != "" {
		paymentIntent = req.PaymentIntentID
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO payments (
			order_id,
			provider,
			stripe_checkout_session_id,
			stripe_payment_intent_id,
			status,
			amount,
			currency
		)
		VALUES ($1, 'stripe', $2, $3, 'paid', $4::numeric / 100, $5)
	`, orderID, req.SessionID, paymentIntent, totalCents, currency); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO tickets (user_id, concert_id, ticket_type_id, seat, status, issued_at)
		SELECT $1, $2, $3, $4 || '-' || gs::text, 'issued', now()
		FROM generate_series(1, $5) AS gs
	`, req.UserID, concertID, req.TicketTypeID, "GA", req.Quantity); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) CreateTicketType(ctx context.Context, concertID string, req CreateTicketTypeRequest) (TicketType, error) {
	var tt TicketType
	err := r.pool.QueryRow(ctx, `
		INSERT INTO ticket_types (
			concert_id,
			name,
			price,
			currency,
			quantity,
			starts,
			ends
		)
		VALUES ($1, $2, $3::numeric / 100, $4, $5, $6, $7)
		RETURNING
			id,
			concert_id,
			name,
			round(price * 100)::bigint AS price_cents,
			currency,
			quantity,
			starts,
			ends
	`, concertID, req.Name, req.PriceCents, req.Currency, req.Quantity, req.Starts, req.Ends).Scan(
		&tt.ID,
		&tt.ConcertID,
		&tt.Name,
		&tt.PriceCents,
		&tt.Currency,
		&tt.Quantity,
		&tt.Starts,
		&tt.Ends,
	)

	return tt, err
}
