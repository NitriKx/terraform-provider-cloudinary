// Package upload_preset implements the cloudinary_upload_preset resource, which
// manages a Cloudinary upload preset via the Admin API.
package upload_preset

import (
	"context"
	"fmt"
	"slices"
	"strings"

	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"

	"github.com/NitriKx/terraform-provider-cloudinary/internal/providerdata"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &uploadPresetResource{}
var _ resource.ResourceWithImportState = &uploadPresetResource{}

// deliveryTypes are the Cloudinary asset delivery types accepted by this resource.
var deliveryTypes = []string{"upload", "authenticated", "private"}

// NewResource returns a new cloudinary_upload_preset resource.
func NewResource() resource.Resource {
	return &uploadPresetResource{}
}

type uploadPresetResource struct {
	client *cloudinary.Cloudinary
}

func (r *uploadPresetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_upload_preset"
}

func (r *uploadPresetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Cloudinary upload preset (signed server-side upload configuration). " +
			"The preset name is the resource identifier.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The upload preset name (same as name).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the upload preset (e.g. \"file-uploader-invoice\"). Changing it forces a new preset.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"asset_folder": schema.StringAttribute{
				Optional: true,
				Description: "The Cloudinary asset folder assets uploaded with this preset are placed in " +
					"(e.g. \"uaas-v2/file-uploader-preprod-eu/invoice\").",
			},
			"use_asset_folder_as_public_id_prefix": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				Description: "Whether Cloudinary prefixes the generated public_id with asset_folder. " +
					"Defaults to true so minted public_ids start with the asset folder path. " +
					"Relevant only when public_id_prefix/folder is not separately specified.",
			},
			"resource_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("auto"),
				Description: "The resource type Cloudinary applies. Defaults to \"auto\".",
			},
			"unsigned": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the preset allows unsigned uploads. Defaults to false (signed uploads).",
			},
			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("upload"),
				Description: "The delivery type controlling asset access: \"upload\" (public URL), " +
					"\"authenticated\" (token-signed delivery), or \"private\". Defaults to \"upload\".",
				Validators: []validator.String{
					oneOfStringValidator{allowed: deliveryTypes},
				},
			},
			"allowed_formats": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The file formats Cloudinary accepts at upload time (e.g. [\"pdf\", \"zip\"]).",
			},
			"moderation": schema.StringAttribute{
				Optional: true,
				Description: "The moderation mode applied to uploads (e.g. \"manual\"). " +
					"For user-origin presets this is usually set dynamically by eval rather than statically.",
			},
			"eval": schema.StringAttribute{
				Optional: true,
				Description: "A server-side JavaScript snippet run by Cloudinary at upload time " +
					"(e.g. to set moderation per file based on format).",
			},
		},
	}
}

func (r *uploadPresetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*providerdata.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *providerdata.ProviderData, got: %T", req.ProviderData),
		)
		return
	}
	r.client = pd.Client
}

func (r *uploadPresetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan uploadPresetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params, diags := uploadParamsFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Admin.CreateUploadPreset(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error creating upload preset", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error creating upload preset", result.Error.Message)
		return
	}

	// The create response only echoes the name, so the plan (with defaults
	// resolved) is the source of truth for the new state.
	plan.ID = types.StringValue(plan.Name.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *uploadPresetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state uploadPresetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}

	result, err := r.client.Admin.GetUploadPreset(ctx, admin.GetUploadPresetParams{Name: name})
	if err != nil {
		resp.Diagnostics.AddError("Error reading upload preset", err.Error())
		return
	}
	if result.Error.Message != "" {
		// Preset was deleted outside of Terraform.
		if isNotFound(result.Error.Message) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Cloudinary API error reading upload preset", result.Error.Message)
		return
	}

	newState, diags := flattenPreset(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *uploadPresetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan uploadPresetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createParams, diags := uploadParamsFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// UpdateUploadPresetParams shares the same shape; reuse the mapped values.
	updateParams := admin.UpdateUploadPresetParams{
		Name:         createParams.Name,
		Unsigned:     createParams.Unsigned,
		UploadParams: createParams.UploadParams,
	}

	result, err := r.client.Admin.UpdateUploadPreset(ctx, updateParams)
	if err != nil {
		resp.Diagnostics.AddError("Error updating upload preset", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error updating upload preset", result.Error.Message)
		return
	}

	plan.ID = types.StringValue(plan.Name.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *uploadPresetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state uploadPresetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Admin.DeleteUploadPreset(ctx, admin.DeleteUploadPresetParams{Name: state.Name.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting upload preset", err.Error())
		return
	}
	if result.Error.Message != "" && !isNotFound(result.Error.Message) {
		resp.Diagnostics.AddError("Cloudinary API error deleting upload preset", result.Error.Message)
		return
	}
}

func (r *uploadPresetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the preset name. Set only id and let Read populate the
	// rest (Read falls back to id when name is unset), which keeps every
	// attribute correctly typed.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// isNotFound reports whether a Cloudinary error message indicates the preset
// does not exist.
func isNotFound(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "not found") || strings.Contains(m, "can't find") || strings.Contains(m, "doesn't exist")
}

// oneOfStringValidator validates that a string attribute is one of a fixed set.
type oneOfStringValidator struct {
	allowed []string
}

func (v oneOfStringValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be one of: %s", strings.Join(v.allowed, ", "))
}

func (v oneOfStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v oneOfStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	if !slices.Contains(v.allowed, val) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value",
			fmt.Sprintf("%q is not valid; must be one of: %s", val, strings.Join(v.allowed, ", ")),
		)
	}
}
