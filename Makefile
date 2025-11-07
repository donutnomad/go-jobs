.PHONY: help build run runLocal clean test example fmt lint deps docker-build docker-run docker-stop docker-clean docker-logs

# 默认目标
help:
	@echo "Available commands:"
	@echo "  make build         - Build the scheduler binary"
	@echo "  make run           - Run the scheduler locally"
	@echo "  make runLocal      - Run the scheduler with local config"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make test          - Run tests"
	@echo "  make example       - Run example executor"
	@echo "  make fmt           - Format code"
	@echo "  make lint          - Run code linter"
	@echo "  make deps          - Update dependencies"
	@echo ""
	@echo "Docker commands:"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-run    - Run Docker container"
	@echo "  make docker-stop   - Stop Docker container"
	@echo "  make docker-logs   - View Docker container logs"
	@echo "  make docker-clean  - Remove Docker container and image"

# 构建二进制文件
build:
	@echo "Building scheduler..."
	@go generate ./...
	@go build -o bin/scheduler cmd/scheduler/main.go cmd/scheduler/wire_gen.go cmd/scheduler/providers.go
	@echo "Build complete: bin/scheduler"

# 运行调度器
run: build
	@echo "Starting scheduler..."
	@./bin/scheduler -config configs/config.yaml

# 运行调度器
runLocal: build
	@echo "Starting scheduler..."
	@./bin/scheduler -config configs/config.local.yaml

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

# 运行示例执行器
example:
	@echo "Starting example executor..."
	@go run examples/executor/main.go

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

# Docker - 构建镜像
# 使用方式: make docker-build [IMAGE_TAG=harbor.abtdev.com/taas/gojobs:1] [PLATFORM=linux/amd64]
IMAGE_TAG ?= job-scheduler:latest
PLATFORM ?= linux/amd64
docker-build:
	@echo "Building Docker image..."
	@echo "Platform: $(PLATFORM)"
	@echo "Image tag: $(IMAGE_TAG)"
	@docker build \
		--platform $(PLATFORM) \
		-f ./Dockerfile \
		-t $(IMAGE_TAG) \
		.
	@echo "Docker image built successfully"

# Docker - 运行容器
docker-run:
	@echo "Starting Docker container..."
	@docker run -d \
		--name job-scheduler \
		-p 8080:8080 \
		-v $(PWD)/configs:/app/configs:ro \
		-v $(PWD)/logs:/app/logs \
		--restart unless-stopped \
		job-scheduler:latest
	@echo "Container started: job-scheduler"
	@echo "API: http://localhost:8080"

# Docker - 停止容器
docker-stop:
	@echo "Stopping Docker container..."
	@docker stop job-scheduler || true
	@echo "Container stopped"

# Docker - 查看日志
docker-logs:
	@docker logs -f job-scheduler

# Docker - 清理
docker-clean:
	@echo "Cleaning Docker resources..."
	@docker stop job-scheduler || true
	@docker rm job-scheduler || true
	@docker rmi job-scheduler:latest || true
	@echo "Docker resources cleaned"

DATE=$(shell date +%Y%m%d%H%M)
DOCKER_IMAGE=harbor.abtdev.com/taas/gojobs:$(DATE)

push:
	@echo "Pushing Docker image..."
	@docker build --platform linux/amd64 -f ./Dockerfile -t $(DOCKER_IMAGE) .
	@docker push $(DOCKER_IMAGE)
	@echo "Docker image pushed successfully"
	@caprover deploy -n abtdev -a taas-dev-gojobs -i $(DOCKER_IMAGE)