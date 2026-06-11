package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/api"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/auth"
	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/service"
)

// AuthHandler serves user authentication endpoints.
type AuthHandler struct {
	auth UserAuthenticator
}

func NewAuthHandler(auth UserAuthenticator) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authSessionResponse struct {
	Token     string              `json:"token"`
	ExpiresAt string              `json:"expiresAt"`
	User      service.UserProfile `json:"user"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var body credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if isRequestBodyTooLarge(err) {
			writeBodyTooLarge(w)
			return
		}
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
		return
	}

	result, err := h.auth.Register(r.Context(), body.Email, body.Password)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, authSessionResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt.UTC().Format(time.RFC3339),
		User:      result.User,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if isRequestBodyTooLarge(err) {
			writeBodyTooLarge(w)
			return
		}
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
		return
	}

	result, err := h.auth.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, authSessionResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt.UTC().Format(time.RFC3339),
		User:      result.User,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.UserFromContext(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "user session required")
		return
	}

	user, err := h.auth.GetUser(r.Context(), session.ID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, user)
}
