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
	_ resource.Resource                = (*remoteFileResource)(nil)
	_ resource.ResourceWithImportState = (*remoteFileResource)(nil)
)

func NewRemoteFileResource() resource.Resource {
	return &remoteFileResource{}
}

// remoteFileResource defines the resource implementation.
type remoteFileResource struct {
	client client.Client
}

// remoteFileModel describes the resource data model.
type remoteFileModel struct {
	// DestinationPath is the file path on the Freebox.
	DestinationPath types.String `tfsdk:"destination_path"`
	// SourceURL is the file URL.
	SourceURL types.String `tfsdk:"source_url"`
	// Checksum is the file checksum.
	// Verify the hash of the downloaded file.
	// The format is sha256:xxxxxx or sha512:xxxxxx; or the URL of a SHA256SUMS, SHA512SUMS, -CHECKSUM or .sha256 file.
	Checksum types.String `tfsdk:"checksum"`

	// Authentication is the credentials to use for the operation.
	Authentication types.Object `tfsdk:"authentication"`

	// Polling is the polling configuration.
	Polling types.Object `tfsdk:"polling"`
}

func (o remoteFileModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"destination_path": types.StringType,
		"source_url":       types.StringType,
		"checksum":         types.StringType,
		"authentication":   types.ObjectType{}.WithAttributeTypes(remoteFileModelAuthenticationsModel{}.AttrTypes()),
		"polling":          types.ObjectType{}.WithAttributeTypes(remoteFilePollingModel{}.AttrTypes()),
	}
}

type remoteFilePollingModel struct {
	Delete          types.Object `tfsdk:"delete"`
	Move            types.Object `tfsdk:"move"`
	Create          types.Object `tfsdk:"create"`
	ChecksumCompute types.Object `tfsdk:"checksum_compute"`
}

func (o remoteFilePollingModel) defaults() basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(remoteFilePollingModel{}.AttrTypes(), map[string]attr.Value{
		"create":           models.NewPollingSpecModel(3*time.Second, 30*time.Minute),
		"move":             models.NewPollingSpecModel(time.Second, time.Minute),
		"delete":           models.NewPollingSpecModel(time.Second, time.Minute),
		"checksum_compute": models.NewPollingSpecModel(time.Second, 2*time.Minute),
	})
}

func (o remoteFilePollingModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"create": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Creation polling configuration",
			Attributes:          models.PollingSpecModelResourceAttributes(3*time.Second, 30*time.Minute),
			Default:             objectdefault.StaticValue(models.NewPollingSpecModel(3*time.Second, 30*time.Minute)),
		},
		"move": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Move polling configuration",
			Attributes:          models.PollingSpecModelResourceAttributes(time.Second, time.Minute),
			Default:             objectdefault.StaticValue(models.NewPollingSpecModel(time.Second, time.Minute)),
		},
		"delete": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Deletion polling configuration",
			Attributes:          models.PollingSpecModelResourceAttributes(time.Second, time.Minute),
			Default:             objectdefault.StaticValue(models.NewPollingSpecModel(time.Second, time.Minute)),
		},
		"checksum_compute": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Checksum compute polling configuration",
			Attributes:          models.PollingSpecModelResourceAttributes(time.Second, 2*time.Minute),
			Default:             objectdefault.StaticValue(models.NewPollingSpecModel(time.Second, 2*time.Minute)),
		},
	}
}

func (o remoteFilePollingModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"create":           types.ObjectType{}.WithAttributeTypes(models.Polling{}.AttrTypes()),
		"move":             types.ObjectType{}.WithAttributeTypes(models.Polling{}.AttrTypes()),
		"delete":           types.ObjectType{}.WithAttributeTypes(models.Polling{}.AttrTypes()),
		"checksum_compute": types.ObjectType{}.WithAttributeTypes(models.Polling{}.AttrTypes()),
	}
}

type remoteFileModelAuthenticationsModel struct {
	// BasicAuth is the basic authentication credentials.
	BasicAuth types.Object `tfsdk:"basic_auth"`
}

func (o remoteFileModelAuthenticationsModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"basic_auth": schema.SingleNestedAttribute{
			Computed:            true,
			Optional:            true,
			MarkdownDescription: "Basic authentication credentials",
			Attributes:          remoteFileModelAuthenticationsBasicAuthModel{}.ResourceAttributes(),
		},
	}
}

func (o remoteFileModelAuthenticationsModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"basic_auth": types.ObjectType{AttrTypes: remoteFileModelAuthenticationsBasicAuthModel{}.AttrTypes()},
	}
}

func (o remoteFileModelAuthenticationsModel) defaults() basetypes.ObjectValue {
	return basetypes.NewObjectNull(remoteFileModelAuthenticationsModel{}.AttrTypes())
}

type remoteFileModelAuthenticationsBasicAuthModel struct {
	// Username is the username.
	Username types.String `tfsdk:"username"`
	// Password is the password.
	Password types.String `tfsdk:"password"`
}

func (o remoteFileModelAuthenticationsBasicAuthModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"username": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Username",
		},
		"password": schema.StringAttribute{
			Optional:            true,
			Sensitive:           true,
			MarkdownDescription: "Password",
		},
	}
}

func (o remoteFileModelAuthenticationsBasicAuthModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"username": types.StringType,
		"password": types.StringType,
	}
}

func (v *remoteFileModel) populateDefaults(ctx context.Context) (diagnostics diag.Diagnostics) {
	if v.Authentication.IsUnknown() {
		v.Authentication = remoteFileModelAuthenticationsModel{}.defaults()
	}
	if v.Authentication.IsUnknown() || v.Authentication.IsNull() {
		v.Authentication = remoteFileModelAuthenticationsModel{}.defaults()
	}
	if v.Polling.IsUnknown() || v.Polling.IsNull() {
		v.Polling = remoteFilePollingModel{}.defaults()
	} else {
		var polling remoteFilePollingModel

		if diags := v.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
			diagnostics.Append(diags...)
			return
		}

		if polling.Create.IsUnknown() {
			polling.Create = models.NewPollingSpecModel(3*time.Second, 30*time.Minute)
		}
		if polling.Move.IsUnknown() {
			polling.Move = models.NewPollingSpecModel(time.Second, time.Minute)
		}
		if polling.Delete.IsUnknown() {
			polling.Delete = models.NewPollingSpecModel(time.Second, time.Minute)
		}
		if polling.ChecksumCompute.IsUnknown() {
			polling.ChecksumCompute = models.NewPollingSpecModel(time.Second, 2*time.Minute)
		}
	}
	return
}

func (v *remoteFileModel) populateFromFileInfo(fileInfo freeboxTypes.FileInfo) diag.Diagnostics {
	v.DestinationPath = basetypes.NewStringValue(string(fileInfo.Path))
	return nil
}

func (v *remoteFileModel) toDownloadPayload() (payload freeboxTypes.DownloadRequest, diagnostics diag.Diagnostics) {
	payload.DownloadURLs = []string{v.SourceURL.ValueString()}
	payload.Hash = v.Checksum.ValueString()

	destinationPath := v.DestinationPath.ValueString()
	payload.DownloadDirectory = go_path.Dir(destinationPath)
	payload.Filename = go_path.Base(destinationPath)

	if !v.Authentication.IsNull() {
		var authentication *remoteFileModelAuthenticationsModel

		if diags := v.Authentication.As(context.Background(), &authentication, basetypes.ObjectAsOptions{}); diags.HasError() {
			diagnostics.Append(diags...)
			return
		}

		if !authentication.BasicAuth.IsUnknown() {
			var basicAuth remoteFileModelAuthenticationsBasicAuthModel
			if diags := authentication.BasicAuth.As(context.Background(), &basicAuth, basetypes.ObjectAsOptions{}); diags.HasError() {
				diagnostics.Append(diags...)
				return
			}

			if username := basicAuth.Username.ValueString(); username != "" {
				payload.Username = username
			}

			if password := basicAuth.Password.ValueString(); password != "" {
				payload.Password = password
			}
		}
	}

	return
}

func (v *remoteFileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_remote_file"
}

