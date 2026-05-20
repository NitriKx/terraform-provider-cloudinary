package trigger

import (
	"context"
	"encoding/json"
	"fmt"

	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"

	"github.com/NitriKx/terraform-provider-cloudinary/internal/providerdata"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &triggerResource{}
var _ resource.ResourceWithImportState = &triggerResource{}

// NewResource returns a new cloudinary_trigger resource.
func NewResource() resource.Resource {
	return &triggerResource{}
}

type triggerResource struct {
	client *cloudinary.Cloudinary
}

type triggerModel struct {
	ID                   types.String `tfsdk:"id"`
	URI                  types.String `tfsdk:"uri"`
	EventType            types.String `tfsdk:"event_type"`
	Additive             types.Bool   `tfsdk:"additive"`
	Filter               types.String `tfsdk:"filter"`
	PayloadTemplate      types.String `tfsdk:"payload_template"`
	AuthScheme           types.String `tfsdk:"auth_scheme"`
	ProductEnvironmentID types.String `tfsdk:"product_environment_id"`
	URIType              types.String `tfsdk:"uri_type"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

var validEventTypes = []string{
	"upload", "delete", "rename", "move", "eager", "explode", "multi",
	"resource_tags_changed", "resource_context_changed", "resource_metadata_changed",
	"resource_display_name_changed", "access_control_changed", "related_assets",
	"create_folder", "delete_folder", "move_or_rename_asset_folder",
	"proof_status_changed", "error", "all",
}

var validAuthSchemes = []string{"default", "legacy_hmac", "eddsa_v2"}

func (r *triggerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trigger"
}

func (r *triggerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Cloudinary webhook notification trigger. " +
			"Triggers fire on specific asset events and POST a payload to a configured URL. " +
			"Up to 30 triggers per product environment (the console UI shows only 10).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the trigger assigned by Cloudinary.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uri": schema.StringAttribute{
				Required:    true,
				Description: "The webhook URL to which Cloudinary sends the notification payload.",
			},
			"event_type": schema.StringAttribute{
				Required: true,
				Description: "The event that fires the trigger. One of: " +
					"upload, delete, rename, move, eager, explode, multi, " +
					"resource_tags_changed, resource_context_changed, resource_metadata_changed, " +
					"resource_display_name_changed, access_control_changed, related_assets, " +
					"create_folder, delete_folder, move_or_rename_asset_folder, " +
					"proof_status_changed, error, all.",
				Validators: []validator.String{
					oneOfStringValidator{values: validEventTypes, fieldName: "event_type"},
				},
			},
			"additive": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "When true, this global trigger fires in addition to any per-asset notification_url. Default false.",
			},
			"filter": schema.StringAttribute{
				Optional: true,
				Description: "A JSONLogic expression (JSON-encoded) that filters which events fire this trigger. " +
					"When set, filter_language is automatically sent as \"jsonlogic\". Example: " +
					`"{\"==\":[{\"var\":\"resource_type\"},\"image\"]}"`,
				Validators: []validator.String{jsonObjectValidator{fieldName: "filter"}},
			},
			"payload_template": schema.StringAttribute{
				Optional: true,
				Description: "A Mustache template object (JSON-encoded, max 16KB) that customises the " +
					"notification payload. Available namespaces: event, asset, folder, eager, error, raw.",
				Validators: []validator.String{
					jsonObjectValidator{fieldName: "payload_template"},
					maxBytesValidator{max: 16 * 1024},
				},
			},
			"auth_scheme": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("default"),
				Description: "Signature method for verifying incoming webhook payloads. One of: default, legacy_hmac, eddsa_v2.",
				Validators: []validator.String{
					oneOfStringValidator{values: validAuthSchemes, fieldName: "auth_scheme"},
				},
			},
			"product_environment_id": schema.StringAttribute{
				Computed:    true,
				Description: "The product environment this trigger is scoped to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uri_type": schema.StringAttribute{
				Computed:    true,
				Description: "The type of the URI. Always \"webhook\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp of when the trigger was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp of when the trigger was last updated.",
			},
		},
	}
}

func (r *triggerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *triggerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan triggerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := admin.CreateTriggerParams{
		URI:        plan.URI.ValueString(),
		EventType:  plan.EventType.ValueString(),
		Additive:   plan.Additive.ValueBool(),
		AuthScheme: plan.AuthScheme.ValueString(),
	}
	if !plan.Filter.IsNull() && !plan.Filter.IsUnknown() {
		m, err := jsonStringToMap(plan.Filter.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid filter JSON", err.Error())
			return
		}
		params.Filter = m
		params.FilterLanguage = "jsonlogic"
	}
	if !plan.PayloadTemplate.IsNull() && !plan.PayloadTemplate.IsUnknown() {
		m, err := jsonStringToMap(plan.PayloadTemplate.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid payload_template JSON", err.Error())
			return
		}
		params.PayloadTemplate = m
	}

	result, err := r.client.Admin.CreateTrigger(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error creating trigger", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error creating trigger", result.Error.Message)
		return
	}

	state := triggerToModel(result.Trigger, plan.Filter, plan.PayloadTemplate)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *triggerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state triggerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Admin.GetTrigger(ctx, admin.GetTriggerParams{
		TriggerID: state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error reading trigger", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Error reading trigger", result.Error.Message)
		return
	}
	if result.ID == "" {
		// Not found — deleted outside of Terraform.
		resp.State.RemoveResource(ctx)
		return
	}

	newState := triggerToModel(result.Trigger, state.Filter, state.PayloadTemplate)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *triggerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan triggerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state triggerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	additive := plan.Additive.ValueBool()
	params := admin.UpdateTriggerParams{
		TriggerID:  state.ID.ValueString(),
		URI:        plan.URI.ValueString(),
		EventType:  plan.EventType.ValueString(),
		Additive:   &additive,
		AuthScheme: plan.AuthScheme.ValueString(),
	}
	if !plan.Filter.IsNull() && !plan.Filter.IsUnknown() {
		m, err := jsonStringToMap(plan.Filter.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid filter JSON", err.Error())
			return
		}
		params.Filter = m
		params.FilterLanguage = "jsonlogic"
	}
	if !plan.PayloadTemplate.IsNull() && !plan.PayloadTemplate.IsUnknown() {
		m, err := jsonStringToMap(plan.PayloadTemplate.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid payload_template JSON", err.Error())
			return
		}
		params.PayloadTemplate = m
	}

	result, err := r.client.Admin.UpdateTrigger(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Error updating trigger", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error updating trigger", result.Error.Message)
		return
	}

	newState := triggerToModel(result.Trigger, plan.Filter, plan.PayloadTemplate)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *triggerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state triggerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Admin.DeleteTrigger(ctx, admin.DeleteTriggerParams{
		TriggerID: state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting trigger", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error deleting trigger", result.Error.Message)
		return
	}
}

func (r *triggerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &triggerModel{
		ID: types.StringValue(req.ID),
	})...)
}

// triggerToModel maps a Trigger API struct to the Terraform state model.
// The filter and payloadTemplate are passed in as-is from the plan/state
// (round-tripping JSON objects through map[string]any can lose ordering).
func triggerToModel(t admin.Trigger, filterState, payloadState types.String) triggerModel {
	m := triggerModel{
		ID:                   types.StringValue(t.ID),
		URI:                  types.StringValue(t.URI),
		EventType:            types.StringValue(t.EventType),
		Additive:             types.BoolValue(t.Additive),
		AuthScheme:           types.StringValue(t.AuthScheme),
		ProductEnvironmentID: types.StringValue(t.ProductEnvironmentID),
		URIType:              types.StringValue(t.URIType),
		CreatedAt:            types.StringValue(t.CreatedAt),
		UpdatedAt:            types.StringValue(t.UpdatedAt),
	}

	// Preserve the original JSON string from plan/state to avoid spurious diffs
	// caused by map key ordering after round-tripping through map[string]any.
	if t.Filter != nil {
		if !filterState.IsNull() && !filterState.IsUnknown() {
			m.Filter = filterState
		} else {
			if s, err := mapToJSONString(t.Filter); err == nil {
				m.Filter = types.StringValue(s)
			}
		}
	} else {
		m.Filter = types.StringNull()
	}

	if t.PayloadTemplate != nil {
		if !payloadState.IsNull() && !payloadState.IsUnknown() {
			m.PayloadTemplate = payloadState
		} else {
			if s, err := mapToJSONString(t.PayloadTemplate); err == nil {
				m.PayloadTemplate = types.StringValue(s)
			}
		}
	} else {
		m.PayloadTemplate = types.StringNull()
	}

	return m
}

func jsonStringToMap(s string) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func mapToJSONString(m map[string]any) (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// oneOfStringValidator validates that a string is one of the allowed values.
type oneOfStringValidator struct {
	values    []string
	fieldName string
}

func (v oneOfStringValidator) Description(_ context.Context) string {
	return fmt.Sprintf("%s must be one of: %v", v.fieldName, v.values)
}

func (v oneOfStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v oneOfStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	val := req.ConfigValue.ValueString()
	for _, allowed := range v.values {
		if val == allowed {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		fmt.Sprintf("Invalid %s", v.fieldName),
		fmt.Sprintf("%q is not a valid value for %s. Must be one of: %v", val, v.fieldName, v.values),
	)
}

// jsonObjectValidator validates that a string is a valid JSON object.
type jsonObjectValidator struct {
	fieldName string
}

func (v jsonObjectValidator) Description(_ context.Context) string {
	return fmt.Sprintf("%s must be a valid JSON object", v.fieldName)
}

func (v jsonObjectValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jsonObjectValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(req.ConfigValue.ValueString()), &m); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			fmt.Sprintf("Invalid JSON in %s", v.fieldName),
			fmt.Sprintf("Value must be a valid JSON object: %s", err.Error()),
		)
	}
}

// maxBytesValidator validates that a string does not exceed a byte limit.
type maxBytesValidator struct {
	max int
}

func (v maxBytesValidator) Description(_ context.Context) string {
	return fmt.Sprintf("Value must not exceed %d bytes", v.max)
}

func (v maxBytesValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v maxBytesValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	if len(req.ConfigValue.ValueString()) > v.max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Value too large",
			fmt.Sprintf("payload_template must not exceed %d bytes, got %d bytes.", v.max, len(req.ConfigValue.ValueString())),
		)
	}
}
