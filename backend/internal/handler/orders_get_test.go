package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/testutil"
	"github.com/go-chi/chi/v5"
	"github.com/rs/xid"
)

func TestOrdersGetByIDFound(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	token, _ := testutil.NewOrderAccessToken(t)
	orderID := xid.New().String()
	order := &db.Order{
		ID: orderID, OrderNumber: "ORD-1", Status: "paid",
		TotalAmountCents: 4900, Currency: "usd",
	}
	testutil.WithAccessToken(order, token)
	store.Orders[orderID] = order
	h := NewOrdersHandler(service.NewOrderService(store, nil, "http://localhost:5173"))
	r := chi.NewRouter()
	r.Get("/api/orders/{id}", h.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/api/orders/"+orderID, nil)
	req.Header.Set("X-Order-Token", token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Data["orderNumber"] != "ORD-1" {
		t.Fatalf("data %+v", env.Data)
	}
}

func TestOrdersGetByIDUnauthorizedWithoutToken(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	token, _ := testutil.NewOrderAccessToken(t)
	orderID := xid.New().String()
	order := &db.Order{ID: orderID, OrderNumber: "ORD-1", Status: "paid"}
	testutil.WithAccessToken(order, token)
	store.Orders[orderID] = order
	h := NewOrdersHandler(service.NewOrderService(store, nil, "http://localhost:5173"))
	r := chi.NewRouter()
	r.Get("/api/orders/{id}", h.GetByID)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/orders/"+orderID, nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestOrdersGetByIDNotFound(t *testing.T) {
	h := NewOrdersHandler(service.NewOrderService(testutil.NewFakeOrderStore(), nil, "http://localhost:5173"))
	r := chi.NewRouter()
	r.Get("/api/orders/{id}", h.GetByID)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/orders/"+xid.New().String(), nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestOrdersGetBySessionFound(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	token, _ := testutil.NewOrderAccessToken(t)
	order := &db.Order{ID: "ord1", OrderNumber: "ORD-1", Status: "paid"}
	testutil.WithAccessToken(order, token)
	store.OrdersBySession["cs_test_found"] = order
	h := NewOrdersHandler(service.NewOrderService(store, nil, "http://localhost:5173"))
	r := chi.NewRouter()
	r.Get("/api/orders/by-session/{sessionId}", h.GetBySession)

	req := httptest.NewRequest(http.MethodGet, "/api/orders/by-session/cs_test_found", nil)
	req.Header.Set("X-Order-Token", token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
}
