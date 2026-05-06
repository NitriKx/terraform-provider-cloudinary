package product_environment_access_key

import (
	"context"
	"fmt"
	"strings"

	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/provisioning"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &accessKeyResource{}
var _ resource.ResourceWithImportState = &accessKeyResource{}

// NewResource returns a new cloudinary_product_environment_access_key resource.
func NewResource() resource.Resource {
	return &accessKeyResource{}
}

type accessKeyResource struct {
	client *cloudinary.Cloudinary
}

type accessKeyModel struct {
	ProductEnvironmentID types.String `tfsdk:"product_environment_id"`
	Name                 types.String `tfsdk:"name"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	APIKey               types.String `tfsdk:"api_key"`
	APISecret            types.String `tfsdk:"api_secret"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

func (r *accessKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product_environment_access_key"
}

func (r *accessKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an API access key for a Cloudinary product environment. " +
			"Access keys provide the credentials needed to interact with a product environment's API.",
		Attributes: map[string]schema.Attribute{
			"product_environment_id": schema.StringAttribute{
				Required: true,
				Description: "The ID of the product environment this access key belongs to. " +
					"Changing this value forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A display name for the access key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the access key is enabled.",
			},
			"api_key": schema.StringAttribute{
				Computed:    true,
				Description: "The API key identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"api_secret": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The API secret. Only available immediately after creation. Preserved in state.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time at which the access key was created (RFC3339).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time at which the access key was last updated (RFC3339).",
			},
		},
	}
}

func (r *accessKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*cloudinary.Cloudinary)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *cloudinary.Cloudinary, got: %T", req.ProviderData),
		)
		return
	}
	if client.Config.Cloud.AccountID == "" {
		resp.Diagnostics.AddError(
			"Missing account_id",
			"The cloudinary_product_environment_access_key resource requires account_id to be set in the provider configuration.",
		)
		return
	}
	r.client = client
}

func (r *accessKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accessKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := provisioning.CreateAccessKeyParams{
		SubAccountID: plan.ProductEnvironmentID.ValueString(),
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		params.Name = plan.Name.ValueString()
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		params.Enabled = &v
	}

	result, err := r.client.Provisioning.CreateAccessKey(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error creating access key", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error creating access key", result.Error.Message)
		return
	}

	state := accessKeyModel{
		ProductEnvironmentID: plan.ProductEnvironmentID,
		Name:                 types.StringValue(result.Name),
		Enabled:              types.BoolValue(result.Enabled),
		APIKey:               types.StringValue(result.APIKey),
		APISecret:            types.StringValue(result.APISecret),
		CreatedAt:            types.StringValue(result.CreatedAt.Format("2006-01-02T15:04:05Z07:00")),
		UpdatedAt:            types.StringValue(result.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *accessKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accessKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Provisioning.GetAccessKey(ctx, provisioning.GetAccessKeyParams{
		SubAccountID: state.ProductEnvironmentID.ValueString(),
		APIKey:       state.APIKey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error reading access key", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.State.RemoveResource(ctx)
		return
	}

	// api_secret is not returned on reads; preserve the value from state.
	state.Name = types.StringValue(result.Name)
	state.Enabled = types.BoolValue(result.Enabled)
	state.APIKey = types.StringValue(result.APIKey)
	state.CreatedAt = types.StringValue(result.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	state.UpdatedAt = types.StringValue(result.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	// api_secret stays as-is from state (UseStateForUnknown plan modifier ensures this).

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *accessKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan accessKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state accessKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := provisioning.UpdateAccessKeyParams{
		SubAccountID: state.ProductEnvironmentID.ValueString(),
		APIKey:       state.APIKey.ValueString(),
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		params.Name = plan.Name.ValueString()
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		params.Enabled = &v
	}

	result, err := r.client.Provisioning.UpdateAccessKey(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error updating access key", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error updating access key", result.Error.Message)
		return
	}

	// Preserve api_secret from state since it's not returned on updates.
	state.Name = types.StringValue(result.Name)
	state.Enabled = types.BoolValue(result.Enabled)
	state.APIKey = types.StringValue(result.APIKey)
	state.UpdatedAt = types.StringValue(result.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *accessKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accessKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Provisioning.DeleteAccessKey(ctx, provisioning.DeleteAccessKeyParams{
		SubAccountID: state.ProductEnvironmentID.ValueString(),
		APIKey:       state.APIKey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting access key", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error deleting access key", result.Error.Message)
		return
	}
}

func (r *accessKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "{product_environment_id}/{api_key}"
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf(
				"Expected format: {product_environment_id}/{api_key}. Got: %q",
				req.ID,
			),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &accessKeyModel{
		ProductEnvironmentID: types.StringValue(parts[0]),
		APIKey:               types.StringValue(parts[1]),
	})...)
}
