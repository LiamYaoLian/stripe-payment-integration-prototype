package domain

import "testing"

func TestWebhookOrderEffectSatisfied(t *testing.T) {
	tests := []struct {
		current  string
		target   string
		refund   bool
		want     bool
	}{
		{OrderStatusPaid, OrderStatusPaid, false, true},
		{OrderStatusPaid, OrderStatusProcessing, false, true},
		{OrderStatusRefunded, OrderStatusPaid, false, true},
		{OrderStatusPending, OrderStatusPaid, false, false},
		{OrderStatusRefunded, OrderStatusRefunded, true, true},
		{OrderStatusPaid, OrderStatusRefunded, true, false},
	}
	for _, tc := range tests {
		got := WebhookOrderEffectSatisfied(tc.current, tc.target, tc.refund)
		if got != tc.want {
			t.Fatalf("current=%s target=%s refund=%v got=%v want=%v", tc.current, tc.target, tc.refund, got, tc.want)
		}
	}
}
