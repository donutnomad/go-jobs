# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个基于Go语言开发的分布式任务调度系统，支持多实例部署、灵活的负载均衡策略和完善的任务管理功能。系统包含Go后端服务和Next.js前端UI界面。

## 常用命令

### Go后端开发
```bash
# 构建项目
make build

# 本地运行调度器
make run

# 运行测试
make test

# 代码格式化
make fmt

# 代码检查
make lint

# 运行示例执行器
make example

# 清理构建产物
make clean

# 更新依赖
make deps
```

### 前端开发 (scheduler-ui目录)
```bash
# 安装依赖
cd scheduler-ui && pnpm install

# 开发模式
cd scheduler-ui && pnpm dev

# 构建生产版本
cd scheduler-ui && pnpm build

# 启动生产服务
cd scheduler-ui && pnpm start

# 代码检查
cd scheduler-ui && pnpm lint
```

### Docker部署
```bash
# 构建Docker镜像
make docker-build

# 启动所有服务
make docker-up

# 停止所有服务
make docker-down
```

### 数据库操作
```bash
# 运行数据库迁移
make migrate
```

## 系统架构

### 核心组件
- **Scheduler**: 任务调度器核心，基于robfig/cron实现定时调度，使用MySQL GET_LOCK实现分布式领导者选举
- **API Server**: 基于Gin框架的RESTful API服务，提供任务管理、执行器管理等接口
- **TaskRunner**: 任务执行器，负责实际的任务分发和执行管理
- **LoadBalance**: 负载均衡模块，支持5种策略：轮询、加权轮询、随机、粘性、最少负载
- **HealthChecker**: 执行器健康检查服务，定期检查执行器状态
- **UI**: Next.js前端界面，提供可视化管理功能

### 项目结构
```
cmd/scheduler/          # 主程序入口
internal/
├── api/               # REST API接口层
├── loadbalance/       # 负载均衡策略实现
├── models/            # 数据模型定义
├── orm/               # 数据库存储层
└── scheduler/         # 调度器核心逻辑
pkg/
├── config/            # 配置管理
└── logger/            # 日志工具
scheduler-ui/          # Next.js前端界面
configs/               # 配置文件
examples/              # 示例执行器代码
```

### 关键实现细节
- **分布式锁**: 使用MySQL的GET_LOCK()函数实现调度器实例间的主从选举
- **任务执行模式**: 支持Sequential(串行)、Parallel(并行)、Skip(跳过)三种模式
- **回调机制**: 执行器通过HTTP回调通知调度器任务执行结果
- **健康检查**: 定期HTTP健康检查，支持执行器状态管理和维护模式
- **数据库**: 使用GORM作为ORM，支持MySQL存储

### 配置文件
配置文件位于`configs/config.yaml`，包含：
- 调度器实例配置(实例ID、锁配置、工作协程数)
- 数据库连接配置(MySQL连接参数)
- 服务器配置(IP、端口、超时设置)
- 健康检查配置(检查间隔、超时、失败阈值)
- 日志配置(级别、格式、输出)

### API接口
系统提供RESTful API，基础路径为`/api/v1`，主要接口包括：
- 任务管理：创建、更新、删除、查询任务
- 执行器管理：注册、状态更新、健康检查
- 执行历史：查询执行记录、统计分析
- 系统监控：调度器状态、系统健康度

### 开发注意事项
- 使用`swagGen`工具进行API文档生成
- 遵循Go标准项目结构和命名规范
- 数据库模型变更需要相应更新迁移脚本
- 新增负载均衡策略需在Manager中注册
- 执行器需实现标准的执行和健康检查接口
- 前端组件使用TypeScript和Tailwind CSS开发

### 测试和调试
- 项目目前没有单元测试文件，建议使用`make test`命令运行测试
- 提供了`test_task_stats.sh`脚本用于测试任务统计API
- 可以通过示例执行器(`examples/executor/`)测试完整流程
- 日志输出到stdout和文件，便于调试和监控