'use client';

import { useQuery } from '@tanstack/react-query';
import { schedulerApi, SchedulerInstance } from '@/lib/api';
import { Server, Activity, CheckCircle, XCircle, Clock, RefreshCw } from 'lucide-react';
import { format } from 'date-fns';

export default function SchedulerPage() {
  const { data: schedulerStatus, isLoading: isStatusLoading } = useQuery({
    queryKey: ['scheduler-status'],
    queryFn: () => schedulerApi.status().then(res => res.data),
    refetchInterval: 5000,
  });

  const { data: healthStatus, isLoading: isHealthLoading } = useQuery({
    queryKey: ['health'],
    queryFn: () => schedulerApi.health().then(res => res.data),
    refetchInterval: 5000,
  });

  const isHealthy = healthStatus?.status === 'healthy';

  const getLeaderBadge = (isLeader: boolean) => {
    if (isLeader) {
      return (
        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
          Leader
        </span>
      );
    }
    return (
      <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
        Follower
      </span>
    );
  };

  if (isStatusLoading || isHealthLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">加载中...</div>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">调度器状态</h1>
        <p className="mt-1 text-sm text-gray-600">监控调度器实例和系统健康状态</p>
      </div>

      {/* 系统健康状态卡片 */}
      <div className="mb-6">
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <div className={`p-3 rounded-lg ${isHealthy ? 'bg-green-100' : 'bg-red-100'}`}>
                <Activity className={`w-6 h-6 ${isHealthy ? 'text-green-600' : 'text-red-600'}`} />
              </div>
              <div>
                <h2 className="text-lg font-semibold text-gray-900">系统健康状态</h2>
                <div className="flex items-center space-x-2 mt-1">
                  {isHealthy ? (
                    <>
                      <CheckCircle className="w-4 h-4 text-green-500" />
                      <span className="text-sm text-green-600">系统运行正常</span>
                    </>
                  ) : (
                    <>
                      <XCircle className="w-4 h-4 text-red-500" />
                      <span className="text-sm text-red-600">系统异常</span>
                    </>
                  )}
                </div>
              </div>
            </div>
            <div className="text-right">
              <p className="text-sm text-gray-600">最后检查时间</p>
              <p className="text-sm font-medium text-gray-900">
                {healthStatus?.time ? format(new Date(healthStatus.time), 'HH:mm:ss') : '-'}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* 调度器实例列表 */}
      <div className="mb-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">调度器实例</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {schedulerStatus?.instances?.map((instance: SchedulerInstance) => {
            return (
              <div
                key={instance.id}
                className="bg-white rounded-lg shadow-sm border border-gray-200 p-6"
              >
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center space-x-3">
                    <div className="p-2 bg-gray-100 rounded-lg">
                      <Server className="w-5 h-5 text-gray-600" />
                    </div>
                    <div>
                      <h3 className="text-md font-semibold text-gray-900">
                        {instance.hostname}
                      </h3>
                      <p className="text-xs text-gray-500">{instance.instance_id}</p>
                    </div>
                  </div>
                  {getLeaderBadge(instance.is_leader)}
                </div>

                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-600">创建时间</span>
                    <span className="text-sm font-medium text-gray-900">
                      {format(new Date(instance.created_at), 'yyyy-MM-dd HH:mm')}
                    </span>
                  </div>

                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-600">状态</span>
                    <div className="flex items-center space-x-1">
                      <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                      <span className="text-sm font-medium text-green-600">在线</span>
                    </div>
                  </div>
                </div>

                {instance.is_leader && (
                  <div className="mt-4 pt-4 border-t border-gray-200">
                    <div className="flex items-center space-x-2 text-blue-600">
                      <RefreshCw className="w-4 h-4" />
                      <span className="text-sm font-medium">正在处理任务调度</span>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
        
        {(!schedulerStatus?.instances || schedulerStatus.instances.length === 0) && (
          <div className="text-center py-12 bg-white rounded-lg">
            <Server className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <p className="text-gray-500">暂无调度器实例</p>
          </div>
        )}
      </div>

      {/* 统计信息 */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">总实例数</p>
              <p className="text-2xl font-bold text-gray-900 mt-1">
                {schedulerStatus?.instances?.length || 0}
              </p>
            </div>
            <Server className="w-8 h-8 text-gray-400" />
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">活跃实例</p>
              <p className="text-2xl font-bold text-green-600 mt-1">
                {schedulerStatus?.instances?.length || 0}
              </p>
            </div>
            <Activity className="w-8 h-8 text-green-400" />
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600">Leader 节点</p>
              <p className="text-2xl font-bold text-blue-600 mt-1">
                {schedulerStatus?.instances?.filter((i: SchedulerInstance) => i.is_leader).length || 0}
              </p>
            </div>
            <RefreshCw className="w-8 h-8 text-blue-400" />
          </div>
        </div>
      </div>
    </div>
  );
}