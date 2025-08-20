//go:build wireinject

package wire

import (
	"github.com/google/wire"
	"go.uber.org/zap"
	"gorm.io/gorm"

	// Domain
	"github.com/jobs/scheduler/internal/app/biz/execution"
	"github.com/jobs/scheduler/internal/app/biz/task"

	// Infrastructure
	executionRepo "github.com/jobs/scheduler/infra/persistence/execution"
	taskRepo "github.com/jobs/scheduler/infra/persistence/task"
	"github.com/jobs/scheduler/infra/queries"

	// Service Layer
	"github.com/jobs/scheduler/internal/app/service"

	// API Layer
	"github.com/jobs/scheduler/internal/app/api"
)

// ProvideApplication 提供完整应用程序
func ProvideApplication(db *gorm.DB, logger *zap.Logger) (*api.Router, error) {
	wire.Build(
		// Repository层（已经返回接口类型）
		taskRepo.NewMysqlRepository,
		executionRepo.NewMysqlRepository,

		// Query层（已经返回接口类型）
		queries.NewTaskQueryService,
		queries.NewExecutionQueryService,

		// Domain Service层
		execution.NewDomainService,

		// UseCase层
		task.NewUseCase,
		execution.NewUseCase,

		// Service层
		service.NewTaskService,
		service.NewExecutionService,

		// API层
		api.NewRouter,
	)
	return nil, nil
}
