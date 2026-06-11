package domain

import "testing"

func TestStripeUIMode(t *testing.T) {
	tests := []struct {
		internal string
		stripe   string
	}{
		{UIModeHosted, "hosted_page"},
		{UIModeEmbedded, "embedded_page"},
		{"unknown", "unknown"},
	}
	for _, tc := range tests {
		if got := StripeUIMode(tc.internal); got != tc.stripe {
			t.Fatalf("StripeUIMode(%q) = %q, want %q", tc.internal, got, tc.stripe)
		}
	}
}
