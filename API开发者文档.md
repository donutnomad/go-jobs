# 分布式任务调度系统 - API 开发者文档

## 目录
- [1. 文档概述](#1-文档概述)
- [2. 数据库设计](#2-数据库设计)
- [3. 数据模型](#3-数据模型)
- [4. API 概览](#4-api-概览)
- [5. 任务管理 API](#5-任务管理-api)
- [6. 执行器管理 API](#6-执行器管理-api)
- [7. 执行历史 API](#7-执行历史-api)
- [8. 系统监控 API](#8-系统监控-api)
- [9. 错误码定义](#9-错误码定义)
- [10. 认证机制](#10-认证机制)
- [11. 开发指南](#11-开发指南)

## 1. 文档概述

### 1.1 文档目标
本文档为分布式任务调度系统的 HTTP API 提供完整的开发者指南，包含所有接口定义、数据模型、错误处理和最佳实践。

### 1.2 API 基本信息
- **Base URL**: `http://localhost:8080/api/v1`
- **协议**: HTTP/HTTPS
- **数据格式**: JSON
- **字符编码**: UTF-8
- **时区**: UTC

### 1.3 版本控制
- **当前版本**: v1
- **版本策略**: 通过 URL 路径进行版本控制
- **向后兼容**: 主版本内保证向后兼容

## 2. 数据库设计

### 2.1 数据库关系图

```mermaid
erDiagram
    tasks {
        varchar id PK
        varchar name UK
        varchar cron_expression
        json parameters
        enum execution_mode
        enum load_balance_strategy
        int max_retry
        int timeout_seconds
        enum status
        timestamp created_at
        timestamp updated_at
    }
    
    executors {
        varchar id PK
        varchar name
        varchar instance_id UK
        varchar base_url
        varchar health_check_url
        enum status
        boolean is_healthy
        timestamp last_health_check
        int health_check_failures
        json metadata
        timestamp created_at
        timestamp updated_at
    }
    
    task_executors {
        varchar id PK
        varchar task_id FK
        varchar executor_id FK
        int priority
        int weight
        timestamp created_at
    }
    
    task_executions {
        varchar id PK
        varchar task_id FK
        varchar executor_id FK
        timestamp scheduled_time
        timestamp start_time
        timestamp end_time
        enum status
        json result
        text logs
        int retry_count
        timestamp created_at
    }
    
    load_balance_state {
        varchar task_id PK,FK
        varchar last_executor_id FK
        int round_robin_index
        varchar sticky_executor_id FK
        timestamp updated_at
    }
    
    scheduler_instances {
        varchar id PK
        varchar instance_id UK
        varchar host
        int port
        boolean is_leader
        timestamp created_at
        timestamp updated_at
    }
    
    tasks ||--o{ task_executors : "has"
    executors ||--o{ task_executors : "assigned to"
    tasks ||--o{ task_executions : "generates"
    executors ||--o{ task_executions : "executes"
    tasks ||--|| load_balance_state : "has"
    executors ||--o{ load_balance_state : "last used"
    executors ||--o{ load_balance_state : "sticky to"
```

### 2.2 表结构详解

#### 2.2.1 tasks - 任务定义表
任务是系统的核心实体，定义了定时执行的作业规则。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | VARCHAR(64) | PRIMARY KEY | 任务唯一标识 |
| name | VARCHAR(255) | NOT NULL, UNIQUE | 任务名称，必须唯一 |
| cron_expression | VARCHAR(100) | NOT NULL | Cron 表达式，支持秒级 |
| parameters | JSON | - | 执行参数，JSON 格式 |
| execution_mode | ENUM | DEFAULT 'parallel' | 执行模式：sequential/parallel/skip |
| load_balance_strategy | ENUM | DEFAULT 'round_robin' | 负载均衡策略 |
| max_retry | INT | DEFAULT 3 | 最大重试次数 |
| timeout_seconds | INT | DEFAULT 300 | 超时时间（秒） |
| status | ENUM | DEFAULT 'active' | 任务状态：active/paused/deleted |
| created_at | TIMESTAMP | AUTO | 创建时间 |
| updated_at | TIMESTAMP | AUTO | 更新时间 |

**索引设计**:
```sql
-- 按状态查询索引
INDEX idx_status (status)
-- 按名称查询索引  
INDEX idx_name (name)
```

#### 2.2.2 executors - 执行器表
记录所有注册的执行器实例信息。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | VARCHAR(64) | PRIMARY KEY | 执行器唯一标识 |
| name | VARCHAR(255) | NOT NULL | 执行器名称 |
| instance_id | VARCHAR(255) | NOT NULL, UNIQUE | 实例ID，全局唯一 |
| base_url | VARCHAR(500) | NOT NULL | 执行器基础URL |
| health_check_url | VARCHAR(500) | - | 健康检查URL |
| status | ENUM | DEFAULT 'online' | 状态：online/offline/maintenance |
| is_healthy | BOOLEAN | DEFAULT TRUE | 是否健康 |
| last_health_check | TIMESTAMP | - | 最后健康检查时间 |
| health_check_failures | INT | DEFAULT 0 | 连续健康检查失败次数 |
| metadata | JSON | - | 元数据信息 |
| created_at | TIMESTAMP | AUTO | 创建时间 |
| updated_at | TIMESTAMP | AUTO | 更新时间 |

**索引设计**:
```sql
-- 状态和健康状态组合索引
INDEX idx_status_healthy (status, is_healthy)
-- 名称和实例组合索引
INDEX idx_name_instance (name, instance_id)
-- 实例ID唯一索引
UNIQUE KEY uk_instance_id (instance_id)
```

#### 2.2.3 task_executors - 任务执行器关联表
定义任务和执行器的多对多关系，支持权重配置。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | VARCHAR(64) | PRIMARY KEY | 关联记录ID |
| task_id | VARCHAR(64) | NOT NULL, FK | 任务ID |
| executor_id | VARCHAR(64) | NOT NULL, FK | 执行器ID |
| priority | INT | DEFAULT 0 | 优先级（数字越小优先级越高） |
| weight | INT | DEFAULT 1 | 权重（用于加权负载均衡） |
| created_at | TIMESTAMP | AUTO | 创建时间 |

**约束设计**:
```sql
-- 任务和执行器唯一约束
UNIQUE KEY uk_task_executor (task_id, executor_id)
-- 外键约束
FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
FOREIGN KEY (executor_id) REFERENCES executors(id) ON DELETE CASCADE
```

#### 2.2.4 task_executions - 任务执行历史表
记录每次任务执行的详细信息。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | VARCHAR(64) | PRIMARY KEY | 执行记录ID |
| task_id | VARCHAR(64) | NOT NULL, FK | 任务ID |
| executor_id | VARCHAR(64) | FK | 执行器ID（可为空） |
| scheduled_time | TIMESTAMP | NOT NULL | 计划执行时间 |
| start_time | TIMESTAMP | - | 实际开始时间 |
| end_time | TIMESTAMP | - | 结束时间 |
| status | ENUM | DEFAULT 'pending' | 执行状态 |
| result | JSON | - | 执行结果 |
| logs | TEXT | - | 执行日志 |
| retry_count | INT | DEFAULT 0 | 重试次数 |
| created_at | TIMESTAMP | AUTO | 创建时间 |

**执行状态枚举值**:
- `pending`: 等待执行
- `running`: 正在执行
- `success`: 执行成功
- `failed`: 执行失败
- `timeout`: 执行超时
- `skipped`: 跳过执行
- `cancelled`: 取消执行

**索引设计**:
```sql
-- 任务和状态组合索引
INDEX idx_task_status (task_id, status)
-- 调度时间索引
INDEX idx_scheduled_time (scheduled_time)
-- 状态索引
INDEX idx_status (status)
```

#### 2.2.5 load_balance_state - 负载均衡状态表
存储各种负载均衡策略的状态信息。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| task_id | VARCHAR(64) | PRIMARY KEY, FK | 任务ID |
| last_executor_id | VARCHAR(64) | FK | 上次使用的执行器ID |
| round_robin_index | INT | DEFAULT 0 | 轮询策略的当前索引 |
| sticky_executor_id | VARCHAR(64) | FK | 粘性策略绑定的执行器ID |
| updated_at | TIMESTAMP | AUTO | 更新时间 |

#### 2.2.6 scheduler_instances - 调度器实例表
支持分布式部署的调度器实例管理。

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | VARCHAR(64) | PRIMARY KEY | 实例记录ID |
| instance_id | VARCHAR(255) | NOT NULL, UNIQUE | 调度器实例ID |
| host | VARCHAR(255) | NOT NULL | 主机地址 |
| port | INT | NOT NULL | 端口号 |
| is_leader | BOOLEAN | DEFAULT FALSE | 是否为主节点 |
| created_at | TIMESTAMP | AUTO | 创建时间 |
| updated_at | TIMESTAMP | AUTO | 更新时间 |

## 3. 数据模型

### 3.1 核心枚举类型

#### 3.1.1 ExecutionMode - 执行模式
```go
type ExecutionMode string

const (
    ExecutionModeSequential ExecutionMode = "sequential" // 串行执行
    ExecutionModeParallel   ExecutionMode = "parallel"   // 并行执行
    ExecutionModeSkip       ExecutionMode = "skip"       // 跳过执行
)
```

#### 3.1.2 LoadBalanceStrategy - 负载均衡策略
```go
type LoadBalanceStrategy string

const (
    LoadBalanceRoundRobin         LoadBalanceStrategy = "round_robin"          // 轮询
    LoadBalanceWeightedRoundRobin LoadBalanceStrategy = "weighted_round_robin" // 加权轮询
    LoadBalanceRandom             LoadBalanceStrategy = "random"               // 随机
    LoadBalanceSticky             LoadBalanceStrategy = "sticky"               // 粘性
    LoadBalanceLeastLoaded        LoadBalanceStrategy = "least_loaded"         // 最少负载
)
```

#### 3.1.3 TaskStatus - 任务状态
```go
type TaskStatus string

const (
    TaskStatusActive  TaskStatus = "active"  // 激活状态
    TaskStatusPaused  TaskStatus = "paused"  // 暂停状态
    TaskStatusDeleted TaskStatus = "deleted" // 删除状态
)
```

#### 3.1.4 ExecutorStatus - 执行器状态
```go
type ExecutorStatus string

const (
    ExecutorStatusOnline      ExecutorStatus = "online"      // 在线
    ExecutorStatusOffline     ExecutorStatus = "offline"     // 离线
    ExecutorStatusMaintenance ExecutorStatus = "maintenance" // 维护中
)
```

### 3.2 请求/响应模型

#### 3.2.1 任务相关模型
```go
// 创建任务请求
type CreateTaskRequest struct {
    Name                   string                     `json:"name" binding:"required"`
    CronExpression         string                     `json:"cron_expression" binding:"required"`
    Parameters             map[string]interface{}     `json:"parameters"`
    ExecutionMode          ExecutionMode              `json:"execution_mode"`
    LoadBalanceStrategy    LoadBalanceStrategy        `json:"load_balance_strategy"`
    MaxRetry               int                        `json:"max_retry"`
    TimeoutSeconds         int                        `json:"timeout_seconds"`
}

// 更新任务请求
type UpdateTaskRequest struct {
    Name                   string                     `json:"name"`
    CronExpression         string                     `json:"cron_expression"`
    Parameters             map[string]interface{}     `json:"parameters"`
    ExecutionMode          ExecutionMode              `json:"execution_mode"`
    LoadBalanceStrategy    LoadBalanceStrategy        `json:"load_balance_strategy"`
    MaxRetry               int                        `json:"max_retry"`
    TimeoutSeconds         int                        `json:"timeout_seconds"`
    Status                 TaskStatus                 `json:"status"`
}

// 任务响应模型
type TaskResponse struct {
    ID                     string                     `json:"id"`
    Name                   string                     `json:"name"`
    CronExpression         string                     `json:"cron_expression"`
    Parameters             map[string]interface{}     `json:"parameters"`
    ExecutionMode          ExecutionMode              `json:"execution_mode"`
    LoadBalanceStrategy    LoadBalanceStrategy        `json:"load_balance_strategy"`
    MaxRetry               int                        `json:"max_retry"`
    TimeoutSeconds         int                        `json:"timeout_seconds"`
    Status                 TaskStatus                 `json:"status"`
    CreatedAt              time.Time                  `json:"created_at"`
    UpdatedAt              time.Time                  `json:"updated_at"`
    TaskExecutors          []TaskExecutorResponse     `json:"task_executors,omitempty"`
}
```

#### 3.2.2 执行器相关模型
```go
// 执行器注册请求
type RegisterRequest struct {
    Name           string                 `json:"name" binding:"required"`
    InstanceID     string                 `json:"instance_id" binding:"required"`
    BaseURL        string                 `json:"base_url" binding:"required"`
    HealthCheckURL string                 `json:"health_check_url"`
    Metadata       map[string]interface{} `json:"metadata"`
}

// 执行器响应模型
type ExecutorResponse struct {
    ID                  string                 `json:"id"`
    Name                string                 `json:"name"`
    InstanceID          string                 `json:"instance_id"`
    BaseURL             string                 `json:"base_url"`
    HealthCheckURL      string                 `json:"health_check_url"`
    Status              ExecutorStatus         `json:"status"`
    IsHealthy           bool                   `json:"is_healthy"`
    LastHealthCheck     *time.Time             `json:"last_health_check"`
    HealthCheckFailures int                    `json:"health_check_failures"`
    Metadata            map[string]interface{} `json:"metadata"`
    CreatedAt           time.Time              `json:"created_at"`
    UpdatedAt           time.Time              `json:"updated_at"`
}
```

#### 3.2.3 执行历史相关模型
```go
// 执行记录响应模型
type TaskExecutionResponse struct {
    ID            string                 `json:"id"`
    TaskID        string                 `json:"task_id"`
    ExecutorID    *string                `json:"executor_id"`
    ScheduledTime time.Time              `json:"scheduled_time"`
    StartTime     *time.Time             `json:"start_time"`
    EndTime       *time.Time             `json:"end_time"`
    Status        ExecutionStatus        `json:"status"`
    Result        map[string]interface{} `json:"result"`
    Logs          string                 `json:"logs"`
    RetryCount    int                    `json:"retry_count"`
    CreatedAt     time.Time              `json:"created_at"`
    Task          *TaskResponse          `json:"task,omitempty"`
    Executor      *ExecutorResponse      `json:"executor,omitempty"`
}

// 执行回调请求
type ExecutionCallbackRequest struct {
    Status  ExecutionStatus        `json:"status" binding:"required"`
    Result  map[string]interface{} `json:"result"`
    Logs    string                 `json:"logs"`
    EndTime *time.Time             `json:"end_time"`
}
```

## 4. API 概览

### 4.1 API 分组结构

```
/api/v1
├── /health                           # 健康检查
├── /tasks                            # 任务管理
│   ├── GET /                         # 获取任务列表
│   ├── POST /                        # 创建任务
│   ├── GET /:id                      # 获取任务详情
│   ├── PUT /:id                      # 更新任务
│   ├── DELETE /:id                   # 删除任务
│   ├── POST /:id/trigger             # 手动触发任务
│   ├── POST /:id/pause               # 暂停任务
│   ├── POST /:id/resume              # 恢复任务
│   ├── GET /:id/stats                # 获取任务统计
│   ├── GET /:id/executors            # 获取任务的执行器
│   ├── POST /:id/executors           # 为任务分配执行器
│   ├── PUT /:id/executors/:executor_id    # 更新执行器分配
│   └── DELETE /:id/executors/:executor_id # 取消执行器分配
├── /executors                        # 执行器管理
│   ├── GET /                         # 获取执行器列表
│   ├── GET /:id                      # 获取执行器详情
│   ├── POST /register                # 注册执行器
│   ├── PUT /:id                      # 更新执行器信息
│   ├── PUT /:id/status               # 更新执行器状态
│   └── DELETE /:id                   # 删除执行器
├── /executions                       # 执行历史
│   ├── GET /                         # 获取执行历史列表
│   ├── GET /stats                    # 获取执行统计
│   ├── GET /:id                      # 获取执行详情
│   ├── POST /:id/callback            # 执行回调
│   └── POST /:id/stop                # 停止执行
└── /scheduler                        # 系统状态
    └── GET /status                   # 获取调度器状态
```

### 4.2 通用响应格式

#### 4.2.1 成功响应格式
```json
{
  "code": 200,
  "message": "success",
  "data": {...}
}
```

#### 4.2.2 错误响应格式
```json
{
  "code": 400,
  "message": "参数验证失败",
  "error": "name field is required"
}
```

#### 4.2.3 分页响应格式
```json
{
  "data": [...],
  "total": 100,
  "page": 1,
  "page_size": 20,
  "total_pages": 5
}
```

## 5. 任务管理 API

### 5.1 获取任务列表

**接口定义**
```
GET /api/v1/tasks
```

**功能描述**：获取任务列表，支持分页和状态筛选。

**查询参数**：
| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| status | string | 否 | 按状态筛选 | active, paused, deleted |
| page | int | 否 | 页码，默认为 1 | 1 |
| page_size | int | 否 | 每页大小，默认为 20 | 20 |
| name | string | 否 | 按名称模糊搜索 | data_sync |

**响应示例**：
```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "data_sync_task",
      "cron_expression": "0 */5 * * * *",
      "parameters": {
        "source": "database_a",
        "target": "database_b"
      },
      "execution_mode": "sequential",
      "load_balance_strategy": "round_robin",
      "max_retry": 3,
      "timeout_seconds": 600,
      "status": "active",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z",
      "task_executors": [
        {
          "executor_id": "exec-001",
          "priority": 1,
          "weight": 2
        }
      ]
    }
  ]
}
```

### 5.2 创建任务

**接口定义**
```
POST /api/v1/tasks
```

**功能描述**：创建新的定时任务。

**请求头**：
```
Content-Type: application/json
```

**请求体**：
```json
{
  "name": "data_sync_task",
  "cron_expression": "0 */5 * * * *",
  "parameters": {
    "source": "database_a",
    "target": "database_b",
    "batch_size": 1000
  },
  "execution_mode": "sequential",
  "load_balance_strategy": "round_robin",
  "max_retry": 3,
  "timeout_seconds": 600
}
```

**字段说明**：
| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| name | string | 是 | 任务名称，全局唯一 |
| cron_expression | string | 是 | Cron 表达式，支持秒级精度 |
| parameters | object | 否 | 任务参数，JSON 对象 |
| execution_mode | string | 否 | 执行模式，默认为 parallel |
| load_balance_strategy | string | 否 | 负载均衡策略，默认为 round_robin |
| max_retry | int | 否 | 最大重试次数，默认为 3 |
| timeout_seconds | int | 否 | 超时时间（秒），默认为 300 |

**Cron 表达式格式**：
```
格式：秒 分 时 日 月 星期
示例：
0 */5 * * * *     # 每5分钟执行一次
0 0 2 * * *       # 每天凌晨2点执行
0 0 0 1 * *       # 每月1号执行
0 30 9-17 * * 1-5 # 工作日9:30-17:30每小时执行
```

**响应示例**：
```json
{
  "code": 201,
  "message": "任务创建成功",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "data_sync_task",
    "cron_expression": "0 */5 * * * *",
    "parameters": {
      "source": "database_a",
      "target": "database_b",
      "batch_size": 1000
    },
    "execution_mode": "sequential",
    "load_balance_strategy": "round_robin",
    "max_retry": 3,
    "timeout_seconds": 600,
    "status": "active",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

### 5.3 获取任务详情

**接口定义**
```
GET /api/v1/tasks/{id}
```

**功能描述**：根据任务 ID 获取任务详细信息。

**路径参数**：
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | string | 是 | 任务ID |

**响应示例**：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "data_sync_task",
    "cron_expression": "0 */5 * * * *",
    "parameters": {
      "source": "database_a",
      "target": "database_b"
    },
    "execution_mode": "sequential",
    "load_balance_strategy": "round_robin",
    "max_retry": 3,
    "timeout_seconds": 600,
    "status": "active",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z",
    "task_executors": [
      {
        "id": "te-001",
        "executor_id": "exec-001",
        "priority": 1,
        "weight": 2,
        "executor": {
          "id": "exec-001",
          "name": "data-processor-1",
          "status": "online",
          "is_healthy": true
        }
      }
    ]
  }
}
```

### 5.4 更新任务

**接口定义**
```
PUT /api/v1/tasks/{id}
```

**功能描述**：更新现有任务的配置信息。

**请求体**（所有字段均为可选）：
```json
{
  "name": "updated_data_sync_task",
  "cron_expression": "0 */10 * * * *",
  "parameters": {
    "source": "database_a",
    "target": "database_b",
    "batch_size": 2000
  },
  "execution_mode": "parallel",
  "load_balance_strategy": "weighted_round_robin",
  "max_retry": 5,
  "timeout_seconds": 900,
  "status": "paused"
}
```

**响应示例**：
```json
{
  "code": 200,
  "message": "任务更新成功",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "updated_data_sync_task",
    "cron_expression": "0 */10 * * * *",
    "parameters": {
      "source": "database_a",
      "target": "database_b",
      "batch_size": 2000
    },
    "execution_mode": "parallel",
    "load_balance_strategy": "weighted_round_robin",
    "max_retry": 5,
    "timeout_seconds": 900,
    "status": "paused",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T01:00:00Z"
  }
}
```

### 5.5 删除任务

**接口定义**
```
DELETE /api/v1/tasks/{id}
```

**功能描述**：软删除任务（将状态设置为 deleted）。

**路径参数**：
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | string | 是 | 任务ID |

**响应示例**：
```json
{
  "code": 200,
  "message": "任务删除成功"
}
```

### 5.6 手动触发任务

**接口定义**
```
POST /api/v1/tasks/{id}/trigger
```

**功能描述**：立即执行指定任务，可以覆盖运行时参数。

**请求体**：
```json
{
  "parameters": {
    "source": "database_c",
    "force_sync": true
  }
}
```

**字段说明**：
| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| parameters | object | 否 | 运行时参数，会与任务默认参数合并 |

**响应示例**：
```json
{
  "code": 200,
  "message": "任务触发成功",
  "data": {
    "id": "exec-550e8400-e29b-41d4-a716-446655440001",
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "scheduled_time": "2024-01-01T01:00:00Z",
    "status": "pending",
    "created_at": "2024-01-01T01:00:00Z"
  }
}
```

### 5.7 暂停任务

**接口定义**
```
POST /api/v1/tasks/{id}/pause
```

**功能描述**：暂停任务调度，不会影响正在执行的任务实例。

**响应示例**：
```json
{
  "code": 200,
  "message": "任务暂停成功",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "previous_status": "active",
    "current_status": "paused"
  }
}
```

### 5.8 恢复任务

**接口定义**
```
POST /api/v1/tasks/{id}/resume
```

**功能描述**：恢复已暂停的任务调度。

**响应示例**：
```json
{
  "code": 200,
  "message": "任务恢复成功",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "previous_status": "paused",
    "current_status": "active"
  }
}
```

### 5.9 获取任务统计

**接口定义**
```
GET /api/v1/tasks/{id}/stats
```

**功能描述**：获取任务的执行统计信息和健康度指标。

**响应示例**：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "success_rate_24h": 95.5,
    "total_24h": 288,
    "success_24h": 275,
    "health_90d": {
      "health_score": 92.3,
      "total_count": 8640,
      "success_count": 8200,
      "failed_count": 340,
      "timeout_count": 100,
      "avg_duration_seconds": 45.2,
      "period_days": 90
    },
    "recent_executions": [
      {
        "date": "2024-01-01",
        "total": 288,
        "success": 275,
        "failed": 13,
        "success_rate": 95.5
      }
    ],
    "daily_stats_90d": [
      {
        "date": "2024-01-01",
        "successRate": 95.5,
        "total": 288
      }
    ]
  }
}
```

### 5.10 任务执行器管理

#### 5.10.1 获取任务的执行器列表

**接口定义**
```
GET /api/v1/tasks/{id}/executors
```

**功能描述**：获取分配给指定任务的执行器列表。

**响应示例**：
```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "id": "te-001",
      "task_id": "550e8400-e29b-41d4-a716-446655440000",
      "executor_id": "exec-001",
      "priority": 1,
      "weight": 2,
      "created_at": "2024-01-01T00:00:00Z",
      "executor": {
        "id": "exec-001",
        "name": "data-processor-1",
        "instance_id": "dp-001",
        "status": "online",
        "is_healthy": true,
        "base_url": "http://192.168.1.100:9090"
      }
    }
  ]
}
```

#### 5.10.2 为任务分配执行器

**接口定义**
```
POST /api/v1/tasks/{id}/executors
```

**功能描述**：为任务分配新的执行器。

**请求体**：
```json
{
  "executor_id": "exec-002",
  "priority": 2,
  "weight": 1
}
```

**字段说明**：
| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| executor_id | string | 是 | 执行器ID |
| priority | int | 否 | 优先级，数字越小优先级越高，默认为 0 |
| weight | int | 否 | 权重，用于加权负载均衡，默认为 1 |

**响应示例**：
```json
{
  "code": 201,
  "message": "执行器分配成功",
  "data": {
    "id": "te-002",
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "executor_id": "exec-002",
    "priority": 2,
    "weight": 1,
    "created_at": "2024-01-01T01:00:00Z"
  }
}
```

#### 5.10.3 更新执行器分配

**接口定义**
```
PUT /api/v1/tasks/{id}/executors/{executor_id}
```

**功能描述**：更新任务执行器的权重和优先级配置。

**请求体**：
```json
{
  "priority": 1,
  "weight": 3
}
```

**响应示例**：
```json
{
  "code": 200,
  "message": "执行器分配更新成功",
  "data": {
    "id": "te-002",
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "executor_id": "exec-002",
    "priority": 1,
    "weight": 3,
    "created_at": "2024-01-01T01:00:00Z"
  }
}
```

#### 5.10.4 取消执行器分配

**接口定义**
```
DELETE /api/v1/tasks/{id}/executors/{executor_id}
```

**功能描述**：取消任务与执行器的分配关系。

**响应示例**：
```json
{
  "code": 200,
  "message": "执行器分配取消成功"
}
```

## 6. 执行器管理 API

### 6.1 获取执行器列表

**接口定义**
```
GET /api/v1/executors
```

**功能描述**：获取所有注册的执行器列表。

**查询参数**：
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| status | string | 否 | 按状态筛选：online/offline/maintenance |
| include_tasks | boolean | 否 | 是否包含任务信息，默认为 false |

**响应示例**：
```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "id": "exec-001",
      "name": "data-processor-1",
      "instance_id": "dp-001",
      "base_url": "http://192.168.1.100:9090",
      "health_check_url": "http://192.168.1.100:9090/health",
      "status": "online",
      "is_healthy": true,
      "last_health_check": "2024-01-01T01:00:00Z",
      "health_check_failures": 0,
      "metadata": {
        "region": "us-west-1",
        "capacity": 10,
        "version": "1.0.0"
      },
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T01:00:00Z"
    }
  ]
}
```

### 6.2 注册执行器

**接口定义**
```
POST /api/v1/executors/register
```

**功能描述**：注册新的执行器到系统中。

**请求体**：
```json
{
  "name": "data-processor-2",
  "instance_id": "dp-002",
  "base_url": "http://192.168.1.101:9090",
  "health_check_url": "http://192.168.1.101:9090/health",
  "metadata": {
    "region": "us-west-1",
    "capacity": 15,
    "version": "1.0.1",
    "tags": ["data-processing", "etl"]
  }
}
```

**字段说明**：
| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| name | string | 是 | 执行器名称 |
| instance_id | string | 是 | 实例ID，全局唯一 |
| base_url | string | 是 | 执行器基础URL |
| health_check_url | string | 否 | 健康检查URL，不填则使用 base_url/health |
| metadata | object | 否 | 元数据信息 |

**响应示例**：
```json
{
  "code": 200,
  "message": "执行器注册成功",
  "data": {
    "id": "exec-002",
    "name": "data-processor-2",
    "instance_id": "dp-002",
    "base_url": "http://192.168.1.101:9090",
    "health_check_url": "http://192.168.1.101:9090/health",
    "status": "online",
    "is_healthy": true,
    "last_health_check": null,
    "health_check_failures": 0,
    "metadata": {
      "region": "us-west-1",
      "capacity": 15,
      "version": "1.0.1",
      "tags": ["data-processing", "etl"]
    },
    "created_at": "2024-01-01T02:00:00Z",
    "updated_at": "2024-01-01T02:00:00Z"
  }
}
```

### 6.3 获取执行器详情

**接口定义**
```
GET /api/v1/executors/{id}
```

**功能描述**：获取指定执行器的详细信息。

**响应示例**：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "exec-001",
    "name": "data-processor-1",
    "instance_id": "dp-001",
    "base_url": "http://192.168.1.100:9090",
    "health_check_url": "http://192.168.1.100:9090/health",
    "status": "online",
    "is_healthy": true,
    "last_health_check": "2024-01-01T01:00:00Z",
    "health_check_failures": 0,
    "metadata": {
      "region": "us-west-1",
      "capacity": 10,
      "version": "1.0.0"
    },
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T01:00:00Z",
    "task_executors": [
      {
        "id": "te-001",
        "task_id": "550e8400-e29b-41d4-a716-446655440000",
        "priority": 1,
        "weight": 2,
        "task": {
          "id": "550e8400-e29b-41d4-a716-446655440000",
          "name": "data_sync_task",
          "status": "active"
        }
      }
    ]
  }
}
```

### 6.4 更新执行器信息

**接口定义**
```
PUT /api/v1/executors/{id}
```

**功能描述**：更新执行器的基本信息。

**请求体**：
```json
{
  "name": "data-processor-1-updated",
  "base_url": "http://192.168.1.100:9091",
  "health_check_url": "http://192.168.1.100:9091/health",
  "metadata": {
    "region": "us-west-1",
    "capacity": 12,
    "version": "1.0.2"
  }
}
```

**响应示例**：
```json
{
  "code": 200,
  "message": "执行器信息更新成功",
  "data": {
    "id": "exec-001",
    "name": "data-processor-1-updated",
    "instance_id": "dp-001",
    "base_url": "http://192.168.1.100:9091",
    "health_check_url": "http://192.168.1.100:9091/health",
    "status": "online",
    "is_healthy": true,
    "metadata": {
      "region": "us-west-1",
      "capacity": 12,
      "version": "1.0.2"
    },
    "updated_at": "2024-01-01T03:00:00Z"
  }
}
```

### 6.5 更新执行器状态

**接口定义**
```
PUT /api/v1/executors/{id}/status
```

**功能描述**：手动更新执行器的运行状态。

**请求体**：
```json
{
  "status": "maintenance",
  "reason": "系统维护升级"
}
```

**字段说明**：
| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| status | string | 是 | 状态：online/offline/maintenance |
| reason | string | 否 | 状态变更原因 |

**响应示例**：
```json
{
  "code": 200,
  "message": "执行器状态更新成功",
  "data": {
    "executor_id": "exec-001",
    "previous_status": "online",
    "current_status": "maintenance",
    "reason": "系统维护升级",
    "updated_at": "2024-01-01T03:30:00Z"
  }
}
```

### 6.6 删除执行器

**接口定义**
```
DELETE /api/v1/executors/{id}
```

**功能描述**：删除执行器及其所有关联关系。

**注意事项**：
- 会同时删除该执行器的所有任务分配关系
- 正在执行的任务不会受到影响，但会失去执行器引用

**响应示例**：
```json
{
  "code": 200,
  "message": "执行器删除成功"
}
```

## 7. 执行历史 API

### 7.1 获取执行历史列表

**接口定义**
```
GET /api/v1/executions
```

**功能描述**：获取任务执行历史记录，支持多维度筛选和分页。

**查询参数**：
| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| task_id | string | 否 | 按任务ID筛选 | 550e8400-e29b-41d4-a716-446655440000 |
| executor_id | string | 否 | 按执行器ID筛选 | exec-001 |
| status | string | 否 | 按状态筛选 | success,failed |
| start_time | string | 否 | 开始时间（ISO 8601） | 2024-01-01T00:00:00Z |
| end_time | string | 否 | 结束时间（ISO 8601） | 2024-01-02T00:00:00Z |
| page | int | 否 | 页码，默认为 1 | 1 |
| page_size | int | 否 | 每页大小，默认为 20，最大 100 | 50 |

**响应示例**：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "data": [
      {
        "id": "exec-550e8400-e29b-41d4-a716-446655440001",
        "task_id": "550e8400-e29b-41d4-a716-446655440000",
        "executor_id": "exec-001",
        "scheduled_time": "2024-01-01T12:00:00Z",
        "start_time": "2024-01-01T12:00:01Z",
        "end_time": "2024-01-01T12:02:15Z",
        "status": "success",
        "result": {
          "processed_records": 1000,
          "sync_duration": 134,
          "errors": []
        },
        "logs": "Task completed successfully. Processed 1000 records in 134 seconds.",
        "retry_count": 0,
        "created_at": "2024-01-01T12:00:00Z",
        "task": {
          "id": "550e8400-e29b-41d4-a716-446655440000",
          "name": "data_sync_task",
          "status": "active"
        },
        "executor": {
          "id": "exec-001",
          "name": "data-processor-1",
          "status": "online"
        }
      }
    ],
    "total": 1524,
    "page": 1,
    "page_size": 20,
    "total_pages": 77
  }
}
```

### 7.2 获取执行统计

**接口定义**
```
GET /api/v1/executions/stats
```

**功能描述**：获取执行历史的统计信息。

**查询参数**：
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| task_id | string | 否 | 按任务ID筛选统计 |
| start_time | string | 否 | 统计开始时间 |
| end_time | string | 否 | 统计结束时间 |

**响应示例**：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "total": 1524,
    "success": 1450,
    "failed": 64,
    "running": 8,
    "pending": 2,
    "success_rate": 95.14,
    "avg_duration": 125.5,
    "total_duration": 181950
  }
}
```

### 7.3 获取执行详情

**接口定义**
```
GET /api/v1/executions/{id}
```

**功能描述**：获取单次执行的详细信息。

**响应示例**：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "exec-550e8400-e29b-41d4-a716-446655440001",
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "executor_id": "exec-001",
    "scheduled_time": "2024-01-01T12:00:00Z",
    "start_time": "2024-01-01T12:00:01Z",
    "end_time": "2024-01-01T12:02:15Z",
    "status": "success",
    "result": {
      "processed_records": 1000,
      "sync_duration": 134,
      "source_count": 1000,
      "target_count": 1000,
      "errors": [],
      "warnings": ["Slow query detected in batch 3"]
    },
    "logs": "2024-01-01T12:00:01Z [INFO] Starting data sync task\n2024-01-01T12:00:01Z [INFO] Connecting to source database\n2024-01-01T12:00:02Z [INFO] Connecting to target database\n2024-01-01T12:00:02Z [INFO] Processing batch 1/10\n...\n2024-01-01T12:02:15Z [INFO] Task completed successfully",
    "retry_count": 0,
    "created_at": "2024-01-01T12:00:00Z",
    "task": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "data_sync_task",
      "cron_expression": "0 */5 * * * *",
      "execution_mode": "sequential",
      "load_balance_strategy": "round_robin",
      "status": "active"
    },
    "executor": {
      "id": "exec-001",
      "name": "data-processor-1",
      "instance_id": "dp-001",
      "base_url": "http://192.168.1.100:9090",
      "status": "online",
      "is_healthy": true
    }
  }
}
```

### 7.4 执行回调

**接口定义**
```
POST /api/v1/executions/{id}/callback
```

**功能描述**：执行器向调度器报告任务执行结果。

**请求体**：
```json
{
  "status": "success",
  "result": {
    "processed_records": 1000,
    "sync_duration": 134,
    "source_count": 1000,
    "target_count": 1000,
    "errors": []
  },
  "logs": "Task completed successfully. Processed 1000 records in 134 seconds.",
  "end_time": "2024-01-01T12:02:15Z"
}
```

**字段说明**：
| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| status | string | 是 | 执行状态：success/failed/timeout |
| result | object | 否 | 执行结果数据 |
| logs | string | 否 | 执行日志 |
| end_time | string | 否 | 执行结束时间 |

**响应示例**：
```json
{
  "code": 200,
  "message": "执行回调处理成功"
}
```

### 7.5 停止执行

**接口定义**
```
POST /api/v1/executions/{id}/stop
```

**功能描述**：停止正在执行的任务。

**注意事项**：
- 只能停止状态为 `running` 的任务
- 系统会向执行器发送停止指令
- 执行状态将变更为 `cancelled`

**响应示例**：
```json
{
  "code": 200,
  "message": "停止指令已发送",
  "data": {
    "execution_id": "exec-550e8400-e29b-41d4-a716-446655440001",
    "previous_status": "running",
    "current_status": "cancelled",
    "stopped_at": "2024-01-01T12:01:30Z"
  }
}
```

## 8. 系统监控 API

### 8.1 健康检查

**接口定义**
```
GET /api/v1/health
```

**功能描述**：检查系统整体健康状态。

**响应示例**：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "status": "healthy",
    "timestamp": "2024-01-01T12:00:00Z",
    "components": {
      "database": {
        "status": "healthy",
        "response_time": 5
      },
      "scheduler": {
        "status": "healthy",
        "is_leader": true,
        "active_tasks": 25
      },
      "executors": {
        "status": "healthy",
        "total": 5,
        "online": 4,
        "offline": 1
      }
    }
  }
}
```

