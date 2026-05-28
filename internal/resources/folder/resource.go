package folder

import (
	"context"
	"fmt"
	"strings"

	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/cloudinary/cloudinary-go/v2/api/admin/search"

	"github.com/NitriKx/terraform-provider-cloudinary/internal/providerdata"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &folderResource{}
var _ resource.ResourceWithImportState = &folderResource{}

// NewResource returns a new cloudinary_folder resource.
func NewResource() resource.Resource {
	return &folderResource{}
}

type folderResource struct {
	client *cloudinary.Cloudinary
}

type folderResourceModel struct {
	ID         types.String `tfsdk:"id"`
	ExternalID types.String `tfsdk:"external_id"`
	Path       types.String `tfsdk:"path"`
	Name       types.String `tfsdk:"name"`
}

func (r *folderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_folder"
}

func (r *folderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Cloudinary folder. " +
			"Folders are used to organize assets in your Cloudinary account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The folder's external ID assigned by Cloudinary (same as external_id).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"external_id": schema.StringAttribute{
				Computed: true,
				Description: "The folder's external ID assigned by Cloudinary. " +
					"Use this value in Cedar policy statements (e.g. resource.ancestor_ids.contains(\"...\")).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Required: true,
				Description: "The full path of the folder (e.g. \"production/images\"). " +
					"Segments are separated by \"/\". Must not start or end with \"/\". " +
					"Forbidden characters: ? & # \\ % < >",
				Validators: []validator.String{
					folderPathValidator{},
				},
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The leaf name of the folder (the last component of the path).",
			},
		},
	}
}

func (r *folderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *folderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan folderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Admin.CreateFolder(ctx, admin.CreateFolderParams{
		Folder: plan.Path.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating folder", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error creating folder", result.Error.Message)
		return
	}

	state := folderResourceModel{
		ID:         types.StringValue(result.ExternalID),
		ExternalID: types.StringValue(result.ExternalID),
		Path:       types.StringValue(result.Path),
		Name:       types.StringValue(result.Name),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *folderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state folderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var found *admin.FolderResult

	if !state.Path.IsNull() && state.Path.ValueString() != "" {
		// Normal lifecycle: look up by path.
		f, err := findFolder(ctx, r.client, state.Path.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading folder", err.Error())
			return
		}
		found = f
	} else if !state.ID.IsNull() && state.ID.ValueString() != "" {
		// Post-import: only external_id is known, resolve via SearchFolders.
		f, err := findFolderByExternalID(ctx, r.client, state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading folder by external ID", err.Error())
			return
		}
		found = f
	}

	if found == nil {
		// Folder was deleted outside of Terraform.
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(found.ExternalID)
	state.ExternalID = types.StringValue(found.ExternalID)
	state.Path = types.StringValue(found.Path)
	state.Name = types.StringValue(found.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *folderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan folderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state folderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Admin.RenameFolder(ctx, admin.RenameFolderParams{
		FromPath: state.Path.ValueString(),
		ToPath:   plan.Path.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error renaming folder", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error renaming folder", result.Error.Message)
		return
	}

	// external_id is the stable identifier — it does not change on rename.
	// The API response omits it, so keep the existing state values for id and external_id.
	state.Path = types.StringValue(result.To.Path)
	state.Name = types.StringValue(result.To.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *folderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state folderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Admin.DeleteFolder(ctx, admin.DeleteFolderParams{
		Folder: state.Path.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting folder", err.Error())
		return
	}
	if result.Error.Message != "" {
		resp.Diagnostics.AddError("Cloudinary API error deleting folder", result.Error.Message)
		return
	}
}

func (r *folderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is the folder's external_id.
	state := folderResourceModel{
		ID: types.StringValue(req.ID),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// folderPathValidator validates Cloudinary folder paths:
//   - Must not start or end with "/" (Cloudinary strips them, causing state inconsistency)
//   - Must not contain the characters forbidden by the Cloudinary API: ? & # \ % < >
//     (Cloudinary returns "The folder name can't include the characters: ?&#\/%<>")
type folderPathValidator struct{}

func (v folderPathValidator) Description(_ context.Context) string {
	return `Folder path must not start or end with "/" and must not contain: ? & # \ % < >`
}

func (v folderPathValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

const folderForbiddenChars = `?&#\%<>`

func (v folderPathValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	p := req.ConfigValue.ValueString()

	if strings.HasPrefix(p, "/") || strings.HasSuffix(p, "/") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid folder path",
			`Folder path must not start or end with "/" (e.g. use "production/images", not "/production/images/").`,
		)
		return
	}

	for _, ch := range folderForbiddenChars {
		if strings.ContainsRune(p, ch) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid folder path",
				fmt.Sprintf("Folder path contains the forbidden character %q. Cloudinary does not allow: %s",
					string(ch), folderForbiddenChars),
			)
			return
		}
	}
}

// findFolder searches for a folder by path using SubFolders/RootFolders, handling pagination.
// Returns nil, nil if not found (deleted outside Terraform).
func findFolder(ctx context.Context, client *cloudinary.Cloudinary, folderPath string) (*admin.FolderResult, error) {
	parentPath := parentOf(folderPath)

	var nextCursor string
	for {
		var folders []admin.FolderResult
		var cursor string
		var apiErr string

		if parentPath == "" {
			result, e := client.Admin.RootFolders(ctx, admin.RootFoldersParams{
				NextCursor: nextCursor,
			})
			if e != nil {
				return nil, e
			}
			folders = result.Folders
			cursor = result.NextCursor
			apiErr = result.Error.Message
		} else {
			result, e := client.Admin.SubFolders(ctx, admin.SubFoldersParams{
				Folder:     parentPath,
				NextCursor: nextCursor,
			})
			if e != nil {
				return nil, e
			}
			folders = result.Folders
			cursor = result.NextCursor
			apiErr = result.Error.Message
		}

		if apiErr != "" {
			// A 404 on SubFolders means the parent itself doesn't exist.
			return nil, nil
		}

		for i := range folders {
			if folders[i].Path == folderPath {
				return &folders[i], nil
			}
		}

		if cursor == "" {
			break
		}
		nextCursor = cursor
	}

	return nil, nil
}

// findFolderByExternalID resolves a folder by its external_id using the SearchFolders API.
// Returns nil, nil if not found.
func findFolderByExternalID(ctx context.Context, client *cloudinary.Cloudinary, externalID string) (*admin.FolderResult, error) {
	result, err := client.Admin.SearchFolders(ctx, search.Query{
		Expression: fmt.Sprintf("external_id=\"%s\"", externalID),
		MaxResults: 1,
	})
	if err != nil {
		return nil, err
	}
	if result.Error.Message != "" {
		return nil, fmt.Errorf("%s", result.Error.Message)
	}
	if len(result.Folders) == 0 {
		return nil, nil
	}

	sf := result.Folders[0]
	return &admin.FolderResult{
		Name:       sf.Name,
		Path:       sf.Path,
		ExternalID: sf.ExternalID,
	}, nil
}

// parentOf returns the parent path of a folder path.
// e.g. "a/b/c" -> "a/b", "a" -> "".
func parentOf(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ""
	}
	return path[:idx]
}
