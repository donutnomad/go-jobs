# 多阶段构建 - 构建阶段
FROM golang:1.25-alpine AS builder

# 安装必要的构建工具
RUN apk add --no-cache git make gcc musl-dev

# 设置工作目录
WORKDIR /build

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 生成wire代码
# RUN go generate ./...

# 构建二进制文件
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o scheduler \
    ./cmd/scheduler/main.go ./cmd/scheduler/wire_gen.go ./cmd/scheduler/providers.go

# 最终阶段 - 运行时镜像
FROM alpine:latest

# 安装必要的运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非root用户
RUN addgroup -g 1000 scheduler && \
    adduser -D -u 1000 -G scheduler scheduler

# 创建必要的目录
RUN mkdir -p /app/configs /app/logs && \
    chown -R scheduler:scheduler /app

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/scheduler /app/scheduler

# 切换到非root用户
USER scheduler

# 暴露端口（默认80，可通过配置文件修改）
EXPOSE 80

# 启动命令（配置文件通过挂载提供）
ENTRYPOINT ["/app/scheduler"]
CMD ["-config", "/app/configs/config.yaml"]
