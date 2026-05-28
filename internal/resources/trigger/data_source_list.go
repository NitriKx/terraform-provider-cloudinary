package trigger

import (
	"context"
	"fmt"

	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"

	"github.com/NitriKx/terraform-provider-cloudinary/internal/providerdata"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &triggerListDataSource{}

// NewListDataSource returns a new cloudinary_triggers data source.
func NewListDataSource() datasource.DataSource {
	return &triggerListDataSource{}
}

type triggerListDataSource struct {
	client *cloudinary.Cloudinary
}

type triggerListModel struct {
	Triggers types.List `tfsdk:"triggers"`
}

// triggerObjectType describes the attr.Type of each trigger in the list.
var triggerObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":                     types.StringType,
		"uri":                    types.StringType,
		"event_type":             types.StringType,
		"additive":               types.BoolType,
		"filter":                 types.StringType,
		"payload_template":       types.StringType,
		"auth_scheme":            types.StringType,
		"product_environment_id": types.StringType,
		"uri_type":               types.StringType,
		"created_at":             types.StringType,
		"updated_at":             types.StringType,
	},
}

func (d *triggerListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_triggers"
}

func (d *triggerListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	triggerAttrs := map[string]schema.Attribute{
		"id":                     schema.StringAttribute{Computed: true, Description: "Trigger ID."},
		"uri":                    schema.StringAttribute{Computed: true, Description: "Webhook URL."},
		"event_type":             schema.StringAttribute{Computed: true, Description: "Event that fires the trigger."},
		"additive":               schema.BoolAttribute{Computed: true, Description: "Whether the trigger is additive."},
		"filter":                 schema.StringAttribute{Computed: true, Description: "JSONLogic filter (JSON-encoded)."},
		"payload_template":       schema.StringAttribute{Computed: true, Description: "Mustache payload template (JSON-encoded)."},
		"auth_scheme":            schema.StringAttribute{Computed: true, Description: "Signature method."},
		"product_environment_id": schema.StringAttribute{Computed: true, Description: "Product environment scope."},
		"uri_type":               schema.StringAttribute{Computed: true, Description: "URI type (always webhook)."},
		"created_at":             schema.StringAttribute{Computed: true, Description: "ISO 8601 creation timestamp."},
		"updated_at":             schema.StringAttribute{Computed: true, Description: "ISO 8601 last-updated timestamp."},
	}

	resp.Schema = schema.Schema{
		Description: "Returns all webhook notification triggers configured for the product environment.",
		Attributes: map[string]schema.Attribute{
			"triggers": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of all configured triggers.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: triggerAttrs,
				},
			},
		},
	}
}

func (d *triggerListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *triggerListDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	result, err := d.client.Admin.ListTriggers(ctx, admin.ListTriggersParams{})
	if err != nil {
		resp.Diagnostics.AddError("Error listing triggers", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Error listing triggers", result.Error.Message)
		return
	}

	objs := make([]attr.Value, 0, len(result.Triggers))
	for _, t := range result.Triggers {
		m := triggerToModel(t, types.StringNull(), types.StringNull())
		obj, diags := types.ObjectValue(triggerObjectType.AttrTypes, map[string]attr.Value{
			"id":                     m.ID,
			"uri":                    m.URI,
			"event_type":             m.EventType,
			"additive":               m.Additive,
			"filter":                 m.Filter,
			"payload_template":       m.PayloadTemplate,
			"auth_scheme":            m.AuthScheme,
			"product_environment_id": m.ProductEnvironmentID,
			"uri_type":               m.URIType,
			"created_at":             m.CreatedAt,
			"updated_at":             m.UpdatedAt,
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		objs = append(objs, obj)
	}

	list, diags := types.ListValue(triggerObjectType, objs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &triggerListModel{Triggers: list})...)
}
