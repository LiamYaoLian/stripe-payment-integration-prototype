package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/testutil"
)

func TestProductsHandlerListsProducts(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	store.Products["p1"] = db.Product{ID: "p1", Name: "Pro", UnitAmountCents: 4900, Currency: "usd", Active: true}
	h := NewProductsHandler(service.NewProductService(store))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/products", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data struct {
			Products []map[string]any `json:"products"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if len(env.Data.Products) != 1 || env.Data.Products[0]["name"] != "Pro" {
		t.Fatalf("products %+v", env.Data.Products)
	}
}

func TestProductsHandlerStoreError(t *testing.T) {
	store := testutil.NewFakeOrderStore()
	store.ListActiveProductsErr = errors.New("db unavailable")
	h := NewProductsHandler(service.NewProductService(store))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/products", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != "INTERNAL_ERROR" {
		t.Fatalf("error %+v", env.Error)
	}
}
