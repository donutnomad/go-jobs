'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'next/navigation';
import { useState, Suspense } from 'react';
import { ArrowLeft, Plus, Edit2, Trash2, Server, Settings, Activity, Clock, Weight, Users, AlertTriangle, Pause, Play, TrendingUp, Shield, BarChart3 } from 'lucide-react';
import Link from 'next/link';

// 90天状态条组件
interface StatusBarsProps {
    data: Array<{
        date: string;
        successRate: number;
        total: number;
    }>;
}

function StatusBars({ data }: StatusBarsProps) {
    // 根据成功率返回颜色
    const getColor = (rate: number) => {
        if (rate >= 99.9) return '#10b981'; // 深绿色
        if (rate >= 99) return '#34d399'; // 绿色
        if (rate >= 95) return '#86efac'; // 浅绿色
        if (rate >= 90) return '#fbbf24'; // 黄色
        if (rate >= 80) return '#fb923c'; // 橙色
        return '#ef4444'; // 红色
    };

    return (
        <div className="w-full">
            {/* 状态条容器 */}
            <div className="flex gap-0.5 w-full" style={{ height: '40px' }}>
                {data.map((day, index) => (
                    <div
                        key={index}
                        className="relative group flex-1 min-w-0"
                        style={{ maxWidth: `${100 / data.length}%` }}
                    >
                        <div
                            className="w-full h-full rounded-sm transition-all cursor-pointer hover:opacity-80"
                            style={{
                                backgroundColor: day.total > 0 ? getColor(day.successRate) : '#e5e7eb',
                            }}
                        />
                        {/* Tooltip */}
                        <div className="absolute bottom-full left-1/2 transform -translate-x-1/2 mb-2 px-3 py-2 bg-gray-900 text-white text-xs rounded opacity-0 group-hover:opacity-100 pointer-events-none z-50 whitespace-nowrap transition-opacity">
                            <div className="font-semibold">{new Date(day.date).toLocaleDateString('zh-CN')}</div>
                            <div>成功率: {day.successRate.toFixed(1)}%</div>
                            <div>执行次数: {day.total}</div>
                            {day.total > 0 && (
                                <div className="text-xs text-gray-300 mt-1">
                                    状态: {day.successRate >= 99.9 ? '完美' :
                                    day.successRate >= 95 ? '良好' :
                                        day.successRate >= 90 ? '正常' : '异常'}
                                </div>
                            )}
                            <div className="absolute top-full left-1/2 transform -translate-x-1/2 -mt-1">
                                <div className="border-4 border-transparent border-t-gray-900"></div>
                            </div>
                        </div>
                    </div>
                ))}
            </div>

            {/* 时间轴标签 */}
            <div className="flex justify-between mt-2 text-xs text-gray-500">
                <span>{data[0] && new Date(data[0].date).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })}</span>
                <span>{data[Math.floor(data.length / 2)] && new Date(data[Math.floor(data.length / 2)].date).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })}</span>
                <span>{data[data.length - 1] && new Date(data[data.length - 1].date).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })}</span>
            </div>

            {/* 图例 */}
            <div className="flex items-center justify-between mt-4">
                <div className="flex items-center gap-4 text-xs">
                    <div className="flex items-center gap-2">
                        <div className="w-3 h-3 rounded-sm bg-green-500"></div>
                        <span className="text-gray-600">100% 正常运行</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <div className="w-3 h-3 rounded-sm bg-yellow-400"></div>
                        <span className="text-gray-600">部分故障</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <div className="w-3 h-3 rounded-sm bg-red-500"></div>
                        <span className="text-gray-600">重大故障</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <div className="w-3 h-3 rounded-sm bg-gray-300"></div>
                        <span className="text-gray-600">无数据</span>
                    </div>
                </div>
                <div className="text-xs text-gray-500">
                    90天平均: <span className="font-semibold">{
                    data.length > 0
                        ? (data.reduce((sum, d) => sum + (d.total > 0 ? d.successRate : 100), 0) / data.length).toFixed(2)
                        : '0'
                }%</span>
                </div>
            </div>
        </div>
    );
}

