package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

type Product struct {
	ID              string
	Name            string
	Description     *string
	UnitAmountCents int32
	Currency        string
	Active          bool
}

type OrderItem struct {
	ID              string
	OrderID         string
	ProductID       *string
	ProductName     string
	Quantity        int32
	UnitAmountCents int32
	LineTotalCents  int32
}

type Order struct {
	ID                      string
	OrderNumber             string
	IdempotencyKey          *string
	Status                  string
	TotalAmountCents        int32
	Currency                string
	CustomerEmail           *string
	StripeCheckoutSessionID *string
	StripePaymentIntentID   *string
	StripeCheckoutURL       *string
	StripeClientSecret      *string
	UIMode                  string
	SuccessURL              *string
	CancelURL               *string
	ReturnURL               *string
	Metadata                json.RawMessage
	PaidAt                  *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
	Items                   []OrderItem
}

type WebhookEvent struct {
	ID               string
	StripeEventID    string
	EventType        string
	OrderID          *string
	ProcessingStatus string
	Payload          json.RawMessage
	ProcessedAt      *time.Time
}

func (s *Store) ListActiveProducts(ctx context.Context) ([]Product, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, description, unit_amount_cents, currency, active
		FROM products WHERE active = true ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.UnitAmountCents, &p.Currency, &p.Active); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (s *Store) GetProduct(ctx context.Context, id string) (*Product, error) {
	var p Product
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, description, unit_amount_cents, currency, active
		FROM products WHERE id = $1 AND active = true`, id).
		Scan(&p.ID, &p.Name, &p.Description, &p.UnitAmountCents, &p.Currency, &p.Active)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) GetOrderByID(ctx context.Context, id string) (*Order, error) {
	return s.scanOrder(ctx, `
		SELECT id, order_number, idempotency_key, status::text, total_amount_cents, currency,
			customer_email, stripe_checkout_session_id, stripe_payment_intent_id,
			stripe_checkout_url, stripe_client_secret, ui_mode::text,
			success_url, cancel_url, return_url, metadata, paid_at, created_at, updated_at
		FROM orders WHERE id = $1`, id)
}

func (s *Store) GetOrderBySessionID(ctx context.Context, sessionID string) (*Order, error) {
	return s.scanOrder(ctx, `
		SELECT id, order_number, idempotency_key, status::text, total_amount_cents, currency,
			customer_email, stripe_checkout_session_id, stripe_payment_intent_id,
			stripe_checkout_url, stripe_client_secret, ui_mode::text,
			success_url, cancel_url, return_url, metadata, paid_at, created_at, updated_at
		FROM orders WHERE stripe_checkout_session_id = $1`, sessionID)
}

func (s *Store) GetOrderByIdempotencyKey(ctx context.Context, key string) (*Order, error) {
	return s.scanOrder(ctx, `
		SELECT id, order_number, idempotency_key, status::text, total_amount_cents, currency,
			customer_email, stripe_checkout_session_id, stripe_payment_intent_id,
			stripe_checkout_url, stripe_client_secret, ui_mode::text,
			success_url, cancel_url, return_url, metadata, paid_at, created_at, updated_at
		FROM orders WHERE idempotency_key = $1`, key)
}

func (s *Store) scanOrder(ctx context.Context, query string, arg string) (*Order, error) {
	var o Order
	err := s.pool.QueryRow(ctx, query, arg).Scan(
		&o.ID, &o.OrderNumber, &o.IdempotencyKey, &o.Status, &o.TotalAmountCents, &o.Currency,
		&o.CustomerEmail, &o.StripeCheckoutSessionID, &o.StripePaymentIntentID,
		&o.StripeCheckoutURL, &o.StripeClientSecret, &o.UIMode,
		&o.SuccessURL, &o.CancelURL, &o.ReturnURL, &o.Metadata, &o.PaidAt, &o.CreatedAt, &o.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items, err := s.ListOrderItems(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return &o, nil
}

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
		var it OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.Quantity, &it.UnitAmountCents, &it.LineTotalCents); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

type CreateOrderParams struct {
	ID               string
	OrderNumber      string
	IdempotencyKey   *string
	TotalAmountCents int32
	Currency         string
	CustomerEmail    *string
	UIMode           string
	SuccessURL       *string
	CancelURL        *string
	ReturnURL        *string
	Metadata         json.RawMessage
	RequestBodyHash  string
}

type CreateOrderItemParams struct {
	ID              string
	OrderID         string
	ProductID       string
	ProductName     string
	Quantity        int32
	UnitAmountCents int32
	LineTotalCents  int32
}

func (s *Store) CreateOrderWithItems(ctx context.Context, order CreateOrderParams, items []CreateOrderItemParams) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	meta := order.Metadata
	if meta == nil {
		meta = json.RawMessage(`{}`)
	}
	// merge request hash into metadata
	var metaMap map[string]any
	_ = json.Unmarshal(meta, &metaMap)
	if metaMap == nil {
		metaMap = map[string]any{}
	}
	metaMap["_request_body_hash"] = order.RequestBodyHash
	metaBytes, _ := json.Marshal(metaMap)

	_, err = tx.Exec(ctx, `
		INSERT INTO orders (id, order_number, idempotency_key, status, total_amount_cents, currency,
			customer_email, ui_mode, success_url, cancel_url, return_url, metadata)
		VALUES ($1, $2, $3, 'pending', $4, $5, $6, $7::ui_mode, $8, $9, $10, $11)`,
		order.ID, order.OrderNumber, order.IdempotencyKey, order.TotalAmountCents, order.Currency,
		order.CustomerEmail, order.UIMode, order.SuccessURL, order.CancelURL, order.ReturnURL, metaBytes,
	)
	if err != nil {
		return err
	}

	for _, it := range items {
		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (id, order_id, product_id, product_name, quantity, unit_amount_cents, line_total_cents)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			it.ID, it.OrderID, it.ProductID, it.ProductName, it.Quantity, it.UnitAmountCents, it.LineTotalCents,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *Store) UpdateOrderSession(ctx context.Context, orderID, sessionID, checkoutURL, clientSecret string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE orders SET stripe_checkout_session_id = $2, stripe_checkout_url = $3,
			stripe_client_secret = $4, updated_at = now()
		WHERE id = $1`,
		orderID, sessionID, nullIfEmpty(checkoutURL), nullIfEmpty(clientSecret),
	)
	return err
}

