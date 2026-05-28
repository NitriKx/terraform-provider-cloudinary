package current_principal

import (
	"context"
	"fmt"

	"github.com/NitriKx/terraform-provider-cloudinary/internal/providerdata"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &currentPrincipalDataSource{}

// NewDataSource returns a new cloudinary_current_principal data source.
func NewDataSource() datasource.DataSource {
	return &currentPrincipalDataSource{}
}

type currentPrincipalDataSource struct {
	principalID string
	cloudName   string
}

type currentPrincipalModel struct {
	PrincipalID   types.String `tfsdk:"principal_id"`
	PrincipalType types.String `tfsdk:"principal_type"`
	CloudName     types.String `tfsdk:"cloud_name"`
}

func (d *currentPrincipalDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_current_principal"
}

func (d *currentPrincipalDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns the identity of the currently authenticated principal " +
			"(the credentials used to configure the provider). " +
			"Use principal_id and principal_type directly in role assignments.",
		Attributes: map[string]schema.Attribute{
			"principal_id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier of the current principal.",
			},
			"principal_type": schema.StringAttribute{
				Computed: true,
				Description: "The type of the current principal. " +
					"Currently always \"apiKey\" when authenticating with a product environment API key.",
			},
			"cloud_name": schema.StringAttribute{
				Computed:    true,
				Description: "The Cloudinary cloud name the provider is configured for.",
			},
		},
	}
}

func (d *currentPrincipalDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.principalID = pd.APIKey
	d.cloudName = pd.CloudName
}

func (d *currentPrincipalDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &currentPrincipalModel{
		PrincipalID:   types.StringValue(d.principalID),
		PrincipalType: types.StringValue("apiKey"),
		CloudName:     types.StringValue(d.cloudName),
	})...)
}
