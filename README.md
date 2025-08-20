# Go任务调度器

一个基于Go语言开发的分布式任务调度系统，支持多实例部署、灵活的负载均衡策略和完善的任务管理功能。

## 特性

- ✅ **分布式架构**：支持多实例部署，通过MySQL GET_LOCK实现主从选举
- ✅ **灵活的调度策略**：支持Cron表达式定时调度
- ✅ **多种执行模式**：
  - Sequential（串行）：等待前一个任务完成后才开始下一个
  - Parallel（并行）：按时触发，不管前一个是否完成
  - Skip（跳过）：如果前一个未完成，跳过本次调度
- ✅ **负载均衡策略**：
  - Round Robin（轮询）
  - Weighted Round Robin（加权轮询）
  - Random（随机）
  - Sticky（粘性）
  - Least Loaded（最少负载）
- ✅ **执行器管理**：
  - 支持执行器动态注册
  - 健康检查和心跳机制
  - 执行器维护模式
- ✅ **任务管理**：
  - RESTful API接口
  - 任务重试机制
  - 执行超时控制
  - 手动触发任务
- ✅ **监控和日志**：
  - 执行历史记录
  - 详细的执行日志
  - 任务状态跟踪

## 快速开始

### 前置要求

- Go 1.21+
- MySQL 8.0+
- Docker & Docker Compose（可选）

### 本地开发

1. **克隆项目**
```bash
git clone https://github.com/jobs/scheduler.git
cd scheduler
```

4. **修改配置**

编辑 `configs/config.yaml` 文件，配置数据库连接信息：
```yaml
database:
  host: 127.0.0.1
  port: 3306
  database: jobs
  user: root
  password: "123456"
```

5. **启动调度器**
```bash
go run cmd/scheduler/main.go
```

### Docker部署

使用Docker Compose一键部署：

```bash
cd docker
docker-compose up -d
```

这将启动：
- 1个MySQL实例
- 3个调度器实例（演示多实例部署）

访问不同的调度器实例：
- http://localhost:8081 - 调度器实例1
- http://localhost:8082 - 调度器实例2
- http://localhost:8083 - 调度器实例3

## API使用

### 注册执行器

```bash
curl -X POST http://localhost:8080/api/v1/executors/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "data-processor",
    "instance_id": "executor-001",
    "base_url": "http://localhost:9090",
    "health_check_url": "http://localhost:9090/health",
    "tasks": [
      {
        "task_name": "daily_report",
        "cron": "0 0 2 * * *",
        "execution_mode": "sequential",
        "load_balance_strategy": "round_robin",
        "timeout": 300
      }
    ]
  }'
```

### 创建任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hourly_sync",
    "cron_expression": "0 0 * * * *",
    "execution_mode": "parallel",
    "load_balance_strategy": "least_loaded",
    "max_retry": 3,
    "timeout_seconds": 600,
    "parameters": {
      "source": "database_a",
      "target": "database_b"
    }
  }'
```

### 手动触发任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks/{task_id}/trigger \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {
      "force": true
    }
  }'
```

### 查询执行历史

```bash
curl http://localhost:8080/api/v1/executions?task_id={task_id}&status=success&limit=10
```

### 更新执行器状态

```bash
curl -X PUT http://localhost:8080/api/v1/executors/{executor_id}/status \
  -H "Content-Type: application/json" \
  -d '{
    "status": "maintenance",
    "reason": "系统升级"
  }'
```

## 示例执行器

项目提供了一个简单的执行器示例，位于 `examples/executor` 目录：

```bash
cd examples/executor
go run main.go
```

执行器将在 9090 端口启动，可以接收调度器的任务调度请求。

## 系统架构

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Scheduler 1   │     │   Scheduler 2   │     │   Scheduler 3   │
│    (Leader)     │     │   (Follower)    │     │   (Follower)    │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                          ┌──────▼──────┐
                          │    MySQL    │
                          │  (GET_LOCK) │
                          └──────┬──────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         │                       │                       │
    ┌────▼─────┐          ┌─────▼─────┐          ┌─────▼─────┐
    │Executor 1│          │Executor 2 │          │Executor 3 │
    └──────────┘          └───────────┘          └───────────┘
```

## 配置说明

### 调度器配置

```yaml
scheduler:
  instance_id: "scheduler-001"        # 实例ID
  lock_key: "scheduler_leader_lock"   # 分布式锁键名
  lock_timeout: 30s                   # 锁超时时间
  heartbeat_interval: 10s             # 心跳间隔
  max_workers: 10                     # 最大工作协程数
```

### 健康检查配置

```yaml
health_check:
  enabled: true         # 是否启用健康检查
  interval: 30s        # 检查间隔
  timeout: 5s          # 检查超时
  failure_threshold: 3  # 失败阈值
  recovery_threshold: 2 # 恢复阈值
```

## 开发指南

### 项目结构

```
jobs/
├── cmd/scheduler/        # 主程序入口
├── internal/            # 内部包
│   ├── api/            # REST API
│   ├── executor/       # 执行器管理
│   ├── loadbalance/    # 负载均衡
│   ├── models/         # 数据模型
│   ├── scheduler/      # 调度器核心
│   └── storage/        # 存储层
├── pkg/                # 公共包
│   ├── config/         # 配置管理
│   └── logger/         # 日志工具
├── examples/           # 示例代码
├── scripts/            # 脚本文件
├── docker/             # Docker相关
└── configs/            # 配置文件
```

### 添加新的负载均衡策略

1. 在 `internal/loadbalance` 目录创建新的策略文件
2. 实现 `Strategy` 接口
3. 在 `Manager` 中注册新策略

### 自定义执行器

执行器需要实现以下接口：

- `POST /execute` - 接收任务执行请求
- `GET /health` - 健康检查端点

执行完成后，需要回调调度器的接口：
- `POST /api/v1/executions/{execution_id}/callback`

## 监控和运维

### 查看调度器状态

```bash
curl http://localhost:8080/api/v1/scheduler/status
```

### 查看执行器列表

```bash
curl http://localhost:8080/api/v1/executors
```

### 日志位置

- 本地开发：`logs/scheduler.log`
- Docker部署：容器内 `/app/logs/scheduler.log`

## License

MIT