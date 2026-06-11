package testutil

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/jackc/pgx/v5/pgconn"
)

type FakeOrderStore struct {
	Products               map[string]db.Product
	Orders                 map[string]*db.Order
	OrdersBySession        map[string]*db.Order
	OrdersByIdempotency    map[string]*db.Order
	ListActiveProductsErr  error
	UpdateSessionErr       error
	CancelReasons          []string
	// RaceInsertViolation simulates a concurrent idempotency-key insert winning between lookup and insert.
	RaceInsertViolation bool
}

func NewFakeOrderStore() *FakeOrderStore {
	return &FakeOrderStore{
		Products:            make(map[string]db.Product),
		Orders:              make(map[string]*db.Order),
		OrdersBySession:     make(map[string]*db.Order),
		OrdersByIdempotency: make(map[string]*db.Order),
	}
}

func (f *FakeOrderStore) Ping(_ context.Context) error {
	return nil
}

func (f *FakeOrderStore) ListActiveProducts(_ context.Context) ([]db.Product, error) {
	if f.ListActiveProductsErr != nil {
		return nil, f.ListActiveProductsErr
	}
	out := make([]db.Product, 0, len(f.Products))
	for _, p := range f.Products {
		if p.Active {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *FakeOrderStore) GetOrderByID(_ context.Context, id string) (*db.Order, error) {
	if o, ok := f.Orders[id]; ok {
		return o, nil
	}
	return nil, nil
}

func (f *FakeOrderStore) GetOrderBySessionID(_ context.Context, sessionID string) (*db.Order, error) {
	if o, ok := f.OrdersBySession[sessionID]; ok {
		return o, nil
	}
	return nil, nil
}

func (f *FakeOrderStore) GetOrderByIdempotencyKey(_ context.Context, key string) (*db.Order, error) {
	if o, ok := f.OrdersByIdempotency[key]; ok {
		return o, nil
	}
	return nil, nil
}

func (f *FakeOrderStore) ClearOrderIdempotencyKey(_ context.Context, orderID string) error {
	for key, o := range f.OrdersByIdempotency {
		if o.ID == orderID {
			delete(f.OrdersByIdempotency, key)
			o.IdempotencyKey = nil
		}
	}
	return nil
}

func (f *FakeOrderStore) GetProduct(_ context.Context, id string) (*db.Product, error) {
	if p, ok := f.Products[id]; ok {
		copy := p
		return &copy, nil
	}
	return nil, nil
}

func (f *FakeOrderStore) GetProductsByIDs(_ context.Context, ids []string) (map[string]db.Product, error) {
	result := make(map[string]db.Product, len(ids))
	for _, id := range ids {
		if product, ok := f.Products[id]; ok && product.Active {
			result[id] = product
		}
	}
	return result, nil
}

func (f *FakeOrderStore) CreateOrderWithItems(_ context.Context, order db.CreateOrderParams, items []db.CreateOrderItemParams) error {
	if f.RaceInsertViolation && order.IdempotencyKey != nil {
		f.RaceInsertViolation = false
		sess := "cs_race_winner"
		url := "https://checkout.stripe.com/race"
		winner := &db.Order{
			ID: "ord-race-winner", OrderNumber: "ORD-RACE", UIMode: order.UIMode, Status: "pending",
			IdempotencyKey: order.IdempotencyKey, Metadata: order.Metadata,
			StripeCheckoutSessionID: &sess, StripeCheckoutURL: &url,
		}
		f.Orders[winner.ID] = winner
		f.OrdersByIdempotency[*order.IdempotencyKey] = winner
		f.OrdersBySession[sess] = winner
		return &pgconn.PgError{Code: "23505"}
	}
	if order.IdempotencyKey != nil {
		if existing, ok := f.OrdersByIdempotency[*order.IdempotencyKey]; ok && existing.ID != order.ID {
			return &pgconn.PgError{Code: "23505"}
		}
	}

	meta := order.Metadata
	if meta == nil {
		meta = json.RawMessage(`{}`)
	}
	var metaMap map[string]any
	_ = json.Unmarshal(meta, &metaMap)
	if metaMap == nil {
		metaMap = map[string]any{}
	}
	metaMap["_request_body_hash"] = order.RequestBodyHash
	metaBytes, _ := json.Marshal(metaMap)

	var tokenHash *string
	if order.AccessTokenHash != "" {
		tokenHash = &order.AccessTokenHash
	}
	o := &db.Order{
		ID: order.ID, OrderNumber: order.OrderNumber, IdempotencyKey: order.IdempotencyKey,
		Status: "pending", TotalAmountCents: order.TotalAmountCents, Currency: order.Currency,
		CustomerEmail: order.CustomerEmail, UIMode: order.UIMode,
		SuccessURL: order.SuccessURL, CancelURL: order.CancelURL, ReturnURL: order.ReturnURL,
		Metadata: metaBytes, AccessTokenHash: tokenHash,
	}
	for _, it := range items {
		o.Items = append(o.Items, db.OrderItem{
			ProductID: &it.ProductID, ProductName: it.ProductName,
			Quantity: it.Quantity, LineTotalCents: it.LineTotalCents,
		})
	}
	f.Orders[o.ID] = o
	if o.IdempotencyKey != nil {
		f.OrdersByIdempotency[*o.IdempotencyKey] = o
	}
	return nil
}

func (f *FakeOrderStore) UpdateOrderAccessTokenHash(_ context.Context, orderID, tokenHash string) error {
	o := f.Orders[orderID]
	if o == nil {
		return nil
	}
	o.AccessTokenHash = &tokenHash
	return nil
}

func (f *FakeOrderStore) UpdateOrderSession(_ context.Context, orderID, sessionID, checkoutURL, clientSecret string) error {
	if f.UpdateSessionErr != nil {
		return f.UpdateSessionErr
	}
	o := f.Orders[orderID]
	if o == nil {
		return nil
	}
	o.StripeCheckoutSessionID = &sessionID
	if checkoutURL != "" {
		o.StripeCheckoutURL = &checkoutURL
	}
	if clientSecret != "" {
		o.StripeClientSecret = &clientSecret
	}
	f.OrdersBySession[sessionID] = o
	return nil
}

func (f *FakeOrderStore) ListOrdersByCustomerID(_ context.Context, customerID string, limit int) ([]db.Order, error) {
	if limit <= 0 {
		limit = 20
	}
	var orders []db.Order
	for _, order := range f.Orders {
		if order.CustomerID != nil && *order.CustomerID == customerID {
			orders = append(orders, *order)
		}
	}
	if len(orders) > limit {
		orders = orders[:limit]
	}
	return orders, nil
}

func (f *FakeOrderStore) ListOrdersByCustomerEmail(_ context.Context, email string, limit int) ([]db.Order, error) {
	if limit <= 0 {
		limit = 20
	}
	var orders []db.Order
	for _, order := range f.Orders {
		if order.CustomerEmail != nil && strings.EqualFold(*order.CustomerEmail, email) {
			orders = append(orders, *order)
		}
	}
	if len(orders) > limit {
		orders = orders[:limit]
	}
	return orders, nil
}

func (f *FakeOrderStore) CancelOrder(_ context.Context, orderID, reason string) error {
	f.CancelReasons = append(f.CancelReasons, reason)
	if o := f.Orders[orderID]; o != nil {
		o.Status = "canceled"
		if o.IdempotencyKey != nil {
			delete(f.OrdersByIdempotency, *o.IdempotencyKey)
			o.IdempotencyKey = nil
		}
	}
	return nil
}
