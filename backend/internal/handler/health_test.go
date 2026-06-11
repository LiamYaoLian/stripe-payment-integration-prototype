package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/LiamYaoLian/stripe-payment-integration-prototype/backend/internal/testutil"
)

func TestHealthLiveAlwaysOK(t *testing.T) {
	h := NewHealthHandlerFromStore(testutil.FakeHealth{Err: os.ErrDeadlineExceeded})
	rec := httptest.NewRecorder()
	h.Live(rec, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHealthReadyUnavailableWhenDBDown(t *testing.T) {
	h := NewHealthHandlerFromStore(testutil.FakeHealth{Err: os.ErrDeadlineExceeded})
	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHealthReadyConnected(t *testing.T) {
	h := NewHealthHandlerFromStore(testutil.FakeHealth{})
	rec := httptest.NewRecorder()
	h.Ready(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var env struct {
		Data map[string]string `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Data["db"] != "connected" {
		t.Fatalf("data %+v", env.Data)
	}
}
