'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Edit, Trash2, Play, Clock, Server, Users, Filter, Settings } from 'lucide-react';
import { format } from 'date-fns';
import Link from 'next/link';

interface Task {
  id: string;
  name: string;
  cron_expression: string;
  parameters: any;
  execution_mode: string;
  load_balance_strategy: string;
  max_retry: number;
  timeout_seconds: number;
  status: string;
  created_at: string;
  updated_at: string;
  task_executors?: TaskExecutor[];
}

interface TaskExecutor {
  id: string;
  task_id: string;
  executor_id: string;
  priority: number;
  weight: number;
  executor?: Executor;
}

interface Executor {
  id: string;
  name: string;
  instance_id: string;
  status: string;
  is_healthy: boolean;
}

export default function TasksPage() {
  const [statusFilter, setStatusFilter] = useState<string>('');
  const queryClient = useQueryClient();
  
  // 从环境变量获取 API URL
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

  // 获取任务列表
  const { data: tasks, isLoading } = useQuery<Task[]>({
    queryKey: ['tasks', statusFilter],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (statusFilter) params.append('status', statusFilter);
      
      const response = await fetch(`${apiUrl}/api/v1/tasks?${params}`);
      if (!response.ok) {
        throw new Error('Failed to fetch tasks');
      }
      const result = await response.json();
      // 适配新的响应格式
      return result.data || result;
    },
    refetchInterval: 30000,
    refetchOnWindowFocus: true,
    refetchOnMount: true,
    staleTime: 0,
  });

  // 删除任务
  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const response = await fetch(`${apiUrl}/api/v1/tasks/${id}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error('Failed to delete task');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
    },
  });

  // 手动触发任务
  const triggerMutation = useMutation({
    mutationFn: async (id: string) => {
      const response = await fetch(`${apiUrl}/api/v1/tasks/${id}/trigger`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      if (!response.ok) {
        throw new Error('Failed to trigger task');
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['executions'] });
    },
  });

  const handleDelete = async (task: Task) => {
    if (confirm(`确定要删除任务 "${task.name}" 吗？`)) {
      deleteMutation.mutate(task.id);
    }
  };

  const handleTrigger = async (task: Task) => {
    try {
      await triggerMutation.mutateAsync(task.id);
      alert('任务已触发执行');
    } catch (error) {
      alert('触发任务失败');
    }
  };

  const getStatusBadge = (status: string) => {
    const statusConfig = {
      active: { color: 'bg-green-100 text-green-800', label: '活跃' },
      paused: { color: 'bg-yellow-100 text-yellow-800', label: '暂停' },
      deleted: { color: 'bg-red-100 text-red-800', label: '已删除' },
    };
    const config = statusConfig[status as keyof typeof statusConfig] || statusConfig.active;
    return (
      <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${config.color}`}>
        {config.label}
      </span>
    );
  };

  const getExecutionModeBadge = (mode: string) => {
    const modeConfig = {
      sequential: { color: 'bg-blue-100 text-blue-800', label: '串行' },
      parallel: { color: 'bg-purple-100 text-purple-800', label: '并行' },
      skip: { color: 'bg-gray-100 text-gray-800', label: '跳过' },
    };
    const config = modeConfig[mode as keyof typeof modeConfig] || modeConfig.parallel;
    return (
      <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${config.color}`}>
        {config.label}
      </span>
    );
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

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">加载中...</div>
      </div>
    );
  }

  return (
    <div>
      {/* 页面头部 */}
      <div className="mb-8 flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">任务管理</h1>
          <p className="mt-1 text-sm text-gray-600">
            创建和管理调度任务，为任务分配执行器
          </p>
        </div>
        <Link
          href="/tasks/create"
          className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          <Plus className="w-4 h-4 mr-2" />
          创建任务
        </Link>
      </div>

      {/* 筛选器 */}
      <div className="mb-6 flex items-center space-x-4">
        <div className="flex items-center space-x-2">
          <Filter className="w-4 h-4 text-gray-400" />
          <span className="text-sm text-gray-700">筛选:</span>
        </div>
        
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="">全部状态</option>
          <option value="active">活跃</option>
          <option value="paused">暂停</option>
        </select>
      </div>

      {/* 统计信息 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-white rounded-lg p-4 border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">总任务数</p>
              <p className="text-2xl font-bold text-gray-900">{tasks?.length || 0}</p>
            </div>
            <Server className="w-8 h-8 text-gray-400" />
          </div>
        </div>

        <div className="bg-white rounded-lg p-4 border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">活跃任务</p>
              <p className="text-2xl font-bold text-green-600">
                {tasks?.filter(t => t.status === 'active').length || 0}
              </p>
            </div>
            <Users className="w-8 h-8 text-green-400" />
          </div>
        </div>

        <div className="bg-white rounded-lg p-4 border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">已配置执行器</p>
              <p className="text-2xl font-bold text-blue-600">
                {tasks?.filter(t => t.task_executors && t.task_executors.length > 0).length || 0}
              </p>
            </div>
            <Settings className="w-8 h-8 text-blue-400" />
          </div>
        </div>

        <div className="bg-white rounded-lg p-4 border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">暂停任务</p>
              <p className="text-2xl font-bold text-yellow-600">
                {tasks?.filter(t => t.status === 'paused').length || 0}
              </p>
            </div>
            <Clock className="w-8 h-8 text-yellow-400" />
          </div>
        </div>
      </div>

      {/* 任务列表 */}
      <div className="bg-white rounded-lg border border-gray-200">
        <div className="px-6 py-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">任务列表</h2>
        </div>

        {(!tasks || tasks.length === 0) ? (
          <div className="text-center py-12">
            <Server className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <p className="text-gray-500 mb-4">暂无任务</p>
            <Link
              href="/tasks/create"
              className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              <Plus className="w-4 h-4 mr-2" />
              创建第一个任务
            </Link>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">任务名称</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Cron 表达式</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">执行模式</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">负载均衡</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">执行器</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">创建时间</th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {tasks.map((task) => (
                  <tr key={task.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4">
                      <div className="text-sm font-medium text-gray-900">{task.name}</div>
                      <div className="text-sm text-gray-500">ID: {task.id}</div>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center space-x-1 text-sm text-gray-900">
                        <Clock className="w-4 h-4 text-gray-400" />
                        <span className="font-mono">{task.cron_expression}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      {getExecutionModeBadge(task.execution_mode)}
                    </td>
                    <td className="px-6 py-4">
                      <span className="text-sm text-gray-900">
                        {getStrategyText(task.load_balance_strategy)}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center space-x-1">
                        <Users className="w-4 h-4 text-gray-400" />
                        <span className="text-sm text-gray-900">
                          {task.task_executors?.length || 0}
                        </span>
                        {(task.task_executors?.length || 0) === 0 && (
                          <span className="text-xs text-red-600 ml-2">未配置</span>
                        )}
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      {getStatusBadge(task.status)}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {format(new Date(task.created_at), 'yyyy-MM-dd HH:mm')}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <div className="flex items-center justify-end space-x-2">
                        <button
                          onClick={() => handleTrigger(task)}
                          className="p-1 text-green-600 hover:text-green-800"
                          title="手动触发"
                          disabled={triggerMutation.isPending}
                        >
                          <Play className="w-4 h-4" />
                        </button>
                        <Link
                          href={`/task-detail?id=${task.id}`}
                          className="p-1 text-blue-600 hover:text-blue-800"
                          title="管理执行器"
                        >
                          <Settings className="w-4 h-4" />
                        </Link>
                        <Link
                          href={`/task-edit?id=${task.id}`}
                          className="p-1 text-blue-600 hover:text-blue-800"
                          title="编辑"
                        >
                          <Edit className="w-4 h-4" />
                        </Link>
                        <button
                          onClick={() => handleDelete(task)}
                          className="p-1 text-red-600 hover:text-red-800"
                          title="删除"
                          disabled={deleteMutation.isPending}
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
    </div>
  );
}