func (v *remoteFileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource downloads a file from a URL and stores it on the Freebox.",
		Attributes: map[string]schema.Attribute{
			"destination_path": schema.StringAttribute{
				MarkdownDescription: "Path to the file on the Freebox",
				Required:            true,
				Validators: []validator.String{
					models.FilePathValidator(),
				},
			},
			"source_url": schema.StringAttribute{
				MarkdownDescription: "The URL of the file to download",
				Required:            true,
				Validators: []validator.String{
					models.DownloadURLValidator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(func(ctx context.Context, sr planmodifier.StringRequest, rrifr *stringplanmodifier.RequiresReplaceIfFuncResponse) {
						var checksum basetypes.StringValue
						rrifr.Diagnostics.Append(sr.Plan.GetAttribute(ctx, path.Root("checksum"), &checksum)...)
						rrifr.RequiresReplace = rrifr.RequiresReplace || checksum.IsNull() || checksum.IsUnknown()
					}, "", "Replace the remote file if the checksum not defined"),
				},
			},
			"checksum": schema.StringAttribute{
				MarkdownDescription: "Checksum to verify the hash of the downloaded file",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					models.ChecksumValidator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"authentication": schema.SingleNestedAttribute{
				MarkdownDescription: "Authentication credentials to use for the operation",
				Optional:            true,
				Computed:            true,
				Attributes:          remoteFileModelAuthenticationsModel{}.ResourceAttributes(),
				Default:             objectdefault.StaticValue(basetypes.NewObjectNull(remoteFileModelAuthenticationsModel{}.AttrTypes())),
			},
			"polling": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Polling configuration",
				Attributes:          remoteFilePollingModel{}.ResourceAttributes(),
				Default:             objectdefault.StaticValue(remoteFilePollingModel{}.defaults()),
			},
		},
	}
}

func (v *remoteFileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (v *remoteFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model remoteFileModel

	if diags := req.Plan.Get(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Checking if the file already exists...")

	if _, err := v.client.GetFileInfo(ctx, model.DestinationPath.ValueString()); err == nil {
		resp.Diagnostics.AddError("File already exists", fmt.Sprintf("Please delete the file %q or import it into the state", model.DestinationPath.ValueString()))
		return
	} else if !errors.Is(err, client.ErrPathNotFound) {
		resp.Diagnostics.AddError("Failed to check file existence", fmt.Sprintf("Path: %s, Error: %s", model.DestinationPath.ValueString(), err.Error()))
		return
	}

	defer func() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}()

	if diags := v.createFromURL(ctx, resp.Private, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	fileInfo, err := v.client.GetFileInfo(ctx, model.DestinationPath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get file", fmt.Sprintf("Path: %s, Error: %s", model.DestinationPath.ValueString(), err.Error()))
		return
	}

	if diags := model.populateFromFileInfo(fileInfo); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	tflog.Debug(ctx, "Verifying the checksum...")

	hMethod, expected := hashSpec(model.Checksum.ValueString())

	result, diags := model.fileChecksum(ctx, resp.Private, v.client, model.DestinationPath.ValueString(), hMethod)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if expected == "" {
		model.setChecksum(hMethod, result)
	} else if expected != result {
		resp.Diagnostics.AddError("Checksum mismatch", fmt.Sprintf("Expected checksum %q, got for %q, Path: %s", expected, result, model.DestinationPath.ValueString()))
		return
	}
}

func (v *remoteFileResource) createFromURL(ctx context.Context, state providerdata.Setter, model *remoteFileModel) (diagnostics diag.Diagnostics) {
	payload, diags := model.toDownloadPayload()
	if diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	taskID, err := v.client.AddDownloadTask(ctx, payload)
	if err != nil {
		diagnostics.AddError("Failed to add download task", err.Error())
		return
	}

	tflog.Debug(ctx, "Start downloading file", map[string]interface{}{
		"task.id": taskID,
	})

	if diags := providerdata.SetCurrentTask(ctx, state, models.TaskTypeDownload, taskID); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	var polling remoteFilePollingModel

	if diags := model.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	var createPolling models.Polling

	if diags := polling.Create.As(ctx, &createPolling, basetypes.ObjectAsOptions{}); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if diags := waitForDownloadTask(ctx, v.client, taskID, createPolling); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	tflog.Info(ctx, "Download task completed", map[string]interface{}{
		"task.id": taskID,
	})

	if err := stopAndDeleteDownloadTask(ctx, v.client, taskID); err != nil {
		diagnostics.AddError("Failed to stop and delete download task", fmt.Sprintf("Task %d, Error: %s", taskID, err.Error()))
		return
	}

	if diags := providerdata.UnsetCurrentTask(ctx, state); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	return
}

func (v *remoteFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model remoteFileModel

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
			tflog.Info(ctx, "File not found", map[string]interface{}{
				"path":  model.DestinationPath.ValueString(),
				"error": err.Error(),
			})

			task, diags := providerdata.GetCurrentTask(ctx, resp.Private)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
			if task == nil { // File is not found and no task is running
				tflog.Debug(ctx, "No task is running")
				resp.State.RemoveResource(ctx)
				return
			}

			var pollingModel remoteFilePollingModel
			if diags := model.Polling.As(ctx, &pollingModel, basetypes.ObjectAsOptions{}); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			var taskPolling types.Object = types.ObjectNull(models.Polling{}.AttrTypes())
			switch models.TaskType(task.Type.ValueString()) {
			case models.TaskTypeDownload:
				taskPolling = pollingModel.Create
			case models.TaskTypeFileSystem:
				taskPolling = pollingModel.Delete
			default:
				resp.Diagnostics.AddWarning("Unknown task type", fmt.Sprintf("Task %q ID: %d", task.Type.ValueString(), task.ID.ValueInt64()))
				providerdata.UnsetCurrentTask(ctx, resp.Private)
				resp.State.RemoveResource(ctx)
				return
			}

			var polling models.Polling
			if diags := taskPolling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			tflog.Info(ctx, "Waiting for the previous download task to complete", map[string]interface{}{
				"task.id": task.ID.ValueInt64(),
			})

			taskType := models.TaskType(task.Type.ValueString())

			if diags := WaitForTask(ctx, v.client, taskType, task.ID.ValueInt64(), &polling); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			tflog.Debug(ctx, "Getting the file info...", map[string]interface{}{
				"path": model.DestinationPath.ValueString(),
			})

			fileInfo, err = v.client.GetFileInfo(ctx, model.DestinationPath.ValueString())
			if err != nil && errors.Is(err, client.ErrPathNotFound) {
				resp.State.RemoveResource(ctx)
				return
			}
		} else {
			resp.Diagnostics.AddError("Failed to get file", fmt.Sprintf("Path: %s, Error: %s", model.DestinationPath.ValueString(), err.Error()))
			return
		}
	}

	defer func() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}()

	if diags := model.populateFromFileInfo(fileInfo); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	tflog.Debug(ctx, "Verifying the checksum...")

	hMethod, _ := hashSpec(model.Checksum.ValueString())

	hash, diags := model.fileChecksum(ctx, resp.Private, v.client, model.DestinationPath.ValueString(), hMethod)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	model.setChecksum(hMethod, hash)
}

