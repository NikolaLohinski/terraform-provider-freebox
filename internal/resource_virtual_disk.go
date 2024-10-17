package internal

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/nikolalohinski/terraform-provider-freebox/internal/models"
	providerdata "github.com/nikolalohinski/terraform-provider-freebox/internal/provider_data"
)

var (
	_ resource.Resource                = &virtualDiskResource{}
	_ resource.ResourceWithImportState = &virtualDiskResource{}
)

func NewVirtualDiskResource() resource.Resource {
	return &virtualDiskResource{}
}

// virtualDiskResource defines the resource implementation.
type virtualDiskResource struct {
	client client.Client
}

// virtualDiskModel describes the resource data model.
type virtualDiskModel struct {
	// Path is the path to the virtual disk.
    Path types.String `tfsdk:"path"`
	// Type is the type of virtual disk.
	Type types.String `tfsdk:"type"`
	// VirtualSize is the size of virtual disk. This is the size the disk will appear inside the VM.
	VirtualSize types.Int64 `tfsdk:"virtual_size"`
	// SizeOnDisk is the space used by virtual image on disk. This is how much filesystem space is consumed on the box.
	SizeOnDisk types.Int64 `tfsdk:"size_on_disk"`

	// polling is the polling configuration.
	Polling types.Object `tfsdk:"polling"`
}

func (v *virtualDiskModel) populateFromVirtualDiskInfo(fileInfo freeboxTypes.VirtualDiskInfo) {
	v.Type = basetypes.NewStringValue(string(fileInfo.Type))
	v.VirtualSize = basetypes.NewInt64Value(fileInfo.VirtualSize)
	v.SizeOnDisk = basetypes.NewInt64Value(fileInfo.ActualSize)
}

func (v *virtualDiskModel) populateDefaults() {
	if v.Polling.IsUnknown() || v.Polling.IsNull() {
		v.Polling = virtualDiskPollingModel{}.defaults()
	}
}

func (v *virtualDiskModel) toCreatePayload() (payload freeboxTypes.VirtualDisksCreatePayload) {
	payload.DiskPath = freeboxTypes.Base64Path(v.Path.ValueString())
	payload.Size = v.VirtualSize.ValueInt64()
	payload.DiskType = v.Type.ValueString()

	return
}

func (v *virtualDiskModel) toResizePayload() (payload freeboxTypes.VirtualDisksResizePayload) {
	payload.DiskPath = freeboxTypes.Base64Path(v.Path.ValueString())
	payload.NewSize = v.VirtualSize.ValueInt64()
	payload.ShrinkAllow = true  // Shrink is always allowed

	return
}

type virtualDiskPollingModel struct {
	// Delete is the polling configuration for delete operation.
	Delete types.Object `tfsdk:"delete"`
	// Move is the polling configuration for move operation.
	Move  types.Object `tfsdk:"move"`
}

func (v virtualDiskPollingModel) defaults() basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(virtualDiskPollingModel{}.AttrTypes(), map[string]attr.Value{
		"delete": models.NewPollingSpecModel(time.Second, time.Minute),
		"move":   models.NewPollingSpecModel(time.Second, time.Minute),
	})
}

func (v virtualDiskPollingModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"delete": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Polling configuration for delete operation",
			Attributes:          models.PollingSpecModelResourceAttributes(time.Second, time.Minute),
			Default:             objectdefault.StaticValue(models.NewPollingSpecModel(time.Second, time.Minute)),
		},
		"move": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Polling configuration for move operation",
			Attributes:          models.PollingSpecModelResourceAttributes(time.Second, time.Minute),
			Default:             objectdefault.StaticValue(models.NewPollingSpecModel(time.Second, time.Minute)),
		},
	}
}

func (v virtualDiskPollingModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"move":   types.ObjectType{}.WithAttributeTypes(models.Polling{}.AttrTypes()),
		"delete": types.ObjectType{}.WithAttributeTypes(models.Polling{}.AttrTypes()),
	}
}

func (v *virtualDiskResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_disk"
}