func (s *Store) ClearOrderIdempotencyKey(ctx context.Context, orderID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE orders SET idempotency_key = NULL, updated_at = now()
		WHERE id = $1 AND status = 'canceled'`, orderID)
	return err
}

func (s *Store) CancelOrder(ctx context.Context, orderID, reason string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE orders SET status = 'canceled',
			metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object('cancel_reason', $2::text),
			updated_at = now()
		WHERE id = $1`, orderID, reason)
	return err
}

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

func (s *Store) GetWebhookEvent(ctx context.Context, stripeEventID string) (*WebhookEvent, error) {
	var e WebhookEvent
	err := s.pool.QueryRow(ctx, `
		SELECT id, stripe_event_id, event_type, order_id, processing_status::text, payload, processed_at
		FROM webhook_events WHERE stripe_event_id = $1`, stripeEventID).
		Scan(&e.ID, &e.StripeEventID, &e.EventType, &e.OrderID, &e.ProcessingStatus, &e.Payload, &e.ProcessedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) InsertWebhookEvent(ctx context.Context, e WebhookEvent) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_events (id, stripe_event_id, event_type, order_id, processing_status, payload)
		VALUES ($1, $2, $3, $4, $5::webhook_processing_status, $6)`,
		e.ID, e.StripeEventID, e.EventType, e.OrderID, e.ProcessingStatus, e.Payload)
	return err
}

func (s *Store) ClaimWebhookEvent(ctx context.Context, stripeEventID string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE webhook_events SET processing_status = 'processed', processed_at = now()
		WHERE stripe_event_id = $1 AND processing_status IN ('received', 'failed')`, stripeEventID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) FailWebhookEvent(ctx context.Context, stripeEventID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_events SET processing_status = 'failed'
		WHERE stripe_event_id = $1`, stripeEventID)
	return err
}

func (s *Store) MarkWebhookIgnored(ctx context.Context, stripeEventID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_events SET processing_status = 'ignored', processed_at = now()
		WHERE stripe_event_id = $1`, stripeEventID)
	return err
}

func (s *Store) UpsertProduct(ctx context.Context, p Product) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO products (id, name, description, unit_amount_cents, currency, active)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING`,
		p.ID, p.Name, p.Description, p.UnitAmountCents, p.Currency, p.Active)
	return err
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
