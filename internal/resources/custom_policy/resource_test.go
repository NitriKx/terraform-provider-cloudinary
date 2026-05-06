package custom_policy_test

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
		t.Skip("CLOUDINARY_ACCOUNT_ID must be set for custom_policy acceptance tests")
	}
	if v := os.Getenv("CLOUDINARY_URL"); v == "" {
		if os.Getenv("CLOUDINARY_CLOUD_NAME") == "" || os.Getenv("CLOUDINARY_API_KEY") == "" || os.Getenv("CLOUDINARY_API_SECRET") == "" {
			t.Skip("CLOUDINARY_URL or (CLOUDINARY_CLOUD_NAME + CLOUDINARY_API_KEY + CLOUDINARY_API_SECRET) must be set for acceptance tests")
		}
	}
}

func TestAccCustomPolicyResource_account_scope(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomPolicyAccountConfig("tf-acc-test-policy"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudinary_custom_policy.test", "name", "tf-acc-test-policy"),
					resource.TestCheckResourceAttr("cloudinary_custom_policy.test", "scope_type", "account"),
					resource.TestCheckResourceAttr("cloudinary_custom_policy.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("cloudinary_custom_policy.test", "id"),
				),
			},
			{
				ResourceName:      "cloudinary_custom_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCustomPolicyAccountConfigUpdated("tf-acc-test-policy-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudinary_custom_policy.test", "name", "tf-acc-test-policy-updated"),
					resource.TestCheckResourceAttr("cloudinary_custom_policy.test", "enabled", "false"),
				),
			},
		},
	})
}

func TestAccCustomPolicyResource_product_env_scope(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomPolicyEnvConfig("tf-acc-test-env", "tf-acc-test-policy-env"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudinary_custom_policy.test", "scope_type", "product_environment"),
					resource.TestCheckResourceAttrSet("cloudinary_custom_policy.test", "scope_id"),
				),
			},
		},
	})
}

func testAccCustomPolicyAccountConfig(name string) string {
	return fmt.Sprintf(`
resource "cloudinary_custom_policy" "test" {
  name        = %q
  scope_type  = "account"
  enabled     = true

  policy_statement = jsonencode({
    effect   = "allow"
    action   = ["read:*"]
    resource = ["*"]
  })
}
`, name)
}

func testAccCustomPolicyAccountConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "cloudinary_custom_policy" "test" {
  name        = %q
  scope_type  = "account"
  enabled     = false

  policy_statement = jsonencode({
    effect   = "allow"
    action   = ["read:*"]
    resource = ["*"]
  })
}
`, name)
}

func testAccCustomPolicyEnvConfig(envName, policyName string) string {
	return fmt.Sprintf(`
resource "cloudinary_product_environment" "test" {
  name    = %q
  enabled = true
}

resource "cloudinary_custom_policy" "test" {
  name       = %q
  scope_type = "product_environment"
  scope_id   = cloudinary_product_environment.test.id
  enabled    = true

  policy_statement = jsonencode({
    effect   = "allow"
    action   = ["read:*"]
    resource = ["*"]
  })
}
`, envName, policyName)
}
