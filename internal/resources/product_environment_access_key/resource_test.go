package product_environment_access_key_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/NitriKx/terraform-provider-cloudinary/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"cloudinary": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("CLOUDINARY_ACCOUNT_ID") == "" {
		t.Skip("CLOUDINARY_ACCOUNT_ID must be set for access_key acceptance tests")
	}
	if v := os.Getenv("CLOUDINARY_URL"); v == "" {
		if os.Getenv("CLOUDINARY_CLOUD_NAME") == "" || os.Getenv("CLOUDINARY_API_KEY") == "" || os.Getenv("CLOUDINARY_API_SECRET") == "" {
			t.Skip("CLOUDINARY_URL or (CLOUDINARY_CLOUD_NAME + CLOUDINARY_API_KEY + CLOUDINARY_API_SECRET) must be set for acceptance tests")
		}
	}
}

func TestAccAccessKeyResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAccessKeyConfig("tf-acc-test-env", "tf-acc-test-key"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudinary_product_environment_access_key.test", "name", "tf-acc-test-key"),
					resource.TestCheckResourceAttr("cloudinary_product_environment_access_key.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("cloudinary_product_environment_access_key.test", "api_key"),
					resource.TestCheckResourceAttrSet("cloudinary_product_environment_access_key.test", "api_secret"),
				),
			},
			{
				ResourceName:      "cloudinary_product_environment_access_key.test",
				ImportState:       true,
				ImportStateIdFunc: testAccAccessKeyImportID("cloudinary_product_environment_access_key.test"),
				ImportStateVerify: true,
				// api_secret is not returned on read, so it can't be verified after import.
				ImportStateVerifyIgnore: []string{"api_secret"},
			},
		},
	})
}

func testAccAccessKeyImportID(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}
		envID := rs.Primary.Attributes["product_environment_id"]
		apiKey := rs.Primary.Attributes["api_key"]
		return envID + "/" + apiKey, nil
	}
}

func testAccAccessKeyConfig(envName, keyName string) string {
	return fmt.Sprintf(`
resource "cloudinary_product_environment" "test" {
  name    = %q
  enabled = true
}

resource "cloudinary_product_environment_access_key" "test" {
  product_environment_id = cloudinary_product_environment.test.id
  name                   = %q
  enabled                = true
}
`, envName, keyName)
}
