# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个分布式任务调度系统，使用Go语言开发，包含以下核心组件：

### 核心架构

- **Scheduler**: 分布式任务调度器，支持领导者选举和高可用性
- **API Server**: 基于Gin框架的REST API服务，提供任务管理接口
- **Load Balancer**: 负载均衡管理器，支持多种策略（轮询、加权轮询、随机、粘性、最少加载）
- **Task Runner**: 任务执行引擎，支持多种执行模式（串行、并行、跳过）
- **Health Checker**: 执行器健康检查系统
- **Executor**: 任务执行器，可动态注册和注销
- **Web UI**: Next.js前端界面，用于任务管理

### 数据模型

- **Task**: 任务定义，包含Cron表达式、执行模式、负载均衡策略
- **TaskExecution**: 任务执行记录，跟踪执行状态和结果
- **Executor**: 执行器实例，支持动态注册
- **SchedulerInstance**: 调度器实例，用于分布式部署

## 常用命令

### 构建和运行
```bash
make build          # 构建调度器二进制文件
make run            # 运行调度器
make clean          # 清理构建产物
make example        # 运行示例执行器
```

### 开发
```bash
make dev            # 开发模式（热重载）
make fmt            # 格式化代码
make lint           # 代码静态检查
make deps           # 更新依赖
```

### 测试
```bash
make test           # 运行所有测试
go test ./internal/scheduler  # 运行单个包测试
go test -v ./...    # 详细测试输出
```

### 数据库
```bash
make migrate        # 运行数据库迁移
```

### Docker
```bash
make docker-build   # 构建Docker镜像
make docker-up      # 启动Docker服务
make docker-down    # 停止Docker服务
```

### 前端UI
```bash
cd scheduler-ui
npm run dev         # 开发模式
npm run build       # 构建生产版本
npm run lint        # ESLint检查
```

## 代码生成

项目使用代码生成工具：
```bash
go generate ./...   # 生成API包装器和接口代码
```

## 配置文件

- `configs/config.yaml`: 主配置文件，包含数据库、服务器、调度器等配置
- 支持通过`-config`参数指定配置文件路径

## 项目结构要点

- `cmd/scheduler/`: 调度器主程序入口
- `internal/scheduler/`: 调度器核心逻辑，包含领导者选举、任务调度
- `internal/api/`: REST API接口实现
- `internal/loadbalance/`: 负载均衡策略实现
- `internal/models/`: 数据模型定义
- `internal/orm/`: 数据库操作层
- `examples/`: 执行器示例代码
- `scheduler-ui/`: Next.js前端界面

## 关键技术栈

- **后端**: Go 1.23.1, Gin Web框架, GORM ORM, Cron调度
- **数据库**: MySQL
- **前端**: Next.js 15, React 19, TypeScript, Tailwind CSS
- **依赖注入**: Google Wire
- **日志**: Uber Zap
- **配置**: Viper

## 分布式特性

- 支持多实例部署，通过分布式锁实现领导者选举
- 执行器可动态注册和健康检查
- 支持多种负载均衡和执行模式
- 具备故障转移和高可用能力

## ⚠️ 重要开发规范

### 数据库关联处理 🚫

**严格禁止使用GORM自带的关联功能，必须在应用层手动实现表关联**

#### 禁止使用的功能：
- ❌ `Preload()` 方法 - 会生成复杂的JOIN查询
- ❌ `Joins()` 方法 - 直接使用JOIN语句
- ❌ 模型结构体中的关联标签，如：
  ```go
  // 禁止这样做
  TaskExecutors []TaskExecutor `gorm:"foreignKey:TaskID"`
  Task *Task `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
  ```

#### 正确的做法：
- ✅ 使用 `gorm:"-"` 标签排除数据库关联但保留JSON序列化
- ✅ 在应用层通过单独查询手动填充关联数据
- ✅ 使用简单的WHERE条件查询关联表数据

#### 示例代码：

```go
// ✅ 正确的模型定义
type Task struct {
    ID   string `gorm:"primaryKey;size:64" json:"id"`
    Name string `gorm:"size:255;not null" json:"name"`
    // ... 其他字段
    
    // 在应用层手动填充的关联字段（不使用GORM关联）
    TaskExecutors []TaskExecutor `gorm:"-" json:"task_executors,omitempty"`
}

// ✅ 正确的关联数据填充方式
func (r *taskRepository) GetByID(ctx context.Context, id string) (*Task, error) {
    var task Task
    if err := r.db.Where("id = ?", id).First(&task).Error; err != nil {
        return nil, err
    }
    
    // 手动查询关联的TaskExecutors
    var taskExecutors []TaskExecutor
    if err := r.db.Where("task_id = ?", id).Find(&taskExecutors).Error; err == nil {
        task.TaskExecutors = taskExecutors
        
        // 进一步查询每个TaskExecutor的关联Executor
        for i := range taskExecutors {
            var executor Executor
            if err := r.db.Where("name = ?", taskExecutors[i].ExecutorName).First(&executor).Error; err == nil {
                task.TaskExecutors[i].Executor = &executor
            }
        }
    }
    
    return &task, nil
}
```

#### 原因说明：
1. **性能考虑**: GORM的Preload会生成复杂的JOIN查询，影响数据库性能
2. **架构清晰**: 在应用层控制关联逻辑，更容易理解和维护
3. **避免循环依赖**: 防止模型间的复杂关联导致的问题
4. **灵活控制**: 可以根据业务需要选择性加载关联数据

#### 违规检查：
- 代码中不应出现 `.Preload(` 调用
- 代码中不应出现 `.Joins(` 调用（除非是必要的统计查询）
- 模型结构体中不应包含GORM关联标签

**此规范为强制性要求，所有开发者必须严格遵守！**