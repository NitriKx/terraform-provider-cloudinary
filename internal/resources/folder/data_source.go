package folder

import (
	"context"
	"fmt"

	cloudinary "github.com/cloudinary/cloudinary-go/v2"

	"github.com/NitriKx/terraform-provider-cloudinary/internal/providerdata"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &folderDataSource{}

// NewDataSource returns a new cloudinary_folder data source.
func NewDataSource() datasource.DataSource {
	return &folderDataSource{}
}

type folderDataSource struct {
	client *cloudinary.Cloudinary
}

type folderDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	ExternalID types.String `tfsdk:"external_id"`
	Path       types.String `tfsdk:"path"`
	Name       types.String `tfsdk:"name"`
}

func (d *folderDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_folder"
}

func (d *folderDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads information about an existing Cloudinary folder.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The folder's external ID assigned by Cloudinary (same as external_id).",
			},
			"external_id": schema.StringAttribute{
				Computed: true,
				Description: "The folder's external ID assigned by Cloudinary. " +
					"Use this value in Cedar policy statements (e.g. resource.ancestor_ids.contains(\"...\")).",
			},
			"path": schema.StringAttribute{
				Required:    true,
				Description: "The full path of the folder to look up (e.g. \"production/images\").",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The leaf name of the folder (the last component of the path).",
			},
		},
	}
}

func (d *folderDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *folderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data folderDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := findFolder(ctx, d.client, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading folder", err.Error())
		return
	}
	if found == nil {
		resp.Diagnostics.AddError(
			"Folder not found",
			fmt.Sprintf("No folder with path %q exists in your Cloudinary account.", data.Path.ValueString()),
		)
		return
	}

	data.ID = types.StringValue(found.ExternalID)
	data.ExternalID = types.StringValue(found.ExternalID)
	data.Path = types.StringValue(found.Path)
	data.Name = types.StringValue(found.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
