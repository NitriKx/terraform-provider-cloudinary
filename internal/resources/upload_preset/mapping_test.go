package upload_preset

import (
	"context"
	"testing"

	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUploadParamsFromModel_fullyPopulated(t *testing.T) {
	ctx := context.Background()
	m := uploadPresetModel{
		Name:                           types.StringValue("file-uploader-invoice"),
		AssetFolder:                    types.StringValue("uaas-v2/file-uploader-preprod-eu/invoice"),
		UseAssetFolderAsPublicIDPrefix: types.BoolValue(true),
		ResourceType:                   types.StringValue("auto"),
		Unsigned:                       types.BoolValue(false),
		Type:                           types.StringValue("authenticated"),
		AllowedFormats: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("pdf"),
			types.StringValue("zip"),
		}),
		Moderation: types.StringValue("manual"),
		Eval:       types.StringValue("upload_options.moderation = 'manual';"),
	}

	params, diags := uploadParamsFromModel(ctx, m)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if params.Name != "file-uploader-invoice" {
		t.Errorf("Name = %q, want %q", params.Name, "file-uploader-invoice")
	}
	if params.AssetFolder != "uaas-v2/file-uploader-preprod-eu/invoice" {
		t.Errorf("AssetFolder = %q, want %q", params.AssetFolder, "uaas-v2/file-uploader-preprod-eu/invoice")
	}
	if params.UseAssetFolderAsPublicIDPrefix == nil || !*params.UseAssetFolderAsPublicIDPrefix {
		t.Errorf("UseAssetFolderAsPublicIDPrefix = %v, want pointer to true", params.UseAssetFolderAsPublicIDPrefix)
	}
	if params.ResourceType != "auto" {
		t.Errorf("ResourceType = %q, want %q", params.ResourceType, "auto")
	}
	if params.Unsigned == nil || *params.Unsigned {
		t.Errorf("Unsigned = %v, want pointer to false", params.Unsigned)
	}
	if string(params.Type) != "authenticated" {
		t.Errorf("Type = %q, want %q", params.Type, "authenticated")
	}
	if params.Moderation != "manual" {
		t.Errorf("Moderation = %q, want %q", params.Moderation, "manual")
	}
	if params.Eval != "upload_options.moderation = 'manual';" {
		t.Errorf("Eval = %q, want the eval snippet", params.Eval)
	}
	if len(params.AllowedFormats) != 2 || params.AllowedFormats[0] != "pdf" || params.AllowedFormats[1] != "zip" {
		t.Errorf("AllowedFormats = %v, want [pdf zip]", params.AllowedFormats)
	}
}

// When optional attributes are null, they must be omitted (left zero) rather
// than sent as empty values that would clobber Cloudinary defaults.
func TestUploadParamsFromModel_optionalNull(t *testing.T) {
	ctx := context.Background()
	m := uploadPresetModel{
		Name:                           types.StringValue("file-uploader-reviews"),
		AssetFolder:                    types.StringValue("uaas-v2/file-uploader-preprod-eu/reviews"),
		UseAssetFolderAsPublicIDPrefix: types.BoolValue(true),
		ResourceType:                   types.StringValue("auto"),
		Unsigned:                       types.BoolValue(false),
		Type:                           types.StringValue("upload"),
		AllowedFormats:                 types.ListNull(types.StringType),
		Moderation:                     types.StringNull(),
		Eval:                           types.StringNull(),
	}

	params, diags := uploadParamsFromModel(ctx, m)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if params.Eval != "" {
		t.Errorf("Eval = %q, want empty when null", params.Eval)
	}
	if params.Moderation != "" {
		t.Errorf("Moderation = %q, want empty when null", params.Moderation)
	}
	if params.AllowedFormats != nil {
		t.Errorf("AllowedFormats = %v, want nil when null", params.AllowedFormats)
	}
}

