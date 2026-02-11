package cart

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListCart(ctx context.Context, userID string) (CartResponse, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			ci.id,
			c.id,
			ci.ticket_type_id,
			ci.quantity,
			round(tt.price * 100)::bigint AS price_cents,
			tt.currency,
			a.name,
			c.city,
			c.country,
			c."when"
		FROM cart_items ci
		JOIN ticket_types tt ON tt.id = ci.ticket_type_id
		JOIN concerts c ON c.id = tt.concert_id
		JOIN artists a ON a.id = c.artist_id
		WHERE ci.user_id = $1
		ORDER BY ci.created_at DESC
	`, userID)
	if err != nil {
		return CartResponse{}, err
	}
	defer rows.Close()

	var items []CartItem
	var total int64
	currency := ""

	for rows.Next() {
		var item CartItem
		if err := rows.Scan(
			&item.ID,
			&item.ConcertID,
			&item.TicketTypeID,
			&item.Quantity,
			&item.PriceCents,
			&item.Currency,
			&item.Title,
			&item.City,
			&item.Country,
			&item.When,
		); err != nil {
			return CartResponse{}, err
		}
		items = append(items, item)
		total += item.PriceCents * int64(item.Quantity)
		if currency == "" {
			currency = item.Currency
		} else if currency != item.Currency {
			currency = ""
		}
	}

	return CartResponse{Items: items, TotalCents: total, Currency: currency}, rows.Err()
}

func (r *Repository) AddItem(ctx context.Context, userID string, req AddItemRequest) (CartItem, error) {
	if req.TicketTypeID == "" {
		return CartItem{}, errors.New("ticket_type_id is required")
	}
	if req.Quantity < 1 {
		return CartItem{}, errors.New("quantity must be >= 1")
	}

	var id string
	if err := r.pool.QueryRow(ctx, `
		INSERT INTO cart_items (user_id, ticket_type_id, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, ticket_type_id)
		DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity, updated_at = now()
		RETURNING id
	`, userID, req.TicketTypeID, req.Quantity).Scan(&id); err != nil {
		return CartItem{}, err
	}

	return r.getCartItem(ctx, userID, id)
}

func (r *Repository) UpdateItem(ctx context.Context, userID, itemID string, req UpdateItemRequest) (CartItem, error) {
	if req.Quantity < 1 {
		return CartItem{}, errors.New("quantity must be >= 1")
	}

	var id string
	if err := r.pool.QueryRow(ctx, `
		UPDATE cart_items
		SET quantity = $1, updated_at = now()
		WHERE id = $2 AND user_id = $3
		RETURNING id
	`, req.Quantity, itemID, userID).Scan(&id); err != nil {
		return CartItem{}, err
	}

	return r.getCartItem(ctx, userID, id)
}

func (r *Repository) DeleteItem(ctx context.Context, userID, itemID string) error {
	cmd, err := r.pool.Exec(ctx, `
		DELETE FROM cart_items
		WHERE id = $1 AND user_id = $2
	`, itemID, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func (r *Repository) GetCheckoutItems(ctx context.Context, userID string) ([]CheckoutItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			ci.id,
			ci.ticket_type_id,
			ci.quantity,
			round(tt.price * 100)::bigint AS price_cents,
			tt.currency,
			a.name,
			c.city,
			c.country,
			c."when",
			c.id
		FROM cart_items ci
		JOIN ticket_types tt ON tt.id = ci.ticket_type_id
		JOIN concerts c ON c.id = tt.concert_id
		JOIN artists a ON a.id = c.artist_id
		WHERE ci.user_id = $1
		ORDER BY ci.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CheckoutItem
	for rows.Next() {
		var item CheckoutItem
		if err := rows.Scan(
			&item.ID,
			&item.TicketTypeID,
			&item.Quantity,
			&item.PriceCents,
			&item.Currency,
			&item.Title,
			&item.City,
			&item.Country,
			&item.When,
			&item.ConcertID,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) getCartItem(ctx context.Context, userID, itemID string) (CartItem, error) {
	var item CartItem
	err := r.pool.QueryRow(ctx, `
		SELECT
			ci.id,
			c.id,
			ci.ticket_type_id,
			ci.quantity,
			round(tt.price * 100)::bigint AS price_cents,
			tt.currency,
			a.name,
			c.city,
			c.country,
			c."when"
		FROM cart_items ci
		JOIN ticket_types tt ON tt.id = ci.ticket_type_id
		JOIN concerts c ON c.id = tt.concert_id
		JOIN artists a ON a.id = c.artist_id
		WHERE ci.user_id = $1 AND ci.id = $2
	`, userID, itemID).Scan(
		&item.ID,
		&item.ConcertID,
		&item.TicketTypeID,
		&item.Quantity,
		&item.PriceCents,
		&item.Currency,
		&item.Title,
		&item.City,
		&item.Country,
		&item.When,
	)
	return item, err
}
