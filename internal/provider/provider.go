package provider

import (
	"context"
	"fmt"
	"os"

	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/config"

	"github.com/NitriKx/terraform-provider-cloudinary/internal/resources/custom_policy"
	"github.com/NitriKx/terraform-provider-cloudinary/internal/resources/folder"
	"github.com/NitriKx/terraform-provider-cloudinary/internal/resources/product_environment"
	"github.com/NitriKx/terraform-provider-cloudinary/internal/resources/product_environment_access_key"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &CloudinaryProvider{}

// CloudinaryProvider implements the Terraform provider for Cloudinary.
type CloudinaryProvider struct {
	version string
}

// New creates a new CloudinaryProvider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CloudinaryProvider{version: version}
	}
}

// providerModel holds the provider configuration read from HCL.
type providerModel struct {
	CloudinaryURL types.String `tfsdk:"cloudinary_url"`
	CloudName     types.String `tfsdk:"cloud_name"`
	APIKey        types.String `tfsdk:"api_key"`
	APISecret     types.String `tfsdk:"api_secret"`
	AccountID     types.String `tfsdk:"account_id"`
}

func (p *CloudinaryProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cloudinary"
	resp.Version = p.version
}

func (p *CloudinaryProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Cloudinary provider manages Cloudinary resources such as folders, " +
			"product environments, access keys, and custom policies.",
		Attributes: map[string]schema.Attribute{
			"cloudinary_url": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Cloudinary URL in the format cloudinary://API_KEY:API_SECRET@CLOUD_NAME. " +
					"Can include account_id as a query parameter (e.g. ?account_id=abc123). " +
					"Falls back to the CLOUDINARY_URL environment variable.",
			},
			"cloud_name": schema.StringAttribute{
				Optional: true,
				Description: "Cloudinary cloud name. " +
					"Falls back to the CLOUDINARY_CLOUD_NAME environment variable.",
			},
			"api_key": schema.StringAttribute{
				Optional: true,
				Description: "Cloudinary API key. " +
					"Falls back to the CLOUDINARY_API_KEY environment variable.",
			},
			"api_secret": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Cloudinary API secret. " +
					"Falls back to the CLOUDINARY_API_SECRET environment variable.",
			},
			"account_id": schema.StringAttribute{
				Optional: true,
				Description: "Cloudinary account ID. Required for Provisioning API resources " +
					"(cloudinary_product_environment, cloudinary_product_environment_access_key, cloudinary_custom_policy). " +
					"Falls back to the CLOUDINARY_ACCOUNT_ID environment variable.",
			},
		},
	}
}

func (p *CloudinaryProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve each value: HCL attribute takes precedence over environment variable.
	cloudinaryURL := resolveValue(data.CloudinaryURL, "CLOUDINARY_URL")
	cloudName := resolveValue(data.CloudName, "CLOUDINARY_CLOUD_NAME")
	apiKey := resolveValue(data.APIKey, "CLOUDINARY_API_KEY")
	apiSecret := resolveValue(data.APISecret, "CLOUDINARY_API_SECRET")
	accountID := resolveValue(data.AccountID, "CLOUDINARY_ACCOUNT_ID")

	// Build the Cloudinary configuration.
	var conf *config.Configuration
	var err error

	switch {
	case cloudinaryURL != "":
		conf, err = config.NewFromURL(cloudinaryURL)
		if err != nil {
			resp.Diagnostics.AddError("Invalid cloudinary_url", fmt.Sprintf("Failed to parse cloudinary_url: %s", err))
			return
		}
	case cloudName != "" && apiKey != "" && apiSecret != "":
		conf, err = config.NewFromParams(cloudName, apiKey, apiSecret)
		if err != nil {
			resp.Diagnostics.AddError("Invalid credentials", fmt.Sprintf("Failed to build Cloudinary config: %s", err))
			return
		}
	default:
		resp.Diagnostics.AddError(
			"Missing Cloudinary credentials",
			"Provide either cloudinary_url (or CLOUDINARY_URL env var), "+
				"or all of cloud_name, api_key, and api_secret "+
				"(or their CLOUDINARY_CLOUD_NAME / CLOUDINARY_API_KEY / CLOUDINARY_API_SECRET equivalents).",
		)
		return
	}

	// Set AccountID if provided. This is required for Provisioning API resources.
	if accountID != "" {
		conf.Cloud.AccountID = accountID
	}

	client, err := cloudinary.NewFromConfiguration(*conf)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Cloudinary client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *CloudinaryProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		folder.NewResource,
		product_environment.NewResource,
		product_environment_access_key.NewResource,
		custom_policy.NewResource,
	}
}

func (p *CloudinaryProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		folder.NewDataSource,
		product_environment.NewDataSource,
	}
}

// resolveValue returns the string value of a types.String attribute if set,
// otherwise falls back to the named environment variable.
func resolveValue(attr types.String, envVar string) string {
	if !attr.IsNull() && !attr.IsUnknown() && attr.ValueString() != "" {
		return attr.ValueString()
	}
	return os.Getenv(envVar)
}
