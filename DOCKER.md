# Go Job Scheduler - Docker 部署指南

## 概述

本文档介绍如何使用 Docker 部署 Go Job Scheduler 后端服务。配置文件通过挂载方式提供，不打包在镜像中，便于灵活配置和更新。

## 快速开始

### 1. 构建镜像

在项目根目录下执行：

```bash
docker build -t job-scheduler:latest .
```

### 2. 准备配置文件

复制示例配置并修改：

```bash
cp configs/config.example.yaml configs/config.docker.yaml
```

编辑 `configs/config.docker.yaml`，修改以下关键配置：

```yaml
database:
  host: your-mysql-host  # MySQL服务器地址
  port: 3306
  database: jobs
  user: root
  password: "your_password"

server:
  ip: 0.0.0.0  # 监听所有接口
  port: 8080

redis:
  host: your-redis-host  # Redis服务器地址
  port: 6379
```

### 3. 运行容器

```bash
docker run -d \
  --name job-scheduler \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/logs:/app/logs \
  job-scheduler:latest
```

## 详细配置

### 端口映射

默认服务端口是 8080（可通过配置文件修改）：

```bash
-p 8080:8080  # 主机端口:容器端口
```

如果配置文件中修改了端口，记得同步修改端口映射：

```bash
# 如果配置文件中 server.port: 7777
-p 7777:7777
```

### 挂载配置

**配置文件挂载**（只读）：

```bash
-v /path/to/configs:/app/configs:ro
```

推荐使用只读模式（`:ro`）防止容器意外修改配置文件。

**日志目录挂载**：

```bash
-v /path/to/logs:/app/logs
```

**使用自定义配置文件名**：

```bash
docker run -d \
  --name job-scheduler \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs:ro \
  job-scheduler:latest \
  -config /app/configs/my-custom-config.yaml
```

### 环境变量

虽然主要配置通过配置文件提供，但也可以通过环境变量覆盖部分配置：

```bash
docker run -d \
  --name job-scheduler \
  -p 8080:8080 \
  -e TZ=Asia/Shanghai \
  -v $(pwd)/configs:/app/configs:ro \
  job-scheduler:latest
```

### 网络配置

**使用自定义网络**（推荐）：

```bash
# 创建网络
docker network create scheduler-network

# 运行数据库（示例）
docker run -d \
  --name mysql \
  --network scheduler-network \
  -e MYSQL_ROOT_PASSWORD=password \
  -e MYSQL_DATABASE=jobs \
  mysql:8.0

# 运行Redis（示例）
docker run -d \
  --name redis \
  --network scheduler-network \
  redis:alpine

# 运行调度器
docker run -d \
  --name job-scheduler \
  --network scheduler-network \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/logs:/app/logs \
  job-scheduler:latest
```

配置文件中对应修改：

```yaml
database:
  host: mysql  # 使用容器名作为主机名

redis:
  host: redis  # 使用容器名作为主机名
```

## 完整部署示例

### 单机部署

```bash
#!/bin/bash

# 1. 创建网络
docker network create scheduler-network

# 2. 启动MySQL
docker run -d \
  --name scheduler-mysql \
  --network scheduler-network \
  -e MYSQL_ROOT_PASSWORD=password123 \
  -e MYSQL_DATABASE=jobs \
  -v mysql-data:/var/lib/mysql \
  -p 3306:3306 \
  mysql:8.0

# 3. 启动Redis
docker run -d \
  --name scheduler-redis \
  --network scheduler-network \
  -v redis-data:/data \
  -p 6379:6379 \
  redis:alpine

# 4. 等待数据库启动
sleep 10

# 5. 初始化数据库
docker exec -i scheduler-mysql mysql -uroot -ppassword123 jobs < scripts/migrate.sql

# 6. 启动调度器
docker run -d \
  --name job-scheduler \
  --network scheduler-network \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/logs:/app/logs \
  --restart unless-stopped \
  job-scheduler:latest

echo "部署完成！"
echo "API地址: http://localhost:8080"
echo "健康检查: http://localhost:8080/api/v1/health"
```

### 分布式部署（多实例）

```bash
# 实例1
docker run -d \
  --name job-scheduler-1 \
  --network scheduler-network \
  -p 8081:8080 \
  -v $(pwd)/configs/config-instance-1.yaml:/app/configs/config.yaml:ro \
  -v $(pwd)/logs/instance-1:/app/logs \
  --restart unless-stopped \
  job-scheduler:latest

# 实例2
docker run -d \
  --name job-scheduler-2 \
  --network scheduler-network \
  -p 8082:8080 \
  -v $(pwd)/configs/config-instance-2.yaml:/app/configs/config.yaml:ro \
  -v $(pwd)/logs/instance-2:/app/logs \
  --restart unless-stopped \
  job-scheduler:latest
```

**注意**：每个实例的配置文件中 `scheduler.instance_id` 必须不同！

## 健康检查

容器内置健康检查：

```bash
# 查看容器健康状态
docker inspect --format='{{.State.Health.Status}}' job-scheduler

# 手动执行健康检查
curl http://localhost:8080/api/v1/health
```

预期响应：

```json
{
  "status": "ok",
  "time": "2024-01-01T00:00:00Z"
}
```

## 日志管理

### 查看日志

```bash
# 查看容器标准输出日志
docker logs job-scheduler

# 实时跟踪日志
docker logs -f job-scheduler

# 查看最近100行
docker logs --tail 100 job-scheduler

# 查看挂载的日志文件
tail -f logs/scheduler.log
```

### 日志配置

在配置文件中可以控制日志输出：

