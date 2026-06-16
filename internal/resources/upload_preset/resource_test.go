package upload_preset_test

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

const testPresetName = "tf-acc-upload-preset"

func TestAccUploadPresetResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUploadPresetConfig(testPresetName, "authenticated", `["pdf", "zip"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudinary_upload_preset.test", "name", testPresetName),
					resource.TestCheckResourceAttr("cloudinary_upload_preset.test", "id", testPresetName),
					resource.TestCheckResourceAttr("cloudinary_upload_preset.test", "type", "authenticated"),
					// The prefix flag defaults on.
					resource.TestCheckResourceAttr("cloudinary_upload_preset.test", "use_asset_folder_as_public_id_prefix", "true"),
					resource.TestCheckResourceAttr("cloudinary_upload_preset.test", "unsigned", "false"),
					resource.TestCheckResourceAttr("cloudinary_upload_preset.test", "resource_type", "auto"),
					resource.TestCheckResourceAttr("cloudinary_upload_preset.test", "allowed_formats.#", "2"),
				),
			},
			{
				ResourceName:      "cloudinary_upload_preset.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// Changing a mutable attribute must update the preset in place (no replacement).
func TestAccUploadPresetResource_updateInPlace(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUploadPresetConfig(testPresetName, "authenticated", `["pdf", "zip"]`),
				Check:  resource.TestCheckResourceAttr("cloudinary_upload_preset.test", "allowed_formats.#", "2"),
			},
			{
				Config: testAccUploadPresetConfig(testPresetName, "authenticated", `["pdf"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudinary_upload_preset.test", "allowed_formats.#", "1"),
					resource.TestCheckResourceAttr("cloudinary_upload_preset.test", "allowed_formats.0", "pdf"),
				),
			},
		},
	})
}

func testAccUploadPresetConfig(name, deliveryType, allowedFormats string) string {
	return fmt.Sprintf(`
resource "cloudinary_upload_preset" "test" {
  name            = %q
  asset_folder    = "uaas-v2/terraform-provider-test/invoice"
  type            = %q
  allowed_formats = %s
  eval            = "upload_options.moderation = 'manual';"
  moderation      = "manual"
}
`, name, deliveryType, allowedFormats)
}
