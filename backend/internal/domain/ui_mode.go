package domain

// Checkout UI mode values stored in the database.
const (
	UIModeHosted    = "hosted"
	UIModeEmbedded  = "embedded"
)

// ValidUIModes lists allowed checkout UI modes.
var ValidUIModes = map[string]bool{
	UIModeHosted:   true,
	UIModeEmbedded: true,
}

// Stripe API 2026-05-27.dahlia renamed ui_mode values (hosted_page, embedded_page).
// Internal API/DB keep hosted and embedded for stable client contracts.
func StripeUIMode(internal string) string {
	switch internal {
	case UIModeHosted:
		return "hosted_page"
	case UIModeEmbedded:
		return "embedded_page"
	default:
		return internal
	}
}