```yaml
log:
  level: info          # debug, info, warn, error
  format: json         # json or console
  output: stdout       # stdout 或文件路径
  file: logs/scheduler.log
```

## 资源限制

生产环境建议设置资源限制：

```bash
docker run -d \
  --name job-scheduler \
  --network scheduler-network \
  -p 8080:8080 \
  --memory="512m" \
  --cpus="1.0" \
  --memory-reservation="256m" \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/logs:/app/logs \
  --restart unless-stopped \
  job-scheduler:latest
```

## 更新部署

### 无停机更新

```bash
# 1. 拉取/构建新镜像
docker build -t job-scheduler:v2.0.0 .

# 2. 启动新容器（使用不同名称和端口）
docker run -d \
  --name job-scheduler-new \
  --network scheduler-network \
  -p 8081:8080 \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/logs:/app/logs \
  job-scheduler:v2.0.0

# 3. 等待新容器健康检查通过
sleep 10
curl http://localhost:8081/api/v1/health

# 4. 更新负载均衡器指向新容器

# 5. 停止旧容器
docker stop job-scheduler
docker rm job-scheduler

# 6. 重命名新容器
docker rename job-scheduler-new job-scheduler
```

### 标准更新（有短暂停机）

```bash
# 1. 停止并删除旧容器
docker stop job-scheduler
docker rm job-scheduler

# 2. 拉取/构建新镜像
docker build -t job-scheduler:latest .

# 3. 启动新容器
docker run -d \
  --name job-scheduler \
  --network scheduler-network \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/logs:/app/logs \
  --restart unless-stopped \
  job-scheduler:latest
```

## 备份和恢复

### 备份配置

```bash
# 备份配置文件
tar -czf scheduler-configs-$(date +%Y%m%d).tar.gz configs/

# 备份日志
tar -czf scheduler-logs-$(date +%Y%m%d).tar.gz logs/
```

### 数据库备份

```bash
# 备份MySQL数据
docker exec scheduler-mysql mysqldump -uroot -ppassword123 jobs > backup-$(date +%Y%m%d).sql

# 恢复
docker exec -i scheduler-mysql mysql -uroot -ppassword123 jobs < backup-20240101.sql
```

## 故障排查

### 容器无法启动

```bash
# 查看容器日志
docker logs job-scheduler

# 查看容器详细信息
docker inspect job-scheduler

# 检查配置文件是否正确挂载
docker exec job-scheduler ls -la /app/configs

# 检查配置文件内容
docker exec job-scheduler cat /app/configs/config.yaml
```

### 数据库连接失败

```bash
# 1. 检查数据库是否可达
docker exec job-scheduler ping mysql

# 2. 检查数据库端口
docker exec job-scheduler nc -zv mysql 3306

# 3. 检查网络
docker network inspect scheduler-network
```

### 健康检查失败

```bash
# 检查端口是否监听
docker exec job-scheduler netstat -tlnp

# 手动访问健康检查端点
docker exec job-scheduler curl http://localhost:8080/api/v1/health

# 查看详细日志
docker logs job-scheduler | grep -i error
```

## 监控和告警

### 使用Prometheus

添加 Prometheus 指标暴露（如果应用支持）：

```bash
docker run -d \
  --name job-scheduler \
  --network scheduler-network \
  -p 8080:8080 \
  -p 9090:9090 \
  -v $(pwd)/configs:/app/configs:ro \
  job-scheduler:latest
```

### 日志收集

集成 ELK 或其他日志系统：

```bash
docker run -d \
  --name job-scheduler \
  --network scheduler-network \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs:ro \
  --log-driver json-file \
  --log-opt max-size=10m \
  --log-opt max-file=3 \
  job-scheduler:latest
```

## 生产环境最佳实践

1. **使用具体的镜像版本标签**，不要使用 `latest`
2. **设置资源限制**，防止单个容器占用过多资源
3. **配置重启策略** `--restart unless-stopped`
4. **使用只读挂载**配置文件 `:ro`
5. **定期备份**配置和数据
6. **监控容器健康状态**
7. **使用容器编排系统**（Kubernetes、Docker Swarm）进行生产部署
8. **启用日志轮转**，防止日志文件过大
9. **定期更新基础镜像**，修复安全漏洞
10. **使用非root用户**运行容器（已在Dockerfile中配置）

## 安全建议

1. **敏感信息管理**：使用 Docker secrets 或环境变量管理敏感配置
2. **网络隔离**：使用自定义网络，限制容器间通信
3. **最小权限原则**：容器已配置为非root用户运行
4. **镜像扫描**：定期扫描镜像漏洞
5. **只读文件系统**：如需要可添加 `--read-only` 参数

## 镜像信息

- **基础镜像**: alpine:latest
- **Go版本**: 1.23
- **用户**: scheduler (UID: 1000)
- **暴露端口**: 8080
- **工作目录**: /app
- **配置路径**: /app/configs
- **日志路径**: /app/logs

## 常用命令

```bash
# 启动
docker start job-scheduler

# 停止
docker stop job-scheduler

# 重启
docker restart job-scheduler

# 删除
docker rm job-scheduler

# 查看日志
docker logs -f job-scheduler

# 进入容器
docker exec -it job-scheduler sh

# 查看容器资源使用
docker stats job-scheduler

# 导出镜像
docker save job-scheduler:latest | gzip > job-scheduler.tar.gz

# 导入镜像
docker load < job-scheduler.tar.gz
```

## 相关链接

- [项目文档](../CLAUDE.md)
- [Makefile使用说明](../Makefile)
- [前端UI Docker部署](../scheduler-ui/DOCKER.md)
