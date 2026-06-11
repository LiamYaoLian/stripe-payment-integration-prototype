package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, map[string]string{"status": "ok"})

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var env Envelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Error != nil {
		t.Fatal("expected no error")
	}
	data, ok := env.Data.(map[string]any)
	if !ok || data["status"] != "ok" {
		t.Fatalf("unexpected data: %v", env.Data)
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusBadRequest, "VALIDATION_ERROR", "bad input")

	var env Envelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Data != nil {
		t.Fatal("expected nil data")
	}
	if env.Error == nil || env.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("unexpected error: %v", env.Error)
	}
}

func TestWriteAppError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteAppError(rec, &AppError{Status: 409, Code: "CONFLICT", Message: "duplicate"})

	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestAppError_Error(t *testing.T) {
	err := &AppError{Message: "oops"}
	if err.Error() != "oops" {
		t.Fatalf("got %q", err.Error())
	}
}
