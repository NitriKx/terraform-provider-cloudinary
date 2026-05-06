package provider_test

import (
	"testing"

	"github.com/NitriKx/terraform-provider-cloudinary/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is used by acceptance tests to create the provider.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"cloudinary": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestProvider(t *testing.T) {
	// Verify the provider can be instantiated without panicking.
	p := provider.New("test")()
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}
