package domain

// WebhookOrderEffectSatisfied reports whether an order already reflects the webhook outcome
// when an UPDATE matched zero rows (idempotent duplicate or already advanced).
func WebhookOrderEffectSatisfied(currentStatus, targetStatus string, refundOnly bool) bool {
	if refundOnly {
		return currentStatus == OrderStatusRefunded
	}
	if currentStatus == targetStatus {
		return true
	}
	switch targetStatus {
	case OrderStatusProcessing:
		return currentStatus == OrderStatusPaid || currentStatus == OrderStatusRefunded
	case OrderStatusPaid:
		return currentStatus == OrderStatusRefunded
	case OrderStatusFailed, OrderStatusExpired:
		return currentStatus == OrderStatusPaid || currentStatus == OrderStatusRefunded
	default:
		return false
	}
}
