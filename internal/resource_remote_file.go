package internal

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	go_path "path"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
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

	// TaskID is the task identifier.
	TaskID types.Int64 `tfsdk:"task_id"`

	// Polling is the polling configuration.
	Polling types.Object `tfsdk:"polling"`
}

func (o remoteFileModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"destination_path": types.StringType,
		"source_url":       types.StringType,
		"checksum":         types.StringType,
		"authentication":   types.ObjectType{}.WithAttributeTypes(remoteFileModelAuthenticationsModel{}.AttrTypes()),
		"task_id":          types.Int64Type,
		"polling":          types.ObjectType{AttrTypes: remoteFilePollingModel{}.AttrTypes()},
	}
}

type remoteFilePollingModel struct {
	Delete          types.Object `tfsdk:"delete"`
	Create          types.Object `tfsdk:"create"`
	ChecksumCompute types.Object `tfsdk:"checksum_compute"`
}

func (o remoteFilePollingModel) defaults() basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(remoteFilePollingModel{}.AttrTypes(), map[string]attr.Value{
		"create":           basetypes.NewObjectValueMust(pollingSpecModel{}.AttrTypes(), map[string]attr.Value{
			"interval": timetypes.NewGoDurationValueFromStringMust("3s"),
			"timeout":  timetypes.NewGoDurationValueFromStringMust("30m"),
		}),
		"delete":           pollingSpecModel{}.defaults(),
		"checksum_compute": pollingSpecModel{}.defaults(),
	})
}

func (o remoteFilePollingModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"create": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Creation polling configuration",
			Attributes:          pollingSpecModel{}.ResourceAttributes(),
			Default:             objectdefault.StaticValue(basetypes.NewObjectValueMust(pollingSpecModel{}.AttrTypes(), map[string]attr.Value{
				"interval": timetypes.NewGoDurationValueFromStringMust("3s"),
				"timeout":  timetypes.NewGoDurationValueFromStringMust("30m"),
			})),
		},
		"delete": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Deletion polling configuration",
			Attributes:          pollingSpecModel{}.ResourceAttributes(),
			Default:             objectdefault.StaticValue(pollingSpecModel{}.defaults()),
		},
		"checksum_compute": schema.SingleNestedAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Checksum compute polling configuration",
			Attributes:          pollingSpecModel{}.ResourceAttributes(),
			Default:             objectdefault.StaticValue(pollingSpecModel{}.defaults()),
		},
	}
}

