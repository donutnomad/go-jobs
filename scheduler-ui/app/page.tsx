'use client';

import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import { 
  BarChart3, Calendar, Server, Activity, Users, Clock, 
  TrendingUp, AlertCircle, CheckCircle, Settings, Play
} from 'lucide-react';
import { format } from 'date-fns';

interface Task {
  id: string;
  name: string;
  cron_expression: string;
  execution_mode: string;
  load_balance_strategy: string;
  status: string;
  created_at: string;
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

interface SystemStats {
  total_tasks: number;
  active_tasks: number;
  total_executors: number;
  healthy_executors: number;
  total_executions_today: number;
  success_rate: number;
}

interface ExecutionStats {
  total: number;
  success: number;
  failed: number;
  pending: number;
  running: number;
}

export default function DashboardPage() {
  // 从环境变量获取 API URL
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
  
  // 获取任务列表
  const { data: tasks, isLoading: isTasksLoading } = useQuery<Task[]>({
    queryKey: ['tasks'],
    queryFn: async () => {
      const response = await fetch(`${apiUrl}/api/v1/tasks`);
      if (!response.ok) {
        throw new Error('Failed to fetch tasks');
      }
      const result = await response.json();
      return result.data || result; // 适配不同的响应格式
    },
    refetchInterval: 30000,
  });

  // 获取执行器统计
  const { data: executors } = useQuery<Executor[]>({
    queryKey: ['executors'],
    queryFn: async () => {
      const response = await fetch(`${apiUrl}/api/v1/executors`);
      if (!response.ok) {
        throw new Error('Failed to fetch executors');
      }
      const result = await response.json();
      return result.data || result; // 适配不同的响应格式
    },
    refetchInterval: 30000,
  });

  // 获取调度器状态
  const { data: schedulerStatus } = useQuery({
    queryKey: ['scheduler-status'],
    queryFn: async () => {
      const response = await fetch(`${apiUrl}/api/v1/scheduler/status`);
      if (!response.ok) {
        throw new Error('Failed to fetch scheduler status');
      }
      const result = await response.json();
      return result.data || result; // 适配不同的响应格式
    },
    refetchInterval: 10000,
  });

  // 获取今日执行统计
  const { data: executionStats } = useQuery<ExecutionStats>({
    queryKey: ['execution-stats'],
    queryFn: async () => {
      // 获取今天的开始时间
      const today = new Date();
      today.setHours(0, 0, 0, 0);
      const startTime = today.toISOString();
      
      const response = await fetch(`${apiUrl}/api/v1/executions/stats?start_time=${startTime}`);
      if (!response.ok) {
        console.error('Failed to fetch execution stats');
        return { total: 0, success: 0, failed: 0, pending: 0, running: 0 };
      }
      
      const result = await response.json();
      return result.data || result; // 适配不同的响应格式
    },
    refetchInterval: 30000,
  });

  // 计算系统统计
  const systemStats: SystemStats = {
    total_tasks: tasks?.length || 0,
    active_tasks: tasks?.filter(t => t.status === 'active').length || 0,
    total_executors: executors?.length || 0,
    healthy_executors: executors?.filter(e => e.status === 'online' && e.is_healthy).length || 0,
    total_executions_today: executionStats?.total || 0,
    success_rate: executionStats && executionStats.total > 0 
      ? (executionStats.success / executionStats.total * 100) 
      : 0,
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

  if (isTasksLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">加载中...</div>
      </div>
    );
  }

  return (
    <div>
      {/* 页面头部 */}
      <div className="mb-8">
        <div className="flex items-center space-x-3">
          <BarChart3 className="w-8 h-8 text-blue-600" />
          <div>
            <h1 className="text-2xl font-bold text-gray-900">系统概览</h1>
            <p className="text-sm text-gray-600">任务调度系统整体状态和任务管理</p>
          </div>
        </div>
      </div>

      {/* 系统统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div className="bg-white rounded-lg p-6 border border-gray-200 shadow-sm">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">总任务数</p>
              <p className="text-2xl font-bold text-gray-900">{systemStats.total_tasks}</p>
              <p className="text-sm text-blue-600">
                {systemStats.active_tasks} 个活跃
              </p>
            </div>
            <div className="p-3 bg-blue-100 rounded-lg">
              <Calendar className="w-6 h-6 text-blue-600" />
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg p-6 border border-gray-200 shadow-sm">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">执行器</p>
              <p className="text-2xl font-bold text-gray-900">{systemStats.total_executors}</p>
              <p className="text-sm text-green-600">
                {systemStats.healthy_executors} 个健康
              </p>
            </div>
            <div className="p-3 bg-green-100 rounded-lg">
              <Server className="w-6 h-6 text-green-600" />
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg p-6 border border-gray-200 shadow-sm">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">今日执行</p>
              <p className="text-2xl font-bold text-gray-900">{systemStats.total_executions_today}</p>
              <p className="text-sm text-blue-600">任务执行次数</p>
            </div>
            <div className="p-3 bg-purple-100 rounded-lg">
              <Activity className="w-6 h-6 text-purple-600" />
            </div>
          </div>
        </div>

        <div className="bg-white rounded-lg p-6 border border-gray-200 shadow-sm">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">成功率</p>
              <p className="text-2xl font-bold text-gray-900">
                {systemStats.total_executions_today > 0 
                  ? `${systemStats.success_rate.toFixed(1)}%`
                  : 'N/A'}
              </p>
              <p className="text-sm text-green-600">
                {systemStats.total_executions_today > 0 ? (
                  <>
                    <TrendingUp className="w-3 h-3 inline mr-1" />
                    {executionStats?.success || 0} 成功 / {executionStats?.failed || 0} 失败
                  </>
                ) : (
                  '暂无执行记录'
                )}
              </p>
            </div>
            <div className="p-3 bg-yellow-100 rounded-lg">
              <CheckCircle className="w-6 h-6 text-yellow-600" />
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* 任务列表 */}
        <div className="bg-white rounded-lg border border-gray-200 shadow-sm">
          <div className="px-6 py-4 border-b border-gray-200">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900">任务概览</h2>
              <Link 
                href="/tasks" 
                className="text-sm text-blue-600 hover:text-blue-800"
              >
                查看全部
              </Link>
            </div>
          </div>

          <div className="p-6">
            {(!tasks || tasks.length === 0) ? (
              <div className="text-center py-8">
                <Calendar className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                <p className="text-gray-500 mb-4">还没有创建任何任务</p>
                <Link 
                  href="/tasks"
                  className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                >
                  创建任务
                </Link>
              </div>
            ) : (
              <div className="space-y-4">
                {tasks.slice(0, 5).map((task) => (
                  <div key={task.id} className="flex items-center justify-between p-4 bg-gray-50 rounded-lg">
                    <div className="flex-1">
                      <div className="flex items-center space-x-2 mb-1">
                        <h3 className="font-medium text-gray-900">{task.name}</h3>
                        <span className={`px-2 py-1 rounded text-xs ${
                          task.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                        }`}>
                          {task.status === 'active' ? '活跃' : task.status}
                        </span>
                      </div>
                      <div className="flex items-center space-x-4 text-sm text-gray-600">
                        <div className="flex items-center space-x-1">
                          <Clock className="w-3 h-3" />
                          <span>{task.cron_expression}</span>
                        </div>
                        <div className="flex items-center space-x-1">
                          <Activity className="w-3 h-3" />
                          <span>{getExecutionModeText(task.execution_mode)}</span>
                        </div>
                        <div className="flex items-center space-x-1">
                          <Users className="w-3 h-3" />
                          <span>{task.task_executors?.length || 0} 执行器</span>
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      <button className="p-1 text-green-600 hover:text-green-800" title="手动触发">
                        <Play className="w-4 h-4" />
                      </button>
                      <Link 
                        href={`/task-detail?id=${task.id}`}
                        className="text-blue-600 hover:text-blue-800"
                      >
                        <Settings className="w-4 h-4" />
                      </Link>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* 系统状态 */}
        <div className="bg-white rounded-lg border border-gray-200 shadow-sm">
          <div className="px-6 py-4 border-b border-gray-200">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900">系统状态</h2>
              <Link 
                href="/scheduler" 
                className="text-sm text-blue-600 hover:text-blue-800"
              >
                查看详情
              </Link>
            </div>
          </div>

          <div className="p-6">
            {/* 调度器状态 */}
            <div className="mb-6">
              <h3 className="text-sm font-medium text-gray-700 mb-3">调度器实例</h3>
              {schedulerStatus?.instances?.length > 0 ? (
                <div className="space-y-2">
                  {schedulerStatus.instances.map((instance: any) => (
                    <div key={instance.id} className="flex items-center justify-between p-3 bg-gray-50 rounded">
                      <div className="flex items-center space-x-3">
                        <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse" />
                        <div>
                          <div className="font-medium text-gray-900">{instance.hostname}</div>
                          <div className="text-xs text-gray-500">{instance.instance_id}</div>
                        </div>
                      </div>
                      {instance.is_leader && (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                          Leader
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="flex items-center space-x-2 text-sm text-red-600">
                  <AlertCircle className="w-4 h-4" />
                  <span>无调度器实例运行</span>
                </div>
              )}
            </div>

            {/* 执行器状态统计 */}
            <div className="mb-6">
              <h3 className="text-sm font-medium text-gray-700 mb-3">执行器状态</h3>
              <div className="grid grid-cols-2 gap-4">
                <div className="text-center p-3 bg-green-50 rounded">
                  <div className="text-lg font-bold text-green-600">{systemStats.healthy_executors}</div>
                  <div className="text-sm text-gray-600">健康</div>
                </div>
                <div className="text-center p-3 bg-red-50 rounded">
                  <div className="text-lg font-bold text-red-600">
                    {systemStats.total_executors - systemStats.healthy_executors}
                  </div>
                  <div className="text-sm text-gray-600">异常</div>
                </div>
              </div>
            </div>

            {/* 快速操作 */}
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-3">快速操作</h3>
              <div className="space-y-2">
                <Link 
                  href="/tasks"
                  className="block w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 rounded"
                >
                  <Calendar className="w-4 h-4 inline mr-2" />
                  管理任务
                </Link>
                <Link 
                  href="/executors"
                  className="block w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 rounded"
                >
                  <Server className="w-4 h-4 inline mr-2" />
                  管理执行器
                </Link>
                <Link 
                  href="/executions"
                  className="block w-full text-left px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 rounded"
                >
                  <Activity className="w-4 h-4 inline mr-2" />
                  执行历史
                </Link>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}