## 任务调度系统重构方案

### 核心设计变更

#### 1. 任务类型（Task Types）
- 系统中预定义或动态注册的任务类型
- 每个任务类型有唯一的名称（如 `send_email`, `http_request`）
- 定义参数 schema 和描述信息

#### 2. 执行器（Executors）
- 执行器注册时声明支持的任务类型列表
- 一个执行器可以支持多个任务类型
- 同一个任务类型可以被多个执行器支持

#### 3. 任务（Tasks）
- 创建任务时必须指定任务类型
- 任务参数必须符合任务类型的 schema
- 调度时自动找到支持该任务类型的执行器

### API 变更

#### 执行器注册
```json
POST /api/v1/executors/register
{
  "name": "email-executor-01",
  "instance_id": "executor-001",
  "base_url": "http://localhost:9090",
  "health_check_url": "http://localhost:9090/health",
  "supported_tasks": [
    {
      "task_type": "send_email",
      "max_concurrent": 10,
      "timeout_seconds": 60
    },
    {
      "task_type": "send_sms",
      "max_concurrent": 20,
      "timeout_seconds": 30
    }
  ]
}
```

#### 创建任务
```json
POST /api/v1/tasks
{
  "name": "每日报告邮件",
  "task_type": "send_email",
  "cron_expression": "0 9 * * *",
  "parameters": {
    "to": "admin@example.com",
    "subject": "Daily Report",
    "template": "daily_report"
  },
  "execution_mode": "sequential",
  "load_balance_strategy": "round_robin"
}
```

#### 获取可用任务类型
```json
GET /api/v1/task-types
Response:
[
  {
    "id": "uuid",
    "name": "send_email",
    "display_name": "发送邮件",
    "description": "发送电子邮件",
    "parameters_schema": {...},
    "available_executors": 3
  }
]
```

### 执行流程

1. **调度触发**：调度器根据 cron 表达式触发任务
2. **查找执行器**：根据任务的 `task_type` 查找所有支持该类型的在线执行器
3. **负载均衡**：根据策略选择一个执行器
4. **发送请求**：向执行器发送执行请求，包含任务类型和参数
5. **执行反馈**：执行器完成后回调调度器

### 数据库变更

1. **task_types 表**：存储任务类型定义
2. **executor_task_types 表**：执行器支持的任务类型映射
3. **tasks 表**：添加 `task_type_id` 字段
4. **删除 task_executors 表**：不再需要任务和执行器的直接关联

### 优势

1. **灵活性**：执行器可以动态注册支持的任务类型
2. **扩展性**：新增任务类型不需要修改代码
3. **解耦**：任务和执行器通过任务类型解耦
4. **负载均衡**：同一任务类型可由多个执行器处理