func (v *virtualDiskResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a virtual machine instance within a Freebox box. See the [Freebox blog](https://dev.freebox.fr/blog/?p=5450) for additional details",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				MarkdownDescription: "Path to the virtual disk on the Freebox",
				Required: 		     true,
				Validators: []validator.String{
					models.FilePathValidator(),
				},
			},
			"type": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("qcow2"),
				MarkdownDescription: "Type of virtual disk",
				Validators: []validator.String{
					models.DiskTypeValidator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"virtual_size": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "Size in bytes of virtual disk. This is the size the disk will appear inside the VM.",
				Validators: []validator.Int64{
					models.VirtualDiskSizeValidator(),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"size_on_disk": schema.Int64Attribute{
				Computed:           true,
				MarkdownDescription: "Space in bytes used by virtual image on the hard drive. This is how much filesystem space is consumed on the box.",
				Validators: []validator.Int64{
					models.DiskSizeValidator(),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"polling": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Polling configuration",
				Attributes:          virtualDiskPollingModel{}.ResourceAttributes(),
				Default:             objectdefault.StaticValue(virtualDiskPollingModel{}.defaults()),
			},
		},
	}
}

func (v *virtualDiskResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (v *virtualDiskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model virtualDiskModel

	if diags := req.Plan.Get(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	taskID, err := v.client.CreateVirtualDisk(ctx, freeboxTypes.VirtualDisksCreatePayload{
		DiskPath: freeboxTypes.Base64Path(model.Path.ValueString()),
		Size:     model.VirtualSize.ValueInt64(),
		DiskType: model.Type.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create virtual disk", fmt.Sprintf("Path: %s, Error: %s", model.Path.ValueString(), err.Error()))
		return
	}

	model.populateDefaults()

	defer func () {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}()

	if diags := providerdata.SetCurrentTask(ctx, resp.Private, models.TaskTypeVirtualDisk, taskID); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if diags := waitForDiskTask(ctx, v.client, taskID); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diskInfo, err := v.client.GetVirtualDiskInfo(ctx, model.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get disk info", fmt.Sprintf("Path: %s, Error: %s", model.Path.ValueString(), err.Error()))
		return
	}

	model.populateFromVirtualDiskInfo(diskInfo)
}

func (v *virtualDiskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model virtualDiskModel

	if diags := req.State.Get(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	model.populateDefaults()

	defer func() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}()

	diskInfo, err := v.client.GetVirtualDiskInfo(ctx, model.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get disk info", fmt.Sprintf("Path: %s, Error: %s", model.Path.ValueString(), err.Error()))
		return
	}

	model.populateFromVirtualDiskInfo(diskInfo)
}

func (v *virtualDiskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var oldModel, newModel virtualDiskModel

	resp.Diagnostics.Append(req.State.Get(ctx, &oldModel)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &newModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	recreate := oldModel.Type.ValueString() != newModel.Type.ValueString()

	task, diags := providerdata.GetCurrentTask(ctx, req.Private)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if task != nil {
		var pollingModel remoteFilePollingModel
		if diags := oldModel.Polling.As(ctx, &pollingModel, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		taskType := models.TaskType(task.Type.ValueString())

		if !recreate && taskType == models.TaskTypeVirtualDisk {
			var polling models.Polling

			if diags := pollingModel.Create.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			tflog.Info(ctx, "Waiting for the previous download task to complete", map[string]interface{}{
				"task.id": task.ID.ValueInt64(),
			})

			if diags := WaitForTask(ctx, v.client, taskType, task.ID.ValueInt64(), &polling); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		}

		if err := stopAndDeleteTask(ctx, v.client, taskType, task.ID.ValueInt64()); err != nil {
			resp.Diagnostics.AddError("Failed to stop and delete task", fmt.Sprintf("Task %q ID: %d, Error: %s", task.Type.ValueString(), task.ID.ValueInt64(), err.Error()))
			return
		}

		if diags := providerdata.UnsetCurrentTask(ctx, resp.Private); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	newModel.populateDefaults()

	defer func() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &newModel)...)
	}()

	if recreate {
		// Delete the old disk and create a new one

		var polling virtualDiskPollingModel

		if diags := newModel.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var deletePolling models.Polling

		if diags := polling.Delete.As(ctx, &deletePolling, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		if diags := deleteFilesIfExist(ctx, resp.Private, v.client, deletePolling, oldModel.Path.ValueString()); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		taskID, err := v.client.CreateVirtualDisk(ctx, newModel.toCreatePayload())
		if err != nil {
			resp.Diagnostics.AddError("Failed to add creation task", fmt.Sprintf("Path: %s, Error: %s", newModel.Path.ValueString(), err.Error()))
			return
		}

		if diags := providerdata.SetCurrentTask(ctx, resp.Private, models.TaskTypeVirtualDisk, taskID); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		if diags := waitForDiskTask(ctx, v.client, taskID); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		diskInfo, err := v.client.GetVirtualDiskInfo(ctx, newModel.Path.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to get disk info", fmt.Sprintf("Path: %s, Error: %s", newModel.Path.ValueString(), err.Error()))
			return
		}

		newModel.populateFromVirtualDiskInfo(diskInfo)

		// Disk fully recreated, no need to continue
		return
	}

	if oldModel.Path.ValueString() != newModel.Path.ValueString() { // Move the disk
		task, err := v.client.MoveFiles(ctx, []string{oldModel.Path.ValueString()}, newModel.Path.ValueString(), freeboxTypes.FileMoveModeSkip)
		if err != nil {
			resp.Diagnostics.AddError("Failed to add move task", fmt.Sprintf("Path: %s, Error: %s", newModel.Path.ValueString(), err.Error()))
			return
		}

		if diags := providerdata.SetCurrentTask(ctx, resp.Private, models.TaskTypeFileSystem, task.ID); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var polling virtualDiskPollingModel

		if diags := newModel.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var movePolling models.Polling

		if diags := polling.Move.As(ctx, &movePolling, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		if diags := waitForFileSystemTask(ctx, v.client, task.ID, movePolling); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		if err := stopAndDeleteFileSystemTask(ctx, v.client, task.ID); err != nil {
			resp.Diagnostics.AddError("Failed to stop and delete file system task", fmt.Sprintf("Task: %d, Error: %s", task.ID, err.Error()))
			return
		}

		if diags := providerdata.UnsetCurrentTask(ctx, resp.Private); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		// Do not return here, continue with resizing the disk if needed
	}

	if oldModel.VirtualSize.ValueInt64() != newModel.VirtualSize.ValueInt64() { // Resize the disk
		taskID, err := v.client.ResizeVirtualDisk(ctx, newModel.toResizePayload())
		if err != nil {
			resp.Diagnostics.AddError("Failed to add resize task", fmt.Sprintf("Path: %s, Error: %s", newModel.Path.ValueString(), err.Error()))
			return
		}

		if diags := providerdata.SetCurrentTask(ctx, resp.Private, models.TaskTypeVirtualDisk, taskID); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		if diags := waitForDiskTask(ctx, v.client, taskID); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		if err := stopAndDeleteTask(ctx, v.client, models.TaskTypeVirtualDisk, taskID); err != nil {
			resp.Diagnostics.AddError("Failed to delete virtual disk task", err.Error())
			return
		}

		if diags := providerdata.UnsetCurrentTask(ctx, resp.Private); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		// Disk resized, update the model

		diskInfo, err := v.client.GetVirtualDiskInfo(ctx, newModel.Path.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to get disk info", fmt.Sprintf("Path: %s, Error: %s", newModel.Path.ValueString(), err.Error()))
			return
		}

		newModel.populateFromVirtualDiskInfo(diskInfo)
	}
}

func (v *virtualDiskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model virtualDiskModel

	if diags := req.State.Get(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	task, diags := providerdata.GetCurrentTask(ctx, resp.Private)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if task != nil {
		taskID := task.ID.ValueInt64()
		taskType := task.Type.ValueString()

		ctx := tflog.SetField(ctx, "task.id", taskID)
		ctx = tflog.SetField(ctx, "task.type", taskType)

		if err := stopAndDeleteTask(ctx, v.client, models.TaskType(taskType), taskID); err != nil {
			resp.Diagnostics.AddWarning("Failed to delete virtual disk task", err.Error())
		}
	}

	var polling virtualDiskPollingModel

	if diags := model.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var deletePolling models.Polling

	if diags := polling.Delete.As(ctx, &deletePolling, basetypes.ObjectAsOptions{}); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if diags := deleteFilesIfExist(ctx, resp.Private, v.client, deletePolling, model.Path.ValueString()); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
}

func (v *virtualDiskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err == nil {
		resp.Diagnostics.AddError("Invalid ID", "Import by task ID is not supported yet")
		return
	}

	var model virtualDiskModel

	model.populateDefaults()

	defer func() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}()

	model.Path = basetypes.NewStringValue(req.ID)

	disk, err := v.client.GetVirtualDiskInfo(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get virtual disk info",err.Error())
		return
	}

	model.populateFromVirtualDiskInfo(disk)
}
