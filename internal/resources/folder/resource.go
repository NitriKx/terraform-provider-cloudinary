package folder

import (
	"context"
	"fmt"
	"strings"

	cloudinary "github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	Path types.String `tfsdk:"path"`
	Name types.String `tfsdk:"name"`
}

func (r *folderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_folder"
}

func (r *folderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Cloudinary folder. " +
			"Folders are used to organize assets in your Cloudinary account.",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Required: true,
				Description: "The full path of the folder (e.g. \"production/images\"). " +
					"Changing this value forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
	client, ok := req.ProviderData.(*cloudinary.Cloudinary)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *cloudinary.Cloudinary, got: %T", req.ProviderData),
		)
		return
	}
	r.client = client
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
		Path: types.StringValue(result.Path),
		Name: types.StringValue(result.Name),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *folderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state folderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	folderPath := state.Path.ValueString()
	found, err := findFolder(ctx, r.client, folderPath)
	if err != nil {
		resp.Diagnostics.AddError("Error reading folder", err.Error())
		return
	}
	if found == nil {
		// Folder was deleted outside of Terraform.
		resp.State.RemoveResource(ctx)
		return
	}

	state.Path = types.StringValue(found.Path)
	state.Name = types.StringValue(found.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op because the only mutable attribute (path) is ForceNew.
func (r *folderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan folderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
	// The import ID is the folder path.
	state := folderResourceModel{
		Path: types.StringValue(req.ID),
		// Name will be populated on the subsequent Read.
		Name: types.StringValue(leafName(req.ID)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// findFolder searches for a folder by path, handling pagination.
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

// parentOf returns the parent path of a folder path.
// e.g. "a/b/c" -> "a/b", "a" -> "".
func parentOf(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ""
	}
	return path[:idx]
}

// leafName returns the last component of a path.
func leafName(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return path
	}
	return path[idx+1:]
}
