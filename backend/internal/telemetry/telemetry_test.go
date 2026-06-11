package telemetry

import (
	"context"
	"testing"
)

func TestSetupDisabledWithoutEndpoint(t *testing.T) {
	shutdown, err := Setup(context.Background(), Config{ServiceName: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestConfigEnabled(t *testing.T) {
	var disabled Config
	if disabled.Enabled() {
		t.Fatal("expected disabled without endpoint")
	}
	enabled := Config{Endpoint: "http://localhost:4318"}
	if !enabled.Enabled() {
		t.Fatal("expected enabled with endpoint")
	}
}
