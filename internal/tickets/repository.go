package tickets

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (r *Repository) FulfillCheckout(ctx context.Context, req FulfillRequest) (bool, error) {
	if req.Quantity < 1 {
		return false, fmt.Errorf("invalid quantity")
	}
	if req.UserID == "" || req.TicketTypeID == "" || req.SessionID == "" {
		return false, fmt.Errorf("missing checkout data")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
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
		return false, err
	}
	if exists {
		return false, tx.Commit(ctx)
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
		return false, err
	}

	if req.ConcertID != "" && concertID != req.ConcertID {
		return false, fmt.Errorf("ticket type does not match concert")
	}
	if available < req.Quantity {
		return false, fmt.Errorf("not enough tickets available")
	}

	if _, err := tx.Exec(ctx, `
		UPDATE ticket_types
		SET quantity = quantity - $1
		WHERE id = $2
	`, req.Quantity, req.TicketTypeID); err != nil {
		return false, err
	}

	totalCents := priceCents * int64(req.Quantity)

	var orderID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO orders (user_id, status, total_amount, currency)
		VALUES ($1, 'paid', $2::numeric / 100, $3)
		RETURNING id
	`, req.UserID, totalCents, currency).Scan(&orderID); err != nil {
		return false, err
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
		return false, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO tickets (user_id, concert_id, ticket_type_id, order_id, seat, status, issued_at)
		SELECT $1, $2, $3, $4, $5 || '-' || gs::text, 'issued', now()
		FROM generate_series(1, $6) AS gs
	`, req.UserID, concertID, req.TicketTypeID, orderID, "GA", req.Quantity); err != nil {
		return false, err
	}

	return true, tx.Commit(ctx)
}

type CartFulfillItem struct {
	ConcertID    string
	TicketTypeID string
	Quantity     int
}

func (r *Repository) FulfillCartCheckout(ctx context.Context, userID, sessionID, paymentIntentID string, itemIDs []string) (bool, []CartFulfillItem, error) {
	if userID == "" || sessionID == "" {
		return false, nil, fmt.Errorf("missing checkout data")
	}
	if len(itemIDs) == 0 {
		return false, nil, fmt.Errorf("missing cart items")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM payments WHERE stripe_checkout_session_id = $1
		)
	`, sessionID).Scan(&exists); err != nil {
		return false, nil, err
	}
	if exists {
		if err := tx.Commit(ctx); err != nil {
			return false, nil, err
		}
		return false, nil, nil
	}

	uuidItems := make([]uuid.UUID, 0, len(itemIDs))
	for _, id := range itemIDs {
		parsed, err := uuid.Parse(strings.TrimSpace(id))
		if err != nil {
			return false, nil, fmt.Errorf("invalid cart item id")
		}
		uuidItems = append(uuidItems, parsed)
	}

	rows, err := tx.Query(ctx, `
		SELECT
			ci.id,
			ci.ticket_type_id,
			ci.quantity,
			tt.concert_id,
			round(tt.price * 100)::bigint AS price_cents,
			tt.currency,
			tt.quantity
		FROM cart_items ci
		JOIN ticket_types tt ON tt.id = ci.ticket_type_id
		WHERE ci.user_id = $1 AND ci.id = ANY($2)
		FOR UPDATE
	`, userID, uuidItems)
	if err != nil {
		return false, nil, err
	}
	defer rows.Close()

	type cartRow struct {
		ItemID       string
		TicketTypeID string
		Quantity     int
		ConcertID    string
		PriceCents   int64
		Currency     string
		Available    int
	}

	var rowsData []cartRow
	for rows.Next() {
		var row cartRow
		if err := rows.Scan(&row.ItemID, &row.TicketTypeID, &row.Quantity, &row.ConcertID, &row.PriceCents, &row.Currency, &row.Available); err != nil {
			return false, nil, err
		}
		rowsData = append(rowsData, row)
	}
	if err := rows.Err(); err != nil {
		return false, nil, err
	}
	if len(rowsData) != len(itemIDs) {
		return false, nil, fmt.Errorf("cart items not found")
	}

	currency := rowsData[0].Currency
	var totalCents int64
	items := make([]CartFulfillItem, 0, len(rowsData))

	for _, row := range rowsData {
		if row.Quantity < 1 {
			return false, nil, fmt.Errorf("invalid quantity")
		}
		if row.Currency != currency {
			return false, nil, fmt.Errorf("multiple currencies in cart")
		}
		if row.Available < row.Quantity {
			return false, nil, fmt.Errorf("not enough tickets available")
		}
		totalCents += row.PriceCents * int64(row.Quantity)
		items = append(items, CartFulfillItem{
			ConcertID:    row.ConcertID,
			TicketTypeID: row.TicketTypeID,
			Quantity:     row.Quantity,
		})
	}

	for _, row := range rowsData {
		if _, err := tx.Exec(ctx, `
			UPDATE ticket_types
			SET quantity = quantity - $1
			WHERE id = $2
		`, row.Quantity, row.TicketTypeID); err != nil {
			return false, nil, err
		}
	}

	var orderID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO orders (user_id, status, total_amount, currency)
		VALUES ($1, 'paid', $2::numeric / 100, $3)
		RETURNING id
	`, userID, totalCents, currency).Scan(&orderID); err != nil {
		return false, nil, err
	}

	var paymentIntent interface{} = nil
	if paymentIntentID != "" {
		paymentIntent = paymentIntentID
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
	`, orderID, sessionID, paymentIntent, totalCents, currency); err != nil {
		return false, nil, err
	}

	for _, row := range rowsData {
		if _, err := tx.Exec(ctx, `
			INSERT INTO tickets (user_id, concert_id, ticket_type_id, order_id, seat, status, issued_at)
			SELECT $1, $2, $3, $4, $5 || '-' || gs::text, 'issued', now()
			FROM generate_series(1, $6) AS gs
		`, userID, row.ConcertID, row.TicketTypeID, orderID, "GA", row.Quantity); err != nil {
			return false, nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM cart_items
		WHERE user_id = $1 AND id = ANY($2)
	`, userID, uuidItems); err != nil {
		return false, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, nil, err
	}

	return true, items, nil
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

