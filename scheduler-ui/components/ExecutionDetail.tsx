'use client';

import { useState } from 'react';
import { TaskExecution, executionApi } from '@/lib/api';
import { format } from 'date-fns';
import { Clock, CheckCircle, XCircle, AlertCircle, Loader, Ban, Server, Calendar, StopCircle } from 'lucide-react';

interface ExecutionDetailProps {
  execution: TaskExecution;
  onStatusChange?: () => void;
}

export default function ExecutionDetail({ execution, onStatusChange }: ExecutionDetailProps) {
  const [isStopping, setIsStopping] = useState(false);

  const handleStop = async () => {
    if (!confirm('确定要停止这个任务吗？')) {
      return;
    }

    setIsStopping(true);
    try {
      await executionApi.stop(execution.id);
      alert('停止请求已发送');
      if (onStatusChange) {
        onStatusChange();
      }
    } catch (error) {
      console.error('Failed to stop execution:', error);
      alert('停止任务失败');
    } finally {
      setIsStopping(false);
    }
  };
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
    return <Icon className={`w-5 h-5 ${config.color}`} />;
  };

  const getStatusLabel = (status: string) => {
    const labels: Record<string, string> = {
      pending: '等待中',
      running: '执行中',
      success: '成功',
      failed: '失败',
      timeout: '超时',
      cancelled: '已取消',
    };
    return labels[status] || status;
  };

  const getDuration = () => {
    if (!execution.start_time) return '未开始';
    if (!execution.end_time) return '执行中...';
    
    const start = new Date(execution.start_time);
    const end = new Date(execution.end_time);
    const diffInMs = end.getTime() - start.getTime();
    const diffInSeconds = Math.floor(diffInMs / 1000);
    
    if (diffInSeconds < 60) {
      return `${diffInSeconds} 秒`;
    } else if (diffInSeconds < 3600) {
      return `${Math.floor(diffInSeconds / 60)} 分钟 ${diffInSeconds % 60} 秒`;
    }
    return `${Math.floor(diffInSeconds / 3600)} 小时 ${Math.floor((diffInSeconds % 3600) / 60)} 分钟`;
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          {getStatusIcon(execution.status)}
          <div>
            <h3 className="text-lg font-semibold text-gray-900">
              {getStatusLabel(execution.status)}
            </h3>
            <p className="text-sm text-gray-500">执行 ID: {execution.id}</p>
          </div>
        </div>
        
        {/* 停止按钮 - 只在运行状态显示 */}
        {execution.status === 'running' && (
          <button
            onClick={handleStop}
            disabled={isStopping}
            className="inline-flex items-center px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isStopping ? (
              <>
                <Loader className="w-4 h-4 mr-2 animate-spin" />
                停止中...
              </>
            ) : (
              <>
                <StopCircle className="w-4 h-4 mr-2" />
                停止任务
              </>
            )}
          </button>
        )}
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-4">
          <div>
            <div className="flex items-center space-x-2 text-sm text-gray-600 mb-1">
              <Calendar className="w-4 h-4" />
              <span>任务信息</span>
            </div>
            <div className="bg-gray-50 rounded-lg p-3">
              <p className="text-sm font-medium text-gray-900">
                {execution.task?.name || 'Unknown Task'}
              </p>
              <p className="text-xs text-gray-500 mt-1">
                ID: {execution.task_id}
              </p>
            </div>
          </div>

          <div>
            <div className="flex items-center space-x-2 text-sm text-gray-600 mb-1">
              <Server className="w-4 h-4" />
              <span>执行器信息</span>
            </div>
            <div className="bg-gray-50 rounded-lg p-3">
              <p className="text-sm font-medium text-gray-900">
                {execution.executor?.name || 'Unknown Executor'}
              </p>
              <p className="text-xs text-gray-500 mt-1">
                实例: {execution.executor?.instance_id || execution.executor_id}
              </p>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <div>
            <div className="flex items-center space-x-2 text-sm text-gray-600 mb-1">
              <Clock className="w-4 h-4" />
              <span>时间信息</span>
            </div>
            <div className="bg-gray-50 rounded-lg p-3 space-y-2">
              <div>
                <p className="text-xs text-gray-600">计划时间</p>
                <p className="text-sm font-medium text-gray-900">
                  {format(new Date(execution.scheduled_time), 'yyyy-MM-dd HH:mm:ss')}
                </p>
              </div>
              {execution.start_time && (
                <div>
                  <p className="text-xs text-gray-600">开始时间</p>
                  <p className="text-sm font-medium text-gray-900">
                    {format(new Date(execution.start_time), 'yyyy-MM-dd HH:mm:ss')}
                  </p>
                </div>
              )}
              {execution.end_time && (
                <div>
                  <p className="text-xs text-gray-600">结束时间</p>
                  <p className="text-sm font-medium text-gray-900">
                    {format(new Date(execution.end_time), 'yyyy-MM-dd HH:mm:ss')}
                  </p>
                </div>
              )}
              <div>
                <p className="text-xs text-gray-600">执行耗时</p>
                <p className="text-sm font-medium text-gray-900">{getDuration()}</p>
              </div>
            </div>
          </div>

          <div>
            <p className="text-sm text-gray-600 mb-1">重试次数</p>
            <div className="bg-gray-50 rounded-lg p-3">
              <p className="text-lg font-semibold text-gray-900">{execution.retry_count}</p>
            </div>
          </div>
        </div>
      </div>

      {execution.result && (
        <div>
          <p className="text-sm text-gray-600 mb-2">执行结果</p>
          <div className="bg-green-50 border border-green-200 rounded-lg p-4">
            <pre className="text-sm text-green-900 whitespace-pre-wrap font-mono">
              {typeof execution.result === 'string' 
                ? execution.result 
                : JSON.stringify(execution.result, null, 2)}
            </pre>
          </div>
        </div>
      )}

      {execution.error && (
        <div>
          <p className="text-sm text-gray-600 mb-2">错误信息</p>
          <div className="bg-red-50 border border-red-200 rounded-lg p-4">
            <pre className="text-sm text-red-900 whitespace-pre-wrap font-mono">
              {typeof execution.error === 'string' 
                ? execution.error 
                : JSON.stringify(execution.error, null, 2)}
            </pre>
          </div>
        </div>
      )}

      {execution.logs && (execution.status === 'failed' || execution.status === 'timeout') && (
        <div>
          <p className="text-sm text-gray-600 mb-2">错误日志</p>
          <div className="bg-red-50 border border-red-200 rounded-lg p-4">
            <pre className="text-sm text-red-900 whitespace-pre-wrap font-mono max-h-60 overflow-y-auto">
              {execution.logs}
            </pre>
          </div>
        </div>
      )}

      {execution.logs && execution.status !== 'failed' && execution.status !== 'timeout' && (
        <div>
          <p className="text-sm text-gray-600 mb-2">执行日志</p>
          <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
            <pre className="text-sm text-gray-900 whitespace-pre-wrap font-mono max-h-60 overflow-y-auto">
              {execution.logs}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}