### 8.2 获取调度器状态

**接口定义**
```
GET /api/v1/scheduler/status
```

**功能描述**：获取调度器集群状态信息。

**响应示例**：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "instances": [
      {
        "id": "scheduler-001",
        "instance_id": "sched-001",
        "host": "192.168.1.10",
        "port": 8080,
        "is_leader": true,
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T12:00:00Z"
      },
      {
        "id": "scheduler-002",
        "instance_id": "sched-002",
        "host": "192.168.1.11",
        "port": 8080,
        "is_leader": false,
        "created_at": "2024-01-01T00:30:00Z",
        "updated_at": "2024-01-01T12:00:00Z"
      }
    ],
    "leader_info": {
      "instance_id": "sched-001",
      "host": "192.168.1.10",
      "since": "2024-01-01T00:00:00Z"
    },
    "cluster_status": {
      "total_instances": 2,
      "healthy_instances": 2,
      "has_leader": true
    },
    "system_metrics": {
      "total_tasks": 25,
      "active_tasks": 20,
      "paused_tasks": 5,
      "total_executors": 5,
      "online_executors": 4,
      "healthy_executors": 4,
      "running_executions": 8,
      "pending_executions": 12
    },
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

## 9. 错误码定义

### 9.1 HTTP 状态码

| 状态码 | 说明 | 使用场景 |
|--------|------|----------|
| 200 | 成功 | 请求处理成功 |
| 201 | 已创建 | 资源创建成功 |
| 400 | 请求错误 | 参数验证失败、格式错误 |
| 401 | 未授权 | 缺少认证信息 |
| 403 | 禁止访问 | 权限不足 |
| 404 | 资源不存在 | 请求的资源未找到 |
| 409 | 冲突 | 资源冲突（如名称重复） |
| 422 | 处理失败 | 业务逻辑验证失败 |
| 500 | 服务器错误 | 系统内部错误 |
| 503 | 服务不可用 | 系统维护或过载 |

