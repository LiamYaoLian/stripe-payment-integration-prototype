package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDGenerated(t *testing.T) {
	var captured string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetRequestID(r.Context())
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	RequestID(next).ServeHTTP(rec, req)

	if captured == "" {
		t.Fatal("expected request id in context")
	}
	if rec.Header().Get("X-Request-ID") != captured {
		t.Fatalf("header %q context %q", rec.Header().Get("X-Request-ID"), captured)
	}
}

func TestRequestIDPreservedFromHeader(t *testing.T) {
	const want = "req-abc-123"
	var captured string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetRequestID(r.Context())
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", want)
	RequestID(next).ServeHTTP(rec, req)

	if captured != want {
		t.Fatalf("got %q want %q", captured, want)
	}
}

func TestGetRequestIDEmpty(t *testing.T) {
	if got := GetRequestID(t.Context()); got != "" {
		t.Fatalf("got %q", got)
	}
}
