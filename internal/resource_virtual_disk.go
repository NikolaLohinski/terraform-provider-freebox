package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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
	// ResizeFrom is the path to the virtual disk to resize from.
	ResizeFrom types.String `tfsdk:"resize_from"`
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

func (v *virtualDiskModel) populateFromFileInfo(fileInfo freeboxTypes.FileInfo) {
	v.Path = basetypes.NewStringValue(string(fileInfo.Path))
	v.SizeOnDisk = basetypes.NewInt64Value(int64(fileInfo.SizeBytes))
}

func (v *virtualDiskModel) populateDefaults() {
	v.SizeOnDisk = basetypes.NewInt64Null()

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
	payload.ShrinkAllow = true // Shrink is always allowed

	return
}

type virtualDiskPollingModel struct {
	// Checksum is the polling configuration for checksum compute operation.
	Checksum types.Object `tfsdk:"checksum"`
	// Copy is the polling configuration for copy operation.
	Copy types.Object `tfsdk:"copy"`
	// Create is the polling configuration for create operation.
	Create types.Object `tfsdk:"create"`
	// Delete is the polling configuration for delete operation.
	Delete types.Object `tfsdk:"delete"`
	// Move is the polling configuration for move operation.
	Move types.Object `tfsdk:"move"`
	// Resize is the polling configuration for resize operation.
	Resize types.Object `tfsdk:"resize"`
}

func (v virtualDiskPollingModel) defaults() basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(virtualDiskPollingModel{}.AttrTypes(), map[string]attr.Value{
		"checksum": models.NewPollingSpecModel(time.Second, time.Minute),
		"copy":     models.NewPollingSpecModel(2*time.Second, 2*time.Minute),
		"create":   models.NewPollingSpecModel(time.Second, time.Minute),
		"delete":   models.NewPollingSpecModel(time.Second, time.Minute),
		"move":     models.NewPollingSpecModel(time.Second, time.Minute),
		"resize":   models.NewPollingSpecModel(time.Second, time.Minute),
	})
}

func (v virtualDiskPollingModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"checksum": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Polling configuration for checksum compute operation",
			Attributes:          models.PollingSpecModelResourceAttributes(time.Second, time.Minute),
			Default:             objectdefault.StaticValue(models.NewPollingSpecModel(time.Second, time.Minute)),
		},
		"copy": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Polling configuration for copy operation",
			Attributes:          models.PollingSpecModelResourceAttributes(time.Second, time.Minute),
			Default:             objectdefault.StaticValue(models.NewPollingSpecModel(time.Second, time.Minute)),
		},
		"create": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Polling configuration for create operation",
			Attributes:          models.PollingSpecModelResourceAttributes(time.Second, time.Minute),
			Default:             objectdefault.StaticValue(models.NewPollingSpecModel(time.Second, time.Minute)),
		},
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
		"resize": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Polling configuration for resize operation",
			Attributes:          models.PollingSpecModelResourceAttributes(time.Second, time.Minute),
			Default:             objectdefault.StaticValue(models.NewPollingSpecModel(time.Second, time.Minute)),
		},
	}
}

func (v virtualDiskPollingModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"checksum": types.ObjectType{}.WithAttributeTypes(models.Polling{}.AttrTypes()),
		"copy":     types.ObjectType{}.WithAttributeTypes(models.Polling{}.AttrTypes()),
		"create":   types.ObjectType{}.WithAttributeTypes(models.Polling{}.AttrTypes()),
		"move":     types.ObjectType{}.WithAttributeTypes(models.Polling{}.AttrTypes()),
		"delete":   types.ObjectType{}.WithAttributeTypes(models.Polling{}.AttrTypes()),
		"resize":   types.ObjectType{}.WithAttributeTypes(models.Polling{}.AttrTypes()),
	}
}

func (v *virtualDiskResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_disk"
}

