package internal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/nikolalohinski/terraform-provider-freebox/internal/models"
	providerdata "github.com/nikolalohinski/terraform-provider-freebox/internal/provider_data"
)

func stopAndDeleteFileSystemTask(ctx context.Context, c client.Client, taskID int64) error {
	errs := make([]error, 0, 2)

	ctx = tflog.SetField(ctx, "task.id", taskID)
	ctx = tflog.SetField(ctx, "task.type", models.TaskTypeFileSystem)

	tflog.Trace(ctx, "Stopping task...")

	if _, err := c.UpdateFileSystemTask(ctx, taskID, freeboxTypes.FileSytemTaskUpdate{
		State: freeboxTypes.FileTaskStatePaused,
	}); err != nil {
		if errors.Is(err, client.ErrTaskNotFound) {
			return nil
		}

		errs = append(errs, fmt.Errorf("stop file system task: %w", err))
	}

	tflog.Trace(ctx, "Deleting task...")

	if err := c.DeleteFileSystemTask(ctx, taskID); err != nil {
		if errors.Is(err, client.ErrTaskNotFound) {
			return nil
		}

		errs = append(errs, fmt.Errorf("erase file system task: %w", err))
	}

	return errors.Join(errs...)
}

func stopAndDeleteVirtualDiskTask(ctx context.Context, c client.Client, taskID int64) error {
	if err := c.DeleteVirtualDiskTask(ctx, taskID); err != nil {
		if errors.Is(err, client.ErrTaskNotFound) {
			return nil
		}

		return fmt.Errorf("delete virtual disk task: %w", err)
	}

	return nil
}

func stopAndDeleteUploadTask(ctx context.Context, c client.Client, taskID int64) error {
	if err := c.CancelUploadTask(ctx, taskID); err != nil {
		if errors.Is(err, client.ErrTaskNotFound) {
			return nil
		}

		return fmt.Errorf("cancel upload task: %w", err)
	}

	if err := c.DeleteUploadTask(ctx, taskID); err != nil {
		if errors.Is(err, client.ErrTaskNotFound) {
			return nil
		}

		return fmt.Errorf("delete upload task: %w", err)
	}

	return nil
}

func stopAndDeleteDownloadTask(ctx context.Context, c client.Client, taskID int64) error {
	errs := make([]error, 0, 2)

	ctx = tflog.SetField(ctx, "task.id", taskID)
	ctx = tflog.SetField(ctx, "task.type", models.TaskTypeFileSystem)

	tflog.Trace(ctx, "Stopping task...")

	if err := c.UpdateDownloadTask(ctx, taskID, freeboxTypes.DownloadTaskUpdate{
		Status: freeboxTypes.DownloadTaskStatusStopped,
	}); err != nil {
		if errors.Is(err, client.ErrTaskNotFound) {
			return nil
		}

		errs = append(errs, fmt.Errorf("stop download task: %w", err))
	}

	tflog.Trace(ctx, "Deleting task...")

	if err := c.DeleteDownloadTask(ctx, taskID); err != nil {
		if errors.Is(err, client.ErrTaskNotFound) {
			return nil
		}

		errs = append(errs, fmt.Errorf("delete download task: %w", err))
	}

	return errors.Join(errs...)
}

