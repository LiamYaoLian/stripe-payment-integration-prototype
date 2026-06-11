package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeHealthStore struct {
	err error
}

func (f *fakeHealthStore) Ping(context.Context) error { return f.err }

func TestHealthHandlerConnected(t *testing.T) {
	h := &HealthHandler{store: &fakeHealthStore{}}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var data map[string]any
	decodeEnvelope(t, rec, &data)
	if data["status"] != "ok" || data["db"] != "connected" {
		t.Fatalf("data %v", data)
	}
}

func TestHealthHandlerDisconnected(t *testing.T) {
	h := &HealthHandler{store: &fakeHealthStore{err: errors.New("down")}}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	var data map[string]any
	decodeEnvelope(t, rec, &data)
	if data["db"] != "disconnected" {
		t.Fatalf("data %v", data)
	}
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder, data *map[string]any) {
	t.Helper()
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	*data = env.Data
}
