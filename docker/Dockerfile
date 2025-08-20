# 构建阶段
FROM golang:1.21-alpine AS builder

# 安装依赖
RUN apk add --no-cache git

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o scheduler cmd/scheduler/main.go

# 运行阶段
FROM alpine:latest

# 安装必要的运行时依赖
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 创建用户
RUN addgroup -S scheduler && adduser -S scheduler -G scheduler

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/scheduler .

# 复制配置文件
COPY configs/config.yaml ./configs/

# 创建日志目录
RUN mkdir -p logs && chown -R scheduler:scheduler logs

# 切换到非root用户
USER scheduler

# 暴露端口
EXPOSE 8080

# 启动应用
CMD ["./scheduler", "-config", "configs/config.yaml"]