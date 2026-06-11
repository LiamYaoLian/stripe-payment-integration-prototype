package handler

import (
	"context"
	"net/http"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/db"
)

type HealthStore interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	store HealthStore
}

func NewHealthHandler(store *db.Store) *HealthHandler {
	return &HealthHandler{store: store}
}

func NewHealthHandlerFromStore(store HealthStore) *HealthHandler {
	return &HealthHandler{store: store}
}

// Live reports process liveness (always OK if the server is running).
func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready reports readiness including database connectivity.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Ping(r.Context()); err != nil {
		api.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unavailable",
			"db":     "disconnected",
		})
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "db": "connected"})
}

// ServeHTTP is an alias for Ready (backward compatible /health).
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Ready(w, r)
}
