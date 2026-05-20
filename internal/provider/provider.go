package provider

import (
	"context"
	"fmt"
	"os"

	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/config"

	"github.com/NitriKx/terraform-provider-cloudinary/internal/providerdata"
	"github.com/NitriKx/terraform-provider-cloudinary/internal/resources/current_principal"
	"github.com/NitriKx/terraform-provider-cloudinary/internal/resources/folder"
	"github.com/NitriKx/terraform-provider-cloudinary/internal/resources/trigger"

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
}

func (p *CloudinaryProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cloudinary"
	resp.Version = p.version
}

func (p *CloudinaryProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Cloudinary provider manages product-environment-level Cloudinary resources such as folders.",
		Attributes: map[string]schema.Attribute{
			"cloudinary_url": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Cloudinary URL in the format cloudinary://API_KEY:API_SECRET@CLOUD_NAME. " +
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
		},
	}
}

func (p *CloudinaryProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read HCL-set values only — do not merge with env vars yet.
	hclURL := attrString(data.CloudinaryURL)
	hclCloudName := attrString(data.CloudName)
	hclAPIKey := attrString(data.APIKey)
	hclAPISecret := attrString(data.APISecret)

	// Catch partial HCL params early to avoid silent env-var fallback masking mistakes.
	hclParamsPartial := (hclCloudName != "") || (hclAPIKey != "") || (hclAPISecret != "")
	hclParamsComplete := (hclCloudName != "") && (hclAPIKey != "") && (hclAPISecret != "")
	if hclParamsPartial && !hclParamsComplete {
		resp.Diagnostics.AddError(
			"Incomplete provider credentials",
			"cloud_name, api_key, and api_secret must all be set together. "+
				"Provide all three, or use cloudinary_url instead.",
		)
		return
	}

	var conf *config.Configuration
	var err error

	// Precedence (standard Terraform provider convention — HCL beats env):
	//   1. HCL params (cloud_name + api_key + api_secret)
	//   2. HCL cloudinary_url
	//   3. Env params (CLOUDINARY_CLOUD_NAME + CLOUDINARY_API_KEY + CLOUDINARY_API_SECRET)
	//   4. Env CLOUDINARY_URL
	envCloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	envAPIKey := os.Getenv("CLOUDINARY_API_KEY")
	envAPISecret := os.Getenv("CLOUDINARY_API_SECRET")
	envURL := os.Getenv("CLOUDINARY_URL")

	switch {
	case hclParamsComplete:
		conf, err = config.NewFromParams(hclCloudName, hclAPIKey, hclAPISecret)
		if err != nil {
			resp.Diagnostics.AddError("Invalid credentials", fmt.Sprintf("Failed to build Cloudinary config: %s", err))
			return
		}
	case hclURL != "":
		conf, err = config.NewFromURL(hclURL)
		if err != nil {
			resp.Diagnostics.AddError("Invalid cloudinary_url", fmt.Sprintf("Failed to parse cloudinary_url: %s", err))
			return
		}
	case envCloudName != "" && envAPIKey != "" && envAPISecret != "":
		conf, err = config.NewFromParams(envCloudName, envAPIKey, envAPISecret)
		if err != nil {
			resp.Diagnostics.AddError("Invalid credentials", fmt.Sprintf("Failed to build Cloudinary config from env: %s", err))
			return
		}
	case envURL != "":
		conf, err = config.NewFromURL(envURL)
		if err != nil {
			resp.Diagnostics.AddError("Invalid CLOUDINARY_URL", fmt.Sprintf("Failed to parse CLOUDINARY_URL: %s", err))
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

	adminClient, err := cloudinary.NewFromConfiguration(*conf)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Cloudinary client", err.Error())
		return
	}

	pd := &providerdata.ProviderData{
		Client:    adminClient,
		APIKey:    conf.Cloud.APIKey,
		CloudName: conf.Cloud.CloudName,
	}
	resp.DataSourceData = pd
	resp.ResourceData = pd
}

func (p *CloudinaryProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		folder.NewResource,
		trigger.NewResource,
	}
}

func (p *CloudinaryProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		folder.NewDataSource,
		current_principal.NewDataSource,
		trigger.NewDataSource,
		trigger.NewListDataSource,
	}
}

// attrString returns the string value of a types.String HCL attribute, or ""
// when the attribute is null or unknown (i.e. not set in the provider block).
func attrString(attr types.String) string {
	if attr.IsNull() || attr.IsUnknown() {
		return ""
	}
	return attr.ValueString()
}