func deleteFilesIfExist(ctx context.Context, state providerdata.Setter, c client.Client, polling models.Polling, paths ...string) (diagnostics diag.Diagnostics) {
	filesToDelete := make([]string, 0, len(paths))

	for _, path := range paths {
		_, err := c.GetFileInfo(ctx, path)
		if err != nil {
			if !errors.Is(err, client.ErrPathNotFound) {
				diagnostics.AddError("Failed to get file info", fmt.Sprintf("Path: %s, Error: %s", path, err.Error()))
			}

			continue
		}

		filesToDelete = append(filesToDelete, path)
	}

	if diagnostics.HasError() {
		return
	}

	if len(filesToDelete) == 0 {
		return nil
	}

	tflog.Debug(ctx, "Deleting files...", map[string]interface{}{
		"files": filesToDelete,
	})

	task, err := c.RemoveFiles(ctx, filesToDelete)
	if err != nil {
		diagnostics.AddError(fmt.Sprintf("Failed to remove %d files", len(filesToDelete)), fmt.Sprintf("Files: %v, Error: %s", filesToDelete, err.Error()))
		return
	}

	if diags := providerdata.SetCurrentTask(ctx, state, models.TaskTypeFileSystem, task.ID); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if diags := waitForFileSystemTask(ctx, c, task.ID, polling); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	if err := stopAndDeleteFileSystemTask(ctx, c, task.ID); err != nil {
		diagnostics.AddError("Failed to stop and delete file system task", fmt.Sprintf("Task: %d, Action: %s, Error: %s", task.ID, task.Type, err.Error()))
		return
	}

	if diags := providerdata.UnsetCurrentTask(ctx, state); diags.HasError() {
		diagnostics.Append(diags...)
	}

	return
}

// waitForFileSystemTask waits for the system task to complete.
func waitForFileSystemTask(ctx context.Context, client client.Client, taskID int64, polling models.Polling) (diagnostics diag.Diagnostics) {
	ctx = tflog.SetField(ctx, "task.id", taskID)
	ctx = tflog.SetField(ctx, "task.type", models.TaskTypeFileSystem)

	interval, diags := polling.Interval.ValueGoDuration()
	if diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	timeout, diags := polling.Timeout.ValueGoDuration()
	if diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	tflog.Debug(ctx, "Waiting for file system task to complete...", map[string]interface{}{
		"timeout":  timeout,
		"interval": interval,
	})

	tick := time.NewTicker(interval)
	defer tick.Stop()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var taskType, currentState string

	for {
		task, taskErr := client.GetFileSystemTask(ctx, taskID)
		if taskErr != nil {
			tflog.Warn(ctx, "Failed to get file system task. Retrying...", map[string]interface{}{
				"error": taskErr.Error(),
			})
		} else {
			if taskType == "" {
				taskType = string(task.Type)
				ctx = tflog.SetField(ctx, "task.action", taskType)
			}

			currentState = string(task.State)

			switch task.State {
			case freeboxTypes.FileTaskStateFailed:
				diagnostics.AddError("File system task failed", fmt.Sprintf("Task: %d, Action: %s, Error code: %s", taskID, task.Type, task.Error))
				return
			case freeboxTypes.FileTaskStateDone:
				return nil // Done
			case freeboxTypes.FileTaskStatePaused:
				tflog.Info(ctx, "File system task paused, please resume it")
			default:
				tflog.Debug(ctx, "File system task not done yet", map[string]interface{}{
					"task.state": task.State,
				})
			}
		}

		select {
		case <-ctx.Done():
			diagnostics.AddError("File system task did not complete in time", fmt.Sprintf("Task: %d, Action: %s, State: %s, Error: %v", taskID, taskType, currentState, ctx.Err()))
			return
		case <-tick.C:
		}
	}
}

