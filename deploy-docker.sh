#!/bin/bash

# Job Scheduler Docker 快速部署脚本

set -e

# 颜色输出
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Job Scheduler Docker 部署脚本 ===${NC}\n"

# 检查是否在项目根目录
if [ ! -f "go.mod" ]; then
    echo -e "${RED}错误: 请在项目根目录下运行此脚本${NC}"
    exit 1
fi

# 检查配置文件
if [ ! -f "configs/config.docker.yaml" ]; then
    echo -e "${YELLOW}警告: 未找到 configs/config.docker.yaml${NC}"
    echo -e "${BLUE}正在从示例配置创建...${NC}"

    if [ -f "configs/config.example.yaml" ]; then
        cp configs/config.example.yaml configs/config.docker.yaml
        echo -e "${GREEN}✓ 已创建 configs/config.docker.yaml${NC}"
        echo -e "${YELLOW}请编辑此文件，修改数据库和Redis配置后重新运行此脚本${NC}"
        exit 0
    else
        echo -e "${RED}错误: 未找到配置示例文件${NC}"
        exit 1
    fi
fi

# 创建必要的目录
echo -e "${BLUE}步骤 1: 创建必要的目录...${NC}"
mkdir -p logs
echo -e "${GREEN}✓ 目录创建完成${NC}\n"

# 构建镜像
echo -e "${BLUE}步骤 2: 构建 Docker 镜像...${NC}"
docker build -t job-scheduler:latest .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 镜像构建成功${NC}\n"
else
    echo -e "${RED}✗ 镜像构建失败${NC}"
    exit 1
fi

# 检查镜像大小
echo -e "${BLUE}步骤 3: 检查镜像信息...${NC}"
docker images job-scheduler:latest
echo ""

# 创建网络（如果不存在）
echo -e "${BLUE}步骤 4: 创建 Docker 网络...${NC}"
if ! docker network inspect scheduler-network >/dev/null 2>&1; then
    docker network create scheduler-network
    echo -e "${GREEN}✓ 网络创建成功${NC}\n"
else
    echo -e "${GREEN}✓ 网络已存在${NC}\n"
fi

# 停止旧容器（如果存在）
if docker ps -a --format '{{.Names}}' | grep -q '^job-scheduler$'; then
    echo -e "${BLUE}步骤 5: 停止旧容器...${NC}"
    docker stop job-scheduler || true
    docker rm job-scheduler || true
    echo -e "${GREEN}✓ 旧容器已移除${NC}\n"
fi

# 启动新容器
echo -e "${BLUE}步骤 6: 启动容器...${NC}"
docker run -d \
  --name job-scheduler \
  --network scheduler-network \
  -p 8080:8080 \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/logs:/app/logs \
  --restart unless-stopped \
  job-scheduler:latest

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 容器启动成功${NC}\n"
else
    echo -e "${RED}✗ 容器启动失败${NC}"
    exit 1
fi

# 等待服务启动
echo -e "${BLUE}步骤 7: 等待服务启动...${NC}"
sleep 5

# 查看容器状态
echo -e "${BLUE}步骤 8: 检查容器状态...${NC}"
docker ps --filter "name=job-scheduler" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo ""

# 查看容器日志
echo -e "${BLUE}步骤 9: 查看启动日志...${NC}"
docker logs --tail 20 job-scheduler
echo ""

# 测试健康检查
echo -e "${BLUE}步骤 10: 测试健康检查...${NC}"
sleep 3
HEALTH_CHECK=$(curl -s http://localhost:8080/api/v1/health || echo "failed")

if echo "$HEALTH_CHECK" | grep -q "ok\|status"; then
    echo -e "${GREEN}✓ 健康检查通过${NC}"
    echo "$HEALTH_CHECK" | head -3
else
    echo -e "${YELLOW}⚠ 健康检查未通过，请查看日志${NC}"
fi

echo ""
echo -e "${GREEN}=== 部署完成 ===${NC}\n"
echo -e "服务信息:"
echo -e "  API地址: ${BLUE}http://localhost:8080${NC}"
echo -e "  健康检查: ${BLUE}http://localhost:8080/api/v1/health${NC}"
echo -e "  调度器状态: ${BLUE}http://localhost:8080/api/v1/scheduler/status${NC}\n"

echo -e "常用命令:"
echo -e "  查看日志: ${BLUE}docker logs -f job-scheduler${NC}"
echo -e "  进入容器: ${BLUE}docker exec -it job-scheduler sh${NC}"
echo -e "  停止服务: ${BLUE}docker stop job-scheduler${NC}"
echo -e "  启动服务: ${BLUE}docker start job-scheduler${NC}"
echo -e "  重启服务: ${BLUE}docker restart job-scheduler${NC}"
echo -e "  查看状态: ${BLUE}docker ps --filter name=job-scheduler${NC}\n"

echo -e "配置文件: ${BLUE}configs/config.docker.yaml${NC}"
echo -e "日志目录: ${BLUE}logs/${NC}\n"
