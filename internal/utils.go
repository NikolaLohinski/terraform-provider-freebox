package internal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

type pollingSpecModel struct {
	Interval timetypes.GoDuration `tfsdk:"interval"`
	Timeout  timetypes.GoDuration `tfsdk:"timeout"`
}

func (o pollingSpecModel) defaults() basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(pollingSpecModel{}.AttrTypes(), map[string]attr.Value{
		"interval": timetypes.NewGoDurationValue(1 * time.Second),
		"timeout":  timetypes.NewGoDurationValue(1 * time.Minute),
	})
}

func (o pollingSpecModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"interval": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			CustomType:          timetypes.GoDurationType{},
			MarkdownDescription: "The interval at which to poll the resource.",
		},
		"timeout": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			CustomType:          timetypes.GoDurationType{},
			MarkdownDescription: "The timeout for the operation.",
		},
	}
}

func (o pollingSpecModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"interval": timetypes.GoDurationType{},
		"timeout":  timetypes.GoDurationType{},
	}
}

func stopAndDeleteDownloadTask(ctx context.Context, c client.Client, taskID int64) error {
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

func deleteFilesIfExist(ctx context.Context, c client.Client, polling pollingSpecModel, paths ...string) (diagnostics diag.Diagnostics) {
	filesToDelete := make([]string, 0, len(paths))

	for _, path := range paths {
		_, err := c.GetFileInfo(ctx, path)
		if err != nil {
			if !errors.Is(err, client.ErrPathNotFound) {
				diagnostics.AddError("Failed to get file info", err.Error())
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

	task, err := c.RemoveFiles(ctx, filesToDelete)
	if err != nil {
		diagnostics.AddError("Failed to remove files", err.Error())
		return
	}

	if diags := waitForFileSystemTask(ctx, c, task.ID, polling); diags.HasError() {
		diagnostics.Append(diags...)
		return
	}

	return
}

type taskError struct {
	taskID    int64
	errorCode string
}

func (e *taskError) Error() string {
	return fmt.Sprintf("task %d failed with error code %q", e.taskID, e.errorCode)
}

// waitForFileSystemTask waits for the system task to complete.
func waitForFileSystemTask(ctx context.Context, client client.Client, taskID int64, polling pollingSpecModel) (diagnostics diag.Diagnostics) {
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

	tick := time.NewTicker(interval)
	defer tick.Stop()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		task, taskErr := client.GetFileSystemTask(ctx, taskID)
		if taskErr != nil {
			diagnostics.AddError("Failed to get file system task", taskErr.Error())
			return
		}

		switch task.State {
		case freeboxTypes.FileTaskStateFailed:
			diagnostics.AddError("File system task failed", fmt.Sprintf("Task %d failed with code: %s", taskID, task.Error))
			return
		case freeboxTypes.FileTaskStateDone:
			return nil // Done
		case freeboxTypes.FileTaskStatePaused:
			diagnostics.AddError("File system task paused", fmt.Sprintf("The file system task %d was paused", taskID))
			return
		default:
			// Nothing to do
		}

		select {
		case <-ctx.Done():
			diagnostics.AddError("File system task timeout", ctx.Err().Error())
			return
		case <-tick.C:
		}
	}
}

const (
	badHashError = "http_bad_hash"
)

// waitForDownloadTask waits for the download task to complete.
func waitForDownloadTask(ctx context.Context, c client.Client, taskID int64, polling pollingSpecModel) (diagnostics diag.Diagnostics) {
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

	tick := time.NewTicker(interval)
	defer tick.Stop()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		task, taskErr := c.GetDownloadTask(ctx, taskID)
		if taskErr != nil {
			diagnostics.AddError("Failed to get download task", taskErr.Error())
		}

		switch task.Status {
		case freeboxTypes.DownloadTaskStatusError:
			diagnostics.AddError("Download task failed", fmt.Sprintf("Error code: %s", task.Error))
			return
		case freeboxTypes.DownloadTaskStatusDone:
			return nil // Done
		case freeboxTypes.DownloadTaskStatusStopped:
			diagnostics.AddError("Download task stopped", "The download task was stopped")
			return
		default:
			// Nothing to do
		}

		select {
		case <-ctx.Done():
			diagnostics.AddError("Download task timeout", ctx.Err().Error())
			return
		case <-tick.C:
		}
	}
}

type errStoppedTask int64

func (e errStoppedTask) Error() string {
	return fmt.Sprintf("task %d is on pause, please resume it", int64(e))
}
