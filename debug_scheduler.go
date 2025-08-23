package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jobs/scheduler/internal/models"
	"github.com/jobs/scheduler/internal/orm"
	"github.com/jobs/scheduler/pkg/config"
)

func main() {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	storage, err := orm.New(orm.Config{
		Host:                  cfg.Database.Host,
		Port:                  cfg.Database.Port,
		Database:              cfg.Database.Database,
		User:                  cfg.Database.User,
		Password:              cfg.Database.Password,
		MaxConnections:        cfg.Database.MaxConnections,
		MaxIdleConnections:    cfg.Database.MaxIdleConnections,
		ConnectionMaxLifetime: cfg.Database.ConnectionMaxLifetime,
	})
	if err != nil {
		log.Fatal(err)
	}

	var instances []models.SchedulerInstance
	if err := storage.DB().WithContext(context.Background()).Find(&instances).Error; err != nil {
		log.Fatal(err)
	}

	fmt.Printf("查询到 %d 个调度器实例:\n", len(instances))
	for _, inst := range instances {
		fmt.Printf("ID: %s, InstanceID: %s, Host: %s, Port: %d, IsLeader: %t\n",
			inst.ID, inst.InstanceID, inst.Host, inst.Port, inst.IsLeader)
	}
}