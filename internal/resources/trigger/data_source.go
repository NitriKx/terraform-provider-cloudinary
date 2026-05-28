package trigger

import (
	"context"
	"fmt"

	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"

	"github.com/NitriKx/terraform-provider-cloudinary/internal/providerdata"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &triggerDataSource{}
var _ datasource.DataSourceWithConfigValidators = &triggerDataSource{}

// NewDataSource returns a new cloudinary_trigger data source.
func NewDataSource() datasource.DataSource {
	return &triggerDataSource{}
}

type triggerDataSource struct {
	client *cloudinary.Cloudinary
}

func (d *triggerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trigger"
}

func (d *triggerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads information about an existing Cloudinary webhook notification trigger. " +
			"Look up by id, or by the combination of event_type and uri.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The unique identifier of the trigger. When set, takes precedence over event_type + uri.",
			},
			"uri": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The webhook URL. Used as a lookup key together with event_type when id is not set.",
			},
			"event_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The event that fires the trigger. Used as a lookup key together with uri when id is not set.",
				Validators: []validator.String{
					oneOfStringValidator{values: validEventTypes, fieldName: "event_type"},
				},
			},
			"additive": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the trigger fires alongside per-asset notification_url callbacks.",
			},
			"filter": schema.StringAttribute{
				Computed:    true,
				Description: "JSONLogic filter expression (JSON-encoded) if configured.",
			},
			"payload_template": schema.StringAttribute{
				Computed:    true,
				Description: "Mustache payload template (JSON-encoded) if configured.",
			},
			"auth_scheme": schema.StringAttribute{
				Computed:    true,
				Description: "Signature method for verifying webhook payloads.",
			},
			"product_environment_id": schema.StringAttribute{
				Computed:    true,
				Description: "The product environment this trigger is scoped to.",
			},
			"uri_type": schema.StringAttribute{
				Computed:    true,
				Description: "The type of the URI. Always \"webhook\".",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp of when the trigger was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "ISO 8601 timestamp of when the trigger was last updated.",
			},
		},
	}
}

// triggerLookupValidator ensures either id, or event_type+uri is provided.
type triggerLookupValidator struct{}

func (v triggerLookupValidator) Description(_ context.Context) string {
	return "Either id, or both event_type and uri must be set."
}

func (v triggerLookupValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v triggerLookupValidator) ValidateDataSource(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var data triggerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !data.ID.IsNull() && !data.ID.IsUnknown() && data.ID.ValueString() != ""
	hasEventType := !data.EventType.IsNull() && !data.EventType.IsUnknown() && data.EventType.ValueString() != ""
	hasURI := !data.URI.IsNull() && !data.URI.IsUnknown() && data.URI.ValueString() != ""

	if !hasID && !(hasEventType && hasURI) {
		resp.Diagnostics.AddError(
			"Missing lookup key",
			"Either 'id', or both 'event_type' and 'uri' must be set to look up a trigger.",
		)
	}
}

func (d *triggerDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{triggerLookupValidator{}}
}

func (d *triggerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.client = pd.Client
}

func (d *triggerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data triggerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var found *admin.Trigger

	if !data.ID.IsNull() && data.ID.ValueString() != "" {
		list, err := d.client.Admin.ListTriggers(ctx, admin.ListTriggersParams{})
		if err != nil {
			resp.Diagnostics.AddError("Error reading trigger", err.Error())
			return
		}
		if list.Error.Message != "" {
			resp.Diagnostics.AddError("Error reading trigger", list.Error.Message)
			return
		}
		wantID := data.ID.ValueString()
		for i := range list.Triggers {
			if list.Triggers[i].ID == wantID {
				found = &list.Triggers[i]
				break
			}
		}
		if found == nil {
			resp.Diagnostics.AddError("Trigger not found",
				fmt.Sprintf("No trigger with id %q exists.", wantID))
			return
		}
	} else {
		wantEventType := data.EventType.ValueString()
		wantURI := data.URI.ValueString()

		result, err := d.client.Admin.ListTriggers(ctx, admin.ListTriggersParams{})
		if err != nil {
			resp.Diagnostics.AddError("Error listing triggers", err.Error())
			return
		}
		if result.Error.Message != "" {
			resp.Diagnostics.AddError("Error listing triggers", result.Error.Message)
			return
		}
		for i := range result.Triggers {
			t := &result.Triggers[i]
			if t.EventType == wantEventType && t.URI == wantURI {
				found = t
				break
			}
		}
		if found == nil {
			resp.Diagnostics.AddError(
				"Trigger not found",
				fmt.Sprintf("No trigger with event_type %q and uri %q exists.", wantEventType, wantURI),
			)
			return
		}
	}

	state := triggerToModel(*found, types.StringNull(), types.StringNull())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