### 9.2 业务错误码

| 错误码 | 错误信息 | 说明 |
|--------|----------|------|
| 10001 | 任务名称已存在 | 创建任务时名称重复 |
| 10002 | Cron表达式格式错误 | Cron表达式解析失败 |
| 10003 | 任务不存在 | 查询或操作不存在的任务 |
| 10004 | 任务状态不允许此操作 | 状态限制操作 |
| 20001 | 执行器不存在 | 查询或操作不存在的执行器 |
| 20002 | 执行器实例ID已存在 | 注册时实例ID重复 |
| 20003 | 执行器状态异常 | 执行器不健康或离线 |
| 30001 | 执行记录不存在 | 查询不存在的执行记录 |
| 30002 | 执行状态不允许此操作 | 状态限制操作 |
| 40001 | 系统维护中 | 系统正在维护 |
| 40002 | 数据库连接失败 | 数据库异常 |

### 9.3 错误响应格式

```json
{
  "code": 400,
  "message": "参数验证失败",
  "error": "cron_expression field is required",
  "details": {
    "field": "cron_expression",
    "value": "",
    "constraint": "required"
  },
  "timestamp": "2024-01-01T12:00:00Z",
  "path": "/api/v1/tasks"
}
```

## 10. 认证机制

### 10.1 当前状态
目前系统未启用身份认证，所有 API 接口都可以直接访问。

