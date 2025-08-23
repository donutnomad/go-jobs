package taskrepo

import (
	domain "github.com/jobs/scheduler/internal/biz/task"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
)

func (po *TaskPo) FromDomain(in *domain.Task) *TaskPo {
	return &TaskPo{
		Mode: commonrepo.Mode{
			ID:        in.ID,
			CreatedAt: in.CreatedAt,
			UpdatedAt: in.UpdatedAt,
		},
		Name:                in.Name,
		CronExpression:      in.CronExpression,
		Parameters:          in.Parameters,
		ExecutionMode:       in.ExecutionMode,
		LoadBalanceStrategy: in.LoadBalanceStrategy,
		Status:              in.Status,
		MaxRetry:            in.MaxRetry,
		TimeoutSeconds:      in.TimeoutSeconds,
	}
}

func (po *TaskPo) ToDomain() *domain.Task {
	return &domain.Task{
		ID:                  po.ID,
		CreatedAt:           po.CreatedAt,
		UpdatedAt:           po.UpdatedAt,
		Name:                po.Name,
		CronExpression:      po.CronExpression,
		Parameters:          po.Parameters,
		ExecutionMode:       po.ExecutionMode,
		LoadBalanceStrategy: po.LoadBalanceStrategy,
		Status:              po.Status,
		MaxRetry:            po.MaxRetry,
		TimeoutSeconds:      po.TimeoutSeconds,
	}
}

func (po *TaskAssignmentPo) FromDomain(in *domain.TaskAssignment) *TaskAssignmentPo {
	return &TaskAssignmentPo{
		Mode: commonrepo.Mode{
			ID: in.ID,
		},
		TaskID:       in.TaskID,
		ExecutorName: in.ExecutorName,
		Priority:     in.Priority,
		Weight:       in.Weight,
	}
}

func (po *TaskAssignmentPo) ToDomain() *domain.TaskAssignment {
	return &domain.TaskAssignment{
		ID:           po.ID,
		TaskID:       po.TaskID,
		ExecutorName: po.ExecutorName,
		Priority:     po.Priority,
		Weight:       po.Weight,
	}
}

func patchToMap(input *domain.TaskPatch) map[string]any {
	var values = make(map[string]any)

	if input.Name != nil {
		values["name"] = *input.Name
	}

	if input.CronExpression != nil {
		values["cron_expression"] = *input.CronExpression
	}

	if input.Parameters != nil {
		values["parameters"] = *input.Parameters
	}

	if input.ExecutionMode != nil {
		values["execution_mode"] = *input.ExecutionMode
	}

	if input.LoadBalanceStrategy != nil {
		values["load_balance_strategy"] = *input.LoadBalanceStrategy
	}

	if input.Status != nil {
		values["status"] = *input.Status
	}

	if input.MaxRetry != nil {
		values["max_retry"] = *input.MaxRetry
	}

	if input.TimeoutSeconds != nil {
		values["timeout_seconds"] = *input.TimeoutSeconds
	}

	return values
}

func assignmentPatchToMap(input *domain.TaskAssignmentPatch) map[string]any {
	var values = make(map[string]any)

	if input.Priority != nil {
		values["priority"] = *input.Priority
	}

	if input.Weight != nil {
		values["weight"] = *input.Weight
	}

	return values
}
