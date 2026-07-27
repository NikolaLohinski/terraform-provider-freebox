package internal

import (
	"context"
	"errors"
	"fmt"
	go_path "path"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/nikolalohinski/terraform-provider-freebox/internal/models"
)

var (
	_ resource.Resource                = (*remoteDirectoryResource)(nil)
	_ resource.ResourceWithImportState = (*remoteDirectoryResource)(nil)
)

func NewRemoteDirectoryResource() resource.Resource {
	return &remoteDirectoryResource{}
}

// remoteDirectoryResource defines the resource implementation.
type remoteDirectoryResource struct {
	client client.Client
}

// remoteDirectoryModel describes the resource data model.
type remoteDirectoryModel struct {
	// DestinationPath is the file path on the Freebox.
	DestinationPath types.String `tfsdk:"destination_path"`

	// Parents is whether to create parent directories.
	Parents types.Bool `tfsdk:"parents"`

	// RemoveAll is whether to remove all files and directories in the directory.
	RemoveAll types.Bool `tfsdk:"remove_all"`

	// Polling is the polling configuration.
	Polling types.Object `tfsdk:"polling"`
}

func (o remoteDirectoryModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"destination_path": types.StringType,
		"parents":          types.BoolType,
		"remove_all":       types.BoolType,
		"polling":          types.ObjectType{}.WithAttributeTypes(remoteDirectoryPollingModel{}.AttrTypes()),
	}
}

func (v *remoteDirectoryModel) populateDefaults() {
	v.Parents = basetypes.NewBoolValue(true)
	v.RemoveAll = basetypes.NewBoolValue(true)
	v.Polling = remoteDirectoryPollingModel{}.defaults()
}

func (v *remoteDirectoryModel) populateFromFileInfo(fileInfo freeboxTypes.FileInfo) {
	v.DestinationPath = basetypes.NewStringValue(go_path.Join(string(fileInfo.Parent), fileInfo.Name))
}

type remoteDirectoryPollingModel struct {
	Delete types.Object `tfsdk:"delete"`
	Move   types.Object `tfsdk:"move"`
}

func (o remoteDirectoryPollingModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"delete": types.ObjectType{}.WithAttributeTypes(models.Polling{}.AttrTypes()),
		"move":   types.ObjectType{}.WithAttributeTypes(models.Polling{}.AttrTypes()),
	}
}

func (o remoteDirectoryPollingModel) defaults() basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(remoteDirectoryPollingModel{}.AttrTypes(), map[string]attr.Value{
		"delete": models.NewPollingSpecModel(time.Second, time.Minute),
		"move":   models.NewPollingSpecModel(time.Second, time.Minute),
	})
}

func (o remoteDirectoryPollingModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"delete": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Deletion polling configuration",
			Attributes:          models.PollingSpecModelResourceAttributes(time.Second, time.Minute),
			Default:             objectdefault.StaticValue(models.NewPollingSpecModel(time.Second, time.Minute)),
		},
		"move": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Move polling configuration",
			Attributes:          models.PollingSpecModelResourceAttributes(time.Second, time.Minute),
			Default:             objectdefault.StaticValue(models.NewPollingSpecModel(time.Second, time.Minute)),
		},
	}
}

func (v *remoteDirectoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_remote_directory"
}

func (v *remoteDirectoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource creates a directory on the Freebox.",
		Attributes: map[string]schema.Attribute{
			"destination_path": schema.StringAttribute{
				MarkdownDescription: "Path to the directory on the Freebox",
				Required:            true,
				Validators: []validator.String{
					models.FilePathValidator(path.Root("destination_path")),
				},
			},
			"parents": schema.BoolAttribute{
				MarkdownDescription: "Whether to create parent directories",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"remove_all": schema.BoolAttribute{
				MarkdownDescription: "Whether to remove all files and directories in the directory",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"polling": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Polling configuration",
				Attributes:          remoteDirectoryPollingModel{}.ResourceAttributes(),
				Default:             objectdefault.StaticValue(remoteDirectoryPollingModel{}.defaults()),
			},
		},
	}
}

func (v *remoteDirectoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	v.client = client
}