func waitForDiskTask(ctx context.Context, c client.Client, taskID int64, timeout timetypes.GoDuration) (diagnostics diag.Diagnostics) {
	ctx = tflog.SetField(ctx, "task.id", taskID)
	ctx = tflog.SetField(ctx, "task.type", models.TaskTypeFileSystem)

	tflog.Debug(ctx, "Waiting for disk task to complete...")

	events, err := c.ListenEvents(ctx, []freeboxTypes.EventDescription{{
		Source: freeboxTypes.EventSourceVMDisk,
		Name:   freeboxTypes.EventDiskTaskDone,
	}})
	if err != nil {
		diagnostics.AddError("Failed to listen for events", fmt.Sprintf("Error: %s", err.Error()))
		return
	}

	task, err := c.GetVirtualDiskTask(ctx, taskID)
	if err != nil {
		diagnostics.AddError("Failed to get virtual disk task", fmt.Sprintf("Task: %d, Error: %s", taskID, err.Error()))
		return
	}

	if task.Done {
		if task.Error {
			diagnostics.AddError("Disk task failed", fmt.Sprintf("Task: %d", taskID))
		}
		return
	}

	timeoutDuration, diags := timeout.ValueGoDuration()
	if diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	for {
		select {
		case event := <-events:
			if !event.Notification.Success {
				diagnostics.AddError("Disk task failed", fmt.Sprintf("Task: %d, Error: %v", taskID, event.Error))
				return
			}

			return nil // Done
		case <-ctx.Done():
			diagnostics.AddError("Disk task did not complete in time", ctx.Err().Error())
			return
		}
	}
}

func waitForUpload(ctx context.Context, c client.Client, taskID int64, polling models.Polling) (diagnostics diag.Diagnostics) {
	ctx = tflog.SetField(ctx, "task.id", taskID)
	ctx = tflog.SetField(ctx, "task.type", models.TaskTypeUpload)

	interval, diags := polling.Interval.ValueGoDuration()
	if diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	timeout, diags := polling.Timeout.ValueGoDuration()
	if diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	tflog.Debug(ctx, "Waiting for upload task to complete...", map[string]interface{}{
		"timeout":  timeout,
		"interval": interval,
	})

	tick := time.NewTicker(interval)
	defer tick.Stop()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var currentStatus string

	for {
		task, taskErr := c.GetUploadTask(ctx, taskID)
		if taskErr != nil {
			diagnostics.AddError("Failed to get upload task", taskErr.Error())
		} else {
			currentStatus = string(task.Status)

			switch task.Status {
			case freeboxTypes.UploadTaskStatusFailed,
				freeboxTypes.UploadTaskStatusCancelled,
				freeboxTypes.UploadTaskStatusConflict,
				freeboxTypes.UploadTaskStatusTimeout:
				diagnostics.AddError("Upload task failed", fmt.Sprintf("Task: %d, Error code: %s", taskID, task.Status))
				return
			case freeboxTypes.UploadTaskStatusDone:
				return nil // Done
			default:
				if task.Uploaded == task.Size {
					return nil // Done
				}

				tflog.Debug(ctx, "Upload task not done yet", map[string]interface{}{
					"task.status": task.Status,
				})
				// Nothing to do
			}
		}

		select {
		case <-ctx.Done():
			diagnostics.AddError("Upload took did not complete in time", fmt.Sprintf("Task: %d, Status: %s, Error: %v", taskID, currentStatus, ctx.Err()))
			return
		case <-tick.C:
		}
	}
}

func waitForUploadTask(ctx context.Context, c client.Client, taskID int64, polling models.Polling) (diagnostics diag.Diagnostics) {
	ctx = tflog.SetField(ctx, "task.id", taskID)
	ctx = tflog.SetField(ctx, "task.type", models.TaskTypeUpload)

	interval, diags := polling.Interval.ValueGoDuration()
	if diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	timeout, diags := polling.Timeout.ValueGoDuration()
	if diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	tflog.Debug(ctx, "Waiting for upload task to complete...", map[string]interface{}{
		"timeout":  timeout,
		"interval": interval,
	})

	tick := time.NewTicker(interval)
	defer tick.Stop()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var currentStatus string

	for {
		task, taskErr := c.GetUploadTask(ctx, taskID)
		if taskErr != nil {
			diagnostics.AddError("Failed to get upload task", taskErr.Error())
		} else {
			currentStatus = string(task.Status)

			switch task.Status {
			case freeboxTypes.UploadTaskStatusFailed,
				freeboxTypes.UploadTaskStatusCancelled,
				freeboxTypes.UploadTaskStatusConflict,
				freeboxTypes.UploadTaskStatusTimeout:
				diagnostics.AddError("Upload task failed", fmt.Sprintf("Task: %d, Error code: %s", taskID, task.Status))
				return
			case freeboxTypes.UploadTaskStatusDone:
				return nil // Done
			default:
				tflog.Debug(ctx, "Upload task not done yet", map[string]interface{}{
					"task.status": task.Status,
				})
				// Nothing to do
			}
		}

		select {
		case <-ctx.Done():
			diagnostics.AddError("Upload took did not complete in time", fmt.Sprintf("Task: %d, Status: %s, Error: %v", taskID, currentStatus, ctx.Err()))
			return
		case <-tick.C:
		}
	}
}

