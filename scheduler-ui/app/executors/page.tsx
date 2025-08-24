'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Server, Activity, CheckCircle, AlertCircle, Calendar, Users, Settings, Filter, Edit, Trash2, ChevronDown, ChevronUp } from 'lucide-react';
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

// 服务器对象类型定义
interface ExecutorServer {
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

// View层对象类型定义
interface ExecutorView {
  id: string;
  name: string;
  displayName: string;
  displayUrl: string;
  type: 'instance' | 'config-only';
  status: 'online' | 'offline' | 'unhealthy' | 'config-only';
  isHealthy: boolean;
  healthCheckFailures: number;
  createdAt: Date;
  lastHealthCheck?: Date;
  healthCheckUrl: string;
  taskExecutors: TaskExecutor[];
  // 原始服务器对象，用于编辑等操作
  _raw: ExecutorServer;
}

interface ExecutorGroupView {
  name: string;
  displaySummary: string;
  instances: ExecutorView[];
  configOnlyItems: ExecutorView[];
  stats: {
    totalInstances: number;
    onlineCount: number;
    healthyCount: number;
    offlineCount: number;
    configOnlyCount: number;
    totalTasks: number;
  };
  sharedTasks: TaskExecutor[];
}

// 数据转换层
class ExecutorTransformer {
  static toView(server: ExecutorServer): ExecutorView {
    const hasInstanceInfo = server.instance_id || server.base_url;
    const type = hasInstanceInfo ? 'instance' : 'config-only';
    
    let status: ExecutorView['status'];
    if (type === 'config-only') {
      status = 'config-only';
    } else if (server.status === 'online' && server.is_healthy) {
      status = 'online';
    } else if (server.status === 'online' && !server.is_healthy) {
      status = 'unhealthy';
    } else {
      status = 'offline';
    }

    return {
      id: server.id,
      name: server.name,
      displayName: server.instance_id || `${server.name}-config`,
      displayUrl: server.base_url || '暂无服务地址',
      type,
      status,
      isHealthy: server.is_healthy,
      healthCheckFailures: server.health_check_failures,
      createdAt: new Date(server.created_at),
      lastHealthCheck: server.last_health_check ? new Date(server.last_health_check) : undefined,
      healthCheckUrl: server.health_check_url || '暂无',
      taskExecutors: server.task_executors || [],
      _raw: server,
    };
  }

  static toGroupView(name: string, servers: ExecutorServer[]): ExecutorGroupView {
    const views = servers.map(this.toView);
    const instances = views.filter(v => v.type === 'instance');
    const configOnlyItems = views.filter(v => v.type === 'config-only');
    
    const stats = {
      totalInstances: instances.length,
      onlineCount: instances.filter(v => v.status === 'online').length,
      healthyCount: instances.filter(v => v.status === 'online').length,
      offlineCount: instances.filter(v => v.status === 'offline').length,
      configOnlyCount: configOnlyItems.length,
      totalTasks: views.length > 0 ? views[0].taskExecutors.length : 0,
    };

    let displaySummary = '';
    if (stats.totalInstances > 0) {
      displaySummary = `${stats.totalInstances} 个实例`;
    } else {
      displaySummary = '暂无实例';
    }
    
    if (stats.totalTasks > 0) {
      displaySummary += ` • 关联 ${stats.totalTasks} 个任务`;
    }

    return {
      name,
      displaySummary,
      instances,
      configOnlyItems,
      stats,
      sharedTasks: views.length > 0 ? views[0].taskExecutors : [],
    };
  }

