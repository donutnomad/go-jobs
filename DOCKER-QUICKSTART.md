# Docker 快速部署指南

## 文件说明

- **[Dockerfile](Dockerfile)** - Docker镜像构建配置（多阶段构建）
- **[.dockerignore](.dockerignore)** - Docker构建时忽略的文件
- **[deploy-docker.sh](deploy-docker.sh)** - 一键部署脚本
- **[DOCKER.md](DOCKER.md)** - 详细的Docker部署文档

## 快速开始

### 1. 准备配置文件

```bash
# 复制示例配置
cp configs/config.example.yaml configs/config.docker.yaml

# 编辑配置文件，修改数据库和Redis连接信息
vim configs/config.docker.yaml
```

关键配置项：

```yaml
database:
  host: mysql  # 数据库地址
  password: "your_password"  # 数据库密码

redis:
  host: redis  # Redis地址

server:
  ip: 0.0.0.0  # 监听所有接口
```

### 2. 一键部署

```bash
./deploy-docker.sh
```

脚本会自动：
- ✅ 构建Docker镜像
- ✅ 创建Docker网络
- ✅ 启动容器（端口8080）
- ✅ 挂载配置文件和日志目录
- ✅ 执行健康检查

### 3. 手动部署

如果想手动控制部署过程：

```bash
# 构建镜像
docker build -t job-scheduler:latest .

# 运行容器
docker run -d \
  --name job-scheduler \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/logs:/app/logs \
  --restart unless-stopped \
  job-scheduler:latest
```

## 配置文件挂载

配置文件通过 **挂载方式** 提供，不打包在镜像中。

### 优势

- ✅ 无需重新构建镜像即可修改配置
- ✅ 支持不同环境使用不同配置
- ✅ 配置更新后重启容器即可生效
- ✅ 方便配置管理和版本控制

### 挂载说明

```bash
-v $(pwd)/configs:/app/configs:ro    # 配置目录（只读）
-v $(pwd)/logs:/app/logs              # 日志目录（可写）
```

### 使用自定义配置文件

```bash
docker run -d \
  --name job-scheduler \
  -p 8080:8080 \
  -v /path/to/my-config.yaml:/app/configs/config.yaml:ro \
  job-scheduler:latest
```

或指定配置文件路径：

```bash
docker run -d \
  --name job-scheduler \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs:ro \
  job-scheduler:latest \
  -config /app/configs/production.yaml
```

## 验证部署

### 检查容器状态

```bash
docker ps --filter "name=job-scheduler"
```

### 查看日志

```bash
# 实时日志
docker logs -f job-scheduler

# 最近日志
docker logs --tail 100 job-scheduler
```

### 健康检查

```bash
curl http://localhost:8080/api/v1/health
```

预期响应：

```json
{
  "status": "ok",
  "time": "2024-01-01T00:00:00Z"
}
```

### 查看调度器状态

```bash
curl http://localhost:8080/api/v1/scheduler/status
```

## 完整环境部署

如果需要同时部署MySQL和Redis：

```bash
# 1. 创建网络
docker network create scheduler-network

# 2. 启动MySQL
docker run -d \
  --name scheduler-mysql \
  --network scheduler-network \
  -e MYSQL_ROOT_PASSWORD=password123 \
  -e MYSQL_DATABASE=jobs \
  -p 3306:3306 \
  mysql:8.0

# 3. 启动Redis
docker run -d \
  --name scheduler-redis \
  --network scheduler-network \
  -p 6379:6379 \
  redis:alpine

# 4. 初始化数据库（等待MySQL启动）
sleep 10
docker exec -i scheduler-mysql mysql -uroot -ppassword123 jobs < scripts/migrate.sql

# 5. 启动调度器
docker run -d \
  --name job-scheduler \
  --network scheduler-network \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/logs:/app/logs \
  --restart unless-stopped \
  job-scheduler:latest
```

配置文件对应修改：

```yaml
database:
  host: scheduler-mysql  # 使用容器名
  port: 3306
  database: jobs
  user: root
  password: "password123"

redis:
  host: scheduler-redis  # 使用容器名
  port: 6379
```

## 常用操作

### 启动/停止/重启

```bash
docker start job-scheduler     # 启动
docker stop job-scheduler      # 停止
docker restart job-scheduler   # 重启
```

### 更新配置

1. 修改配置文件 `configs/config.docker.yaml`
2. 重启容器：`docker restart job-scheduler`

### 查看资源使用

```bash
docker stats job-scheduler
```

### 进入容器

```bash
docker exec -it job-scheduler sh
```

### 清理容器

```bash
docker stop job-scheduler
docker rm job-scheduler
```

## 镜像特性

- **大小**: ~30-40MB（基于Alpine Linux）
- **Go版本**: 1.23
- **架构**: linux/amd64
- **用户**: 非root用户（scheduler:1000）
- **健康检查**: 内置
- **时区**: Asia/Shanghai

## 故障排查

### 容器无法启动

```bash
# 查看错误日志
docker logs job-scheduler

# 检查配置文件是否正确挂载
docker exec job-scheduler ls -la /app/configs
docker exec job-scheduler cat /app/configs/config.yaml
```

### 数据库连接失败

```bash
# 检查数据库连通性
docker exec job-scheduler ping mysql

# 检查网络
docker network inspect scheduler-network
```

### 端口被占用

```bash
# 查找占用端口的进程
lsof -i :8080

# 使用其他端口
docker run -d \
  --name job-scheduler \
  -p 9090:8080 \
  -v $(pwd)/configs:/app/configs:ro \
  job-scheduler:latest
```

## 生产环境建议

1. ✅ 使用具体版本标签，不要用 `latest`
2. ✅ 设置资源限制 `--memory` `--cpus`
3. ✅ 配置重启策略 `--restart unless-stopped`
4. ✅ 使用只读挂载配置 `:ro`
5. ✅ 定期备份配置和日志
6. ✅ 监控容器健康状态
7. ✅ 使用容器编排工具（K8s）

## 更多信息

详细的部署指南、网络配置、监控、备份等请参考：

📖 [完整 Docker 部署文档](DOCKER.md)

## 支持

如遇问题，请查看：
- [项目文档](CLAUDE.md)
- [Docker详细文档](DOCKER.md)
- 容器日志：`docker logs job-scheduler`
