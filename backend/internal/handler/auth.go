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
	auth          UserAuthenticator
	secureCookies bool
}

func NewAuthHandler(auth UserAuthenticator, secureCookies bool) *AuthHandler {
	return &AuthHandler{auth: auth, secureCookies: secureCookies}
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authSessionResponse struct {
	ExpiresAt string              `json:"expiresAt"`
	User      service.UserProfile `json:"user"`
}

type emailRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type verifyEmailRequest struct {
	Token string `json:"token"`
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

	auth.SetSessionCookie(w, result.SessionToken, result.ExpiresAt, h.secureCookies)
	api.WriteJSON(w, http.StatusCreated, authSessionResponse{
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

	auth.SetSessionCookie(w, result.SessionToken, result.ExpiresAt, h.secureCookies)
	api.WriteJSON(w, http.StatusOK, authSessionResponse{
		ExpiresAt: result.ExpiresAt.UTC().Format(time.RFC3339),
		User:      result.User,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	_ = h.auth.Logout(r.Context(), auth.SessionTokenFromRequest(r))
	auth.ClearSessionCookie(w, h.secureCookies)
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "signed_out"})
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

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var body emailRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if isRequestBodyTooLarge(err) {
			writeBodyTooLarge(w)
			return
		}
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
		return
	}

	if err := h.auth.RequestPasswordReset(r.Context(), body.Email); err != nil {
		writeServiceError(w, r, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "If an account exists for that email, a reset link has been sent.",
	})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var body resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if isRequestBodyTooLarge(err) {
			writeBodyTooLarge(w)
			return
		}
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
		return
	}

	if err := h.auth.ResetPassword(r.Context(), body.Token, body.Password); err != nil {
		writeServiceError(w, r, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var body verifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if isRequestBodyTooLarge(err) {
			writeBodyTooLarge(w)
			return
		}
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json body")
		return
	}

	if err := h.auth.VerifyEmail(r.Context(), body.Token); err != nil {
		writeServiceError(w, r, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]string{"message": "email verified"})
}
