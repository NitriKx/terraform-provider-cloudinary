package product_environment_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/NitriKx/terraform-provider-cloudinary/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"cloudinary": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("CLOUDINARY_ACCOUNT_ID") == "" {
		t.Skip("CLOUDINARY_ACCOUNT_ID must be set for product_environment acceptance tests")
	}
	if v := os.Getenv("CLOUDINARY_URL"); v == "" {
		if os.Getenv("CLOUDINARY_CLOUD_NAME") == "" || os.Getenv("CLOUDINARY_API_KEY") == "" || os.Getenv("CLOUDINARY_API_SECRET") == "" {
			t.Skip("CLOUDINARY_URL or (CLOUDINARY_CLOUD_NAME + CLOUDINARY_API_KEY + CLOUDINARY_API_SECRET) must be set for acceptance tests")
		}
	}
}

func TestAccProductEnvironmentResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProductEnvironmentConfig("tf-acc-test-env"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudinary_product_environment.test", "name", "tf-acc-test-env"),
					resource.TestCheckResourceAttr("cloudinary_product_environment.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("cloudinary_product_environment.test", "id"),
					resource.TestCheckResourceAttrSet("cloudinary_product_environment.test", "cloud_name"),
				),
			},
			{
				ResourceName:      "cloudinary_product_environment.test",
				ImportState:       true,
				ImportStateVerify: true,
				// base_sub_account_id is not returned by the API on read.
				ImportStateVerifyIgnore: []string{"base_sub_account_id"},
			},
			{
				Config: testAccProductEnvironmentConfigUpdated("tf-acc-test-env-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudinary_product_environment.test", "name", "tf-acc-test-env-updated"),
					resource.TestCheckResourceAttr("cloudinary_product_environment.test", "enabled", "false"),
				),
			},
		},
	})
}

func TestAccProductEnvironmentResource_customAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProductEnvironmentConfigWithAttributes("tf-acc-test-env-attrs"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudinary_product_environment.test", "name", "tf-acc-test-env-attrs"),
					resource.TestCheckResourceAttr("cloudinary_product_environment.test", "custom_attributes.team", "platform"),
				),
			},
		},
	})
}

func testAccProductEnvironmentConfig(name string) string {
	return fmt.Sprintf(`
resource "cloudinary_product_environment" "test" {
  name    = %q
  enabled = true
}
`, name)
}

func testAccProductEnvironmentConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "cloudinary_product_environment" "test" {
  name    = %q
  enabled = false
}
`, name)
}

func testAccProductEnvironmentConfigWithAttributes(name string) string {
	return fmt.Sprintf(`
resource "cloudinary_product_environment" "test" {
  name    = %q
  enabled = true

  custom_attributes = {
    team        = "platform"
    environment = "test"
  }
}
`, name)
}