func (r *Repository) ListTicketTypes(ctx context.Context, concertID string) ([]TicketType, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, concert_id, name, round(price * 100)::bigint AS price_cents, currency, quantity, starts, ends
		FROM ticket_types
		WHERE concert_id = $1
		ORDER BY price ASC
	`, concertID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TicketType
	for rows.Next() {
		var tt TicketType
		if err := rows.Scan(&tt.ID, &tt.ConcertID, &tt.Name, &tt.PriceCents, &tt.Currency, &tt.Quantity, &tt.Starts, &tt.Ends); err != nil {
			return nil, err
		}
		out = append(out, tt)
	}
	return out, rows.Err()
}

type ConcertEmailInfo struct {
	ConcertTitle string
	When         time.Time
	City         string
	Country      string
}

func (r *Repository) GetConcertEmailInfo(ctx context.Context, concertID string) (ConcertEmailInfo, error) {
	var info ConcertEmailInfo
	err := r.pool.QueryRow(ctx, `
		SELECT a.name, c."when", c.city, c.country
		FROM concerts c
		JOIN artists a ON a.id = c.artist_id
		WHERE c.id = $1
	`, concertID).Scan(&info.ConcertTitle, &info.When, &info.City, &info.Country)
	return info, err
}

func (r *Repository) ListUserTickets(ctx context.Context, userID string) (UserTicketsResponse, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.concert_id, a.name, c."when", c.city, c.country, t.seat, t.status, t.issued_at, t.used_at
		FROM tickets t
		JOIN concerts c ON c.id = t.concert_id
		JOIN artists a ON a.id = c.artist_id
		WHERE t.user_id = $1
		ORDER BY t.issued_at DESC
	`, userID)
	if err != nil {
		return UserTicketsResponse{}, err
	}
	defer rows.Close()

	var active []TicketSummary
	var past []TicketSummary
	for rows.Next() {
		var t TicketSummary
		if err := rows.Scan(&t.ID, &t.ConcertID, &t.Artist, &t.When, &t.City, &t.Country, &t.Seat, &t.Status, &t.IssuedAt, &t.UsedAt); err != nil {
			return UserTicketsResponse{}, err
		}
		if t.When.After(time.Now()) {
			active = append(active, t)
		} else {
			past = append(past, t)
		}
	}

	return UserTicketsResponse{Active: active, Past: past}, rows.Err()
}

