package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// GetOrderByID returns an order by primary key.
func (s *Store) GetOrderByID(ctx context.Context, id string) (*Order, error) {
	return s.scanOrder(ctx, `
		SELECT `+orderSelectColumns+`
		FROM orders WHERE id = $1`, id)
}

// GetOrderBySessionID returns an order by Stripe checkout session ID.
func (s *Store) GetOrderBySessionID(ctx context.Context, sessionID string) (*Order, error) {
	return s.scanOrder(ctx, `
		SELECT `+orderSelectColumns+`
		FROM orders WHERE stripe_checkout_session_id = $1`, sessionID)
}

// GetOrderByPaymentIntentID returns an order by Stripe payment intent ID.
func (s *Store) GetOrderByPaymentIntentID(ctx context.Context, paymentIntentID string) (*Order, error) {
	return s.scanOrder(ctx, `
		SELECT `+orderSelectColumns+`
		FROM orders WHERE stripe_payment_intent_id = $1`, paymentIntentID)
}

// ListOrdersByCustomerID returns recent orders for an authenticated customer.
func (s *Store) ListOrdersByCustomerID(ctx context.Context, customerID string, limit int) ([]Order, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT `+orderSelectColumns+`
		FROM orders
		WHERE customer_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, customerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(
			&order.ID, &order.OrderNumber, &order.IdempotencyKey, &order.Status, &order.TotalAmountCents, &order.Currency,
			&order.CustomerEmail, &order.StripeCheckoutSessionID, &order.StripePaymentIntentID,
			&order.StripeCheckoutURL, &order.StripeClientSecret, &order.UIMode,
			&order.SuccessURL, &order.CancelURL, &order.ReturnURL, &order.Metadata, &order.PaidAt, &order.CreatedAt, &order.UpdatedAt,
			&order.AccessTokenHash,
		); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

// ListOrdersByCustomerEmail returns recent orders for a guest email.
func (s *Store) ListOrdersByCustomerEmail(ctx context.Context, email string, limit int) ([]Order, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT `+orderSelectColumns+`
		FROM orders
		WHERE lower(customer_email) = lower($1)
		ORDER BY created_at DESC
		LIMIT $2`, email, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(
			&order.ID, &order.OrderNumber, &order.IdempotencyKey, &order.Status, &order.TotalAmountCents, &order.Currency,
			&order.CustomerEmail, &order.StripeCheckoutSessionID, &order.StripePaymentIntentID,
			&order.StripeCheckoutURL, &order.StripeClientSecret, &order.UIMode,
			&order.SuccessURL, &order.CancelURL, &order.ReturnURL, &order.Metadata, &order.PaidAt, &order.CreatedAt, &order.UpdatedAt,
			&order.AccessTokenHash,
		); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

// GetOrderByIdempotencyKey returns an order by idempotency key.
func (s *Store) GetOrderByIdempotencyKey(ctx context.Context, key string) (*Order, error) {
	return s.scanOrder(ctx, `
		SELECT `+orderSelectColumns+`
		FROM orders WHERE idempotency_key = $1`, key)
}

func (s *Store) scanOrder(ctx context.Context, query string, arg string) (*Order, error) {
	var order Order
	err := s.pool.QueryRow(ctx, query, arg).Scan(
		&order.ID, &order.OrderNumber, &order.IdempotencyKey, &order.Status, &order.TotalAmountCents, &order.Currency,
		&order.CustomerEmail, &order.StripeCheckoutSessionID, &order.StripePaymentIntentID,
		&order.StripeCheckoutURL, &order.StripeClientSecret, &order.UIMode,
		&order.SuccessURL, &order.CancelURL, &order.ReturnURL, &order.Metadata, &order.PaidAt, &order.CreatedAt, &order.UpdatedAt,
		&order.AccessTokenHash,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items, err := s.ListOrderItems(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	order.Items = items
	return &order, nil
}

// ListOrderItems returns line items for an order.
func (s *Store) ListOrderItems(ctx context.Context, orderID string) ([]OrderItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, order_id, product_id, product_name, quantity, unit_amount_cents, line_total_cents
		FROM order_items WHERE order_id = $1`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.ProductName, &item.Quantity, &item.UnitAmountCents, &item.LineTotalCents); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// CreateOrderWithItems inserts an order and its line items in a transaction.
func (s *Store) CreateOrderWithItems(ctx context.Context, order CreateOrderParams, items []CreateOrderItemParams) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	metaBytes, err := mergeRequestBodyHash(order.Metadata, order.RequestBodyHash)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO orders (id, order_number, idempotency_key, status, total_amount_cents, currency,
			customer_email, ui_mode, success_url, cancel_url, return_url, metadata, access_token_hash)
		VALUES ($1, $2, $3, 'pending', $4, $5, $6, $7::ui_mode, $8, $9, $10, $11, $12)`,
		order.ID, order.OrderNumber, order.IdempotencyKey, order.TotalAmountCents, order.Currency,
		order.CustomerEmail, order.UIMode, order.SuccessURL, order.CancelURL, order.ReturnURL, metaBytes,
		nullIfEmpty(order.AccessTokenHash),
	)
	if err != nil {
		return err
	}

	for _, item := range items {
		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (id, order_id, product_id, product_name, quantity, unit_amount_cents, line_total_cents)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			item.ID, item.OrderID, item.ProductID, item.ProductName, item.Quantity, item.UnitAmountCents, item.LineTotalCents,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// UpdateOrderSession persists Stripe session details on an order.
func (s *Store) UpdateOrderSession(ctx context.Context, orderID, sessionID, checkoutURL, clientSecret string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE orders SET stripe_checkout_session_id = $2, stripe_checkout_url = $3,
			stripe_client_secret = $4, updated_at = now()
		WHERE id = $1`,
		orderID, sessionID, nullIfEmpty(checkoutURL), nullIfEmpty(clientSecret),
	)
	return err
}