func (v *virtualDiskResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a virtual disk image within a Freebox",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				MarkdownDescription: "Path to the virtual disk on the Freebox",
				Required:            true,
				Validators: []validator.String{
					models.FilePathValidator(),
				},
			},
			"type": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "Type of virtual disk. If not specified, the type will be inferred from the resize from file or be set to qcow2",
				Validators: []validator.String{
					models.DiskTypeValidator(),
					stringvalidator.ConflictsWith(path.MatchRoot("resize_from")),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resize_from": schema.StringAttribute{
				MarkdownDescription: "Path to the virtual disk to resize from",
				Optional:            true,
				Computed:            false,
				Validators: []validator.String{
					models.FilePathValidator(),
					stringvalidator.ConflictsWith(path.MatchRoot("type")),
				},
			},
			"virtual_size": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "Size in bytes of virtual disk. This is the size the disk will appear inside the VM.",
				Validators: []validator.Int64{
					models.VirtualDiskSizeValidator(),
					int64validator.AtLeast(4_096),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"size_on_disk": schema.Int64Attribute{
				Computed:            true,
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

	model.populateDefaults()

	defer func() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}()

	if diags := v.create(ctx, &model, resp.Private); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
}

func (v *virtualDiskResource) create(ctx context.Context, model *virtualDiskModel, private providerdata.Setter) (diagnostics diag.Diagnostics) {
	if !model.ResizeFrom.IsNull() {
		if diags := v.createFromExistingDisk(ctx, model.ResizeFrom.ValueString(), model, private); diags.HasError() {
			diagnostics.Append(diags...)
			return
		}
	} else {
		if diags := v.createFromScratch(ctx, model, private); diags.HasError() {
			diagnostics.Append(diags...)
			return
		}
	}

	diskInfo, err := v.client.GetVirtualDiskInfo(ctx, model.Path.ValueString())
	if err != nil {
		diagnostics.AddError("Failed to get disk info", fmt.Sprintf("Path: %s, Error: %s", model.Path.ValueString(), err.Error()))
		return
	}

	model.populateFromVirtualDiskInfo(diskInfo)

	return
}