### 10.2 计划支持的认证方式

#### 10.2.1 API Key 认证
```http
GET /api/v1/tasks
Authorization: Bearer your-api-key-here
```

#### 10.2.2 JWT Token 认证
```http
GET /api/v1/tasks
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

#### 10.2.3 Basic 认证
```http
GET /api/v1/tasks
Authorization: Basic dXNlcm5hbWU6cGFzc3dvcmQ=
```

## 11. 开发指南

### 11.1 快速开始

#### 11.1.1 环境准备
```bash
# 1. 安装 Go 1.23+
go version

# 2. 克隆代码
git clone <repository-url>
cd scheduler

# 3. 安装依赖
go mod download

# 4. 启动 MySQL 数据库
docker run -d --name mysql \
  -e MYSQL_ROOT_PASSWORD=123456 \
  -e MYSQL_DATABASE=jobs \
  -p 3306:3306 \
  mysql:8.0

# 5. 执行数据库迁移
mysql -h 127.0.0.1 -P 3306 -u root -p123456 jobs < scripts/migrate.sql

# 6. 启动调度器
go run cmd/scheduler/main.go
```

#### 11.1.2 验证安装
```bash
# 健康检查
curl http://localhost:8080/api/v1/health

# 创建测试任务
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test_task",
    "cron_expression": "*/10 * * * * *",
    "execution_mode": "parallel"
  }'
