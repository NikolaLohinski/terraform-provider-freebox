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
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

var (
	_ resource.Resource                = &remoteFileResource{}
	_ resource.ResourceWithImportState = &remoteFileResource{}
)

func NewRemoteFileResource() resource.Resource {
	return &remoteFileResource{}
}

// remoteFileResource defines the resource implementation.
type remoteFileResource struct {
	client client.Client
}

//go:generate stringer -type=fileType -linecomment
type fileType uint8

const (
	// fileTypeDir represents a directory.
	fileTypeDir fileType = iota // dir
	// fileTypeFile represents a file.
	fileTypeFile // file
)

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

	// Credentials is the credentials to use for the operation.
	Authentication *remoteFileModelCredentialsModel `tfsdk:"authentication"`

	// taskID is the task identifier.
	TaskID types.Int64 `tfsdk:"task_id"`
}

type remoteFileModelCredentialsModel struct {
	// BasicAuth is the basic authentication credentials.
	BasicAuth *remoteFileModelCredentialsBasicAuthModel `tfsdk:"basic_auth"`
}

type remoteFileModelCredentialsBasicAuthModel struct {
	// Username is the username.
	Username types.String `tfsdk:"username"`
	// Password is the password.
	Password types.String `tfsdk:"password"`
}

func (v *remoteFileModel) populateDestinationFromFileInfo(fileInfo freeboxTypes.FileInfo) {
	v.DestinationPath = basetypes.NewStringValue(string(fileInfo.Path))
}

func (v *remoteFileModel) populateFromDownloadTask(downloadTask freeboxTypes.DownloadTask) {
	v.TaskID = basetypes.NewInt64Value(downloadTask.ID)
	v.DestinationPath = basetypes.NewStringValue(go_path.Join(string(downloadTask.DownloadDirectory), downloadTask.Name))

	// Do we really want to get the checksum from the download task?
	if v.Checksum.IsUnknown() && downloadTask.InfoHash != "" {
		v.Checksum = basetypes.NewStringValue(downloadTask.InfoHash)
	}

	// TODO: Implement credentials
}

func (v *remoteFileModel) toDownloadPayload() (payload freeboxTypes.DownloadRequest) {
	payload.DownloadURLs = []string{v.SourceURL.ValueString()}
	payload.Hash = v.Checksum.ValueString()

	destinationPath := v.DestinationPath.ValueString()
	payload.DownloadDirectory = go_path.Dir(destinationPath)
	payload.Filename = go_path.Base(destinationPath)

	if v.Authentication != nil && v.Authentication.BasicAuth != nil {
		if username := v.Authentication.BasicAuth.Username.ValueString(); username != "" {
			payload.Username = username
		}

		if password := v.Authentication.BasicAuth.Password.ValueString(); password != "" {
			payload.Password = password
		}
	}

	return
}

func (v *remoteFileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_remote_file"
}

func (v *remoteFileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a virtual machine instance within a Freebox box. See the [Freebox blog](https://dev.freebox.fr/blog/?p=5450) for additional details",
		Attributes: map[string]schema.Attribute{
			"destination_path": schema.StringAttribute{
				MarkdownDescription: "Path to the file on the Freebox",
				Required: 		     true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"source_url": schema.StringAttribute{
				MarkdownDescription: "VM ethernet interface MAC address",
				Required: 		     true,
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
				Attributes: map[string]schema.Attribute{
					"basic_auth": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Basic authentication credentials",
						Attributes: map[string]schema.Attribute{
							"username": schema.StringAttribute{
								Optional:            true,
								MarkdownDescription: "Username",
							},
							"password": schema.StringAttribute{
								Optional:            true,
								Sensitive:           true,
								MarkdownDescription: "Password",
							},
						},
					},
				},
			},
			"task_id": schema.Int64Attribute{
				Optional:           true,
				Computed:           true,
				MarkdownDescription: "Task identifier",
				//PlanModifiers: []planmodifier.Int64{
					// int64planmodifier.UseStateForUnknown(),
				//},
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

	resp.Diagnostics.Append(v.createFromURL(ctx, &model, resp.State)...)

	fileInfo, err := v.client.GetFileInfo(ctx, model.DestinationPath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get file",
			err.Error(),
		)
		return
	} else {
		model.populateDestinationFromFileInfo(fileInfo)
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}

	checksum := model.Checksum.ValueString()
	if checksum == "" {
		checksum, err = fileChecksum(ctx, v.client, model.DestinationPath.ValueString(), string(freeboxTypes.HashTypeSHA256))
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to read hash",
				err.Error(),
			)
			return
		} else {
			model.setChecksum(string(freeboxTypes.HashTypeSHA256), checksum)
			resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
		}
	}

	resp.Diagnostics.Append(verifyFileChecksum(ctx, v.client, model.DestinationPath.ValueString(), checksum)...)
}

