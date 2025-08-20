'use client';

import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { executionApi, TaskExecution } from '@/lib/api';
import { Clock, CheckCircle, XCircle, AlertCircle, Loader, Ban, Eye, Layers, List, ChevronDown, ChevronUp } from 'lucide-react';
import { format } from 'date-fns';
import Modal from '@/components/Modal';
import ExecutionDetail from '@/components/ExecutionDetail';
import Pagination from '@/components/Pagination';

export default function ExecutionsPage() {
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [taskIdFilter, setTaskIdFilter] = useState<string>('');
  const [selectedExecution, setSelectedExecution] = useState<TaskExecution | null>(null);
  const [groupByTask, setGroupByTask] = useState<boolean>(false);
  const [expandedTasks, setExpandedTasks] = useState<Set<string>>(new Set());
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const { data: response, isLoading, error, refetch } = useQuery({
    queryKey: ['executions', statusFilter, taskIdFilter, currentPage, pageSize],
    queryFn: () => executionApi.list({
      status: statusFilter || undefined,
      task_id: taskIdFilter || undefined,
      page: currentPage,
      page_size: pageSize,
    }).then(res => res.data), // res.data 是 ApiResponse<PaginatedResponse<TaskExecution>>
    refetchInterval: 10000, // 每10秒刷新一次
    refetchOnWindowFocus: true,
    refetchOnMount: true,
    staleTime: 0,
  });

  const executions = response?.data || [];
  const total = response?.total || 0;

  const getStatusIcon = (status: string) => {
    const iconConfig = {
      pending: { icon: Clock, color: 'text-gray-500' },
      running: { icon: Loader, color: 'text-blue-500 animate-spin' },
      success: { icon: CheckCircle, color: 'text-green-500' },
      failed: { icon: XCircle, color: 'text-red-500' },
      timeout: { icon: AlertCircle, color: 'text-orange-500' },
      cancelled: { icon: Ban, color: 'text-gray-500' },
    };
    const config = iconConfig[status as keyof typeof iconConfig] || iconConfig.pending;
    const Icon = config.icon;
    return <Icon className={`w-4 h-4 ${config.color}`} />;
  };

  const getStatusBadge = (status: string) => {
    const statusConfig = {
      pending: { color: 'bg-gray-100 text-gray-800', label: '等待中' },
      running: { color: 'bg-blue-100 text-blue-800', label: '执行中' },
      success: { color: 'bg-green-100 text-green-800', label: '成功' },
      failed: { color: 'bg-red-100 text-red-800', label: '失败' },
      timeout: { color: 'bg-orange-100 text-orange-800', label: '超时' },
      cancelled: { color: 'bg-gray-100 text-gray-800', label: '已取消' },
    };
    const config = statusConfig[status as keyof typeof statusConfig] || statusConfig.pending;
    return (
      <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${config.color}`}>
        {config.label}
      </span>
    );
  };

  const getDuration = (startTime?: string, endTime?: string) => {
    if (!startTime) return '-';
    if (!endTime) return '执行中...';
    
    const start = new Date(startTime);
    const end = new Date(endTime);
    const diffInMs = end.getTime() - start.getTime();
    const diffInSeconds = Math.floor(diffInMs / 1000);
    
    if (diffInSeconds < 60) {
      return `${diffInSeconds} 秒`;
    } else if (diffInSeconds < 3600) {
      return `${Math.floor(diffInSeconds / 60)} 分钟 ${diffInSeconds % 60} 秒`;
    }
    return `${Math.floor(diffInSeconds / 3600)} 小时 ${Math.floor((diffInSeconds % 3600) / 60)} 分钟`;
  };

  // 按任务分组执行记录
  const groupedExecutions = useMemo(() => {
    if (!executions || !groupByTask) return null;

    const groups: Record<string, TaskExecution[]> = {};
    executions.forEach((execution) => {
      const taskId = execution.task_id;
      if (!groups[taskId]) {
        groups[taskId] = [];
      }
      groups[taskId].push(execution);
    });

    // 按任务名称排序分组，并对每组内的执行记录按时间倒序排列
    return Object.entries(groups).sort(([, aExecutions], [, bExecutions]) => {
      const aTaskName = aExecutions[0]?.task?.name || 'Unknown';
      const bTaskName = bExecutions[0]?.task?.name || 'Unknown';
      return aTaskName.localeCompare(bTaskName);
    }).map(([taskId, taskExecutions]) => [
      taskId, 
      taskExecutions.sort((a, b) => 
        new Date(b.scheduled_time).getTime() - new Date(a.scheduled_time).getTime()
      )
    ] as [string, TaskExecution[]]);
  }, [executions, groupByTask]);

  // 处理分页变化
  const handlePageChange = (page: number) => {
    setCurrentPage(page);
  };

  const handlePageSizeChange = (size: number) => {
    setPageSize(size);
    setCurrentPage(1); // 重置到第一页
  };

  // 处理筛选器变化
  const handleStatusFilterChange = (status: string) => {
    setStatusFilter(status);
    setCurrentPage(1); // 重置到第一页
  };

  const handleTaskIdFilterChange = (taskId: string) => {
    setTaskIdFilter(taskId);
    setCurrentPage(1); // 重置到第一页
  };

  // 切换任务组展开/折叠状态
  const toggleTaskExpansion = (taskId: string) => {
    const newExpanded = new Set(expandedTasks);
    if (newExpanded.has(taskId)) {
      newExpanded.delete(taskId);
    } else {
      newExpanded.add(taskId);
    }
    setExpandedTasks(newExpanded);
  };

  const renderExecutionRow = (execution: TaskExecution, showTaskInfo: boolean = true) => (
    <tr key={execution.id} className="hover:bg-gray-50">
      {showTaskInfo && (
        <td className="px-6 py-4 whitespace-nowrap">
          <div className="text-sm font-medium text-gray-900">
            {execution.task?.name || 'Unknown'}
          </div>
          <div className="text-sm text-gray-500">
            ID: {execution.task_id.slice(0, 8)}...
          </div>
        </td>
      )}
      <td className="px-6 py-4 whitespace-nowrap">
        <div className="text-sm text-gray-900">
          {execution.executor?.name || 'N/A'}
        </div>
        <div className="text-sm text-gray-500">
          {execution.executor?.instance_id || (execution.executor_id ? execution.executor_id.slice(0, 8) + '...' : 'N/A')}
        </div>
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
        {format(new Date(execution.scheduled_time), 'yyyy-MM-dd HH:mm:ss')}
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
        {execution.start_time 
          ? format(new Date(execution.start_time), 'HH:mm:ss')
          : '-'}
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
        {getDuration(execution.start_time, execution.end_time)}
      </td>
      <td className="px-6 py-4 whitespace-nowrap">
        <div className="flex items-center space-x-2">
          {getStatusIcon(execution.status)}
          {getStatusBadge(execution.status)}
        </div>
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 text-center">
        {execution.retry_count}
      </td>
      <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
        <button
          onClick={() => setSelectedExecution(execution)}
          className="text-indigo-600 hover:text-indigo-900"
          title="查看详情"
        >
          <Eye className="w-4 h-4" />
        </button>
      </td>
    </tr>
  );

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">加载中...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-red-500">加载失败</div>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">执行历史</h1>
          <p className="mt-1 text-sm text-gray-600">查看任务执行的历史记录和详情</p>
        </div>
      </div>

      <div className="mb-4 flex items-center space-x-4">
        <select
          value={statusFilter}
          onChange={(e) => handleStatusFilterChange(e.target.value)}
          className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="">全部状态</option>
          <option value="pending">等待中</option>
          <option value="running">执行中</option>
          <option value="success">成功</option>
          <option value="failed">失败</option>
          <option value="timeout">超时</option>
          <option value="cancelled">已取消</option>
        </select>
        
        <input
          type="text"
          value={taskIdFilter}
          onChange={(e) => handleTaskIdFilterChange(e.target.value)}
          placeholder="任务 ID 过滤"
          className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        />

        <div className="flex items-center space-x-2 ml-auto">
          <button
            onClick={() => setGroupByTask(!groupByTask)}
            className={`flex items-center space-x-2 px-3 py-2 rounded-md text-sm font-medium border transition-colors ${
              groupByTask
                ? 'bg-blue-100 text-blue-800 border-blue-300'
                : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
            }`}
            title={groupByTask ? '切换到列表视图' : '切换到分组视图'}
          >
            {groupByTask ? <List className="w-4 h-4" /> : <Layers className="w-4 h-4" />}
            <span>{groupByTask ? '列表视图' : '分组视图'}</span>
          </button>
        </div>
      </div>

      {/* 分组视图提示 */}
      {groupByTask && (
        <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
          <p className="text-sm text-yellow-800">
            <AlertCircle className="w-4 h-4 inline mr-1" />
            分组视图显示当前页的数据分组。如需查看更多数据，请切换到列表视图并使用分页功能。
          </p>
        </div>
      )}

      <div className="bg-white shadow-sm rounded-lg overflow-hidden">
        {groupByTask && groupedExecutions ? (
          // 分组视图
          <div className="space-y-6">
            {groupedExecutions.map(([taskId, taskExecutions]) => {
              const isExpanded = expandedTasks.has(taskId);
              const visibleExecutions = isExpanded ? taskExecutions : taskExecutions.slice(0, 2);
              const hasMore = taskExecutions.length > 2;
              
              return (
                <div key={taskId} className="border rounded-lg">
                  <div className="bg-gray-50 px-6 py-4 border-b">
                    <div className="flex items-center justify-between">
                      <div className="flex-1">
                        <h3 className="text-lg font-semibold text-gray-900">
                          {taskExecutions[0]?.task?.name || 'Unknown Task'}
                        </h3>
                        <p className="text-sm text-gray-500">
                          任务ID: {taskId.slice(0, 8)}... | 执行次数: {taskExecutions.length}
                        </p>
                      </div>
                      <div className="flex items-center space-x-4">
                        <div className="flex items-center space-x-2">
                          {/* 显示该任务的状态统计 */}
                          {['success', 'failed', 'running', 'pending'].map(status => {
                            const count = taskExecutions.filter(exec => exec.status === status).length;
                            if (count === 0) return null;
                            return (
                              <div key={status} className="flex items-center space-x-1">
                                {getStatusIcon(status)}
                                <span className="text-sm text-gray-600">{count}</span>
                              </div>
                            );
                          })}
                        </div>
                        {hasMore && (
                          <button
                            onClick={() => toggleTaskExpansion(taskId)}
                            className="flex items-center space-x-1 px-2 py-1 text-sm text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded transition-colors"
                            title={isExpanded ? '折叠执行记录' : `展开查看全部${taskExecutions.length}条记录`}
                          >
                            {isExpanded ? (
                              <>
                                <ChevronUp className="w-4 h-4" />
                                <span>折叠</span>
                              </>
                            ) : (
                              <>
                                <ChevronDown className="w-4 h-4" />
                                <span>展开 ({taskExecutions.length - 2})</span>
                              </>
                            )}
                          </button>
                        )}
                      </div>
                    </div>
                  </div>
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          执行器
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          计划时间
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          开始时间
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          耗时
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          状态
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          重试次数
                        </th>
                        <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                          操作
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                      {visibleExecutions.map((execution) => renderExecutionRow(execution, false))}
                    </tbody>
                  </table>
                  {!isExpanded && hasMore && (
                    <div className="bg-gray-50 px-6 py-2 border-t">
                      <button
                        onClick={() => toggleTaskExpansion(taskId)}
                        className="text-sm text-blue-600 hover:text-blue-800 font-medium"
                      >
                        查看更多 {taskExecutions.length - 2} 条记录...
                      </button>
                    </div>
                  )}
                </div>
              );
            })}
            {groupedExecutions.length === 0 && (
              <div className="text-center py-12">
                <p className="text-gray-500">暂无执行记录</p>
              </div>
            )}
          </div>
        ) : (
          // 列表视图
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  任务
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  执行器
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  计划时间
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  开始时间
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  耗时
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  状态
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  重试次数
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {executions?.map((execution) => renderExecutionRow(execution, true))}
            </tbody>
          </table>
        )}
        {!groupByTask && executions?.length === 0 && (
          <div className="text-center py-12">
            <p className="text-gray-500">暂无执行记录</p>
          </div>
        )}
        
        {/* 分页组件 */}
        {!groupByTask && total > 0 && (
          <Pagination
            currentPage={currentPage}
            totalPages={Math.ceil(total / pageSize)}
            pageSize={pageSize}
            total={total}
            onPageChange={handlePageChange}
            onPageSizeChange={handlePageSizeChange}
          />
        )}
      </div>

      <Modal
        isOpen={!!selectedExecution}
        onClose={() => setSelectedExecution(null)}
        title="执行详情"
      >
        {selectedExecution && (
          <ExecutionDetail 
            execution={selectedExecution} 
            onStatusChange={() => {
              refetch();
              setSelectedExecution(null);
            }}
          />
        )}
      </Modal>
    </div>
  );
}