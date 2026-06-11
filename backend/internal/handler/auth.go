package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
)

// AuthHandler serves guest session endpoints.
type AuthHandler struct {
	auth GuestAuthenticator
}

func NewAuthHandler(auth GuestAuthenticator) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type guestSessionRequest struct {
	Email       string `json:"email"`
	OrderID     string `json:"orderId"`
	AccessToken string `json:"accessToken"`
}

type guestSessionResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
	Role      string `json:"role"`
}

func (h *AuthHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var body guestSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if isRequestBodyTooLarge(err) {
			writeBodyTooLarge(w)
			return
		}
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
		return
	}

	result, err := h.auth.CreateGuestSession(r.Context(), body.Email, body.OrderID, body.AccessToken)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, guestSessionResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt.UTC().Format(time.RFC3339),
		Role:      result.Role,
	})
}