func (v *remoteDirectoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model remoteDirectoryModel

	if diags := req.Plan.Get(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	path := model.DestinationPath.ValueString()

	if diags := v.createParentDirectoriesIfNeeded(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	parent, name := go_path.Split(path)

	_, err := v.client.CreateDirectory(ctx, parent, name)
	if err != nil {
		if !errors.Is(err, client.ErrDestinationConflict) {
			resp.Diagnostics.AddError("Failed to create directory", fmt.Sprintf("Path: %s, Error: %s", path, err.Error()))
			return
		}
	}

	if diags := resp.State.Set(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
}

func (v *remoteDirectoryResource) createParentDirectoriesIfNeeded(ctx context.Context, model *remoteDirectoryModel) (diagnostics diag.Diagnostics) {
	if !model.Parents.ValueBool() {
		return
	}

	parent := ""
	for _, path := range strings.Split(strings.TrimPrefix(go_path.Dir(model.DestinationPath.ValueString()), "/"), "/") {
		_, err := v.client.CreateDirectory(ctx, parent, path)

		parent += "/" + path

		if err != nil {
			if errors.Is(err, client.ErrDestinationConflict) {
				continue
			}
			/*
				if i == 0 { // The first path is the parent directory, so we probably don't have access to it
					var apiError *client.APIError
					if errors.As(err, &apiError) {
						if apiError.Code == string(freeboxTypes.AccessDeniedErrorCode) {
							continue
						}
					}
				}
			*/
			diagnostics.AddWarning("Failed to create parent directory", fmt.Sprintf("Path: %s, Error: %s", go_path.Join(parent, path), err.Error()))
			continue
		}

		tflog.Debug(ctx, "Created directory", map[string]interface{}{
			"parent": parent,
		})
	}

	return
}

func (v *remoteDirectoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model remoteDirectoryModel

	if diags := req.State.Get(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	fileInfo, err := v.client.GetFileInfo(ctx, model.DestinationPath.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrPathNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Failed to get directory info", fmt.Sprintf("Path: %s, Error: %s", model.DestinationPath.ValueString(), err.Error()))
		return
	}

	if fileInfo.Type != freeboxTypes.FileTypeDirectory {
		resp.Diagnostics.AddError("Path is not a directory", fmt.Sprintf("Path: %s, Type: %s", model.DestinationPath.ValueString(), fileInfo.Type))
		return
	}

	model.populateFromFileInfo(fileInfo)

	if diags := resp.State.Set(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
}

func (v *remoteDirectoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var oldModel, newModel remoteDirectoryModel

	resp.Diagnostics.Append(req.State.Get(ctx, &oldModel)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &newModel)...)

	if resp.Diagnostics.HasError() {
		return
	}

	newModel.populateDefaults()

	oldPath := oldModel.DestinationPath.ValueString()
	newPath := newModel.DestinationPath.ValueString()

	var polling remoteDirectoryPollingModel

	if diags := newModel.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if oldPath != newPath {
		if diags := v.createParentDirectoriesIfNeeded(ctx, &newModel); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var movePolling models.Polling

		if diags := polling.Move.As(ctx, &movePolling, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		task, err := v.client.MoveFiles(ctx, []string{oldPath}, newPath, freeboxTypes.FileMoveModeSkip)
		if err != nil {
			resp.Diagnostics.AddError("Failed to move directory", fmt.Sprintf("From: %s, To: %s, Error: %s", oldModel.DestinationPath.ValueString(), newModel.DestinationPath.ValueString(), err.Error()))
			return
		}

		defer func() {
			if err := stopAndDeleteFileSystemTask(ctx, v.client, task.ID); err != nil {
				resp.Diagnostics.AddError("Failed to stop and delete file system task", fmt.Sprintf("Task %d, Error: %s", task.ID, err.Error()))
				return
			}
		}()

		if diags := waitForFileSystemTask(ctx, v.client, task.ID, movePolling); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	if diags := resp.State.Set(ctx, &newModel); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
}

func (v *remoteDirectoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model remoteDirectoryModel

	if diags := req.State.Get(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if !model.RemoveAll.ValueBool() {
		files, err := v.client.ListFiles(ctx, model.DestinationPath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to list files", fmt.Sprintf("Path: %s, Error: %s", model.DestinationPath.ValueString(), err.Error()))
			return
		}

		if len(files) > 0 {
			resp.Diagnostics.AddError("Directory is not empty", fmt.Sprintf("Path: %s, Files count: %d", model.DestinationPath.ValueString(), len(files)))
			return
		}
	}

	// Delete the file
	var polling remoteDirectoryPollingModel

	if diags := model.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var deletePolling models.Polling

	if diags := polling.Delete.As(ctx, &deletePolling, basetypes.ObjectAsOptions{}); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	tflog.Info(ctx, "Deleting the file...", map[string]interface{}{
		"path": model.DestinationPath.ValueString(),
	})

	if diags := deleteFilesIfExist(ctx, resp.Private, v.client, deletePolling, model.DestinationPath.ValueString()); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
}

func (v *remoteDirectoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	path := req.ID

	tflog.Info(ctx, "Reading the file metadata...", map[string]interface{}{
		"path": path,
	})

	fileInfo, err := v.client.GetFileInfo(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get file", fmt.Sprintf("Path: %s, Error: %s", path, err.Error()))
		return
	}

	if fileInfo.Type != freeboxTypes.FileTypeDirectory {
		resp.Diagnostics.AddError("Path is not a directory", fmt.Sprintf("Path: %s, Type: %s", path, fileInfo.Type))
		return
	}

	var model remoteDirectoryModel

	defer func() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}()

	model.DestinationPath = types.StringValue(path)

	model.populateDefaults()
}
