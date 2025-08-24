//go:build wireinject
// +build wireinject

package main

//go:generate go run -mod=mod github.com/google/wire/cmd/wire

import (
	"github.com/google/wire"
	"github.com/jobs/scheduler/internal/api"
	"github.com/jobs/scheduler/internal/biz/executor"
	"github.com/jobs/scheduler/internal/biz/task"
	"github.com/jobs/scheduler/internal/infra/persistence/commonrepo"
	"github.com/jobs/scheduler/internal/infra/persistence/executionrepo"
	"github.com/jobs/scheduler/internal/infra/persistence/executorrepo"
	"github.com/jobs/scheduler/internal/infra/persistence/loadbalancerepo"
	"github.com/jobs/scheduler/internal/infra/persistence/schedulerinstancerepo"
	"github.com/jobs/scheduler/internal/infra/persistence/taskrepo"
	"github.com/jobs/scheduler/internal/loadbalance"
	"github.com/jobs/scheduler/internal/scheduler"
	"github.com/jobs/scheduler/pkg/config"
	"go.uber.org/zap"
)

func InitilizeApp(logger *zap.Logger, cfg config.Config, db commonrepo.DB) (*App, error) {
	wire.Build(
		NewApp,

		ProvideHealthCheckConfig,
		ProvideTaskRunnerConfig,

		wire.Bind(new(scheduler.IEmitter), new(*scheduler.EventBus)),
		wire.Bind(new(scheduler.ITaskRunner), new(*scheduler.TaskRunner)),

		// other
		scheduler.Provider,
		loadbalance.Provider,

		// http api providers
		api.Provider,

		// biz providers
		executor.Provider,
		task.Provider,

		// infra providers
		taskrepo.Provider,
		executorrepo.Provider,
		executionrepo.Provider,
		schedulerinstancerepo.Provider,
		loadbalancerepo.Provider,
	)
	return nil, nil
}
