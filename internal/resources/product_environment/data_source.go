package product_environment

import (
	"context"
	"fmt"

	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/provisioning"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &productEnvironmentDataSource{}

// NewDataSource returns a new cloudinary_product_environment data source.
func NewDataSource() datasource.DataSource {
	return &productEnvironmentDataSource{}
}

type productEnvironmentDataSource struct {
	client *cloudinary.Cloudinary
}

type productEnvironmentDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	CloudName        types.String `tfsdk:"cloud_name"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	CustomAttributes types.Map    `tfsdk:"custom_attributes"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (d *productEnvironmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product_environment"
}

func (d *productEnvironmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads information about an existing Cloudinary product environment (sub-account).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier of the product environment.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The display name of the product environment.",
			},
			"cloud_name": schema.StringAttribute{
				Computed:    true,
				Description: "The Cloudinary cloud name for this product environment.",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the product environment is enabled.",
			},
			"custom_attributes": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Custom key-value attributes associated with the product environment.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time at which the product environment was created (RFC3339).",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The time at which the product environment was last updated (RFC3339).",
			},
		},
	}
}

func (d *productEnvironmentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"The cloudinary_product_environment data source requires account_id to be set in the provider configuration.",
		)
		return
	}
	d.client = client
}

func (d *productEnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data productEnvironmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.Provisioning.GetProductEnvironment(ctx, provisioning.GetProductEnvironmentParams{
		SubAccountID: data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error reading product environment", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error reading product environment", result.Error.Message)
		return
	}

	customAttrs, diags := types.MapValueFrom(ctx, types.StringType, result.CustomAttributes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(result.ID)
	data.Name = types.StringValue(result.Name)
	data.CloudName = types.StringValue(result.CloudName)
	data.Enabled = types.BoolValue(result.Enabled)
	data.CustomAttributes = customAttrs
	data.CreatedAt = types.StringValue(result.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	data.UpdatedAt = types.StringValue(result.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