```

### 11.2 客户端开发示例

#### 11.2.1 Go 客户端
```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type Client struct {
    baseURL string
    client  *http.Client
}

func NewClient(baseURL string) *Client {
    return &Client{
        baseURL: baseURL,
        client:  &http.Client{},
    }
}

func (c *Client) CreateTask(task CreateTaskRequest) (*TaskResponse, error) {
    data, _ := json.Marshal(task)
    resp, err := c.client.Post(
        c.baseURL+"/api/v1/tasks",
        "application/json",
        bytes.NewBuffer(data),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Data TaskResponse `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&result)
    return &result.Data, nil
}
```

#### 11.2.2 Python 客户端
```python
import requests
import json

class SchedulerClient:
    def __init__(self, base_url):
        self.base_url = base_url
        self.session = requests.Session()
    
    def create_task(self, task_data):
        """创建任务"""
        response = self.session.post(
            f"{self.base_url}/api/v1/tasks",
            json=task_data,
            headers={"Content-Type": "application/json"}
        )
        response.raise_for_status()
        return response.json()["data"]
    
    def list_tasks(self, status=None):
        """获取任务列表"""
        params = {}
        if status:
            params["status"] = status
        
        response = self.session.get(
            f"{self.base_url}/api/v1/tasks",
            params=params
        )
        response.raise_for_status()
        return response.json()["data"]

# 使用示例
client = SchedulerClient("http://localhost:8080")

# 创建任务
task = client.create_task({
    "name": "python_test_task",
    "cron_expression": "0 */5 * * * *",
    "execution_mode": "sequential"
})
print(f"Created task: {task['id']}")

# 获取任务列表
tasks = client.list_tasks(status="active")
print(f"Active tasks: {len(tasks)}")
```

#### 11.2.3 JavaScript 客户端
```javascript
class SchedulerClient {
    constructor(baseURL) {
        this.baseURL = baseURL;
    }

    async createTask(taskData) {
        const response = await fetch(`${this.baseURL}/api/v1/tasks`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(taskData),
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const result = await response.json();
        return result.data;
    }

    async listTasks(filters = {}) {
        const params = new URLSearchParams(filters);
        const response = await fetch(`${this.baseURL}/api/v1/tasks?${params}`);
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const result = await response.json();
        return result.data;
    }

    async triggerTask(taskId, parameters = {}) {
        const response = await fetch(`${this.baseURL}/api/v1/tasks/${taskId}/trigger`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ parameters }),
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const result = await response.json();
        return result.data;
    }
}

// 使用示例
const client = new SchedulerClient('http://localhost:8080');

// 创建任务
const task = await client.createTask({
    name: 'js_test_task',
    cron_expression: '0 */5 * * * *',
    execution_mode: 'parallel',
    parameters: {
        source: 'api',
        target: 'database'
    }
});

console.log('Created task:', task.id);

// 手动触发任务
const execution = await client.triggerTask(task.id, {
    force: true
});

console.log('Triggered execution:', execution.id);
```

### 11.3 最佳实践

#### 11.3.1 任务设计原则
1. **幂等性**：任务应该支持重复执行而不产生副作用
2. **快速失败**：尽早发现和报告错误
3. **详细日志**：记录足够的执行信息用于故障排查
4. **资源清理**：确保任务执行后正确释放资源

#### 11.3.2 错误处理
```go
// 执行器端错误处理示例
func (e *Executor) ExecuteTask(ctx context.Context, task *Task) error {
    // 1. 参数验证
    if err := validateParameters(task.Parameters); err != nil {
        return fmt.Errorf("parameter validation failed: %w", err)
    }

    // 2. 超时控制
    ctx, cancel := context.WithTimeout(ctx, time.Duration(task.TimeoutSeconds)*time.Second)
    defer cancel()

    // 3. 执行任务
    result, err := e.doExecute(ctx, task)
    if err != nil {
        return fmt.Errorf("task execution failed: %w", err)
    }

    // 4. 回调结果
    return e.reportResult(task.ExecutionID, result)
}
```

#### 11.3.3 性能优化建议
1. **数据库查询优化**：使用适当的索引和查询条件
2. **连接池管理**：合理配置数据库和HTTP连接池
3. **并发控制**：根据系统资源调整最大并发数
4. **监控告警**：设置关键指标的监控和告警

### 11.4 常见问题

#### 11.4.1 任务不执行
**可能原因**：
- 任务状态为 paused 或 deleted
- 调度器不是 leader 节点
- Cron 表达式格式错误
- 没有可用的执行器

**排查步骤**：
1. 检查任务状态：`GET /api/v1/tasks/{id}`
2. 检查调度器状态：`GET /api/v1/scheduler/status`
3. 检查执行器状态：`GET /api/v1/executors`
4. 检查执行历史：`GET /api/v1/executions?task_id={id}`

#### 11.4.2 执行器健康检查失败
**可能原因**：
- 执行器服务不可达
- 健康检查端点异常
- 网络连接问题

**解决方法**：
1. 验证执行器服务状态
2. 检查健康检查URL配置
3. 测试网络连通性
4. 查看执行器日志

#### 11.4.3 任务执行超时
**可能原因**：
- 超时时间设置过短
- 任务逻辑执行时间过长
- 系统资源不足

**解决方法**：
1. 调整任务超时配置
2. 优化任务执行逻辑
3. 检查系统资源使用情况
4. 考虑任务拆分

---

**文档版本**：v1.0  
**最后更新**：2024年8月  
**维护者**：开发团队