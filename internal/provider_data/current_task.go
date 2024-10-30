package providerdata

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/terraform-provider-freebox/internal/models"
)

const taskKey = "task"

func UnsetCurrentTask(ctx context.Context, state Setter) (diagnotics diag.Diagnostics) {
	return state.SetKey(ctx, taskKey, nil)
}

func SetCurrentTask(ctx context.Context, state Setter, taskType models.TaskType, id int64) (diagnotics diag.Diagnostics) {
	taskBytes, err := json.Marshal(&models.Task{
		ID:   basetypes.NewInt64Value(id),
		Type: basetypes.NewStringValue(string(taskType)),
	})
	if err != nil {
		diagnotics.AddError("Failed to marshal task", fmt.Sprintf("Task: %d, Type: %s, Errorf: %v", id, taskType, err))
		return
	}

	return state.SetKey(ctx, taskKey, taskBytes)
}

func GetCurrentTask(ctx context.Context, state Getter) (task *models.Task, diagnotics diag.Diagnostics) {
	taskBytes, diags := state.GetKey(ctx, taskKey)
	if diags.HasError() {
		return nil, diags
	}

	if len(taskBytes) == 0 {
		return nil, nil
	}

	task = new(models.Task)
	if err := json.Unmarshal(taskBytes, task); err != nil {
		diagnotics.AddError("Failed to unmarshal task", fmt.Sprintf("Task: %s, Error: %v", string(taskBytes), err))
		return nil, diagnotics
	}

	if task.ID.IsNull() || task.ID.Equal(basetypes.NewInt64Value(0)) {
		return nil, nil
	}

	return task, nil
}
