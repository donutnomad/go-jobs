#!/bin/bash
# 测试任务统计API

TASK_ID="b8c5e2f1-9b4a-4e3d-8f7a-1a2b3c4d5e6f"  # 需要替换为实际的任务ID
BASE_URL="http://localhost:8080/api/v1"

echo "🧪 测试任务统计API"
echo "=================="
echo ""

# 获取任务列表，取第一个任务ID
echo "1. 获取任务列表..."
TASK_ID=$(curl -s "$BASE_URL/tasks" | jq -r '.[0].id // empty')

if [ -z "$TASK_ID" ]; then
    echo "❌ 没有找到任务，请先创建一个任务"
    exit 1
fi

echo "✅ 使用任务ID: $TASK_ID"
echo ""

# 获取任务统计
echo "2. 获取任务统计数据..."
STATS=$(curl -s "$BASE_URL/tasks/$TASK_ID/stats")

if [ $? -eq 0 ]; then
    echo "✅ 成功获取统计数据:"
    echo ""
    
    # 解析并显示统计数据
    echo "📊 24小时统计:"
    echo "  - 成功率: $(echo $STATS | jq -r '.success_rate_24h // 0')%"
    echo "  - 总执行: $(echo $STATS | jq -r '.total_24h // 0')"
    echo "  - 成功数: $(echo $STATS | jq -r '.success_24h // 0')"
    echo ""
    
    echo "📈 90天健康度:"
    echo "  - 健康分数: $(echo $STATS | jq -r '.health_90d.health_score // 0')"
    echo "  - 总执行次数: $(echo $STATS | jq -r '.health_90d.total_count // 0')"
    echo "  - 成功次数: $(echo $STATS | jq -r '.health_90d.success_count // 0')"
    echo "  - 失败次数: $(echo $STATS | jq -r '.health_90d.failed_count // 0')"
    echo "  - 超时次数: $(echo $STATS | jq -r '.health_90d.timeout_count // 0')"
    echo "  - 平均执行时间: $(echo $STATS | jq -r '.health_90d.avg_duration_seconds // 0')秒"
    echo ""
    
    echo "📅 最近7天执行趋势:"
    echo $STATS | jq -r '.recent_executions[] | "  \(.date): 总数=\(.total), 成功=\(.success), 失败=\(.failed), 成功率=\(.success_rate)%"'
    
else
    echo "❌ 获取统计数据失败"
fi

echo ""
echo "✅ 测试完成!"