package handler

import (
	"context"
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
)

type healthStore interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	store healthStore
}

func NewHealthHandler(store *db.Store) *HealthHandler {
	return &HealthHandler{store: store}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dbStatus := "connected"
	if err := h.store.Ping(r.Context()); err != nil {
		dbStatus = "disconnected"
	}
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "db": dbStatus})
}
