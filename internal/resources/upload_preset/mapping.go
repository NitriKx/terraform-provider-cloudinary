package upload_preset

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// uploadPresetModel is the Terraform state/plan model for cloudinary_upload_preset.
type uploadPresetModel struct {
	ID                             types.String `tfsdk:"id"`
	Name                           types.String `tfsdk:"name"`
	AssetFolder                    types.String `tfsdk:"asset_folder"`
	UseAssetFolderAsPublicIDPrefix types.Bool   `tfsdk:"use_asset_folder_as_public_id_prefix"`
	ResourceType                   types.String `tfsdk:"resource_type"`
	Unsigned                       types.Bool   `tfsdk:"unsigned"`
	Type                           types.String `tfsdk:"type"`
	AllowedFormats                 types.List   `tfsdk:"allowed_formats"`
	Moderation                     types.String `tfsdk:"moderation"`
	Eval                           types.String `tfsdk:"eval"`
}

// uploadParamsFromModel translates the Terraform model into Cloudinary
// CreateUploadPresetParams. It is the single source of the request shape, kept
// as a pure function so it can be unit-tested without a live API.
func uploadParamsFromModel(ctx context.Context, m uploadPresetModel) (admin.CreateUploadPresetParams, diag.Diagnostics) {
	var diags diag.Diagnostics

	params := admin.CreateUploadPresetParams{
		Name: m.Name.ValueString(),
	}

	// Unsigned and the public-id-prefix flag are *bool on the SDK: set the
	// pointer only when the attribute is known, so unset never clobbers.
	if !m.Unsigned.IsNull() && !m.Unsigned.IsUnknown() {
		params.Unsigned = m.Unsigned.ValueBoolPointer()
	}
	if !m.UseAssetFolderAsPublicIDPrefix.IsNull() && !m.UseAssetFolderAsPublicIDPrefix.IsUnknown() {
		params.UseAssetFolderAsPublicIDPrefix = m.UseAssetFolderAsPublicIDPrefix.ValueBoolPointer()
	}

	// String fields carry omitempty in the SDK, so a null/empty value is simply
	// dropped from the request rather than overwriting a Cloudinary default.
	params.AssetFolder = m.AssetFolder.ValueString()
	params.ResourceType = m.ResourceType.ValueString()
	params.Type = api.DeliveryType(m.Type.ValueString())
	params.Moderation = m.Moderation.ValueString()
	params.Eval = m.Eval.ValueString()

	if !m.AllowedFormats.IsNull() && !m.AllowedFormats.IsUnknown() {
		var formats []string
		diags.Append(m.AllowedFormats.ElementsAs(ctx, &formats, false)...)
		if diags.HasError() {
			return params, diags
		}
		params.AllowedFormats = api.CldAPIArray(formats)
	}

	return params, diags
}

// flattenPreset parses a GET /upload_presets response into the Terraform model
// so Read can reconcile out-of-band drift.
func flattenPreset(ctx context.Context, res *admin.GetUploadPresetResult) (uploadPresetModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	m := uploadPresetModel{
		ID:       types.StringValue(res.Name),
		Name:     types.StringValue(res.Name),
		Unsigned: types.BoolValue(res.Unsigned),
	}

	s, err := decodeSettings(res.Settings)
	if err != nil {
		diags.AddError("Error parsing upload preset settings", err.Error())
		return m, diags
	}

	m.AssetFolder = types.StringValue(s.AssetFolder)
	// Cloudinary accepts resource_type on create but does not return it in the
	// preset settings, so reconcile an absent value to the "auto" default rather
	// than reporting empty (which would drift against the schema default).
	if s.ResourceType == "" {
		m.ResourceType = types.StringValue("auto")
	} else {
		m.ResourceType = types.StringValue(s.ResourceType)
	}
	m.Type = types.StringValue(s.Type)
	m.Moderation = optionalString(s.Moderation)
	m.Eval = optionalString(s.Eval)

	if s.UseAssetFolderAsPublicIDPrefix != nil {
		m.UseAssetFolderAsPublicIDPrefix = types.BoolValue(*s.UseAssetFolderAsPublicIDPrefix)
	} else {
		m.UseAssetFolderAsPublicIDPrefix = types.BoolValue(false)
	}

	formats := s.allowedFormatsSlice()
	if len(formats) == 0 {
		m.AllowedFormats = types.ListNull(types.StringType)
	} else {
		lv, d := types.ListValueFrom(ctx, types.StringType, formats)
		diags.Append(d...)
		m.AllowedFormats = lv
	}

	return m, diags
}

// presetSettings mirrors the nested "settings" object of a GET /upload_presets
// response. allowed_formats is left as interface{} because Cloudinary may return
// it as a JSON array or as a comma-separated string.
type presetSettings struct {
	AssetFolder                    string      `json:"asset_folder"`
	UseAssetFolderAsPublicIDPrefix *bool       `json:"use_asset_folder_as_public_id_prefix"`
	ResourceType                   string      `json:"resource_type"`
	Type                           string      `json:"type"`
	AllowedFormats                 interface{} `json:"allowed_formats"`
	Eval                           string      `json:"eval"`
	Moderation                     string      `json:"moderation"`
}

// decodeSettings re-marshals the untyped settings blob through JSON into the
// typed presetSettings struct.
func decodeSettings(raw interface{}) (presetSettings, error) {
	var s presetSettings
	if raw == nil {
		return s, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return s, fmt.Errorf("marshal settings: %w", err)
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("unmarshal settings: %w", err)
	}
	return s, nil
}

// allowedFormatsSlice normalises allowed_formats from either a JSON array or a
// comma-separated string into a string slice.
func (s presetSettings) allowedFormatsSlice() []string {
	switch v := s.AllowedFormats.(type) {
	case string:
		if v == "" {
			return nil
		}
		return strings.Split(v, ",")
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, e := range v {
			if str, ok := e.(string); ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}

// optionalString maps an empty API string to a null attribute so an unset
// configuration value does not register as drift.
func optionalString(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}
