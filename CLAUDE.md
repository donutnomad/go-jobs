# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 架构必须遵守规范
`.claude/architecture-guide.md`

## 项目概述

这是一个基于 Go 的分布式任务调度系统，支持多实例部署、灵活的负载均衡策略和完善的任务管理功能。

## 常用命令

### 构建和运行
```bash
# 构建调度器
make build

# 运行调度器
make run
# 或直接运行
go run cmd/scheduler/main.go

# 运行示例执行器
make example
# 或直接运行
go run examples/executor/main.go

# 开发模式（热重载）
make dev
```

### 测试
```bash
# 运行所有测试
make test
# 或
go test -v ./...

# 运行特定测试
go test -v ./test/integration
```

### 代码质量
```bash
# 格式化代码
make fmt
# 或
go fmt ./...

# 代码检查
make lint
# 需要安装 golangci-lint
```

### 数据库
```bash
# 执行数据库迁移
make migrate
# 或直接执行
mysql -h 127.0.0.1 -P 3306 -u root -p123456 < scripts/migrate.sql
```

### Docker
```bash
# 构建 Docker 镜像
make docker-build

# 启动 Docker Compose 服务
make docker-up

# 停止 Docker Compose 服务
make docker-down
```

### Next.js UI
```bash
# 进入 UI 目录
cd scheduler-ui

# 安装依赖
npm install

# 开发模式
npm run dev

# 构建
npm run build

# 生产模式运行
npm start

# 代码检查
npm run lint
```

## 系统架构

### 核心组件

1. **Scheduler (调度器)**
   - 位置：`internal/scheduler/`
   - 负责任务调度，使用 cron 表达式定时触发任务
   - 实现分布式锁（MySQL GET_LOCK）进行主从选举
   - 管理任务执行和回调处理

2. **Executor Manager (执行器管理器)**
   - 位置：`internal/executor/`
   - 管理执行器注册和健康检查
   - 维护执行器状态（active/maintenance/offline）
   - 定期健康检查（30秒间隔）

3. **Load Balancer (负载均衡器)**
   - 位置：`internal/loadbalance/`
   - 实现 5 种策略：Round Robin、Weighted、Random、Sticky、Least Loaded
   - 状态持久化到数据库

4. **API Server**
   - 位置：`internal/api/`
   - 提供 RESTful API 接口
   - 基于 Gin 框架实现

5. **Storage Layer**
   - 位置：`internal/storage/`
   - 基于 GORM 的数据持久层
   - 处理所有数据库操作

### 数据库结构

- `tasks`：任务定义
- `executors`：执行器实例
- `task_executors`：任务与执行器关联
- `task_executions`：执行历史
- `load_balance_state`：负载均衡状态
- `scheduler_instances`：调度器实例

### 配置文件

主配置文件：`configs/config.yaml`

重要配置项：
- 数据库连接（默认：127.0.0.1:3306, root/123456, database: jobs）
- API 端口（默认：8080）
- 调度器实例 ID
- 健康检查配置

## 开发注意事项

1. **任务执行模式**
   - Sequential：串行执行
   - Parallel：并行执行
   - Skip：跳过执行

2. **回调机制**
   - 执行器完成任务后需要回调 `/api/v1/executions/{execution_id}/callback`
   - 回调包含执行状态、输出和错误信息

3. **健康检查**
   - 执行器需要实现 `/health` 端点
   - 失败阈值：3 次
   - 恢复阈值：2 次

4. **分布式锁**
   - 使用 MySQL GET_LOCK 实现
   - 锁键名：`scheduler_leader_lock`
   - 超时时间：30 秒

## API 测试示例

### 创建任务
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test_task",
    "cron_expression": "*/10 * * * * *",
    "execution_mode": "parallel",
    "load_balance_strategy": "round_robin"
  }'
```

### 注册执行器
```bash
curl -X POST http://localhost:8080/api/v1/executors/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "sample-executor",
    "instance_id": "executor-001",
    "base_url": "http://localhost:9090",
    "health_check_url": "http://localhost:9090/health"
  }'
```

### 手动触发任务
```bash
curl -X POST http://localhost:8080/api/v1/tasks/{task_id}/trigger
```

## 依赖包

主要依赖：
- gin-gonic/gin：Web 框架
- gorm.io/gorm：ORM
- robfig/cron/v3：Cron 调度
- google/uuid：UUID 生成
- zap：日志库
- viper：配置管理

## 端口说明

- 8080：调度器 API 端口
- 9090：示例执行器端口
- 3000：Next.js UI 端口（scheduler-ui）
- 3306：MySQL 数据库端口