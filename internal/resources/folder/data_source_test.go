package folder_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFolderDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFolderDataSourceConfig("tf-acc-test-ds-folder"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudinary_folder.test", "path", "tf-acc-test-ds-folder"),
					resource.TestCheckResourceAttr("data.cloudinary_folder.test", "name", "tf-acc-test-ds-folder"),
					resource.TestCheckResourceAttrSet("data.cloudinary_folder.test", "id"),
				),
			},
		},
	})
}

func testAccFolderDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "cloudinary_folder" "test" {
  path = %q
}

data "cloudinary_folder" "test" {
  path = cloudinary_folder.test.path
}
`, name)
}
