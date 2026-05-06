package product_environment

import (
	"context"
	"fmt"

	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/provisioning"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &productEnvironmentResource{}
var _ resource.ResourceWithImportState = &productEnvironmentResource{}

// NewResource returns a new cloudinary_product_environment resource.
func NewResource() resource.Resource {
	return &productEnvironmentResource{}
}

type productEnvironmentResource struct {
	client *cloudinary.Cloudinary
}

type productEnvironmentModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	CloudName          types.String `tfsdk:"cloud_name"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	CustomAttributes   types.Map    `tfsdk:"custom_attributes"`
	BaseSubAccountID   types.String `tfsdk:"base_sub_account_id"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (r *productEnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product_environment"
}

func (r *productEnvironmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Cloudinary product environment (sub-account). " +
			"Product environments allow you to segment your Cloudinary usage for different projects or teams.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the product environment assigned by Cloudinary.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The display name of the product environment.",
			},
			"cloud_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The Cloudinary cloud name for this product environment. Auto-generated if not provided.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the product environment is enabled.",
			},
			"custom_attributes": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A map of custom key-value attributes to associate with the product environment.",
			},
			"base_sub_account_id": schema.StringAttribute{
				Optional: true,
				Description: "The ID of an existing product environment to copy settings from when creating this one. " +
					"Changing this value forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time at which the product environment was created (RFC3339).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time at which the product environment was last updated (RFC3339).",
			},
		},
	}
}

func (r *productEnvironmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The cloudinary_product_environment resource requires account_id to be set in the provider configuration.",
		)
		return
	}
	r.client = client
}

func (r *productEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan productEnvironmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := provisioning.CreateProductEnvironmentParams{
		Name: plan.Name.ValueString(),
	}
	if !plan.CloudName.IsNull() && !plan.CloudName.IsUnknown() {
		params.CloudName = plan.CloudName.ValueString()
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		params.Enabled = &v
	}
	if !plan.BaseSubAccountID.IsNull() && !plan.BaseSubAccountID.IsUnknown() {
		params.BaseSubAccountID = plan.BaseSubAccountID.ValueString()
	}
	if !plan.CustomAttributes.IsNull() && !plan.CustomAttributes.IsUnknown() {
		attrs := make(map[string]string)
		resp.Diagnostics.Append(plan.CustomAttributes.ElementsAs(ctx, &attrs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params.CustomAttributes = attrs
	}

	result, err := r.client.Provisioning.CreateProductEnvironment(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error creating product environment", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error creating product environment", result.Error.Message)
		return
	}

	state, diags := productEnvironmentFromResult(&result.ProductEnvironmentResult, plan.BaseSubAccountID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *productEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state productEnvironmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Provisioning.GetProductEnvironment(ctx, provisioning.GetProductEnvironmentParams{
		SubAccountID: state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error reading product environment", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.State.RemoveResource(ctx)
		return
	}

	newState, diags := productEnvironmentFromResult(&result.ProductEnvironmentResult, state.BaseSubAccountID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *productEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan productEnvironmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state productEnvironmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := provisioning.UpdateProductEnvironmentParams{
		SubAccountID: state.ID.ValueString(),
		Name:         plan.Name.ValueString(),
	}
	if !plan.CloudName.IsNull() && !plan.CloudName.IsUnknown() {
		params.CloudName = plan.CloudName.ValueString()
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		params.Enabled = &v
	}
	if !plan.CustomAttributes.IsNull() && !plan.CustomAttributes.IsUnknown() {
		attrs := make(map[string]string)
		resp.Diagnostics.Append(plan.CustomAttributes.ElementsAs(ctx, &attrs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params.CustomAttributes = attrs
	}

	result, err := r.client.Provisioning.UpdateProductEnvironment(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error updating product environment", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error updating product environment", result.Error.Message)
		return
	}

	newState, diags := productEnvironmentFromResult(&result.ProductEnvironmentResult, plan.BaseSubAccountID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *productEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state productEnvironmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Provisioning.DeleteProductEnvironment(ctx, provisioning.DeleteProductEnvironmentParams{
		SubAccountID: state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting product environment", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error deleting product environment", result.Error.Message)
		return
	}
}

func (r *productEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &productEnvironmentModel{
		ID: types.StringValue(req.ID),
	})...)
}

// productEnvironmentFromResult maps a Cloudinary API result to the Terraform state model.
func productEnvironmentFromResult(result *provisioning.ProductEnvironmentResult, baseSubAccountID types.String) (productEnvironmentModel, diag.Diagnostics) {
	// Build custom_attributes map.
	customAttrs, diags := types.MapValueFrom(context.Background(), types.StringType, result.CustomAttributes)

	state := productEnvironmentModel{
		ID:               types.StringValue(result.ID),
		Name:             types.StringValue(result.Name),
		CloudName:        types.StringValue(result.CloudName),
		Enabled:          types.BoolValue(result.Enabled),
		CustomAttributes: customAttrs,
		BaseSubAccountID: baseSubAccountID,
		CreatedAt:        types.StringValue(result.CreatedAt.Format("2006-01-02T15:04:05Z07:00")),
		UpdatedAt:        types.StringValue(result.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")),
	}
	return state, diags
}