func (v *remoteFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var oldModel, newModel remoteFileModel

	resp.Diagnostics.Append(req.State.Get(ctx, &oldModel)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &newModel)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if diags := newModel.populateDefaults(ctx); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var polling remoteFilePollingModel

	if diags := newModel.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	defer func() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &newModel)...)
	}()

	recreate := !oldModel.SourceURL.Equal(newModel.SourceURL)

	if !oldModel.Checksum.IsNull() && !oldModel.Checksum.IsUnknown() &&
		!newModel.Checksum.IsNull() && !newModel.Checksum.IsUnknown() {
		recreate = !oldModel.Checksum.Equal(newModel.Checksum)
	}

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

		if !recreate && taskType == models.TaskTypeDownload {
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

	if recreate {
		tflog.Info(ctx, "Recreating the file...")

		tflog.Debug(ctx, "Deleting the file...")

		var deletePolling models.Polling

		if diags := polling.Delete.As(ctx, &deletePolling, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		if diags := deleteFilesIfExist(ctx, resp.Private, v.client, deletePolling, oldModel.DestinationPath.ValueString()); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		tflog.Debug(ctx, "Downloading the file...")

		payload, diags := newModel.toDownloadPayload()
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		taskID, err := v.client.AddDownloadTask(ctx, payload)
		if err != nil {
			resp.Diagnostics.AddError("Failed to add download task", err.Error())
			return
		}

		if diags := providerdata.SetCurrentTask(ctx, resp.Private, models.TaskTypeDownload, taskID); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var createPolling models.Polling

		if diags := polling.Create.As(ctx, &createPolling, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		if diags := waitForDownloadTask(ctx, v.client, taskID, createPolling); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		if err := stopAndDeleteDownloadTask(ctx, v.client, taskID); err != nil {
			resp.Diagnostics.AddError("Failed to stop and delete download task", err.Error())
			return
		}

		if diags := providerdata.UnsetCurrentTask(ctx, resp.Private); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		tflog.Debug(ctx, "Verifying the checksum...")

		hMethod, expected := hashSpec(newModel.Checksum.ValueString())

		result, diags := newModel.fileChecksum(ctx, resp.Private, v.client, newModel.DestinationPath.ValueString(), hMethod)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		if expected == "" {
			newModel.setChecksum(hMethod, result)
		} else if expected != result {
			resp.Diagnostics.AddError("Checksum mismatch", fmt.Sprintf("Expected checksum %q, got %q, Path: %s", expected, result, newModel.DestinationPath.ValueString()))
			return
		}

		return
	}

	if oldModel.DestinationPath.ValueString() != newModel.DestinationPath.ValueString() {
		tflog.Info(ctx, "Moving the file...")

		tflog.Debug(ctx, "Checking if the file already exists...")

		if _, err := v.client.GetFileInfo(ctx, newModel.DestinationPath.ValueString()); err == nil {
			resp.Diagnostics.AddError("File already exists", "Please delete the file or import it into the state")
			return
		} else if !errors.Is(err, client.ErrPathNotFound) {
			resp.Diagnostics.AddError("Failed to check file existence", fmt.Sprintf("Path: %s, Error: %s", newModel.DestinationPath.ValueString(), err.Error()))
			return
		}

		tflog.Debug(ctx, "Moving the file...")

		task, err := v.client.MoveFiles(ctx, []string{oldModel.DestinationPath.ValueString()}, newModel.DestinationPath.ValueString(), freeboxTypes.FileMoveModeOverwrite)
		if err != nil {
			resp.Diagnostics.AddError("Failed to move file", fmt.Sprintf("From: %s, To: %s, Error: %s", oldModel.DestinationPath.ValueString(), newModel.DestinationPath.ValueString(), err.Error()))
			return
		}

		if diags := providerdata.SetCurrentTask(ctx, resp.Private, models.TaskTypeFileSystem, task.ID); diags.HasError() {
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
			resp.Diagnostics.AddError("Failed to stop and delete file system task", fmt.Sprintf("Task %d, Error: %s", task.ID, err.Error()))
			return
		}

		if diags := providerdata.UnsetCurrentTask(ctx, resp.Private); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		return
	}
}

func (v *remoteFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model remoteFileModel

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

		ctx := tflog.SetField(ctx, "task.id", taskID)
		ctx = tflog.SetField(ctx, "task.type", task.Type.ValueString())

		if task.Type.ValueString() == string(models.TaskTypeDownload) {
			tflog.Debug(ctx, "Stopping the download task...")

			if err := v.client.UpdateDownloadTask(ctx, taskID, freeboxTypes.DownloadTaskUpdate{
				Status: freeboxTypes.DownloadTaskStatusStopped,
			}); err != nil {
				if !errors.Is(err, client.ErrTaskNotFound) {
					resp.Diagnostics.AddError("Failed to stop download task", fmt.Sprintf("Task %d, Error: %s", taskID, err.Error()))
					return
				}
			} else {
				tflog.Info(ctx, "Deleting the download task and its file...")

				if err := v.client.EraseDownloadTask(ctx, taskID); err != nil {
					if !errors.Is(err, client.ErrTaskNotFound) {
						resp.Diagnostics.AddError("Failed to erase download task and its files", fmt.Sprintf("Task %d, Error: %s", taskID, err.Error()))
						return
					}
					task, err := v.client.RemoveFiles(ctx, []string{model.DestinationPath.ValueString()})
					if err != nil {
						resp.Diagnostics.AddError("Failed to remove file", fmt.Sprintf("Path: %s, Error: %s", model.DestinationPath.ValueString(), err.Error()))
						return
					}

					if diags := providerdata.SetCurrentTask(ctx, resp.Private, models.TaskTypeFileSystem, task.ID); diags.HasError() {
						resp.Diagnostics.Append(diags...)
						return
					}

					var polling remoteFilePollingModel

					if diags := model.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
						resp.Diagnostics.Append(diags...)
						return
					}

					var deletePolling models.Polling

					if diags := polling.Delete.As(ctx, &deletePolling, basetypes.ObjectAsOptions{}); diags.HasError() {
						resp.Diagnostics.Append(diags...)
						return
					}

					if diags := waitForFileSystemTask(ctx, v.client, task.ID, deletePolling); diags.HasError() {
						resp.Diagnostics.Append(diags...)
						return
					}

					if err := stopAndDeleteFileSystemTask(ctx, v.client, task.ID); err != nil {
						resp.Diagnostics.AddError("Failed to stop and delete file system task", fmt.Sprintf("Task %d, Error: %s", task.ID, err.Error()))
						return
					}

					if diags := providerdata.UnsetCurrentTask(ctx, resp.Private); diags.HasError() {
						resp.Diagnostics.Append(diags...)
						return
					}
				}
			}
		} else {
			tflog.Info(ctx, "Deleting the task...")

			if err := stopAndDeleteTask(ctx, v.client, models.TaskTypeFileSystem, taskID); err != nil {
				resp.Diagnostics.AddError("Failed to stop and delete task", fmt.Sprintf("Task %d, Error: %s", taskID, err.Error()))
				return
			}

			if diags := providerdata.UnsetCurrentTask(ctx, resp.Private); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		}
	}

	// Delete the file
	var polling remoteFilePollingModel

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

func (v *remoteFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	path := req.ID

	tflog.Info(ctx, "Reading the file metadata...", map[string]interface{}{
		"path": path,
	})

	fileInfo, err := v.client.GetFileInfo(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get file", fmt.Sprintf("Path: %s, Error: %s", path, err.Error()))
		return
	}

	var model remoteFileModel

	defer func() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}()

	if diags := model.populateDefaults(ctx); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if diags := model.populateFromFileInfo(fileInfo); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Set checksum

	tflog.Info(ctx, "Computing the checksum...")

	hMethod, _ := hashSpec(model.Checksum.ValueString())

	hash, diags := model.fileChecksum(ctx, resp.Private, v.client, model.DestinationPath.ValueString(), hMethod)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	model.setChecksum(hMethod, hash)
}