func (r *Repository) ListAdminTickets(ctx context.Context) ([]AdminTicket, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.user_id, u.email, t.concert_id, a.name, c."when", c.city, c.country, t.seat, t.status, t.issued_at, t.used_at, t.ticket_type_id, t.order_id
		FROM tickets t
		JOIN users u ON u.id = t.user_id
		JOIN concerts c ON c.id = t.concert_id
		JOIN artists a ON a.id = c.artist_id
		ORDER BY t.issued_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AdminTicket
	for rows.Next() {
		var t AdminTicket
		if err := rows.Scan(&t.ID, &t.UserID, &t.UserEmail, &t.ConcertID, &t.Artist, &t.When, &t.City, &t.Country, &t.Seat, &t.Status, &t.IssuedAt, &t.UsedAt, &t.TicketTypeID, &t.OrderID); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) CreateAdminTicket(ctx context.Context, req CreateTicketRequest) (AdminTicket, error) {
	issuedAt := time.Now()
	if req.IssuedAt != nil {
		issuedAt = *req.IssuedAt
	}

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "issued"
	}

	var ticketID string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO tickets (user_id, concert_id, ticket_type_id, order_id, seat, status, issued_at, used_at)
		VALUES ($1, $2, $3, NULL, $4, $5, $6, $7)
		RETURNING id
	`, req.UserID, req.ConcertID, req.TicketTypeID, req.Seat, status, issuedAt, req.UsedAt).Scan(&ticketID)
	if err != nil {
		return AdminTicket{}, err
	}

	return r.GetAdminTicket(ctx, ticketID)
}

func (r *Repository) GetAdminTicket(ctx context.Context, ticketID string) (AdminTicket, error) {
	var t AdminTicket
	err := r.pool.QueryRow(ctx, `
		SELECT t.id, t.user_id, u.email, t.concert_id, a.name, c."when", c.city, c.country, t.seat, t.status, t.issued_at, t.used_at, t.ticket_type_id, t.order_id
		FROM tickets t
		JOIN users u ON u.id = t.user_id
		JOIN concerts c ON c.id = t.concert_id
		JOIN artists a ON a.id = c.artist_id
		WHERE t.id = $1
	`, ticketID).Scan(&t.ID, &t.UserID, &t.UserEmail, &t.ConcertID, &t.Artist, &t.When, &t.City, &t.Country, &t.Seat, &t.Status, &t.IssuedAt, &t.UsedAt, &t.TicketTypeID, &t.OrderID)

	return t, err
}

func (r *Repository) UpdateAdminTicket(ctx context.Context, ticketID string, req UpdateTicketRequest) (AdminTicket, error) {
	set := make([]string, 0, 3)
	args := make([]any, 0, 4)

	if req.Seat != nil {
		args = append(args, *req.Seat)
		set = append(set, "seat = $"+fmt.Sprint(len(args)))
	}
	if req.Status != nil {
		args = append(args, *req.Status)
		set = append(set, "status = $"+fmt.Sprint(len(args)))
	}
	if req.UsedAt != nil {
		args = append(args, *req.UsedAt)
		set = append(set, "used_at = $"+fmt.Sprint(len(args)))
	}
	if len(set) == 0 {
		return AdminTicket{}, fmt.Errorf("no fields to update")
	}

	args = append(args, ticketID)
	cmd, err := r.pool.Exec(ctx, "UPDATE tickets SET "+strings.Join(set, ", ")+" WHERE id = $"+fmt.Sprint(len(args)), args...)
	if err != nil {
		return AdminTicket{}, err
	}
	if cmd.RowsAffected() == 0 {
		return AdminTicket{}, pgx.ErrNoRows
	}

	return r.GetAdminTicket(ctx, ticketID)
}

func (r *Repository) DeleteAdminTicket(ctx context.Context, ticketID string) error {
	cmd, err := r.pool.Exec(ctx, "DELETE FROM tickets WHERE id = $1", ticketID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) GetTicketStats(ctx context.Context) (TicketStats, error) {
	var stats TicketStats
	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*)::int AS total,
			SUM(CASE WHEN status = 'issued' THEN 1 ELSE 0 END)::int AS active,
			SUM(CASE WHEN status = 'used' THEN 1 ELSE 0 END)::int AS used
		FROM tickets
	`).Scan(&stats.Total, &stats.Active, &stats.Used)
	return stats, err
}

func (r *Repository) DeleteTicketType(ctx context.Context, id string) error {
	cmd, err := r.pool.Exec(ctx, "DELETE FROM ticket_types WHERE id = $1", id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
