-- 优化数据库性能的索引创建脚本
-- 执行命令: mysql -h 127.0.0.1 -P 3306 -u root -p123456 jobs < scripts/optimize_indexes.sql

-- 任务执行表索引
ALTER TABLE task_executions ADD INDEX idx_task_executions_status (status);
ALTER TABLE task_executions ADD INDEX idx_task_executions_task_id (task_id);
ALTER TABLE task_executions ADD INDEX idx_task_executions_executor_id (executor_id);
ALTER TABLE task_executions ADD INDEX idx_task_executions_scheduled_time (scheduled_time);
ALTER TABLE task_executions ADD INDEX idx_task_executions_status_executor (status, executor_id);

-- 任务表索引
ALTER TABLE tasks ADD INDEX idx_tasks_status (status);
ALTER TABLE tasks ADD INDEX idx_tasks_name (name);
ALTER TABLE tasks ADD INDEX idx_tasks_cron (cron_expression);

-- 执行器表索引
ALTER TABLE executors ADD INDEX idx_executors_status (status);
ALTER TABLE executors ADD INDEX idx_executors_instance_id (instance_id);
ALTER TABLE executors ADD INDEX idx_executors_health_check (status, last_heartbeat);

-- 任务执行器关联表索引
ALTER TABLE task_executors ADD INDEX idx_task_executors_task (task_id);
ALTER TABLE task_executors ADD INDEX idx_task_executors_executor (executor_id);

-- 负载均衡状态表索引
ALTER TABLE load_balance_state ADD INDEX idx_load_balance_task (task_id);

-- 调度器实例表索引
ALTER TABLE scheduler_instances ADD INDEX idx_scheduler_instances_status (status, last_heartbeat);

-- 分析表以更新统计信息
ANALYZE TABLE task_executions;
ANALYZE TABLE tasks;
ANALYZE TABLE executors;
ANALYZE TABLE task_executors;
ANALYZE TABLE load_balance_state;
ANALYZE TABLE scheduler_instances;