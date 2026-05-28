package provider_test

import (
	"testing"

	"github.com/NitriKx/terraform-provider-cloudinary/internal/provider"
)

func TestProvider(t *testing.T) {
	// Verify the provider can be instantiated without panicking.
	p := provider.New("test")()
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}
