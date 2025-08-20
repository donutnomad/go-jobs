'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Server, Activity, CheckCircle, AlertCircle, Calendar, Users, Settings, Filter, Edit, Trash2 } from 'lucide-react';
import { format } from 'date-fns';
import Link from 'next/link';

interface Task {
  id: string;
  name: string;
  cron_expression: string;
  execution_mode: string;
  load_balance_strategy: string;
  status: string;
}

interface TaskExecutor {
  id: string;
  task_id: string;
  executor_id: string;
  weight: number;
  priority: number;
  created_at: string;
  task?: Task;
}

interface Executor {
  id: string;
  name: string;
  instance_id: string;
  base_url: string;
  health_check_url: string;
  status: string;
  is_healthy: boolean;
  last_health_check: string;
  health_check_failures: number;
  created_at: string;
  task_executors?: TaskExecutor[];
}

export default function ExecutorsPage() {
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [editingExecutor, setEditingExecutor] = useState<Executor | null>(null);
  const [editForm, setEditForm] = useState({
    name: '',
    base_url: '',
    health_check_url: '',
  });
  const queryClient = useQueryClient();

  // 获取执行器列表（包含关联的任务信息）- 不再传递筛选参数给后端
  const { data: allExecutors, isLoading } = useQuery<Executor[]>({
    queryKey: ['executors'],
    queryFn: async () => {
      const params = new URLSearchParams();
      params.append('include_tasks', 'true');
      
      const response = await fetch(`/api/v1/executors?${params}`);
      if (!response.ok) {
        throw new Error('Failed to fetch executors');
      }
      return response.json();
    },
    refetchInterval: 30000,
    refetchOnWindowFocus: true,
    refetchOnMount: true,
    staleTime: 0,
  });

  // 在前端本地筛选执行器
  const executors = allExecutors?.filter(executor => {
    if (!statusFilter) return true;
    
    if (statusFilter === 'online') {
      return executor.status === 'online';
    } else if (statusFilter === 'offline') {
      return executor.status === 'offline';
    }
    
    return true;
  });

  // 删除执行器
  const deleteMutation = useMutation({
    mutationFn: async (executorId: string) => {
      const response = await fetch(`/api/v1/executors/${executorId}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error('Failed to delete executor');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['executors'] });
    },
  });

  // 更新执行器
  const updateMutation = useMutation({
    mutationFn: async ({ executorId, data }: { executorId: string; data: any }) => {
      const response = await fetch(`/api/v1/executors/${executorId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      });
      if (!response.ok) {
        throw new Error('Failed to update executor');
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['executors'] });
      setEditingExecutor(null);
    },
  });

  const handleDelete = async (executor: Executor) => {
    if (confirm(`确定要删除执行器 \"${executor.name}\" 吗？`)) {
      deleteMutation.mutate(executor.id);
    }
  };

  const handleEdit = (executor: Executor) => {
    setEditingExecutor(executor);
    setEditForm({
      name: executor.name,
      base_url: executor.base_url,
      health_check_url: executor.health_check_url,
    });
  };

  const handleSaveEdit = () => {
    if (!editingExecutor) return;
    
    updateMutation.mutate({
      executorId: editingExecutor.id,
      data: editForm,
    });
  };

  const handleCancelEdit = () => {
    setEditingExecutor(null);
    setEditForm({
      name: '',
      base_url: '',
      health_check_url: '',
    });
  };

  const getStatusBadge = (executor: Executor) => {
    if (executor.status === 'offline') {
      return (
        <span className="inline-flex items-center px-2 py-1 text-xs font-medium bg-red-100 text-red-800 rounded-full">
          <AlertCircle className="w-3 h-3 mr-1" />
          离线
        </span>
      );
    }
    if (executor.status === 'online' && executor.is_healthy) {
      return (
        <span className="inline-flex items-center px-2 py-1 text-xs font-medium bg-green-100 text-green-800 rounded-full">
          <CheckCircle className="w-3 h-3 mr-1" />
          健康
        </span>
      );
    }
    // online but not healthy
    return (
      <span className="inline-flex items-center px-2 py-1 text-xs font-medium bg-yellow-100 text-yellow-800 rounded-full">
        <AlertCircle className="w-3 h-3 mr-1" />
        异常
      </span>
    );
  };

  const getStrategyText = (strategy: string) => {
    const strategyMap: Record<string, string> = {
      'round_robin': '轮询',
      'weighted': '加权',
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
          <h1 className="text-2xl font-bold text-gray-900">执行器管理</h1>
          <p className="mt-1 text-sm text-gray-600">
            管理任务执行器，查看它们关联的任务
          </p>
        </div>
        <button
          onClick={() => window.location.href = '/executors/register'}
          className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          <Plus className="w-4 h-4 mr-2" />
          注册执行器
        </button>
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
          <option value="online">在线</option>
          <option value="offline">离线</option>
        </select>
      </div>

      {/* 统计信息 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-white rounded-lg p-4 border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">总执行器</p>
              <p className="text-2xl font-bold text-gray-900">{allExecutors?.length || 0}</p>
            </div>
            <Server className="w-8 h-8 text-gray-400" />
          </div>
        </div>

        <div className="bg-white rounded-lg p-4 border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">健康执行器</p>
              <p className="text-2xl font-bold text-green-600">
                {allExecutors?.filter(e => e.status === 'online' && e.is_healthy).length || 0}
              </p>
            </div>
            <CheckCircle className="w-8 h-8 text-green-400" />
          </div>
        </div>

        <div className="bg-white rounded-lg p-4 border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">异常执行器</p>
              <p className="text-2xl font-bold text-yellow-600">
                {allExecutors?.filter(e => e.status === 'online' && !e.is_healthy).length || 0}
              </p>
            </div>
            <AlertCircle className="w-8 h-8 text-yellow-400" />
          </div>
        </div>

        <div className="bg-white rounded-lg p-4 border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">离线执行器</p>
              <p className="text-2xl font-bold text-red-600">
                {allExecutors?.filter(e => e.status === 'offline').length || 0}
              </p>
            </div>
            <Activity className="w-8 h-8 text-red-400" />
          </div>
        </div>
      </div>

      {/* 执行器列表 */}
      <div className="space-y-4">
        {executors?.map((executor) => (
          <div key={executor.id} className="bg-white rounded-lg border border-gray-200 p-6">
            <div className="flex items-start justify-between mb-4">
              <div className="flex items-center space-x-4">
                <div className="p-3 bg-gray-100 rounded-lg">
                  <Server className="w-6 h-6 text-gray-600" />
                </div>
                <div>
                  <h3 className="text-lg font-semibold text-gray-900">{executor.name}</h3>
                  <p className="text-sm text-gray-500">{executor.instance_id}</p>
                  <p className="text-sm text-gray-600 mt-1">{executor.base_url}</p>
                </div>
              </div>
              <div className="flex items-center space-x-3">
                {getStatusBadge(executor)}
                <button
                  onClick={() => handleEdit(executor)}
                  className="p-2 text-gray-400 hover:text-blue-600 transition-colors"
                  title="编辑执行器"
                >
                  <Edit className="w-4 h-4" />
                </button>
                <button
                  onClick={() => handleDelete(executor)}
                  className="p-2 text-gray-400 hover:text-red-600 transition-colors"
                  title="删除执行器"
                  disabled={deleteMutation.isPending}
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-4">
              <div className="flex items-center space-x-2">
                <Activity className="w-4 h-4 text-gray-400" />
                <span className="text-sm text-gray-600">状态:</span>
                <span className="text-sm font-medium">
                  {executor.status === 'offline' ? '离线' : executor.is_healthy ? '健康' : '异常'}
                </span>
              </div>
              
              <div className="flex items-center space-x-2">
                <CheckCircle className="w-4 h-4 text-gray-400" />
                <span className="text-sm text-gray-600">失败次数:</span>
                <span className="text-sm font-medium">
                  {executor.health_check_failures}
                </span>
              </div>
              
              <div className="flex items-center space-x-2">
                <Calendar className="w-4 h-4 text-gray-400" />
                <span className="text-sm text-gray-600">注册时间:</span>
                <span className="text-sm font-medium">
                  {format(new Date(executor.created_at), 'yyyy-MM-dd HH:mm')}
                </span>
              </div>
              
              <div className="flex items-center space-x-2">
                <Users className="w-4 h-4 text-gray-400" />
                <span className="text-sm text-gray-600">关联任务:</span>
                <span className="text-sm font-medium">
                  {executor.task_executors?.length || 0} 个
                </span>
              </div>
            </div>

            {/* 关联的任务 */}
            {executor.task_executors && executor.task_executors.length > 0 && (
              <div className="border-t pt-4">
                <h4 className="text-sm font-medium text-gray-700 mb-3">关联的任务</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  {executor.task_executors.map((assignment) => (
                    <div key={assignment.id} className="bg-gray-50 rounded-lg p-3">
                      <div className="flex items-center justify-between mb-2">
                        <Link
                          href={`/tasks/${assignment.task_id}`}
                          className="text-sm font-medium text-blue-600 hover:text-blue-800"
                        >
                          {assignment.task?.name || '未知任务'}
                        </Link>
                        <span className={`text-xs px-2 py-1 rounded ${
                          assignment.task?.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                        }`}>
                          {assignment.task?.status === 'active' ? '活跃' : assignment.task?.status || '未知'}
                        </span>
                      </div>
                      <div className="grid grid-cols-2 gap-2 text-xs text-gray-600">
                        <div>权重: {assignment.weight}</div>
                        <div>优先级: {assignment.priority}</div>
                        <div>Cron: {assignment.task?.cron_expression || 'N/A'}</div>
                        <div>模式: {assignment.task?.execution_mode || 'N/A'}</div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {(!executor.task_executors || executor.task_executors.length === 0) && (
              <div className="border-t pt-4">
                <div className="text-center py-4">
                  <Calendar className="w-8 h-8 text-gray-400 mx-auto mb-2" />
                  <p className="text-sm text-gray-500">该执行器未分配任何任务</p>
                  <Link
                    href="/tasks"
                    className="text-sm text-blue-600 hover:text-blue-800 mt-1 inline-block"
                  >
                    前往任务管理进行分配
                  </Link>
                </div>
              </div>
            )}
          </div>
        ))}
      </div>

      {(!executors || executors.length === 0) && (
        <div className="text-center py-12 bg-white rounded-lg">
          <Server className="w-12 h-12 text-gray-400 mx-auto mb-4" />
          <p className="text-gray-500 mb-4">暂无执行器</p>
          <button
            onClick={() => window.location.href = '/executors/register'}
            className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
          >
            <Plus className="w-4 h-4 mr-2" />
            注册第一个执行器
          </button>
        </div>
      )}

      {/* 编辑执行器模态框 */}
      {editingExecutor && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">
              编辑执行器
            </h3>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  执行器名称
                </label>
                <input
                  type="text"
                  value={editForm.name}
                  onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="执行器名称"
                />
              </div>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  基础URL
                </label>
                <input
                  type="url"
                  value={editForm.base_url}
                  onChange={(e) => setEditForm({ ...editForm, base_url: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="http://localhost:9090"
                />
              </div>
              
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  健康检查URL
                </label>
                <input
                  type="url"
                  value={editForm.health_check_url}
                  onChange={(e) => setEditForm({ ...editForm, health_check_url: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="http://localhost:9090/health"
                />
              </div>
            </div>
            
            <div className="flex justify-end space-x-3 mt-6">
              <button
                onClick={handleCancelEdit}
                className="px-4 py-2 text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 transition-colors"
                disabled={updateMutation.isPending}
              >
                取消
              </button>
              <button
                onClick={handleSaveEdit}
                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50"
                disabled={updateMutation.isPending}
              >
                {updateMutation.isPending ? '保存中...' : '保存'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}