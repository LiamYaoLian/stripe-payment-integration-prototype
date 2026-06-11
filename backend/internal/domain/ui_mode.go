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