func (v *remoteFileResource) createFromURL(ctx context.Context, model *remoteFileModel, state tfsdk.State) (diagnostics diag.Diagnostics) {
	taskID, err := v.client.AddDownloadTask(ctx, model.toDownloadPayload())
	if err != nil {
		diagnostics.AddError(
			"Failed to add download task",
			err.Error(),
		)
		return
	}

	diagnostics.Append(state.SetAttribute(ctx, path.Root("task_id"), taskID)...)

	downloadTask, err := v.client.GetDownloadTask(ctx, taskID)
	if err != nil {
		diagnostics.AddError(
			"Failed to get download task",
			err.Error(),
		)
	} else {
		model.populateFromDownloadTask(downloadTask)
		diagnostics.Append(state.Set(ctx, &model)...)
	}

	if err := waitForDownloadTask(ctx, v.client, taskID); err != nil {
		diagnostics.AddError(
			"Failed to wait for download task",
			err.Error(),
		)
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

	fileInfo, err := v.client.GetFileInfo(ctx, model.DestinationPath.ValueString())
	if err != nil {
		// File does not exist yet, wait for download task to complete
		if err := waitForDownloadTask(ctx,v.client, model.TaskID.ValueInt64()); err != nil {
			resp.Diagnostics.AddError(
				"Failed to wait for download task",
				err.Error(),
			)
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

	hash, err := fileChecksum(ctx, v.client, model.DestinationPath.ValueString(), hMethod)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read hash",
			err.Error(),
		)
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
		return freeboxTypes.DownloadTask{}, &taskError{taskID: 0, errorCode: string(freeboxTypes.DownloadTaskErrorNotFound)}
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

	if err := stopAndDeleteDownloadTask(ctx, v.client, oldModel.TaskID.ValueInt64()); err != nil {
		resp.Diagnostics.AddWarning(
			"Failed to delete download task and its files",
			err.Error(),
		)
	}

	if err := deleteFilesIfExist(ctx, v.client,
		oldModel.DestinationPath.ValueString(),
		newModel.DestinationPath.ValueString(),
	); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete file(s)",
			err.Error(),
		)
		return
	}

	taskID, err := v.client.AddDownloadTask(ctx, newModel.toDownloadPayload())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to add download task",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("task_id"), taskID)...)
	newModel.TaskID = basetypes.NewInt64Value(taskID)

	if err := waitForDownloadTask(ctx, v.client, taskID); err != nil {
		resp.Diagnostics.AddError(
			"Failed to wait for download task",
			err.Error(),
		)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newModel)...)

	resp.Diagnostics.Append(verifyFileChecksum(ctx, v.client, newModel.DestinationPath.ValueString(), newModel.Checksum.ValueString())...)
}

func (v *remoteFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model remoteFileModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := stopAndDeleteDownloadTask(ctx, v.client, model.TaskID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete download task and its files",
			err.Error(),
		)
		return
	}

	if err := deleteFilesIfExist(ctx, v.client, model.DestinationPath.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete file",
			err.Error(),
		)
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
		resp.Diagnostics.AddError(
			"Failed to get download task",
			err.Error(),
		)
		return
	}

	var model remoteFileModel

	model.populateFromDownloadTask(task)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *remoteFileResource) importStateFromPath(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	path := req.ID

	fileInfo, err := v.client.GetFileInfo(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get file",
			err.Error(),
		)
		return
	}

	var model remoteFileModel
	model.populateDestinationFromFileInfo(fileInfo)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)

	task, err := v.findTaskByPath(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to find download task",
			err.Error(),
		)
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

func stopAndDeleteDownloadTask(ctx context.Context, c client.Client, taskID int64) (error) {
	errs := make([]error, 0, 2)

	if err := c.UpdateDownloadTask(ctx, taskID, freeboxTypes.DownloadTaskUpdate{
		Status: freeboxTypes.DownloadTaskStatusStopped,
	}); err != nil {
		if errors.Is(err, client.ErrTaskNotFound) {
			return nil
		}

		errs = append(errs, fmt.Errorf("stop download task: %w", err))
	}

	if err := c.EraseDownloadTask(ctx, taskID); err != nil {
		if errors.Is(err, client.ErrTaskNotFound) {
			return nil
		}

		errs = append(errs, fmt.Errorf("erase download task: %w", err))
	}

	return errors.Join(errs...)
}

