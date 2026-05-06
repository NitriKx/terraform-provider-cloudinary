package product_environment_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProductEnvironmentDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProductEnvironmentDataSourceConfig("tf-acc-test-ds-env"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudinary_product_environment.test", "id"),
					resource.TestCheckResourceAttr("data.cloudinary_product_environment.test", "name", "tf-acc-test-ds-env"),
					resource.TestCheckResourceAttr("data.cloudinary_product_environment.test", "enabled", "true"),
				),
			},
		},
	})
}

func testAccProductEnvironmentDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "cloudinary_product_environment" "test" {
  name    = %q
  enabled = true
}

data "cloudinary_product_environment" "test" {
  id = cloudinary_product_environment.test.id
}
`, name)
}
