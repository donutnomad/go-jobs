'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { ArrowLeft, Save, Calendar, Server, Settings } from 'lucide-react';
import Link from 'next/link';

interface CreateTaskRequest {
  name: string;
  cron_expression: string;
  parameters?: any;
  execution_mode: string;
  load_balance_strategy: string;
  max_retry: number;
  timeout_seconds: number;
}

export default function CreateTaskPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [formData, setFormData] = useState<CreateTaskRequest>({
    name: '',
    cron_expression: '',
    parameters: null,
    execution_mode: 'parallel',
    load_balance_strategy: 'round_robin',
    max_retry: 3,
    timeout_seconds: 300,
  });
  const [parametersJson, setParametersJson] = useState('{}');

  const createTaskMutation = useMutation({
    mutationFn: async (data: CreateTaskRequest) => {
      const response = await fetch('/api/v1/tasks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      router.push('/tasks');
    },
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    try {
      let parameters = null;
      if (parametersJson.trim()) {
        parameters = JSON.parse(parametersJson);
      }

      await createTaskMutation.mutateAsync({
        ...formData,
        parameters,
      });
    } catch (error) {
      if (error instanceof SyntaxError) {
        alert('参数 JSON 格式错误，请检查');
        return;
      }
      alert('创建任务失败: ' + (error as Error).message);
    }
  };

  const cronPresets = [
    { label: '每分钟', value: '0 * * * * *' },
    { label: '每5分钟', value: '0 */5 * * * *' },
    { label: '每10分钟', value: '0 */10 * * * *' },
    { label: '每小时', value: '0 0 * * * *' },
    { label: '每天 0:00', value: '0 0 0 * * *' },
    { label: '每天 6:00', value: '0 0 6 * * *' },
    { label: '工作日 9:00', value: '0 0 9 * * 1-5' },
  ];

  return (
    <div className="max-w-4xl mx-auto">
      {/* 面包屑导航 */}
      <div className="mb-6 flex items-center space-x-2 text-sm">
        <Link href="/tasks" className="text-blue-600 hover:text-blue-800 flex items-center space-x-1">
          <ArrowLeft className="w-4 h-4" />
          <span>任务管理</span>
        </Link>
        <span className="text-gray-500">/</span>
        <span className="text-gray-900">创建任务</span>
      </div>

      {/* 页面头部 */}
      <div className="mb-8">
        <div className="flex items-center space-x-3">
          <Calendar className="w-8 h-8 text-blue-600" />
          <div>
            <h1 className="text-2xl font-bold text-gray-900">创建新任务</h1>
            <p className="text-sm text-gray-600">定义一个新的调度任务</p>
          </div>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* 基本信息 */}
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">基本信息</h2>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                任务名称 *
              </label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="输入任务名称"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                执行模式 *
              </label>
              <select
                value={formData.execution_mode}
                onChange={(e) => setFormData({ ...formData, execution_mode: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                required
              >
                <option value="parallel">并行执行</option>
                <option value="sequential">串行执行</option>
                <option value="skip">跳过执行</option>
              </select>
            </div>
          </div>
        </div>

        {/* 调度配置 */}
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">调度配置</h2>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Cron 表达式 *
            </label>
            <div className="mb-3">
              <input
                type="text"
                value={formData.cron_expression}
                onChange={(e) => setFormData({ ...formData, cron_expression: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="例如: 0 0 * * * *"
                required
              />
              <p className="text-xs text-gray-500 mt-1">
                格式: 秒 分 时 日 月 周，支持标准 cron 语法
              </p>
            </div>
            
            <div>
              <p className="text-sm font-medium text-gray-700 mb-2">常用预设：</p>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                {cronPresets.map((preset) => (
                  <button
                    key={preset.value}
                    type="button"
                    onClick={() => setFormData({ ...formData, cron_expression: preset.value })}
                    className="px-3 py-2 text-sm bg-gray-100 text-gray-700 rounded hover:bg-gray-200 transition-colors"
                  >
                    {preset.label}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* 负载均衡和执行配置 */}
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">执行配置</h2>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                负载均衡策略 *
              </label>
              <select
                value={formData.load_balance_strategy}
                onChange={(e) => setFormData({ ...formData, load_balance_strategy: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                required
              >
                <option value="round_robin">轮询</option>
                <option value="weighted_round_robin">加权轮询</option>
                <option value="random">随机</option>
                <option value="sticky">粘性</option>
                <option value="least_loaded">最少负载</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                最大重试次数
              </label>
              <input
                type="number"
                value={formData.max_retry}
                onChange={(e) => setFormData({ ...formData, max_retry: parseInt(e.target.value) })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                min="0"
                max="10"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                超时时间（秒）
              </label>
              <input
                type="number"
                value={formData.timeout_seconds}
                onChange={(e) => setFormData({ ...formData, timeout_seconds: parseInt(e.target.value) })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                min="1"
                max="3600"
              />
            </div>
          </div>
        </div>

        {/* 任务参数 */}
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">任务参数</h2>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              参数（JSON 格式）
            </label>
            <textarea
              value={parametersJson}
              onChange={(e) => setParametersJson(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={4}
              placeholder="请输入有效的 JSON 格式参数，例如: {&quot;key&quot;: &quot;value&quot;}"
            />
            <p className="text-xs text-gray-500 mt-1">
              可选，如果任务需要参数，请以 JSON 格式输入
            </p>
          </div>
        </div>

        {/* 提交按钮 */}
        <div className="flex items-center justify-end space-x-4 bg-white rounded-lg border border-gray-200 p-6">
          <Link
            href="/tasks"
            className="px-6 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
          >
            取消
          </Link>
          <button
            type="submit"
            disabled={createTaskMutation.isPending}
            className="inline-flex items-center px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
          >
            <Save className="w-4 h-4 mr-2" />
            {createTaskMutation.isPending ? '创建中...' : '创建任务'}
          </button>
        </div>
      </form>
    </div>
  );
}