func (o remoteFilePollingModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"create":           types.ObjectType{AttrTypes: pollingSpecModel{}.AttrTypes()},
		"delete":           types.ObjectType{AttrTypes: pollingSpecModel{}.AttrTypes()},
		"checksum_compute": types.ObjectType{AttrTypes: pollingSpecModel{}.AttrTypes()},
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

func (v *remoteFileModel) populateDefaults() {
	v.Authentication = remoteFileModelAuthenticationsModel{}.defaults()
	v.Polling = remoteFilePollingModel{}.defaults()
}

func (v *remoteFileModel) populateDestinationFromFileInfo(fileInfo freeboxTypes.FileInfo) {
	v.DestinationPath = basetypes.NewStringValue(string(fileInfo.Path))
}

func (v *remoteFileModel) populateFromDownloadTask(downloadTask freeboxTypes.DownloadTask) {
	v.TaskID = basetypes.NewInt64Value(downloadTask.ID)
	v.DestinationPath = basetypes.NewStringValue(go_path.Join(string(downloadTask.DownloadDirectory), downloadTask.Name))

	// Do we really want to get the checksum from the download task?
	if v.Checksum.IsUnknown() && downloadTask.InfoHash != "" {
		method, value := hashSpec(downloadTask.InfoHash)
		v.setChecksum(method, value)
	}

	// Credentials are not returned by the API
	// Source URL is not returned by the API
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
		MarkdownDescription: "The resource downloads a file from a URL and stores it on the Freebox.",
		Attributes: map[string]schema.Attribute{
			"destination_path": schema.StringAttribute{
				MarkdownDescription: "Path to the file on the Freebox",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"source_url": schema.StringAttribute{
				MarkdownDescription: "The URL of the file to download",
				Required:            true,
				Validators: []validator.String{
					&sourceURLValidator{},
				},
			},
			"checksum": schema.StringAttribute{
				MarkdownDescription: "Checksum to verify the hash of the downloaded file",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"authentication": schema.SingleNestedAttribute{
				MarkdownDescription: "Authentication credentials to use for the operation",
				Optional:            true,
				Computed:            true,
				Attributes:          remoteFileModelAuthenticationsModel{}.ResourceAttributes(),
				Default:             objectdefault.StaticValue(basetypes.NewObjectNull(remoteFileModelAuthenticationsModel{}.AttrTypes())),
			},
			"task_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Task identifier",
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
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

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := v.client.GetFileInfo(ctx, model.DestinationPath.ValueString()); err == nil {
		resp.Diagnostics.AddError("File already exists", "Please delete the file or import it into the state")
		return
	} else if !errors.Is(err, client.ErrPathNotFound) {
		resp.Diagnostics.AddError("Failed to get file", err.Error())
		return
	}

	resp.Diagnostics.Append(v.createFromURL(ctx, &model, resp.State)...)

	fileInfo, err := v.client.GetFileInfo(ctx, model.DestinationPath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get file", err.Error())
		return
	} else {
		model.populateDestinationFromFileInfo(fileInfo)
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}

	var polling remoteFilePollingModel

	if diags := model.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	checksum := model.Checksum.ValueString()
	if checksum == "" {
		result, diags := fileChecksum(ctx, v.client, model.DestinationPath.ValueString(), string(freeboxTypes.HashTypeSHA256), polling)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		checksum = result
		model.setChecksum(string(freeboxTypes.HashTypeSHA256), checksum)
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	} else {
		resp.Diagnostics.Append(verifyFileChecksum(ctx, v.client, model.DestinationPath.ValueString(), checksum, polling)...)
	}
}

func (v *remoteFileResource) createFromURL(ctx context.Context, model *remoteFileModel, state tfsdk.State) (diagnostics diag.Diagnostics) {
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

	diagnostics.Append(state.SetAttribute(ctx, path.Root("task_id"), taskID)...)

	downloadTask, err := v.client.GetDownloadTask(ctx, taskID)
	if err != nil {
		diagnostics.AddError("Failed to get download task", err.Error())
	} else {
		model.populateFromDownloadTask(downloadTask)
		diagnostics.Append(state.Set(ctx, &model)...)
	}

	if diagnostics.HasError() {
		return
	}

	var polling remoteFilePollingModel

	if diags := tfsdk.ValueFrom(ctx, model.Polling, model.Polling.Type(ctx), &polling); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	var createPolling pollingSpecModel

	if diags := polling.Create.As(ctx, &createPolling, basetypes.ObjectAsOptions{}); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if diags := waitForDownloadTask(ctx, v.client, taskID, createPolling); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	return
}

func (v *remoteFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model remoteFileModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var polling remoteFilePollingModel

	if diags := model.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	fileInfo, err := v.client.GetFileInfo(ctx, model.DestinationPath.ValueString())
	if err != nil {
		var createPolling pollingSpecModel

		if diags := polling.Create.As(ctx, &createPolling, basetypes.ObjectAsOptions{}); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		// File does not exist yet, wait for download task to complete
		if diags := waitForDownloadTask(ctx, v.client, model.TaskID.ValueInt64(), createPolling); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	if model.TaskID.IsNull() {
		task, err := v.findTaskByPath(ctx, model.DestinationPath.ValueString())
		if err != nil {
			return
		}

		model.populateFromDownloadTask(task)
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}

	model.populateDestinationFromFileInfo(fileInfo)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)

	hMethod, _ := hashSpec(model.Checksum.ValueString())

	hash, diags := fileChecksum(ctx, v.client, model.DestinationPath.ValueString(), hMethod, polling)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	} else {
		model.setChecksum(hMethod, hash)
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (v *remoteFileResource) findTaskByPath(ctx context.Context, destinationPath string) (freeboxTypes.DownloadTask, error) {
	tasks, err := v.client.ListDownloadTasks(ctx)
	if err != nil {
		return freeboxTypes.DownloadTask{}, fmt.Errorf("list download tasks: %w", err)
	}

	name := go_path.Base(destinationPath)
	directory := go_path.Dir(destinationPath)

	var task freeboxTypes.DownloadTask

	for _, t := range tasks {
		if string(t.DownloadDirectory) == directory && t.Name == name && t.CreatedTimestamp.After(task.CreatedTimestamp.Time) {
			task = t
		}
	}

	if task.ID == 0 {
		return freeboxTypes.DownloadTask{}, &taskError{taskID: 0, errorCode: string("task_not_found")}
	}

	return task, nil
}

func (v *remoteFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var oldModel, newModel remoteFileModel

	resp.Diagnostics.Append(req.State.Get(ctx, &oldModel)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &newModel)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if oldModel.SourceURL.ValueString() != newModel.SourceURL.ValueString() {
		if _, err := v.client.GetFileInfo(ctx, newModel.DestinationPath.ValueString()); err == nil {
			resp.Diagnostics.AddError("File already exists", "Please delete the file or import it into the state")
			return
		} else if !errors.Is(err, client.ErrPathNotFound) {
			resp.Diagnostics.AddError("Failed to get file", err.Error())
			return
		}
	}

	if err := stopAndDeleteDownloadTask(ctx, v.client, oldModel.TaskID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Failed to stop and delete download task", err.Error())
	}

	var polling remoteFilePollingModel

	if diags := newModel.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var deletePolling pollingSpecModel

	if diags := polling.Delete.As(ctx, &deletePolling, basetypes.ObjectAsOptions{}); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if diags := deleteFilesIfExist(ctx, v.client, deletePolling, oldModel.DestinationPath.ValueString()); diags.HasError() {
		resp.Diagnostics.Append(diags...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

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

	newModel.TaskID = basetypes.NewInt64Value(taskID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newModel)...)

	var createPolling pollingSpecModel

	if diags := polling.Create.As(ctx, &createPolling, basetypes.ObjectAsOptions{}); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if diags := waitForDownloadTask(ctx, v.client, taskID, createPolling); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	checksum := newModel.Checksum.ValueString()
	if checksum == "" {
		result, diags := fileChecksum(ctx, v.client, newModel.DestinationPath.ValueString(), string(freeboxTypes.HashTypeSHA256), polling)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		checksum = result
		newModel.setChecksum(string(freeboxTypes.HashTypeSHA256), checksum)
		resp.Diagnostics.Append(resp.State.Set(ctx, &newModel)...)
	} else {
		resp.Diagnostics.Append(verifyFileChecksum(ctx, v.client, newModel.DestinationPath.ValueString(), newModel.Checksum.ValueString(), polling)...)
	}
}

func (v *remoteFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model remoteFileModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := stopAndDeleteDownloadTask(ctx, v.client, model.TaskID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Failed to stop and delete download task", err.Error())
		return
	}

	var polling remoteFilePollingModel

	if diags := model.Polling.As(ctx, &polling, basetypes.ObjectAsOptions{}); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var deletePolling pollingSpecModel

	if diags := polling.Delete.As(ctx, &deletePolling, basetypes.ObjectAsOptions{}); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if diags := deleteFilesIfExist(ctx, v.client, deletePolling, model.DestinationPath.ValueString()); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
}

func (v *remoteFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.Atoi(req.ID)
	if err != nil {
		v.importStateFromPath(ctx, req, resp)
		return
	}

	task, err := v.client.GetDownloadTask(ctx, int64(id))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get download task", err.Error())
		return
	}

	var model remoteFileModel

	model.populateDefaults()

	model.populateFromDownloadTask(task)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *remoteFileResource) importStateFromPath(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	path := req.ID

	fileInfo, err := v.client.GetFileInfo(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get file", err.Error())
		return
	}

	var model remoteFileModel

	model.populateDefaults()

	model.populateDestinationFromFileInfo(fileInfo)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)

	task, err := v.findTaskByPath(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError("Failed to find download task", err.Error())
		return
	}

	model.populateFromDownloadTask(task)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
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

// verifyFileChecksum verifies the checksum of the file.
func verifyFileChecksum(ctx context.Context, client client.Client, path string, checksum string, polling remoteFilePollingModel) (diagnostics diag.Diagnostics) {
	hMethod, hValue := hashSpec(checksum)

	hash, diags := fileChecksum(ctx, client, path, hMethod, polling)
	if diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if hash != hValue {
		diagnostics.AddError("Checksum mismatch", fmt.Sprintf("Expected %q, got %q", checksum, hash))
	}

	return
}

// fileChecksum verifies the checksum of the file.
func fileChecksum(ctx context.Context, client client.Client, path string, hashType string, polling remoteFilePollingModel) (checsum string, diagnostics diag.Diagnostics) {
	if hashType == "" {
		hashType = string(freeboxTypes.HashTypeSHA256)
	}

	task, err := client.AddHashFileTask(ctx, freeboxTypes.HashPayload{
		HashType: freeboxTypes.HashType(hashType),
		Path:     freeboxTypes.Base64Path(path),
	})

	if err != nil {
		diagnostics.AddError("Failed to add hash file task", err.Error())
		return
	}

	var checksumPolling pollingSpecModel

	if diags := polling.ChecksumCompute.As(ctx, &checksumPolling, basetypes.ObjectAsOptions{}); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if diags := waitForFileSystemTask(ctx, client, task.ID, checksumPolling); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	hash, err := client.GetHashResult(ctx, task.ID)
	if err != nil {
		diagnostics.AddError("Failed to get hash result", err.Error())
		return
	}

	return hash, nil
}

type sourceURLValidator struct{}

var supportedShemes = []string{"http", "https", "ftp", "magnet"}

func (s *sourceURLValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	u, err := url.Parse(req.ConfigValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid URL", err.Error())
		return
	}

	if !slices.Contains(supportedShemes, u.Scheme) {
		resp.Diagnostics.AddError("Unsupported URL scheme", fmt.Sprintf("Scheme %q is not supported, expected one of %v", u.Scheme, supportedShemes))
	}

	return
}

func (s *sourceURLValidator) Description(ctx context.Context) string {
	return "source URL validator"
}

func (s *sourceURLValidator) MarkdownDescription(ctx context.Context) string {
	return "source URL validator"
}

type checksumValidator struct{}

func (s *checksumValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	parts := strings.SplitN(req.ConfigValue.ValueString(), ":", 2)
	if len(parts) != 2 {
		// Use the default hash function
		return
	}

	if len(parts[0]) == 0 {
		resp.Diagnostics.AddError("Invalid checksum", "Checksum must have a hash function")
	}
	if len(parts[1]) == 0 {
		resp.Diagnostics.AddError("Invalid checksum", "Checksum must have a value")
	}

	return
}

func (s *checksumValidator) Description(ctx context.Context) string {
	return "checksum validator"
}

func (s *checksumValidator) MarkdownDescription(ctx context.Context) string {
	return "checksum validator"
}
