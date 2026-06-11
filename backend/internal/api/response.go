package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type Envelope struct {
	Data  any        `json:"data"`
	Error *ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details []any  `json:"details,omitempty"`
}

// AppError is a domain error with an HTTP status and machine-readable code.
type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	WriteEnvelope(w, status, data, nil)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteEnvelope(w, status, nil, &ErrorBody{Code: code, Message: message})
}

func WriteEnvelope(w http.ResponseWriter, status int, data any, err *ErrorBody) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if encodeErr := json.NewEncoder(w).Encode(Envelope{Data: data, Error: err}); encodeErr != nil {
		slog.Error("failed to encode JSON response", "error", encodeErr)
	}
}

func WriteAppError(w http.ResponseWriter, err *AppError) {
	WriteError(w, err.Status, err.Code, err.Message)
}