// UpdateOrderAccessTokenHash replaces the stored access-token hash for an order.
func (s *Store) UpdateOrderAccessTokenHash(ctx context.Context, orderID, tokenHash string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE orders SET access_token_hash = $2, updated_at = now()
		WHERE id = $1`, orderID, tokenHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("order %s not found", orderID)
	}
	return nil
}

// ClearOrderIdempotencyKey removes the idempotency key from a canceled order.
func (s *Store) ClearOrderIdempotencyKey(ctx context.Context, orderID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE orders SET idempotency_key = NULL, updated_at = now()
		WHERE id = $1 AND status = 'canceled'`, orderID)
	return err
}

// CancelOrder marks an order as canceled with a reason in metadata.
func (s *Store) CancelOrder(ctx context.Context, orderID, reason string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE orders SET status = 'canceled',
			idempotency_key = NULL,
			metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object('cancel_reason', $2::text),
			updated_at = now()
		WHERE id = $1`, orderID, reason)
	return err
}

// UpdateOrderStatus sets order status unconditionally.
func (s *Store) UpdateOrderStatus(ctx context.Context, orderID, status string, paymentIntentID *string, paidAt *time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE orders SET status = $2::order_status,
			stripe_payment_intent_id = COALESCE($3, stripe_payment_intent_id),
			paid_at = COALESCE($4, paid_at),
			updated_at = now()
		WHERE id = $1`,
		orderID, status, paymentIntentID, paidAt)
	return err
}

// UpdateOrderStatusIfAllowed transitions order status only from allowed prior states.
func (s *Store) UpdateOrderStatusIfAllowed(ctx context.Context, orderID, newStatus string, paymentIntentID *string, paidAt *time.Time, allowedFrom []string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE orders SET status = $2::order_status,
			stripe_payment_intent_id = COALESCE($3, stripe_payment_intent_id),
			paid_at = COALESCE($4, paid_at),
			updated_at = now()
		WHERE id = $1 AND status::text = ANY($5) AND status::text NOT IN ('paid', 'refunded')`,
		orderID, newStatus, paymentIntentID, paidAt, allowedFrom)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// CancelStalePendingOrders cancels pending orders that never received a Stripe session.
func (s *Store) CancelStalePendingOrders(ctx context.Context, olderThan time.Duration) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE orders SET status = 'canceled',
			metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object('cancel_reason', 'stale_checkout'),
			updated_at = now()
		WHERE status = 'pending'
			AND stripe_checkout_session_id IS NULL
			AND created_at < now() - $1::interval`,
		olderThan.String(),
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
