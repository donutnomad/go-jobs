'use client';

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { ArrowLeft, Save, Server, Globe, Activity } from 'lucide-react';
import Link from 'next/link';

type RegistrationMode = 'full' | 'name-only';

interface RegisterExecutorRequest {
  name: string;
  instance_id?: string;
  base_url?: string;
  health_check_url?: string;
}

export default function RegisterExecutorPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [registrationMode, setRegistrationMode] = useState<RegistrationMode>('full');
  const [formData, setFormData] = useState<RegisterExecutorRequest>({
    name: '',
    instance_id: '',
    base_url: '',
    health_check_url: '',
  });
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

  const registerMutation = useMutation({
    mutationFn: async (data: RegisterExecutorRequest) => {
      let apiData: any;
      
      if (registrationMode === 'name-only') {
        // 仅名称模式：只发送名称
        apiData = {
          executor_name: data.name,
          name_only: true, // 标识这是仅名称注册
        };
      } else {
        // 完整模式：转换字段名以匹配后端API期望的格式
        apiData = {
          executor_id: data.instance_id,
          executor_name: data.name,
          executor_url: data.base_url,
          health_check_url: data.health_check_url,
        };
      }
      
      const response = await fetch(`${apiUrl}/api/v1/executors/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(apiData),
      });
      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['executors'] });
      router.push('/executors');
    },
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // 根据注册模式验证必填字段
    if (registrationMode === 'full') {
      if (!formData.name || !formData.instance_id || !formData.base_url) {
        alert('请填写所有必填字段');
        return;
      }
    } else {
      if (!formData.name) {
        alert('请填写执行器名称');
        return;
      }
    }
    
    try {
      await registerMutation.mutateAsync(formData);
    } catch (error) {
      alert('注册执行器失败: ' + (error as Error).message);
    }
  };

  const handleModeChange = (mode: RegistrationMode) => {
    setRegistrationMode(mode);
    // 当切换到仅名称模式时，清空其他字段
    if (mode === 'name-only') {
      setFormData({
        name: formData.name,
        instance_id: '',
        base_url: '',
        health_check_url: '',
      });
    }
  };

  const handleBaseUrlChange = (value: string) => {
    setFormData({ 
      ...formData, 
      base_url: value,
      // 自动填充健康检查URL
      health_check_url: value ? `${value}/health` : ''
    });
  };

  const generateInstanceId = () => {
    const timestamp = Date.now();
    const random = Math.random().toString(36).substr(2, 5);
    const instanceId = `executor-${timestamp}-${random}`;
    setFormData({ ...formData, instance_id: instanceId });
  };

  return (
    <div className="max-w-2xl mx-auto">
      {/* 面包屑导航 */}
      <div className="mb-6 flex items-center space-x-2 text-sm">
        <Link href="/executors" className="text-blue-600 hover:text-blue-800 flex items-center space-x-1">
          <ArrowLeft className="w-4 h-4" />
          <span>执行器管理</span>
        </Link>
        <span className="text-gray-500">/</span>
        <span className="text-gray-900">注册执行器</span>
      </div>

      {/* 页面头部 */}
      <div className="mb-8">
        <div className="flex items-center space-x-3">
          <Server className="w-8 h-8 text-blue-600" />
          <div>
            <h1 className="text-2xl font-bold text-gray-900">注册新执行器</h1>
            <p className="text-sm text-gray-600">添加一个新的任务执行器到系统中</p>
          </div>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* 注册模式选择 */}
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">注册方式</h2>
          <div className="space-y-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                选择注册方式 *
              </label>
              <select
                value={registrationMode}
                onChange={(e) => handleModeChange(e.target.value as RegistrationMode)}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="full">完整实例信息注册</option>
                <option value="name-only">仅名称注册</option>
              </select>
              <p className="text-xs text-gray-500 mt-1">
                {registrationMode === 'full' 
                  ? '需要提供完整的执行器实例信息，包括URL和健康检查端点'
                  : '仅需提供执行器名称，适用于只需要标识执行器的场景'
                }
              </p>
            </div>
          </div>
        </div>

        {/* 基本信息 */}
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">基本信息</h2>
          
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                执行器名称 *
              </label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="例如: user-service-executor"
                required
              />
              <p className="text-xs text-gray-500 mt-1">
                执行器的显示名称，用于识别和管理
              </p>
            </div>

            {registrationMode === 'full' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  实例 ID *
                </label>
                <div className="flex space-x-2">
                  <input
                    type="text"
                    value={formData.instance_id || ''}
                    onChange={(e) => setFormData({ ...formData, instance_id: e.target.value })}
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="例如: executor-001"
                    required={registrationMode === 'full'}
                  />
                  <button
                    type="button"
                    onClick={generateInstanceId}
                    className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors"
                  >
                    生成
                  </button>
                </div>
                <p className="text-xs text-gray-500 mt-1">
                  唯一标识符，用于区分同一执行器的不同实例
                </p>
              </div>
            )}
          </div>
        </div>

        {registrationMode === 'full' && (
          /* 网络配置 */
          <div className="bg-white rounded-lg border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4 flex items-center space-x-2">
              <Globe className="w-5 h-5" />
              <span>网络配置</span>
            </h2>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  基础 URL *
                </label>
                <input
                  type="url"
                  value={formData.base_url || ''}
                  onChange={(e) => handleBaseUrlChange(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="http://localhost:9090"
                  required={registrationMode === 'full'}
                />
                <p className="text-xs text-gray-500 mt-1">
                  执行器服务的基础访问地址
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  健康检查 URL *
                </label>
                <input
                  type="url"
                  value={formData.health_check_url || ''}
                  onChange={(e) => setFormData({ ...formData, health_check_url: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="http://localhost:9090/health"
                  required={registrationMode === 'full'}
                />
                <p className="text-xs text-gray-500 mt-1">
                  用于检查执行器健康状态的端点
                </p>
              </div>
            </div>
          </div>
        )}

        {/* 注意事项 */}
        <div className="bg-blue-50 rounded-lg p-4">
          <div className="flex items-start space-x-3">
            <Activity className="w-5 h-5 text-blue-600 mt-0.5" />
            <div className="text-sm">
              <h3 className="font-medium text-blue-900 mb-1">
                {registrationMode === 'full' ? '注册前请确保' : '注意事项'}
              </h3>
              <ul className="text-blue-800 space-y-1">
                {registrationMode === 'full' ? (
                  <>
                    <li>• 执行器服务已启动并可访问</li>
                    <li>• 健康检查端点返回 200 状态码</li>
                    <li>• 执行器实现了必要的任务处理接口</li>
                    <li>• 网络连接正常，调度器可以访问执行器</li>
                  </>
                ) : (
                  <>
                    <li>• 仅名称注册不会进行实际的网络连接</li>
                    <li>• 该执行器将显示为离线状态，直到实际实例连接</li>
                    <li>• 适用于预先定义执行器配置的场景</li>
                    <li>• 后续可以通过编辑功能补充完整信息</li>
                  </>
                )}
              </ul>
            </div>
          </div>
        </div>

        {/* 提交按钮 */}
        <div className="flex items-center justify-end space-x-4 bg-white rounded-lg border border-gray-200 p-6">
          <Link
            href="/executors"
            className="px-6 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
          >
            取消
          </Link>
          <button
            type="submit"
            disabled={registerMutation.isPending}
            className="inline-flex items-center px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
          >
            <Save className="w-4 h-4 mr-2" />
            {registerMutation.isPending ? '注册中...' : '注册执行器'}
          </button>
        </div>
      </form>

      {/* 示例代码 */}
      <div className="mt-8 bg-white rounded-lg border border-gray-200 p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">执行器示例代码</h3>
        <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto">
          <pre className="text-green-400 text-sm">
{`// Go 执行器示例
package main

import (
    "encoding/json"
    "net/http"
    "log"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "ok",
        "message": "Executor is healthy"
    })
}

func executeTaskHandler(w http.ResponseWriter, r *http.Request) {
    // 处理任务执行逻辑
    // 读取请求参数，执行任务，返回结果
}

func main() {
    http.HandleFunc("/health", healthHandler)
    http.HandleFunc("/execute", executeTaskHandler)
    
    log.Println("Executor started on :9090")
    log.Fatal(http.ListenAndServe(":9090", nil))
}`}
          </pre>
        </div>
      </div>
    </div>
  );
}