func (v *virtualDiskResource) createFromExistingDisk(ctx context.Context, resizeFrom string, model *virtualDiskModel, private providerdata.Setter) (diagnostics diag.Diagnostics) {
	var polling virtualDiskPollingModel
	if diags := model.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	// Start the hash task

	hashTask, err := v.client.AddHashFileTask(ctx, freeboxTypes.HashPayload{
		HashType: freeboxTypes.HashTypeSHA512,
		Path:     freeboxTypes.Base64Path(resizeFrom),
	})
	if err != nil {
		diagnostics.AddError("Failed to add hash file task", fmt.Sprintf("Source: %s, Error: %s", resizeFrom, err.Error()))
		return
	}

	// Note: No need to store the task id as we need to restart the full process if the hash is not the same

	tflog.Debug(ctx, "Computing checksum of the existing virtual disk", map[string]interface{}{
		"path":      resizeFrom,
		"task.id":   hashTask.ID,
		"task.type": models.TaskTypeFileSystem,
	})

	// Start the copy task

	copyTask, err := v.client.CopyFiles(ctx, []string{resizeFrom}, model.Path.ValueString(), freeboxTypes.FileCopyModeOverwrite)
	if err != nil {
		diagnostics.AddError("Failed to copy virtual disk", fmt.Sprintf("Source: %s, Destination: %s, Error: %s", resizeFrom, model.Path.ValueString(), err.Error()))
		return
	}

	tflog.Debug(ctx, "Copying virtual disk", map[string]interface{}{
		"source":    resizeFrom,
		"path":      model.Path.ValueString(),
		"task.id":   copyTask.ID,
		"task.type": models.TaskTypeFileSystem,
	})

	if diags := providerdata.SetCurrentTask(ctx, private, models.TaskTypeFileSystem, copyTask.ID); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	var copyPolling models.Polling
	if diags := polling.Copy.As(ctx, &copyPolling, basetypes.ObjectAsOptions{}); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if diags := waitForFileSystemTask(ctx, v.client, copyTask.ID, copyPolling); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if err := stopAndDeleteFileSystemTask(ctx, v.client, copyTask.ID); err != nil {
		diagnostics.AddError("Failed to stop and delete file system task", fmt.Sprintf("Task: %d, Error: %s", copyTask.ID, err.Error()))
		return
	}

	if diags := providerdata.UnsetCurrentTask(ctx, private); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	// Wait for the hash task to complete

	var checksumPolling models.Polling
	if diags := polling.Checksum.As(ctx, &checksumPolling, basetypes.ObjectAsOptions{}); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if diags := waitForFileSystemTask(ctx, v.client, hashTask.ID, checksumPolling); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	hash, err := v.client.GetHashResult(ctx, hashTask.ID)
	if err != nil {
		diagnostics.AddError("Failed to get hash result", fmt.Sprintf("Task: %d, Error: %s", hashTask.ID, err.Error()))
		return
	}

	hashJson, err := json.Marshal(hash)
	if err != nil {
		diagnostics.AddError("Failed to marshal hash", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	if diags := private.SetKey(ctx, "resize_from_checksum", hashJson); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if err := stopAndDeleteFileSystemTask(ctx, v.client, hashTask.ID); err != nil {
		diagnostics.AddError("Failed to stop and delete file system task", fmt.Sprintf("Task: %d, Error: %s", hashTask.ID, err.Error()))
		return
	}

	// Start the resize task

	resizeTaskID, err := v.client.ResizeVirtualDisk(ctx, freeboxTypes.VirtualDisksResizePayload{
		DiskPath:    freeboxTypes.Base64Path(model.Path.ValueString()),
		NewSize:     model.VirtualSize.ValueInt64() - 4_096, // The freebox API automatically adds 4KB to the size
		ShrinkAllow: true,
	})
	if err != nil {
		diagnostics.AddError("Failed to resize virtual disk", fmt.Sprintf("Path: %s, Error: %s", resizeFrom, err.Error()))
		return
	}

	tflog.Debug(ctx, "Resizing virtual disk", map[string]interface{}{
		"path":     model.Path.ValueString(),
		"new_size": model.VirtualSize.ValueInt64(),
	})

	if diags := providerdata.SetCurrentTask(ctx, private, models.TaskTypeVirtualDisk, resizeTaskID); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if diags := model.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	var resizePolling models.Polling
	if diags := polling.Resize.As(ctx, &resizePolling, basetypes.ObjectAsOptions{}); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if diags := waitForDiskTask(ctx, v.client, resizeTaskID, resizePolling.Timeout); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if err := stopAndDeleteTask(ctx, v.client, models.TaskTypeVirtualDisk, resizeTaskID); err != nil {
		diagnostics.AddError("Failed to delete virtual disk task", fmt.Sprintf("Task: %d, Error: %s", resizeTaskID, err.Error()))
		return
	}

	if diags := providerdata.UnsetCurrentTask(ctx, private); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	return
}

func (v *virtualDiskResource) createFromScratch(ctx context.Context, model *virtualDiskModel, private providerdata.Setter) (diagnostics diag.Diagnostics) {
	if model.Type.IsUnknown() || model.Type.IsNull() {
		model.Type = basetypes.NewStringValue(freeboxTypes.QCow2Disk)
	}

	taskID, err := v.client.CreateVirtualDisk(ctx, model.toCreatePayload())
	if err != nil {
		diagnostics.AddError("Failed to create virtual disk", fmt.Sprintf("Path: %s, Error: %s", model.Path.ValueString(), err.Error()))
		return
	}

	if diags := providerdata.SetCurrentTask(ctx, private, models.TaskTypeVirtualDisk, taskID); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	var polling virtualDiskPollingModel

	if diags := model.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	var createPolling models.Polling

	if diags := polling.Create.As(ctx, &createPolling, basetypes.ObjectAsOptions{}); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if diags := waitForDiskTask(ctx, v.client, taskID, createPolling.Timeout); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if err := stopAndDeleteTask(ctx, v.client, models.TaskTypeVirtualDisk, taskID); err != nil {
		diagnostics.AddError("Failed to delete virtual disk task", fmt.Sprintf("Task: %d, Error: %s", taskID, err.Error()))
		return
	}

	if diags := providerdata.UnsetCurrentTask(ctx, private); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	return
}

func (v *virtualDiskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model virtualDiskModel

	if diags := req.State.Get(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	model.populateDefaults()

	var removed bool

	defer func() {
		if !removed {
			resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
		}
	}()

	diskPath := model.Path.ValueString()

	diskInfo, err := v.client.GetVirtualDiskInfo(ctx, diskPath)
	if err == nil {
		model.populateFromVirtualDiskInfo(diskInfo)
		return
	}

	tflog.Info(ctx, "Failed to get virtual disk info", map[string]interface{}{
		"error": err,
	})

	target := new(client.APIError)
	if errors.As(err, &target) {
		switch target.Code {
		case freeboxTypes.DiskErrorNotFound:
			tflog.Debug(ctx, "Virtual disk is not found, removing the resource from the state", map[string]interface{}{
				"error": err,
			})

			removed = true
			resp.State.RemoveResource(ctx)
			return
		case freeboxTypes.DiskErrorInfo:
			resp.Diagnostics.AddWarning("Failed to get virtual disk, It is probably in use", fmt.Sprintf("Path: %s, Error: %s", diskPath, err.Error()))
			return
		default:
			tflog.Warn(ctx, "Failed to get disk info", map[string]interface{}{
				"error": err,
			})

			fileInfo, err := v.client.GetFileInfo(ctx, diskPath)
			if err != nil {
				resp.Diagnostics.AddError("Failed to get file info", fmt.Sprintf("Path: %s, Error: %s", diskPath, err.Error()))
				return
			}

			model.populateFromFileInfo(fileInfo)
		}
	}
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
		var pollingModel virtualDiskPollingModel
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

			tflog.Info(ctx, "Waiting for the previous create task to complete", map[string]interface{}{
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

	if !recreate { // Compare the checksum of the resize from file only if we are not recreating the disk
		if hashJson, diags := req.Private.GetKey(ctx, "resize_from_checksum"); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		} else if hashJson != nil {
			var resizeFromChecksum string
			if err := json.Unmarshal(hashJson, &resizeFromChecksum); err != nil {
				resp.Diagnostics.AddError("Failed to unmarshal resize from checksum", fmt.Sprintf("Error: %s", err.Error()))
				return
			}

			// Compute the checksum of the new resize from file

			resizeFromPath := newModel.ResizeFrom.ValueString()

			hashTask, err := v.client.AddHashFileTask(ctx, freeboxTypes.HashPayload{
				HashType: freeboxTypes.HashTypeSHA512,
				Path:     freeboxTypes.Base64Path(resizeFromPath),
			})
			if err != nil {
				resp.Diagnostics.AddError("Failed to add hash file task", fmt.Sprintf("Source: %s, Error: %s", resizeFromPath, err.Error()))
				return
			}

			var polling virtualDiskPollingModel
			if diags := newModel.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			var checksumPolling models.Polling
			if diags := polling.Checksum.As(ctx, &checksumPolling, basetypes.ObjectAsOptions{}); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			if diags := waitForFileSystemTask(ctx, v.client, hashTask.ID, checksumPolling); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			hash, err := v.client.GetHashResult(ctx, hashTask.ID)
			if err != nil {
				resp.Diagnostics.AddError("Failed to get hash result", fmt.Sprintf("Task: %d, Error: %s", hashTask.ID, err.Error()))
				return
			}

			if err := stopAndDeleteFileSystemTask(ctx, v.client, hashTask.ID); err != nil {
				resp.Diagnostics.AddError("Failed to stop and delete file system task", fmt.Sprintf("Task: %d, Error: %s", hashTask.ID, err.Error()))
				return
			}

			recreate = resizeFromChecksum != hash // Recreate the disk if the checksum is different
		}
	}

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

		if diags := v.create(ctx, &newModel, resp.Private); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

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
		tflog.Info(ctx, "Resizing virtual disk", map[string]interface{}{
			"old_size": oldModel.VirtualSize.ValueInt64(),
			"new_size": newModel.VirtualSize.ValueInt64(),
		})

		taskID, err := v.client.ResizeVirtualDisk(ctx, newModel.toResizePayload())
		if err != nil {
			resp.Diagnostics.AddError("Failed to add resize task", fmt.Sprintf("Path: %s, Error: %s", newModel.Path.ValueString(), err.Error()))
			return
		}

		if diags := providerdata.SetCurrentTask(ctx, resp.Private, models.TaskTypeVirtualDisk, taskID); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var polling virtualDiskPollingModel

		if diags := newModel.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var resizePolling models.Polling

		if diags := polling.Resize.As(ctx, &resizePolling, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		if diags := waitForDiskTask(ctx, v.client, taskID, resizePolling.Timeout); diags.HasError() {
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
		resp.Diagnostics.AddError("Failed to get virtual disk info", err.Error())
		return
	}

	model.populateFromVirtualDiskInfo(disk)
}