  static getStatusBadge(view: ExecutorView) {
    switch (view.status) {
      case 'online':
        return {
          className: 'bg-green-100 text-green-800',
          icon: 'CheckCircle',
          text: '健康'
        };
      case 'unhealthy':
        return {
          className: 'bg-yellow-100 text-yellow-800',
          icon: 'AlertCircle',
          text: '异常'
        };
      case 'offline':
        return {
          className: 'bg-red-100 text-red-800',
          icon: 'AlertCircle',
          text: '离线'
        };
      case 'config-only':
        return {
          className: 'bg-orange-100 text-orange-800',
          icon: 'Settings',
          text: '仅配置'
        };
      default:
        return {
          className: 'bg-gray-100 text-gray-800',
          icon: 'AlertCircle',
          text: '未知'
        };
    }
  }
}

export default function ExecutorsPage() {
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [editingExecutor, setEditingExecutor] = useState<ExecutorServer | null>(null);
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const [editForm, setEditForm] = useState({
    name: '',
    base_url: '',
    health_check_url: '',
  });
  const queryClient = useQueryClient();
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

  // 分组处理函数 - 使用转换器
  const groupExecutorsByName = (executors: ExecutorServer[]): ExecutorGroupView[] => {
    const groups = new Map<string, ExecutorServer[]>();
    
    // 按名称分组
    executors.forEach(executor => {
      if (!groups.has(executor.name)) {
        groups.set(executor.name, []);
      }
      groups.get(executor.name)!.push(executor);
    });

    // 使用转换器转换为View对象
    return Array.from(groups.entries())
      .map(([name, servers]) => ExecutorTransformer.toGroupView(name, servers))
      .sort((a, b) => a.name.localeCompare(b.name));
  };

  // 切换分组展开状态
  const toggleGroup = (groupName: string) => {
    const newExpanded = new Set(expandedGroups);
    if (newExpanded.has(groupName)) {
      newExpanded.delete(groupName);
    } else {
      newExpanded.add(groupName);
    }
    setExpandedGroups(newExpanded);
  };

  // 获取执行器列表（包含关联的任务信息）- 不再传递筛选参数给后端
  const { data: allExecutors, isLoading } = useQuery<ExecutorServer[]>({
    queryKey: ['executors'],
    queryFn: async () => {
      const params = new URLSearchParams();
      params.append('include_tasks', 'true');
      
      const response = await fetch(`${apiUrl}/api/v1/executors?${params}`);
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

  // 在前端本地筛选执行器 - 使用view对象
  const filteredExecutors = allExecutors?.filter(executor => {
    if (!statusFilter) return true;
    
    const view = ExecutorTransformer.toView(executor);
    
    if (statusFilter === 'online') {
      return view.status === 'online';
    } else if (statusFilter === 'offline') {
      return view.status === 'offline' || view.status === 'config-only';
    }
    
    return true;
  });

  // 对筛选后的执行器进行分组
  const executorGroups = filteredExecutors ? groupExecutorsByName(filteredExecutors) : [];

  // 删除执行器
  const deleteMutation = useMutation({
    mutationFn: async (executorId: string) => {
      const response = await fetch(`${apiUrl}/api/v1/executors/${executorId}`, {
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
      const response = await fetch(`${apiUrl}/api/v1/executors/${executorId}`, {
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

  const handleDelete = async (view: ExecutorView) => {
    if (confirm(`确定要删除执行器 \"${view.name}\" 吗？`)) {
      deleteMutation.mutate(view.id);
    }
  };

  const handleEdit = (view: ExecutorView) => {
    setEditingExecutor(view._raw);
    setEditForm({
      name: view._raw.name,
      base_url: view._raw.base_url,
      health_check_url: view._raw.health_check_url,
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

  const renderStatusBadge = (view: ExecutorView) => {
    const badge = ExecutorTransformer.getStatusBadge(view);
    const IconComponent = {
      CheckCircle,
      AlertCircle,
      Settings,
    }[badge.icon] || AlertCircle;

    return (
      <span className={`inline-flex items-center px-2 py-1 text-xs font-medium rounded-full ${badge.className}`}>
        <IconComponent className="w-3 h-3 mr-1" />
        {badge.text}
      </span>
    );
  };

  // 计算全局统计信息 - 使用view对象
  const globalStats = allExecutors ? (() => {
    const allViews = allExecutors.map(ExecutorTransformer.toView);
    const instances = allViews.filter(v => v.type === 'instance');
    
    return {
      total: instances.length,
      healthy: instances.filter(v => v.status === 'online').length,
      unhealthy: instances.filter(v => v.status === 'unhealthy').length,
      offline: instances.filter(v => v.status === 'offline').length,
    };
  })() : { total: 0, healthy: 0, unhealthy: 0, offline: 0 };

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
              <p className="text-2xl font-bold text-gray-900">{globalStats.total}</p>
            </div>
            <Server className="w-8 h-8 text-gray-400" />
          </div>
        </div>

        <div className="bg-white rounded-lg p-4 border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">健康执行器</p>
              <p className="text-2xl font-bold text-green-600">{globalStats.healthy}</p>
            </div>
            <CheckCircle className="w-8 h-8 text-green-400" />
          </div>
        </div>

        <div className="bg-white rounded-lg p-4 border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">异常执行器</p>
              <p className="text-2xl font-bold text-yellow-600">{globalStats.unhealthy}</p>
            </div>
            <AlertCircle className="w-8 h-8 text-yellow-400" />
          </div>
        </div>

        <div className="bg-white rounded-lg p-4 border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">离线执行器</p>
              <p className="text-2xl font-bold text-red-600">{globalStats.offline}</p>
            </div>
            <Activity className="w-8 h-8 text-red-400" />
          </div>
        </div>
      </div>

      {/* 执行器分组列表 */}
      <div className="space-y-4">
        {executorGroups.map((group) => (
          <div key={group.name} className="bg-white rounded-lg border border-gray-200">
            {/* 分组头部 */}
            <div 
              className="p-6 cursor-pointer hover:bg-gray-50 transition-colors"
              onClick={() => toggleGroup(group.name)}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-4">
                  <div className="p-3 bg-gray-100 rounded-lg">
                    <Server className="w-6 h-6 text-gray-600" />
                  </div>
                  <div>
                    <h3 className="text-lg font-semibold text-gray-900">{group.name}</h3>
                    <p className="text-sm text-gray-600">{group.displaySummary}</p>
                  </div>
                </div>
                
                <div className="flex items-center space-x-4">
                  {/* 状态统计 */}
                  <div className="flex items-center space-x-3">
                    {group.stats.healthyCount > 0 && (
                      <span className="inline-flex items-center px-2 py-1 text-xs font-medium bg-green-100 text-green-800 rounded-full">
                        <CheckCircle className="w-3 h-3 mr-1" />
                        {group.stats.healthyCount} 健康
                      </span>
                    )}
                    {group.stats.onlineCount - group.stats.healthyCount > 0 && (
                      <span className="inline-flex items-center px-2 py-1 text-xs font-medium bg-yellow-100 text-yellow-800 rounded-full">
                        <AlertCircle className="w-3 h-3 mr-1" />
                        {group.stats.onlineCount - group.stats.healthyCount} 异常
                      </span>
                    )}
                    {group.stats.offlineCount > 0 && (
                      <span className="inline-flex items-center px-2 py-1 text-xs font-medium bg-red-100 text-red-800 rounded-full">
                        <AlertCircle className="w-3 h-3 mr-1" />
                        {group.stats.offlineCount} 离线
                      </span>
                    )}
                    {group.stats.configOnlyCount > 0 && (
                      <span className="inline-flex items-center px-2 py-1 text-xs font-medium bg-orange-100 text-orange-800 rounded-full">
                        <Settings className="w-3 h-3 mr-1" />
                        {group.stats.configOnlyCount} 待配置
                      </span>
                    )}
                  </div>
                  
                  {/* 展开/折叠按钮 */}
                  <button className="p-2 text-gray-400 hover:text-gray-600 transition-colors">
                    {expandedGroups.has(group.name) ? (
                      <ChevronUp className="w-5 h-5" />
                    ) : (
                      <ChevronDown className="w-5 h-5" />
                    )}
                  </button>
                </div>
              </div>
            </div>

            {/* 展开的内容 */}
            {expandedGroups.has(group.name) && (
              <div className="border-t border-gray-200 bg-gray-50">
                <div className="p-6">
                  {/* 共享任务部分 */}
                  {group.sharedTasks && group.sharedTasks.length > 0 && (
                    <div className="mb-6">
                      <h4 className="text-sm font-medium text-gray-700 mb-3 flex items-center">
                        <Users className="w-4 h-4 mr-2" />
                        关联任务 ({group.sharedTasks.length} 个)
                      </h4>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-4">
                        {group.sharedTasks.map((assignment) => (
                          <div key={assignment.id} className="bg-white rounded-lg border border-gray-200 p-3">
                            <div className="flex items-center justify-between mb-2">
                              <Link
                                href={`/task-detail?id=${assignment.task_id}`}
                                className="text-sm font-medium text-blue-600 hover:text-blue-800 truncate"
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

                  {/* 无任务提示 */}
                  {(!group.sharedTasks || group.sharedTasks.length === 0) && (
                    <div className="mb-6">
                      <div className="text-center py-4 bg-white rounded-lg border border-gray-200">
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

                  {/* 真实实例部分 */}
                  {group.instances.length > 0 && (
                    <>
                      <h4 className="text-sm font-medium text-gray-700 mb-3 flex items-center">
                        <Server className="w-4 h-4 mr-2" />
                        执行器实例 ({group.stats.totalInstances} 个)
                      </h4>
                      <div className="space-y-3 mb-4">
                        {group.instances.map((view) => (
                          <div key={view.id} className="bg-white rounded-lg border border-gray-200 p-4">
                            <div className="flex items-start justify-between mb-3">
                              <div className="flex items-center space-x-3">
                                <div className="p-2 bg-gray-100 rounded-lg">
                                  <Activity className="w-4 h-4 text-gray-600" />
                                </div>
                                <div>
                                  <h5 className="text-sm font-semibold text-gray-900">
                                    {view.displayName}
                                  </h5>
                                  <p className="text-xs text-gray-600 mt-1">
                                    {view.displayUrl}
                                  </p>
                                </div>
                              </div>
                              <div className="flex items-center space-x-3">
                                {renderStatusBadge(view)}
                                <button
                                  onClick={() => handleEdit(view)}
                                  className="p-1 text-gray-400 hover:text-blue-600 transition-colors"
                                  title="编辑执行器"
                                >
                                  <Edit className="w-3 h-3" />
                                </button>
                                <button
                                  onClick={() => handleDelete(view)}
                                  className="p-1 text-gray-400 hover:text-red-600 transition-colors"
                                  title="删除执行器"
                                  disabled={deleteMutation.isPending}
                                >
                                  <Trash2 className="w-3 h-3" />
                                </button>
                              </div>
                            </div>

                            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 text-xs">
                              <div className="flex items-center space-x-2">
                                <CheckCircle className="w-3 h-3 text-gray-400" />
                                <span className="text-gray-600">失败次数:</span>
                                <span className="font-medium">{view.healthCheckFailures}</span>
                              </div>
                              
                              <div className="flex items-center space-x-2">
                                <Calendar className="w-3 h-3 text-gray-400" />
                                <span className="text-gray-600">注册时间:</span>
                                <span className="font-medium">
                                  {format(view.createdAt, 'MM-dd HH:mm')}
                                </span>
                              </div>
                              
                              <div className="flex items-center space-x-2">
                                <Activity className="w-3 h-3 text-gray-400" />
                                <span className="text-gray-600">健康检查URL:</span>
                                <span className="font-medium text-xs truncate" title={view.healthCheckUrl}>
                                  {view.healthCheckUrl === '暂无' ? '暂无' : 
                                    (view._raw.base_url ? 
                                      view.healthCheckUrl.replace(view._raw.base_url, '') : 
                                      view.healthCheckUrl
                                    )
                                  }
                                </span>
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                    </>
                  )}

                  {/* 仅名称配置部分 */}
                  {group.configOnlyItems.length > 0 && (
                    <>
                      <h4 className="text-sm font-medium text-orange-700 mb-3 flex items-center">
                        <Settings className="w-4 h-4 mr-2" />
                        仅名称配置 ({group.stats.configOnlyCount} 个)
                      </h4>
                      <div className="space-y-3">
                        {group.configOnlyItems.map((view) => (
                          <div key={view.id} className="bg-orange-50 rounded-lg border border-orange-200 p-4">
                            <div className="flex items-start justify-between mb-3">
                              <div className="flex items-center space-x-3">
                                <div className="p-2 bg-orange-200 rounded-lg">
                                  <Settings className="w-4 h-4 text-orange-600" />
                                </div>
                                <div>
                                  <h5 className="text-sm font-semibold text-gray-900">
                                    {view.name}
                                  </h5>
                                  <p className="text-xs text-orange-600 mt-1">
                                    需要完善实例信息后才能上线
                                  </p>
                                </div>
                              </div>
                              <div className="flex items-center space-x-3">
                                {renderStatusBadge(view)}
                                <button
                                  onClick={() => handleEdit(view)}
                                  className="p-1 text-orange-400 hover:text-orange-600 transition-colors"
                                  title="完善执行器信息"
                                >
                                  <Edit className="w-3 h-3" />
                                </button>
                                <button
                                  onClick={() => handleDelete(view)}
                                  className="p-1 text-orange-400 hover:text-red-600 transition-colors"
                                  title="删除执行器"
                                  disabled={deleteMutation.isPending}
                                >
                                  <Trash2 className="w-3 h-3" />
                                </button>
                              </div>
                            </div>

                            <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs">
                              <div className="flex items-center space-x-2">
                                <Calendar className="w-3 h-3 text-orange-400" />
                                <span className="text-gray-600">创建时间:</span>
                                <span className="font-medium">
                                  {format(view.createdAt, 'MM-dd HH:mm')}
                                </span>
                              </div>
                              <div className="flex items-center space-x-2">
                                <Settings className="w-3 h-3 text-orange-400" />
                                <span className="text-gray-600">状态:</span>
                                <span className="font-medium text-orange-600">待完善</span>
                              </div>
                            </div>
                          </div>
                        ))}
                      </div>
                    </>
                  )}
                </div>
              </div>
            )}
          </div>
        ))}
      </div>

      {(!filteredExecutors || filteredExecutors.length === 0) && (
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