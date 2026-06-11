package config

import "testing"

func TestLoadSeed(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://custom/seed")
	url, err := LoadSeed()
	if err != nil {
		t.Fatal(err)
	}
	if url != "postgresql://custom/seed" {
		t.Fatalf("url %q", url)
	}
}