func hashSpec(checksum string) (string, string) {
	parts := strings.SplitN(checksum, ":", 2)
	if len(parts) == 1 {
		parts = []string{string(freeboxTypes.HashTypeSHA256), parts[0]}
	} else if parts[0] == "" {
		parts[0] = string(freeboxTypes.HashTypeSHA256)
	}

	return parts[0], parts[1]
}

func (v *remoteFileModel) setChecksum(method, value string) {
	v.Checksum = basetypes.NewStringValue(fmt.Sprintf("%s:%s", method, value))
}

// fileChecksum verifies the checksum of the file.
func (v *remoteFileModel) fileChecksum(ctx context.Context, state providerdata.Setter, client client.Client, path string, hashType string) (checksum string, diagnostics diag.Diagnostics) {
	var polling remoteFilePollingModel

	if diags := v.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if hashType == "" {
		hashType = string(freeboxTypes.HashTypeSHA256)
	}

	task, err := client.AddHashFileTask(ctx, freeboxTypes.HashPayload{
		HashType: freeboxTypes.HashType(hashType),
		Path:     freeboxTypes.Base64Path(path),
	})
	if err != nil {
		diagnostics.AddError("Failed to request file hash", fmt.Sprintf("Path: %s, Hash type: %s, Error: %s", path, hashType, err.Error()))
		return
	}

	ctx = tflog.SetField(ctx, "task.id", task.ID)

	tflog.Debug(ctx, "Hasing the file...")

	if diags := providerdata.SetCurrentTask(ctx, state, models.TaskTypeFileSystem, task.ID); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	var checksumPolling models.Polling

	if diags := polling.ChecksumCompute.As(ctx, &checksumPolling, basetypes.ObjectAsOptions{}); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if diags := waitForFileSystemTask(ctx, client, task.ID, checksumPolling); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	tflog.Debug(ctx, "Getting the hash result...")

	hash, err := client.GetHashResult(ctx, task.ID)
	if err != nil {
		diagnostics.AddError("Failed to get hash result", fmt.Sprintf("Task %d, Path: %s, Hash type: %s, Error: %s", task.ID, path, hashType, err.Error()))
		return
	}

	if err := stopAndDeleteFileSystemTask(ctx, client, task.ID); err != nil {
		diagnostics.AddError("Failed to stop and delete file system task", fmt.Sprintf("Task %d, Error: %s", task.ID, err.Error()))
		return
	}

	if diags := providerdata.UnsetCurrentTask(ctx, state); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	return hash, nil
}
