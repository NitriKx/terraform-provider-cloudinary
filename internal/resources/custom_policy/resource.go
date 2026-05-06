package custom_policy

import (
	"context"
	"fmt"
	"time"

	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/provisioning"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &customPolicyResource{}
var _ resource.ResourceWithImportState = &customPolicyResource{}
var _ resource.ResourceWithConfigValidators = &customPolicyResource{}

// NewResource returns a new cloudinary_custom_policy resource.
func NewResource() resource.Resource {
	return &customPolicyResource{}
}

type customPolicyResource struct {
	client *cloudinary.Cloudinary
}

type customPolicyModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	PolicyStatement types.String `tfsdk:"policy_statement"`
	Description     types.String `tfsdk:"description"`
	ScopeType       types.String `tfsdk:"scope_type"`
	ScopeID         types.String `tfsdk:"scope_id"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

func (r *customPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_policy"
}

func (r *customPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Cloudinary custom permissions policy. " +
			"Custom policies allow fine-grained access control over Cloudinary resources. " +
			"Use jsonencode() to construct the policy_statement value.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the custom policy assigned by Cloudinary.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The display name of the custom policy.",
			},
			"policy_statement": schema.StringAttribute{
				Required:    true,
				Description: "The policy statement JSON string. Use jsonencode() to construct this value.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A human-readable description of the custom policy.",
			},
			"scope_type": schema.StringAttribute{
				Required: true,
				Description: "The scope of the policy. Must be one of: " +
					"\"account\" (applies to the whole account) or " +
					"\"product_environment\" (applies to a specific product environment).",
				Validators: []validator.String{
					stringvalidator.OneOf("account", "product_environment"),
				},
			},
			"scope_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "The ID of the product environment this policy applies to. " +
					"Required when scope_type is \"product_environment\". " +
					"Must be omitted or empty when scope_type is \"account\".",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the custom policy is enabled.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The Unix timestamp at which the custom policy was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The Unix timestamp at which the custom policy was last updated.",
			},
		},
	}
}

// ConfigValidators enforces that scope_id is required when scope_type is "product_environment".
func (r *customPolicyResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		scopeIDValidator{},
	}
}

// scopeIDValidator validates the scope_id / scope_type relationship.
type scopeIDValidator struct{}

func (v scopeIDValidator) Description(_ context.Context) string {
	return "scope_id is required when scope_type is \"product_environment\""
}

func (v scopeIDValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v scopeIDValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var scopeType types.String
	var scopeID types.String

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("scope_type"), &scopeType)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("scope_id"), &scopeID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if scopeType.IsUnknown() || scopeType.IsNull() {
		return
	}

	if scopeType.ValueString() == "product_environment" {
		if scopeID.IsNull() || scopeID.IsUnknown() || scopeID.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("scope_id"),
				"scope_id required",
				"scope_id must be set when scope_type is \"product_environment\".",
			)
		}
	}
}

func (r *customPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The cloudinary_custom_policy resource requires account_id to be set in the provider configuration.",
		)
		return
	}
	r.client = client
}

func (r *customPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := provisioning.CreateCustomPolicyParams{
		Name:            plan.Name.ValueString(),
		PolicyStatement: plan.PolicyStatement.ValueString(),
		ScopeType:       plan.ScopeType.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = plan.Description.ValueString()
	}
	if !plan.ScopeID.IsNull() && !plan.ScopeID.IsUnknown() {
		params.ScopeID = plan.ScopeID.ValueString()
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		params.Enabled = &v
	}

	result, err := r.client.Provisioning.CreateCustomPolicy(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error creating custom policy", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error creating custom policy", result.Error.Message)
		return
	}

	state := policyFromResult(result.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Provisioning.GetCustomPolicy(ctx, provisioning.GetCustomPolicyParams{
		PolicyID: state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error reading custom policy", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.State.RemoveResource(ctx)
		return
	}

	newState := policyFromResult(result.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *customPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state customPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := provisioning.UpdateCustomPolicyParams{
		PolicyID:        state.ID.ValueString(),
		Name:            plan.Name.ValueString(),
		PolicyStatement: plan.PolicyStatement.ValueString(),
		ScopeType:       plan.ScopeType.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = plan.Description.ValueString()
	}
	if !plan.ScopeID.IsNull() && !plan.ScopeID.IsUnknown() {
		params.ScopeID = plan.ScopeID.ValueString()
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		params.Enabled = &v
	}

	result, err := r.client.Provisioning.UpdateCustomPolicy(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error updating custom policy", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error updating custom policy", result.Error.Message)
		return
	}

	newState := policyFromResult(result.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *customPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Provisioning.DeleteCustomPolicy(ctx, provisioning.DeleteCustomPolicyParams{
		PolicyID: state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting custom policy", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error deleting custom policy", result.Error.Message)
		return
	}
}

func (r *customPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &customPolicyModel{
		ID: types.StringValue(req.ID),
	})...)
}

// policyFromResult maps a Cloudinary API result to the Terraform state model.
func policyFromResult(result provisioning.CustomPolicyResult) customPolicyModel {
	return customPolicyModel{
		ID:              types.StringValue(result.ID),
		Name:            types.StringValue(result.Name),
		PolicyStatement: types.StringValue(result.PolicyStatement),
		Description:     types.StringValue(result.Description),
		ScopeType:       types.StringValue(result.ScopeType),
		ScopeID:         types.StringValue(result.ScopeID),
		Enabled:         types.BoolValue(result.Enabled),
		CreatedAt:       types.StringValue(time.Unix(result.CreatedAt, 0).UTC().Format("2006-01-02T15:04:05Z")),
		UpdatedAt:       types.StringValue(time.Unix(result.UpdatedAt, 0).UTC().Format("2006-01-02T15:04:05Z")),
	}
}