func deleteFilesIfExist(ctx context.Context, c client.Client, paths ...string) (error) {
	filesToDelete := make([]string, 0, len(paths))

	var errs []error
	for _, path := range paths {
		_, err := c.GetFileInfo(ctx, path)
		if err != nil {
			if !errors.Is(err, client.ErrPathNotFound) {
				errs = append(errs, err)
			}

			continue
		}

		filesToDelete = append(filesToDelete, path)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	if len(filesToDelete) == 0 {
		return nil
	}

	task, err := c.RemoveFiles(ctx, filesToDelete)
	if err != nil {
		return fmt.Errorf("remove files: %w", err)
	}

	if err := waitForFileSystemTask(ctx, c, task.ID); err != nil {
		var taskError *taskError
		if errors.As(err, &taskError) && taskError.errorCode == string(freeboxTypes.FileTaskErrorFileNotFound) {
			return nil
		}

		return fmt.Errorf("wait for file system task: %w", err)
	}

	return nil
}

// verifyFileChecksum verifies the checksum of the file.
func verifyFileChecksum(ctx context.Context, client client.Client, path string, checksum string) (diagnostics diag.Diagnostics) {
	hMethod, hValue := hashSpec(checksum)

	hash, err := fileChecksum(ctx, client, path, hMethod)
	if err != nil {
		diagnostics.AddError(
			"Failed to read hash",
			err.Error(),
		)
		return
	}

	if hash != hValue {
		diagnostics.AddError(
			"Checksum mismatch",
			fmt.Sprintf("Expected %q, got %q", checksum, hash),
		)
	}

	return
}

// fileChecksum verifies the checksum of the file.
func fileChecksum(ctx context.Context, client client.Client, path string, hashType string) (string, error) {
	if hashType == "" {
		hashType = string(freeboxTypes.HashTypeSHA256)
	}

	task, err := client.AddHashFileTask(ctx, freeboxTypes.HashPayload{
		HashType: freeboxTypes.HashType(hashType),
		Path:     freeboxTypes.Base64Path(path),
	})

	if err != nil {
		return "", fmt.Errorf("hash file task: %w", err)
	}

	if err := waitForFileSystemTask(ctx, client, task.ID); err != nil {
		return "", fmt.Errorf("wait for file system task: %w", err)
	}

	hash, err := client.GetHashResult(ctx, task.ID)
	if err != nil {
		return "", fmt.Errorf("get hash result: %w", err)
	}

	return hash, nil
}

type taskError struct {
	taskID int64
	errorCode string
}

func (e *taskError) Error() string {
	return fmt.Sprintf("task %d failed with error code %q", e.taskID, e.errorCode)
}

// waitForFileSystemTask waits for the system task to complete.
func waitForFileSystemTask(ctx context.Context, client client.Client, taskID int64) (error) {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		task, taskErr := client.GetFileSystemTask(ctx, taskID)
		if taskErr != nil {
			return fmt.Errorf("get task: %w", taskErr)
		}

		switch task.State {
		case freeboxTypes.FileTaskStateFailed:
			return &taskError{taskID: taskID, errorCode: string(task.Error)}
		case freeboxTypes.FileTaskStateDone:
			return nil // Done
		case freeboxTypes.FileTaskStatePaused:
			return errStoppedTask(taskID)
		default:
			// Nothing to do
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}

const (
	badHashError = "http_bad_hash"
)

// waitForDownloadTask waits for the download task to complete.
func waitForDownloadTask(ctx context.Context, c client.Client, taskID int64) (error) {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		task, taskErr := c.GetDownloadTask(ctx, taskID)
		if taskErr != nil {
			return fmt.Errorf("get task: %w", taskErr)
		}

		switch task.Status {
		case freeboxTypes.DownloadTaskStatusError:
			return &taskError{taskID: taskID, errorCode: string(task.Error)}
		case freeboxTypes.DownloadTaskStatusDone:
			return nil // Done
		case freeboxTypes.DownloadTaskStatusStopped:
			return errStoppedTask(taskID)
		default:
			// Nothing to do
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}

type errStoppedTask int64

func (e errStoppedTask) Error() string {
	return fmt.Sprintf("task %d is on pause, please resume it", int64(e))
}

type sourceURLValidator struct {}

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

type checksumValidator struct {}

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
