package executionrepo

import (
	domain "github.com/jobs/scheduler/internal/biz/execution"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
)

func (po *TaskExecution) ToDomain() *domain.TaskExecution {
	return &domain.TaskExecution{
		ID:            po.ID,
		CreatedAt:     po.CreatedAt,
		UpdatedAt:     po.UpdatedAt,
		TaskID:        po.TaskID,
		ExecutorID:    po.ExecutorID,
		ScheduledTime: po.ScheduledTime,
		StartTime:     po.StartTime,
		EndTime:       po.EndTime,
		Status:        po.Status,
		Result:        po.Result,
		Logs:          po.Logs,
		RetryCount:    po.RetryCount,
	}
}

func (po *TaskExecution) FromDomain(domain *domain.TaskExecution) *TaskExecution {
	return &TaskExecution{
		Mode: commonrepo.Mode{
			ID:        domain.ID,
			CreatedAt: domain.CreatedAt,
			UpdatedAt: domain.UpdatedAt,
		},
		TaskID:        domain.TaskID,
		ExecutorID:    domain.ExecutorID,
		ScheduledTime: domain.ScheduledTime,
		StartTime:     domain.StartTime,
		EndTime:       domain.EndTime,
		Status:        domain.Status,
		Result:        domain.Result,
		Logs:          domain.Logs,
		RetryCount:    domain.RetryCount,
	}
}

func patchToMap(input *domain.TaskExecutionPatch) map[string]any {
	var values = make(map[string]any)
	if input.StartTime != nil {
		values["start_time"] = input.StartTime
	}
	if input.EndTime != nil {
		values["end_time"] = input.EndTime
	}
	if input.Status != nil {
		values["status"] = input.Status
	}
	if input.Result != nil {
		values["result"] = input.Result
	}
	if input.Logs != nil {
		values["logs"] = input.Logs
	}
	if input.RetryCount != nil {
		values["retry_count"] = input.RetryCount
	}
	return values
}