// waitForDownloadTask waits for the download task to complete.
func waitForDownloadTask(ctx context.Context, c client.Client, taskID int64, polling models.Polling) (diagnostics diag.Diagnostics) {
	ctx = tflog.SetField(ctx, "task.id", taskID)
	ctx = tflog.SetField(ctx, "task.type", models.TaskTypeDownload)

	interval, diags := polling.Interval.ValueGoDuration()
	if diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	timeout, diags := polling.Timeout.ValueGoDuration()
	if diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	tflog.Debug(ctx, "Waiting for download task to complete...", map[string]interface{}{
		"timeout":  timeout,
		"interval": interval,
	})

	tick := time.NewTicker(interval)
	defer tick.Stop()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var currentStatus string

	for {
		task, taskErr := c.GetDownloadTask(ctx, taskID)
		if taskErr != nil {
			diagnostics.AddError("Failed to get download task", taskErr.Error())
		} else {
			currentStatus = string(task.Status)

			switch task.Status {
			case freeboxTypes.DownloadTaskStatusError:
				diagnostics.AddError("Download task failed", fmt.Sprintf("Task: %d, Error code: %s", taskID, task.Error))
				return
			case freeboxTypes.DownloadTaskStatusDone:
				return nil // Done
			case freeboxTypes.DownloadTaskStatusStopped:
				tflog.Info(ctx, "Download task stopped, please resume it")
			default:
				tflog.Debug(ctx, "Download task not done yet", map[string]interface{}{
					"task.status": task.Status,
				})
				// Nothing to do
			}
		}

		select {
		case <-ctx.Done():
			diagnostics.AddError("Download took did not complete in time", fmt.Sprintf("Task: %d, Status: %s, Error: %v", taskID, currentStatus, ctx.Err()))
			return
		case <-tick.C:
		}
	}
}

func stopAndDeleteTask(ctx context.Context, c client.Client, taskType models.TaskType, taskID int64) error {
	switch taskType {
	case models.TaskTypeFileSystem:
		return stopAndDeleteFileSystemTask(ctx, c, taskID)
	case models.TaskTypeDownload:
		return stopAndDeleteDownloadTask(ctx, c, taskID)
	case models.TaskTypeVirtualDisk:
		return stopAndDeleteVirtualDiskTask(ctx, c, taskID)
	case models.TaskTypeUpload:
		return stopAndDeleteUploadTask(ctx, c, taskID)
	default:
		return fmt.Errorf("unknown task type: %s", taskType)
	}
}

func WaitForTask(ctx context.Context, c client.Client, taskType models.TaskType, taskID int64, polling *models.Polling) (diagnostics diag.Diagnostics) {
	switch taskType {
	case models.TaskTypeFileSystem:
		return waitForFileSystemTask(ctx, c, taskID, *polling)
	case models.TaskTypeDownload:
		return waitForDownloadTask(ctx, c, taskID, *polling)
	case models.TaskTypeVirtualDisk:
		return waitForDiskTask(ctx, c, taskID, polling.Timeout)
	case models.TaskTypeUpload:
		return waitForUploadTask(ctx, c, taskID, *polling)
	default:
		diagnostics.AddError("Unknown task type", fmt.Sprintf("Task: %d, Type: %s", taskID, taskType))
	}

	return
}
