'use client';

import { useState, useEffect } from 'react';
import { useMutation } from '@tanstack/react-query';
import { taskApi, Task } from '@/lib/api';

interface TaskFormProps {
  task?: Task;
  onSuccess: () => void;
  onCancel: () => void;
}

export default function TaskForm({ task, onSuccess, onCancel }: TaskFormProps) {
  const [formData, setFormData] = useState({
    name: '',
    cron_expression: '*/10 * * * * *',
    execution_mode: 'sequential',
    load_balance_strategy: 'round_robin',
    max_retry: 3,
    timeout_seconds: 300,
    status: 'active',
    parameters: '{}',
  });

  useEffect(() => {
    if (task) {
      setFormData({
        name: task.name,
        cron_expression: task.cron_expression,
        execution_mode: task.execution_mode,
        load_balance_strategy: task.load_balance_strategy,
        max_retry: task.max_retry,
        timeout_seconds: task.timeout_seconds,
        status: task.status,
        parameters: JSON.stringify(task.parameters || {}),
      });
    }
  }, [task]);

  const createMutation = useMutation({
    mutationFn: (data: any) => taskApi.create(data),
    onSuccess: () => {
      onSuccess();
    },
  });

  const updateMutation = useMutation({
    mutationFn: (data: any) => taskApi.update(task!.id, data),
    onSuccess: () => {
      onSuccess();
    },
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    let parameters = {};
    try {
      parameters = JSON.parse(formData.parameters);
    } catch (e) {
      alert('参数格式错误，必须是有效的 JSON');
      return;
    }

    const data = {
      ...formData,
      parameters,
      max_retry: Number(formData.max_retry),
      timeout_seconds: Number(formData.timeout_seconds),
    };

    if (task) {
      await updateMutation.mutateAsync(data);
    } else {
      await createMutation.mutateAsync(data);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label htmlFor="name" className="block text-sm font-medium text-gray-700">
          任务名称
        </label>
        <input
          type="text"
          id="name"
          name="name"
          value={formData.name}
          onChange={handleChange}
          required
          className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        />
      </div>

      <div>
        <label htmlFor="cron_expression" className="block text-sm font-medium text-gray-700">
          Cron 表达式
        </label>
        <input
          type="text"
          id="cron_expression"
          name="cron_expression"
          value={formData.cron_expression}
          onChange={handleChange}
          required
          placeholder="*/10 * * * * *"
          className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 font-mono"
        />
        <p className="mt-1 text-xs text-gray-500">格式：秒 分 时 日 月 周</p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label htmlFor="execution_mode" className="block text-sm font-medium text-gray-700">
            执行模式
          </label>
          <select
            id="execution_mode"
            name="execution_mode"
            value={formData.execution_mode}
            onChange={handleChange}
            className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
          >
            <option value="sequential">顺序执行</option>
            <option value="parallel">并行执行</option>
            <option value="skip">跳过</option>
          </select>
        </div>

        <div>
          <label htmlFor="load_balance_strategy" className="block text-sm font-medium text-gray-700">
            负载均衡策略
          </label>
          <select
            id="load_balance_strategy"
            name="load_balance_strategy"
            value={formData.load_balance_strategy}
            onChange={handleChange}
            className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
          >
            <option value="round_robin">轮询</option>
            <option value="weighted">加权</option>
            <option value="random">随机</option>
            <option value="sticky">粘性</option>
            <option value="least_loaded">最少负载</option>
          </select>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label htmlFor="max_retry" className="block text-sm font-medium text-gray-700">
            最大重试次数
          </label>
          <input
            type="number"
            id="max_retry"
            name="max_retry"
            value={formData.max_retry}
            onChange={handleChange}
            min="0"
            max="10"
            className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
          />
        </div>

        <div>
          <label htmlFor="timeout_seconds" className="block text-sm font-medium text-gray-700">
            超时时间（秒）
          </label>
          <input
            type="number"
            id="timeout_seconds"
            name="timeout_seconds"
            value={formData.timeout_seconds}
            onChange={handleChange}
            min="1"
            max="3600"
            className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
          />
        </div>
      </div>

      <div>
        <label htmlFor="parameters" className="block text-sm font-medium text-gray-700">
          任务参数（JSON 格式）
        </label>
        <textarea
          id="parameters"
          name="parameters"
          value={formData.parameters}
          onChange={handleChange}
          rows={3}
          className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 font-mono text-sm"
        />
      </div>

      {task && (
        <div>
          <label htmlFor="status" className="block text-sm font-medium text-gray-700">
            状态
          </label>
          <select
            id="status"
            name="status"
            value={formData.status}
            onChange={handleChange}
            className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
          >
            <option value="active">活动</option>
            <option value="paused">暂停</option>
          </select>
        </div>
      )}

      <div className="flex justify-end space-x-3 pt-4">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50 transition-colors"
        >
          取消
        </button>
        <button
          type="submit"
          disabled={createMutation.isPending || updateMutation.isPending}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50"
        >
          {createMutation.isPending || updateMutation.isPending
            ? '保存中...'
            : task
            ? '更新'
            : '创建'}
        </button>
      </div>
    </form>
  );
}