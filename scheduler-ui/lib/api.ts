import axios from 'axios';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export const api = axios.create({
  baseURL: `${API_BASE_URL}/api/v1`,
  headers: {
    'Content-Type': 'application/json',
  },
});

export interface Task {
  id: string;
  name: string;
  task_type_id?: string;
  cron_expression: string;
  parameters?: Record<string, any>;
  execution_mode: 'sequential' | 'parallel' | 'skip';
  load_balance_strategy: 'round_robin' | 'weighted' | 'random' | 'sticky' | 'least_loaded';
  max_retry: number;
  timeout_seconds: number;
  status: 'active' | 'paused' | 'deleted';
  created_at: string;
  updated_at: string;
  task_executors?: TaskExecutor[];
}

export interface TaskExecutor {
  id: string;
  task_id: string;
  executor_id: string;
  priority: number;
  weight: number;
  executor?: Executor;
}

export interface Executor {
  id: string;
  name: string;
  instance_id: string;
  base_url: string;
  health_check_url: string;
  status: 'online' | 'offline';
  created_at: string;
  updated_at: string;
}

export interface TaskExecution {
  id: string;
  task_id: string;
  executor_id: string;
  scheduled_time: string;
  start_time?: string;
  end_time?: string;
  status: 'pending' | 'running' | 'success' | 'failed' | 'timeout' | 'cancelled';
  result?: string;
  logs?: string;
  error?: string;
  retry_count: number;
  task?: Task;
  executor?: Executor;
}

export interface SchedulerInstance {
  id: string;
  instance_id: string;
  hostname: string;
  is_leader: boolean;
  created_at: string;
  updated_at: string;
}

export const taskApi = {
  list: (status?: string) => api.get<ApiResponse<Task[]>>('/tasks', { params: { status } }),
  get: (id: string) => api.get<ApiResponse<Task>>(`/tasks/${id}`),
  create: (data: Partial<Task>) => api.post<ApiResponse<Task>>('/tasks', data),
  update: (id: string, data: Partial<Task>) => api.put<ApiResponse<Task>>(`/tasks/${id}`, data),
  delete: (id: string) => api.delete(`/tasks/${id}`),
  trigger: (id: string, parameters?: Record<string, any>) => 
    api.post<ApiResponse<TaskExecution>>(`/tasks/${id}/trigger`, { parameters }),
  getExecutors: (id: string) => api.get<ApiResponse<TaskExecutor[]>>(`/tasks/${id}/executors`),
};

export const executorApi = {
  list: () => api.get<Executor[]>('/executors'),
  get: (id: string) => api.get<Executor>(`/executors/${id}`),
  register: (data: {
    name: string;
    instance_id: string;
    base_url: string;
    health_check_url: string;
    tasks: Array<{ task_name: string; cron_expression: string }>;
  }) => api.post<Executor>('/executors/register', data),
  updateStatus: (id: string, status: string, reason?: string) => 
    api.put(`/executors/${id}/status`, { status, reason }),
  delete: (id: string) => api.delete(`/executors/${id}`),
};

// 新的API响应包装格式
export interface ApiResponse<T> {
  code: number;
  data: T;
  message: string;
  total: number;
}

export interface PaginatedResponse<T> {
  total: number;
  items: T[];
}

export const executionApi = {
  list: (params?: {
    task_id?: string;
    status?: string;
    start_time?: string;
    end_time?: string;
    page?: number;
    page_size?: number;
  }) => api.get<ApiResponse<TaskExecution[]>>('/executions', { params }),
  get: (id: string) => api.get<ApiResponse<TaskExecution>>(`/executions/${id}`),
  callback: (id: string, data: {
    status: string;
    result?: string;
    error?: string;
  }) => api.post(`/executions/${id}/callback`, data),
  stop: (id: string) => api.post(`/executions/${id}/stop`),
};

export const schedulerApi = {
  status: () => api.get<{ instances: SchedulerInstance[]; time: string }>('/scheduler/status'),
  health: () => api.get<{ status: string; time: string }>('/health'),
};