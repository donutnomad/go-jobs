'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { executorApi } from '@/lib/api';

interface ExecutorFormProps {
  onSuccess: () => void;
  onCancel: () => void;
}

export default function ExecutorForm({ onSuccess, onCancel }: ExecutorFormProps) {
  const queryClient = useQueryClient();
  const [formData, setFormData] = useState({
    name: '',
    instance_id: '',
    base_url: 'http://localhost:9090',
    health_check_url: 'http://localhost:9090/health',
    tasks: [{ task_name: '', cron_expression: '' }],
  });

  const registerMutation = useMutation({
    mutationFn: (data: any) => executorApi.register(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['executors'] });
      onSuccess();
    },
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    const validTasks = formData.tasks.filter(task => task.task_name && task.cron_expression);
    
    if (validTasks.length === 0) {
      alert('请至少添加一个任务');
      return;
    }

    await registerMutation.mutateAsync({
      ...formData,
      tasks: validTasks,
    });
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleTaskChange = (index: number, field: string, value: string) => {
    const newTasks = [...formData.tasks];
    newTasks[index] = { ...newTasks[index], [field]: value };
    setFormData(prev => ({ ...prev, tasks: newTasks }));
  };

  const addTask = () => {
    setFormData(prev => ({
      ...prev,
      tasks: [...prev.tasks, { task_name: '', cron_expression: '' }],
    }));
  };

  const removeTask = (index: number) => {
    if (formData.tasks.length > 1) {
      setFormData(prev => ({
        ...prev,
        tasks: prev.tasks.filter((_, i) => i !== index),
      }));
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label htmlFor="name" className="block text-sm font-medium text-gray-700">
          执行器名称
        </label>
        <input
          type="text"
          id="name"
          name="name"
          value={formData.name}
          onChange={handleChange}
          required
          placeholder="例如：worker-01"
          className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        />
      </div>

      <div>
        <label htmlFor="instance_id" className="block text-sm font-medium text-gray-700">
          实例 ID
        </label>
        <input
          type="text"
          id="instance_id"
          name="instance_id"
          value={formData.instance_id}
          onChange={handleChange}
          required
          placeholder="例如：executor-001"
          className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        />
      </div>

      <div>
        <label htmlFor="base_url" className="block text-sm font-medium text-gray-700">
          基础 URL
        </label>
        <input
          type="url"
          id="base_url"
          name="base_url"
          value={formData.base_url}
          onChange={handleChange}
          required
          placeholder="http://localhost:9090"
          className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        />
      </div>

      <div>
        <label htmlFor="health_check_url" className="block text-sm font-medium text-gray-700">
          健康检查 URL
        </label>
        <input
          type="url"
          id="health_check_url"
          name="health_check_url"
          value={formData.health_check_url}
          onChange={handleChange}
          required
          placeholder="http://localhost:9090/health"
          className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        />
      </div>

      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="block text-sm font-medium text-gray-700">
            支持的任务
          </label>
          <button
            type="button"
            onClick={addTask}
            className="text-sm text-blue-600 hover:text-blue-700"
          >
            + 添加任务
          </button>
        </div>
        <div className="space-y-2">
          {formData.tasks.map((task, index) => (
            <div key={index} className="flex space-x-2">
              <input
                type="text"
                value={task.task_name}
                onChange={(e) => handleTaskChange(index, 'task_name', e.target.value)}
                placeholder="任务名称"
                className="flex-1 px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
              />
              <input
                type="text"
                value={task.cron_expression}
                onChange={(e) => handleTaskChange(index, 'cron_expression', e.target.value)}
                placeholder="Cron 表达式"
                className="flex-1 px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 font-mono"
              />
              {formData.tasks.length > 1 && (
                <button
                  type="button"
                  onClick={() => removeTask(index)}
                  className="px-3 py-2 text-red-600 hover:text-red-700"
                >
                  删除
                </button>
              )}
            </div>
          ))}
        </div>
      </div>

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
          disabled={registerMutation.isPending}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50"
        >
          {registerMutation.isPending ? '注册中...' : '注册'}
        </button>
      </div>
    </form>
  );
}