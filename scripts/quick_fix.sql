-- 快速修复脚本：为现有任务创建默认任务类型关联
-- 执行命令: /opt/homebrew/opt/mysql-client/bin/mysql -h 127.0.0.1 -P 3306 -u root -p123456 jobs < scripts/quick_fix.sql

-- 1. 首先确保有默认任务类型
INSERT IGNORE INTO task_types (id, name, display_name, description, default_lb_strategy) 
VALUES ('default-type-id', 'default', '默认任务类型', '系统默认任务类型，用于兼容现有任务', 'round_robin');

-- 2. 为所有没有task_type_id的任务设置默认任务类型
UPDATE tasks 
SET task_type_id = 'default-type-id' 
WHERE task_type_id IS NULL OR task_type_id = '';

-- 3. 为默认任务类型分配所有现有的在线执行器
INSERT IGNORE INTO task_type_executors (id, task_type_id, executor_id, weight, priority, enabled)
SELECT 
    CONCAT('tte-', e.id) as id,
    'default-type-id' as task_type_id,
    e.id as executor_id,
    1 as weight,
    0 as priority,
    TRUE as enabled
FROM executors e 
WHERE e.status = 'online';

-- 4. 说明
-- 这个脚本确保现有系统能够平滑过渡到新的任务类型架构
-- 所有现有任务都会关联到默认任务类型
-- 所有在线执行器都会被分配给默认任务类型