interface Task {
    id: string;
    name: string;
    cron_expression: string;
    parameters: Record<string, unknown>;
    execution_mode: string;
    load_balance_strategy: string;
    max_retry: number;
    timeout_seconds: number;
    status: string;
    created_at: string;
    updated_at: string;
    task_executors: TaskExecutor[];
}

interface TaskExecutor {
    id: string;
    task_id: string;
    executor_id: string;
    priority: number;
    weight: number;
    created_at: string;
    executor: Executor;
}

interface Executor {
    id: string;
    name: string;
    instance_id: string;
    base_url: string;
    status: string;
    is_healthy: boolean;
    last_health_check: string;
    health_check_failures: number;
}

interface AvailableExecutor extends Executor {
    is_assigned: boolean;
}

function TaskDetailContent() {
    const searchParams = useSearchParams();
    const taskId = searchParams.get('id');
    const queryClient = useQueryClient();
    const [isAssignModalOpen, setIsAssignModalOpen] = useState(false);
    const [editingAssignment, setEditingAssignment] = useState<TaskExecutor | null>(null);

    // 如果没有 id 参数，显示错误
    if (!taskId) {
        return (
            <div className="container mx-auto p-6">
                <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
                    缺少任务 ID 参数
                </div>
                <Link href="/tasks" className="mt-4 inline-block text-blue-600 hover:underline">
                    返回任务列表
                </Link>
            </div>
        );
    }

    // 获取任务详情
    const { data: task, isLoading: isTaskLoading } = useQuery<Task>({
        queryKey: ['task', taskId],
        queryFn: async () => {
            const response = await fetch(`/api/v1/tasks/${taskId}`);
            if (!response.ok) {
                throw new Error('Failed to fetch task');
            }
            return response.json();
        },
        refetchInterval: 10000,
    });

    // 获取任务统计数据
    const { data: stats } = useQuery({
        queryKey: ['task-stats', taskId],
        queryFn: async () => {
            const response = await fetch(`/api/v1/tasks/${taskId}/stats`);
            if (!response.ok) {
                throw new Error('Failed to fetch task stats');
            }
            return response.json();
        },
        refetchInterval: 30000, // 每30秒刷新一次统计数据
    });

    // 获取所有可用的执行器
    const { data: allExecutors, isLoading: isExecutorsLoading } = useQuery<Executor[]>({
        queryKey: ['executors'],
        queryFn: async () => {
            const response = await fetch('/api/v1/executors');
            if (!response.ok) {
                throw new Error('Failed to fetch executors');
            }
            return response.json();
        },
    });

    // 删除执行器分配
    const unassignMutation = useMutation({
        mutationFn: async (executorId: string) => {
            const response = await fetch(`/api/v1/tasks/${taskId}/executors/${executorId}`, {
                method: 'DELETE',
            });
            if (!response.ok) {
                throw new Error('Failed to unassign executor');
            }
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['task', taskId] });
        },
    });

    // 暂停任务
    const pauseMutation = useMutation({
        mutationFn: async () => {
            const response = await fetch(`/api/v1/tasks/${taskId}/pause`, {
                method: 'POST',
            });
            if (!response.ok) {
                throw new Error('Failed to pause task');
            }
            return response.json();
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['task', taskId] });
        },
    });

    // 恢复任务
    const resumeMutation = useMutation({
        mutationFn: async () => {
            const response = await fetch(`/api/v1/tasks/${taskId}/resume`, {
                method: 'POST',
            });
            if (!response.ok) {
                throw new Error('Failed to resume task');
            }
            return response.json();
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['task', taskId] });
        },
    });

    const handlePauseResume = () => {
        if (!task) return;
        if (task.status === 'active') {
            if (confirm(`确定要暂停任务 "${task.name}" 的调度吗？`)) {
                pauseMutation.mutate();
            }
        } else if (task.status === 'paused') {
            if (confirm(`确定要恢复任务 "${task.name}" 的调度吗？`)) {
                resumeMutation.mutate();
            }
        }
    };

    const handleUnassign = async (assignment: TaskExecutor) => {
        if (confirm(`确定要从该任务中移除执行器 "${assignment.executor.name}" 吗？`)) {
            unassignMutation.mutate(assignment.executor_id);
        }
    };

    const getStrategyText = (strategy: string) => {
        const strategyMap: Record<string, string> = {
            'round_robin': '轮询',
            'weighted_round_robin': '加权轮询',
            'random': '随机',
            'sticky': '粘性',
            'least_loaded': '最少负载'
        };
        return strategyMap[strategy] || strategy;
    };

    const getExecutionModeText = (mode: string) => {
        const modeMap: Record<string, string> = {
            'sequential': '串行',
            'parallel': '并行',
            'skip': '跳过'
        };
        return modeMap[mode] || mode;
    };

    const getStatusBadge = (executor: Executor) => {
        if (executor.status === 'online' && executor.is_healthy) {
            return <span className="inline-flex px-2 py-1 text-xs font-medium bg-green-100 text-green-800 rounded-full">健康</span>;
        }
        if (executor.status === 'online' && !executor.is_healthy) {
            return <span className="inline-flex px-2 py-1 text-xs font-medium bg-yellow-100 text-yellow-800 rounded-full">异常</span>;
        }
        return <span className="inline-flex px-2 py-1 text-xs font-medium bg-red-100 text-red-800 rounded-full">离线</span>;
    };

    if (isTaskLoading || isExecutorsLoading) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="text-gray-500">加载中...</div>
            </div>
        );
    }

    if (!task) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="text-gray-500">任务不存在</div>
            </div>
        );
    }

    const availableExecutors: AvailableExecutor[] = (allExecutors || []).map(executor => ({
        ...executor,
        is_assigned: (task.task_executors || []).some(a => a.executor_id === executor.id)
    }));

    const healthyCount = (task.task_executors || []).filter(a => a.executor.is_healthy && a.executor.status === 'online').length;
    const totalWeight = (task.task_executors || []).reduce((sum, a) => sum + a.weight, 0);

    return (
        <div>
            {/* 面包屑导航 */}
            <div className="mb-6 flex items-center space-x-2 text-sm">
                <Link href="/tasks" className="text-blue-600 hover:text-blue-800 flex items-center space-x-1">
                    <ArrowLeft className="w-4 h-4" />
                    <span>任务管理</span>
                </Link>
                <span className="text-gray-500">/</span>
                <span className="text-gray-900">{task.name}</span>
            </div>

            {/* 任务信息 */}
            <div className="bg-white rounded-lg border border-gray-200 p-6 mb-6">
                <div className="flex items-start justify-between">
                    <div className="flex-1">
                        <div className="flex items-center space-x-3 mb-2">
                            <Settings className="w-6 h-6 text-blue-600" />
                            <h1 className="text-2xl font-bold text-gray-900">{task.name}</h1>
                            <span className={`px-2 py-1 rounded text-xs ${
                                task.status === 'active' ? 'bg-green-100 text-green-800' :
                                    task.status === 'paused' ? 'bg-yellow-100 text-yellow-800' :
                                        'bg-gray-100 text-gray-800'
                            }`}>
                {task.status === 'active' ? '活跃' :
                    task.status === 'paused' ? '已暂停' :
                        task.status}
              </span>
                        </div>

                        <div className="grid grid-cols-2 md:grid-cols-4 gap-6 text-sm">
                            <div className="flex items-center space-x-2">
                                <Clock className="w-4 h-4 text-gray-400" />
                                <span className="text-gray-600">Cron:</span>
                                <span className="font-mono font-medium">{task.cron_expression}</span>
                            </div>
                            <div className="flex items-center space-x-2">
                                <Activity className="w-4 h-4 text-gray-400" />
                                <span className="text-gray-600">执行模式:</span>
                                <span className="font-medium">{getExecutionModeText(task.execution_mode)}</span>
                            </div>
                            <div className="flex items-center space-x-2">
                                <Server className="w-4 h-4 text-gray-400" />
                                <span className="text-gray-600">负载均衡:</span>
                                <span className="font-medium">{getStrategyText(task.load_balance_strategy)}</span>
                            </div>
                            <div className="flex items-center space-x-2">
                                <Users className="w-4 h-4 text-gray-400" />
                                <span className="text-gray-600">分配执行器:</span>
                                <span className="font-medium">{(task.task_executors || []).length}</span>
                            </div>
                        </div>
                    </div>

                    <div className="flex items-center space-x-2">
                        <Link
                            href={`/task-edit?id=${task.id}`}
                            className="inline-flex items-center px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
                        >
                            <Edit2 className="w-4 h-4 mr-2" />
                            编辑任务
                        </Link>
                        <button
                            onClick={handlePauseResume}
                            disabled={pauseMutation.isPending || resumeMutation.isPending || task.status === 'deleted'}
                            className={`inline-flex items-center px-4 py-2 rounded-lg transition-colors ${
                                task.status === 'active'
                                    ? 'bg-yellow-600 text-white hover:bg-yellow-700'
                                    : task.status === 'paused'
                                        ? 'bg-green-600 text-white hover:bg-green-700'
                                        : 'bg-gray-400 text-gray-200 cursor-not-allowed'
                            }`}
                        >
                            {task.status === 'active' ? (
                                <>
                                    <Pause className="w-4 h-4 mr-2" />
                                    {pauseMutation.isPending ? '暂停中...' : '暂停调度'}
                                </>
                            ) : task.status === 'paused' ? (
                                <>
                                    <Play className="w-4 h-4 mr-2" />
                                    {resumeMutation.isPending ? '恢复中...' : '恢复调度'}
                                </>
                            ) : (
                                <>
                                    <Pause className="w-4 h-4 mr-2" />
                                    暂停调度
                                </>
                            )}
                        </button>
                        <button
                            onClick={() => setIsAssignModalOpen(true)}
                            className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                        >
                            <Plus className="w-4 h-4 mr-2" />
                            分配执行器
                        </button>
                    </div>
                </div>
            </div>

            {/* 统计信息 */}
            <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-6">
                <div className="bg-white rounded-lg p-4 border border-gray-200">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-sm text-gray-600">总分配数</p>
                            <p className="text-2xl font-bold text-gray-900">{(task.task_executors || []).length}</p>
                        </div>
                        <Server className="w-8 h-8 text-gray-400" />
                    </div>
                </div>

                <div className="bg-white rounded-lg p-4 border border-gray-200">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-sm text-gray-600">健康执行器</p>
                            <p className="text-2xl font-bold text-green-600">{healthyCount}</p>
                        </div>
                        <Activity className="w-8 h-8 text-green-400" />
                    </div>
                </div>

                <div className="bg-white rounded-lg p-4 border border-gray-200">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-sm text-gray-600">24h成功率</p>
                            <p className="text-2xl font-bold text-blue-600">
                                {stats ? `${stats.success_rate_24h?.toFixed(1)}%` : '-'}
                            </p>
                            {stats && stats.total_24h > 0 && (
                                <p className="text-xs text-gray-500 mt-1">
                                    {stats.success_24h}/{stats.total_24h}
                                </p>
                            )}
                        </div>
                        <TrendingUp className="w-8 h-8 text-blue-400" />
                    </div>
                </div>

                <div className="bg-white rounded-lg p-4 border border-gray-200">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-sm text-gray-600">90天健康度</p>
                            <p className="text-2xl font-bold" style={{
                                color: stats?.health_90d?.health_score >= 80 ? '#10b981' :
                                    stats?.health_90d?.health_score >= 60 ? '#f59e0b' : '#ef4444'
                            }}>
                                {stats?.health_90d ? `${stats.health_90d.health_score?.toFixed(0)}` : '-'}
                            </p>
                            {stats?.health_90d && (
                                <p className="text-xs text-gray-500 mt-1">
                                    总执行: {stats.health_90d.total_count}
                                </p>
                            )}
                        </div>
                        <Shield className="w-8 h-8" style={{
                            color: stats?.health_90d?.health_score >= 80 ? '#86efac' :
                                stats?.health_90d?.health_score >= 60 ? '#fcd34d' : '#fca5a5'
                        }} />
                    </div>
                </div>

                <div className="bg-white rounded-lg p-4 border border-gray-200">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-sm text-gray-600">总权重</p>
                            <p className="text-2xl font-bold text-purple-600">{totalWeight}</p>
                        </div>
                        <Weight className="w-8 h-8 text-purple-400" />
                    </div>
                </div>

                <div className="bg-white rounded-lg p-4 border border-gray-200">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-sm text-gray-600">重试次数</p>
                            <p className="text-2xl font-bold text-indigo-600">{task.max_retry}</p>
                        </div>
                        <Users className="w-8 h-8 text-indigo-400" />
                    </div>
                </div>
            </div>

            {/* 90天服务状态图 */}
            {stats?.daily_stats_90d && (
                <div className="bg-white rounded-lg border border-gray-200 p-6 mb-6">
                    <div className="flex items-center justify-between mb-6">
                        <h3 className="text-lg font-semibold text-gray-900 flex items-center">
                            <BarChart3 className="w-5 h-5 mr-2 text-gray-600" />
                            90天服务状态
                        </h3>
                        <div className="text-sm text-gray-500">
                            总体健康度: <span className="font-semibold" style={{
                            color: stats?.health_90d?.health_score >= 80 ? '#10b981' :
                                stats?.health_90d?.health_score >= 60 ? '#f59e0b' : '#ef4444'
                        }}>{stats?.health_90d?.health_score?.toFixed(1)}%</span>
                        </div>
                    </div>
                    <StatusBars
                        data={stats.daily_stats_90d.map((day: any) => ({
                            date: day.date,
                            successRate: day.successRate,
                            total: day.total
                        }))}
                    />
                </div>
            )}

            {/* 执行器分配列表 */}
            <div className="bg-white rounded-lg border border-gray-200">
                <div className="px-6 py-4 border-b border-gray-200">
                    <h2 className="text-lg font-semibold text-gray-900">执行器分配</h2>
                </div>

                {(task.task_executors || []).length === 0 ? (
                    <div className="p-12 text-center">
                        <AlertTriangle className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                        <p className="text-gray-500 mb-4">该任务还没有分配任何执行器</p>
                        <button
                            onClick={() => setIsAssignModalOpen(true)}
                            className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                        >
                            <Plus className="w-4 h-4 mr-2" />
                            分配第一个执行器
                        </button>
                    </div>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="w-full">
                            <thead className="bg-gray-50">
                            <tr>
                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">执行器</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">优先级</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">权重</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">基础URL</th>
                                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">分配时间</th>
                                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">操作</th>
                            </tr>
                            </thead>
                            <tbody className="divide-y divide-gray-200">
                            {(task.task_executors || [])
                                .sort((a, b) => b.priority - a.priority)
                                .map((assignment) => (
                                    <tr key={assignment.id} className="hover:bg-gray-50">
                                        <td className="px-6 py-4">
                                            <div className="flex items-center space-x-3">
                                                <div className={`w-2 h-2 rounded-full ${
                                                    assignment.executor.is_healthy && assignment.executor.status === 'online'
                                                        ? 'bg-green-400'
                                                        : 'bg-red-400'
                                                }`} />
                                                <div>
                                                    <div className="font-medium text-gray-900">{assignment.executor.name}</div>
                                                    <div className="text-sm text-gray-500">{assignment.executor.instance_id}</div>
                                                </div>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4">
                                            {getStatusBadge(assignment.executor)}
                                        </td>
                                        <td className="px-6 py-4">
                      <span className="inline-flex items-center px-2 py-1 text-sm font-medium bg-purple-100 text-purple-800 rounded">
                        {assignment.priority}
                      </span>
                                        </td>
                                        <td className="px-6 py-4">
                      <span className="inline-flex items-center px-2 py-1 text-sm font-medium bg-blue-100 text-blue-800 rounded">
                        {assignment.weight}
                      </span>
                                        </td>
                                        <td className="px-6 py-4">
                                            <span className="text-sm text-gray-900">{assignment.executor.base_url}</span>
                                        </td>
                                        <td className="px-6 py-4">
                      <span className="text-sm text-gray-500">
                        {new Date(assignment.created_at).toLocaleDateString()}
                      </span>
                                        </td>
                                        <td className="px-6 py-4 text-right">
                                            <div className="flex items-center justify-end space-x-2">
                                                <button
                                                    onClick={() => setEditingAssignment(assignment)}
                                                    className="p-1 text-gray-400 hover:text-gray-600"
                                                    title="编辑分配"
                                                >
                                                    <Edit2 className="w-4 h-4" />
                                                </button>
                                                <button
                                                    onClick={() => handleUnassign(assignment)}
                                                    className="p-1 text-gray-400 hover:text-red-600"
                                                    title="移除分配"
                                                    disabled={unassignMutation.isPending}
                                                >
                                                    <Trash2 className="w-4 h-4" />
                                                </button>
                                            </div>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            {/* TODO: 分配和编辑执行器的模态框 */}

            {/* 分配执行器模态框 */}
            {isAssignModalOpen && (
                <AssignExecutorModal
                    isOpen={isAssignModalOpen}
                    onClose={() => setIsAssignModalOpen(false)}
                    taskId={taskId}
                    availableExecutors={availableExecutors}
                    onSuccess={() => {
                        queryClient.invalidateQueries({ queryKey: ['task', taskId] });
                        setIsAssignModalOpen(false);
                    }}
                />
            )}

            {/* 编辑分配模态框 */}
            {editingAssignment && (
                <EditAssignmentModal
                    isOpen={!!editingAssignment}
                    onClose={() => setEditingAssignment(null)}
                    taskId={taskId}
                    assignment={editingAssignment}
                    onSuccess={() => {
                        queryClient.invalidateQueries({ queryKey: ['task', taskId] });
                        setEditingAssignment(null);
                    }}
                />
            )}
        </div>
    );
}

// 分配执行器模态框组件
interface AssignExecutorModalProps {
    isOpen: boolean;
    onClose: () => void;
    taskId: string;
    availableExecutors: AvailableExecutor[];
    onSuccess: () => void;
}

function AssignExecutorModal({
                                 isOpen,
                                 onClose,
                                 taskId,
                                 availableExecutors,
                                 onSuccess
                             }: AssignExecutorModalProps) {
    const [selectedExecutorId, setSelectedExecutorId] = useState('');
    const [priority, setPriority] = useState(10);
    const [weight, setWeight] = useState(1);

    const assignMutation = useMutation({
        mutationFn: async (data: { executor_id: string; priority: number; weight: number }) => {
            const response = await fetch(`/api/v1/tasks/${taskId}/executors`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data),
            });
            if (!response.ok) {
                throw new Error('Failed to assign executor');
            }
            return response.json();
        },
        onSuccess: () => {
            onSuccess();
        },
    });

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (selectedExecutorId) {
            assignMutation.mutate({
                executor_id: selectedExecutorId,
                priority,
                weight,
            });
        }
    };

    const unassignedExecutors = (availableExecutors || []).filter(e => !e.is_assigned && e.status === 'online');

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg max-w-md w-full p-6">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">分配执行器</h3>

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                            选择执行器 *
                        </label>
                        <select
                            value={selectedExecutorId}
                            onChange={(e) => setSelectedExecutorId(e.target.value)}
                            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                            required
                        >
                            <option value="">请选择执行器</option>
                            {unassignedExecutors.map((executor) => (
                                <option key={executor.id} value={executor.id}>
                                    {executor.name} ({executor.instance_id})
                                </option>
                            ))}
                        </select>
                        {unassignedExecutors.length === 0 && (
                            <p className="text-sm text-gray-500 mt-1">没有可用的执行器</p>
                        )}
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                            优先级
                        </label>
                        <input
                            type="number"
                            value={priority}
                            onChange={(e) => setPriority(parseInt(e.target.value))}
                            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                            min="1"
                            max="100"
                        />
                        <p className="text-xs text-gray-500 mt-1">数值越大优先级越高</p>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                            权重
                        </label>
                        <input
                            type="number"
                            value={weight}
                            onChange={(e) => setWeight(parseInt(e.target.value))}
                            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                            min="1"
                            max="100"
                        />
                        <p className="text-xs text-gray-500 mt-1">用于加权负载均衡</p>
                    </div>

                    <div className="flex justify-end space-x-3 pt-4">
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50"
                        >
                            取消
                        </button>
                        <button
                            type="submit"
                            disabled={assignMutation.isPending || !selectedExecutorId}
                            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                        >
                            {assignMutation.isPending ? '分配中...' : '分配'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}

// 编辑分配模态框组件
interface EditAssignmentModalProps {
    isOpen: boolean;
    onClose: () => void;
    taskId: string;
    assignment: TaskExecutor;
    onSuccess: () => void;
}

function EditAssignmentModal({
                                 isOpen,
                                 onClose,
                                 taskId,
                                 assignment,
                                 onSuccess
                             }: EditAssignmentModalProps) {
    const [priority, setPriority] = useState(assignment.priority);
    const [weight, setWeight] = useState(assignment.weight);

    const updateMutation = useMutation({
        mutationFn: async (data: { priority: number; weight: number }) => {
            const response = await fetch(`/api/v1/tasks/${taskId}/executors/${assignment.executor_id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data),
            });
            if (!response.ok) {
                throw new Error('Failed to update assignment');
            }
            return response.json();
        },
        onSuccess: () => {
            onSuccess();
        },
    });

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        updateMutation.mutate({ priority, weight });
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg max-w-md w-full p-6">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">编辑执行器分配</h3>
                <p className="text-sm text-gray-600 mb-4">
                    执行器：{assignment.executor?.name || '未知'} ({assignment.executor?.instance_id || '未知'})
                </p>

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                            优先级
                        </label>
                        <input
                            type="number"
                            value={priority}
                            onChange={(e) => setPriority(parseInt(e.target.value))}
                            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                            min="1"
                            max="100"
                        />
                        <p className="text-xs text-gray-500 mt-1">数值越大优先级越高</p>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                            权重
                        </label>
                        <input
                            type="number"
                            value={weight}
                            onChange={(e) => setWeight(parseInt(e.target.value))}
                            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                            min="1"
                            max="100"
                        />
                        <p className="text-xs text-gray-500 mt-1">用于加权负载均衡</p>
                    </div>

                    <div className="flex justify-end space-x-3 pt-4">
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50"
                        >
                            取消
                        </button>
                        <button
                            type="submit"
                            disabled={updateMutation.isPending}
                            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                        >
                            {updateMutation.isPending ? '保存中...' : '保存'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}

export default function TaskDetailPage() {
    return (
        <Suspense fallback={<div className="flex items-center justify-center h-64"><div className="text-gray-500">加载中...</div></div>}>
            <TaskDetailContent />
        </Suspense>
    );
}