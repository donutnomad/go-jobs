#!/bin/bash

# 执行器自动注册测试脚本

echo "=== 执行器自动注册测试 ==="

# API 端点
API_URL="http://localhost:8080/api/v1/executors/register"

# 测试数据文件
TEST_DATA="examples/register_example.json"

echo "正在向 $API_URL 发送注册请求..."
echo "使用测试数据: $TEST_DATA"
echo ""

# 发送注册请求
curl -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -d @"$TEST_DATA" \
  -w "\nHTTP状态码: %{http_code}\n" \
  | jq '.'

echo ""
echo "=== 测试完成 ==="

# 查看注册的任务
echo ""
echo "=== 查看注册的任务 ==="
curl -X GET "http://localhost:8080/api/v1/tasks" \
  -H "Content-Type: application/json" \
  | jq '.[] | {id: .id, name: .name, status: .status, cron_expression: .cron_expression}'

# 查看注册的执行器
echo ""
echo "=== 查看注册的执行器 ==="
curl -X GET "http://localhost:8080/api/v1/executors" \
  -H "Content-Type: application/json" \
  | jq '.[] | {id: .id, name: .name, instance_id: .instance_id, status: .status}'