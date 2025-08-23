package executorrepo

import (
	domain "github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
)

func (po *Executor) FromDomain(in *domain.Executor) *Executor {
	return &Executor{
		Mode: commonrepo.Mode{
			ID:        in.ID,
			CreatedAt: in.CreatedAt,
			UpdatedAt: in.UpdatedAt,
		},
		Name:                in.Name,
		InstanceID:          in.InstanceID,
		BaseURL:             in.BaseURL,
		HealthCheckURL:      in.HealthCheckURL,
		Status:              in.Status,
		IsHealthy:           in.IsHealthy,
		LastHealthCheck:     in.LastHealthCheck,
		HealthCheckFailures: in.HealthCheckFailures,
		Metadata:            in.Metadata,
	}
}

func (po *Executor) ToDomain() *domain.Executor {
	return &domain.Executor{
		ID:                  po.ID,
		CreatedAt:           po.CreatedAt,
		UpdatedAt:           po.UpdatedAt,
		Name:                po.Name,
		InstanceID:          po.InstanceID,
		BaseURL:             po.BaseURL,
		HealthCheckURL:      po.HealthCheckURL,
		Status:              po.Status,
		IsHealthy:           po.IsHealthy,
		LastHealthCheck:     po.LastHealthCheck,
		HealthCheckFailures: po.HealthCheckFailures,
		Metadata:            po.Metadata,
	}
}

func patchToMap(input *domain.ExecutorPatch) map[string]any {
	if input == nil {
		return nil
	}
	var values = make(map[string]any)

	if input.Name != nil {
		values["name"] = *input.Name
	}

	if input.InstanceID != nil {
		values["instance_id"] = *input.InstanceID
	}

	if input.BaseURL != nil {
		values["base_url"] = *input.BaseURL
	}

	if input.HealthCheckURL != nil {
		values["health_check_url"] = *input.HealthCheckURL
	}

	if input.Status != nil {
		values["status"] = *input.Status
	}

	if input.IsHealthy != nil {
		values["is_healthy"] = *input.IsHealthy
	}

	if input.LastHealthCheck != nil {
		values["last_health_check"] = *input.LastHealthCheck
	}

	if input.HealthCheckFailures != nil {
		values["health_check_failures"] = *input.HealthCheckFailures
	}

	if input.Metadata != nil {
		values["metadata"] = *input.Metadata
	}

	return values
}