// flattenPreset must parse the GET /upload_presets settings blob back into the
// model so Read can detect out-of-band drift.
func TestFlattenPreset_arrayAllowedFormats(t *testing.T) {
	ctx := context.Background()
	res := &admin.GetUploadPresetResult{
		Name:     "file-uploader-invoice",
		Unsigned: false,
		Settings: map[string]interface{}{
			"asset_folder":                         "uaas-v2/file-uploader-preprod-eu/invoice",
			"use_asset_folder_as_public_id_prefix": true,
			"resource_type":                        "auto",
			"type":                                 "authenticated",
			"allowed_formats":                      []interface{}{"pdf", "zip"},
			"eval":                                 "upload_options.moderation = 'manual';",
			"moderation":                           "manual",
		},
	}

	m, diags := flattenPreset(ctx, res)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if m.ID.ValueString() != "file-uploader-invoice" || m.Name.ValueString() != "file-uploader-invoice" {
		t.Errorf("ID/Name = %q/%q, want preset name", m.ID.ValueString(), m.Name.ValueString())
	}
	if m.AssetFolder.ValueString() != "uaas-v2/file-uploader-preprod-eu/invoice" {
		t.Errorf("AssetFolder = %q", m.AssetFolder.ValueString())
	}
	if !m.UseAssetFolderAsPublicIDPrefix.ValueBool() {
		t.Errorf("UseAssetFolderAsPublicIDPrefix = false, want true")
	}
	if m.ResourceType.ValueString() != "auto" {
		t.Errorf("ResourceType = %q, want auto", m.ResourceType.ValueString())
	}
	if m.Type.ValueString() != "authenticated" {
		t.Errorf("Type = %q, want authenticated", m.Type.ValueString())
	}
	if m.Moderation.ValueString() != "manual" {
		t.Errorf("Moderation = %q, want manual", m.Moderation.ValueString())
	}
	if m.Eval.ValueString() != "upload_options.moderation = 'manual';" {
		t.Errorf("Eval = %q", m.Eval.ValueString())
	}
	var formats []string
	m.AllowedFormats.ElementsAs(ctx, &formats, false)
	if len(formats) != 2 || formats[0] != "pdf" || formats[1] != "zip" {
		t.Errorf("AllowedFormats = %v, want [pdf zip]", formats)
	}
}

// Cloudinary may return allowed_formats as a comma-separated string instead of
// an array; flattenPreset must accept both.
func TestFlattenPreset_commaStringAllowedFormats(t *testing.T) {
	ctx := context.Background()
	res := &admin.GetUploadPresetResult{
		Name: "file-uploader-invoice",
		Settings: map[string]interface{}{
			"asset_folder":    "uaas-v2/file-uploader-preprod-eu/invoice",
			"allowed_formats": "pdf,zip",
		},
	}

	m, diags := flattenPreset(ctx, res)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	var formats []string
	m.AllowedFormats.ElementsAs(ctx, &formats, false)
	if len(formats) != 2 || formats[0] != "pdf" || formats[1] != "zip" {
		t.Errorf("AllowedFormats = %v, want [pdf zip]", formats)
	}
}

// Cloudinary accepts resource_type on create but does not echo it back in the
// preset settings (verified against preprod). flattenPreset must reconcile an
// absent resource_type to the "auto" default rather than clobbering it to empty.
func TestFlattenPreset_absentResourceTypeDefaultsToAuto(t *testing.T) {
	ctx := context.Background()
	res := &admin.GetUploadPresetResult{
		Name: "file-uploader-invoice",
		Settings: map[string]interface{}{
			"asset_folder": "uaas-v2/file-uploader-preprod-eu/invoice",
			"type":         "upload",
		},
	}

	m, diags := flattenPreset(ctx, res)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if m.ResourceType.ValueString() != "auto" {
		t.Errorf("ResourceType = %q, want auto when absent from settings", m.ResourceType.ValueString())
	}
}

// Optional string fields absent from settings must be null, so they compare
// equal to an unset configuration (no spurious drift).
func TestFlattenPreset_absentOptionalsAreNull(t *testing.T) {
	ctx := context.Background()
	res := &admin.GetUploadPresetResult{
		Name: "file-uploader-reviews",
		Settings: map[string]interface{}{
			"asset_folder": "uaas-v2/file-uploader-preprod-eu/reviews",
			"type":         "upload",
		},
	}

	m, diags := flattenPreset(ctx, res)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !m.Eval.IsNull() {
		t.Errorf("Eval = %q, want null", m.Eval.ValueString())
	}
	if !m.Moderation.IsNull() {
		t.Errorf("Moderation = %q, want null", m.Moderation.ValueString())
	}
	if !m.AllowedFormats.IsNull() {
		t.Errorf("AllowedFormats = %v, want null", m.AllowedFormats)
	}
}
