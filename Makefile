.PHONY: help build run clean test docker-build docker-up docker-down migrate

# 默认目标
help:
	@echo "Available commands:"
	@echo "  make build        - Build the scheduler binary"
	@echo "  make run          - Run the scheduler locally"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make test         - Run tests"
	@echo "  make migrate      - Run database migrations"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-up    - Start services with Docker Compose"
	@echo "  make docker-down  - Stop Docker Compose services"
	@echo "  make example      - Run example executor"

# 构建二进制文件
build:
	@echo "Building scheduler..."
	@go build -o bin/scheduler cmd/scheduler/main.go
	@echo "Build complete: bin/scheduler"
	go generate ./...

# 运行调度器
run: build
	@echo "Starting scheduler..."
	@./bin/scheduler -config configs/config.yaml

# 清理构建产物
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean
	@echo "Clean complete"

# 运行测试
test:
	@echo "Running tests..."
	@go test -v ./...

# 数据库迁移
migrate:
	@echo "Running database migrations..."
	@mysql -h 127.0.0.1 -P 3306 -u root -p123456 < scripts/migrate.sql
	@echo "Migration complete"

# Docker相关
docker-build:
	@echo "Building Docker image..."
	@docker build -f docker/Dockerfile -t job-scheduler:latest .
	@echo "Docker build complete"

docker-up:
	@echo "Starting services with Docker Compose..."
	@cd docker && docker-compose up -d
	@echo "Services started"

docker-down:
	@echo "Stopping Docker Compose services..."
	@cd docker && docker-compose down
	@echo "Services stopped"

# 运行示例执行器
example:
	@echo "Starting example executor..."
	@go run examples/executor/main.go

# 开发模式 - 使用 air 热重载
dev:
	@echo "Starting in development mode with hot reload..."
	@air -c .air.toml

# 格式化代码
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete"

# 代码检查
lint:
	@echo "Running linter..."
	@golangci-lint run
	@echo "Lint complete"

# 依赖管理
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies updated"