package folder_test

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
	if v := os.Getenv("CLOUDINARY_URL"); v == "" {
		if os.Getenv("CLOUDINARY_CLOUD_NAME") == "" || os.Getenv("CLOUDINARY_API_KEY") == "" || os.Getenv("CLOUDINARY_API_SECRET") == "" {
			t.Skip("CLOUDINARY_URL or (CLOUDINARY_CLOUD_NAME + CLOUDINARY_API_KEY + CLOUDINARY_API_SECRET) must be set for acceptance tests")
		}
	}
}

func TestAccFolderResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFolderConfig("tf-acc-test-folder"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudinary_folder.test", "path", "tf-acc-test-folder"),
					resource.TestCheckResourceAttr("cloudinary_folder.test", "name", "tf-acc-test-folder"),
				),
			},
			{
				ResourceName:      "cloudinary_folder.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFolderResource_nested(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFolderNestedConfig("tf-acc-test-parent", "tf-acc-test-child"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudinary_folder.parent", "path", "tf-acc-test-parent"),
					resource.TestCheckResourceAttr("cloudinary_folder.child", "path", "tf-acc-test-parent/tf-acc-test-child"),
					resource.TestCheckResourceAttr("cloudinary_folder.child", "name", "tf-acc-test-child"),
				),
			},
		},
	})
}

func testAccFolderConfig(name string) string {
	return fmt.Sprintf(`
resource "cloudinary_folder" "test" {
  path = %q
}
`, name)
}

func testAccFolderNestedConfig(parent, child string) string {
	return fmt.Sprintf(`
resource "cloudinary_folder" "parent" {
  path = %q
}

resource "cloudinary_folder" "child" {
  path = "${cloudinary_folder.parent.path}/%s"
}
`, parent, child)
}
