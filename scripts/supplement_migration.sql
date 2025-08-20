-- 补充迁移脚本：添加缺失的字段和数据
-- 执行命令: /opt/homebrew/opt/mysql-client/bin/mysql -h 127.0.0.1 -P 3306 -u root -p123456 jobs < scripts/supplement_migration.sql

-- 1. 为task_types表添加缺失的default_lb_strategy字段
ALTER TABLE task_types 
ADD COLUMN default_lb_strategy ENUM('round_robin','weighted','random','sticky','least_loaded') DEFAULT 'round_robin';

-- 2. 创建任务类型与执行器关联表（如果不存在）
CREATE TABLE IF NOT EXISTS task_type_executors (
    id VARCHAR(64) PRIMARY KEY,
    task_type_id VARCHAR(64) NOT NULL,
    executor_id VARCHAR(64) NOT NULL,
    weight INT DEFAULT 1,
    priority INT DEFAULT 0,
    max_concurrent INT DEFAULT 10,
    timeout_seconds INT DEFAULT 300,
    lb_strategy ENUM('round_robin','weighted','random','sticky','least_loaded') NULL,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_type_executor (task_type_id, executor_id),
    INDEX idx_type_executor (task_type_id, executor_id),
    INDEX idx_enabled (enabled)
);

-- 3. 创建任务类型负载均衡状态表（如果不存在）
CREATE TABLE IF NOT EXISTS task_type_load_balance_state (
    task_type_id VARCHAR(64) PRIMARY KEY,
    last_executor_id VARCHAR(64),
    round_robin_index INT DEFAULT 0,
    sticky_executor_id VARCHAR(64),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 4. 插入默认任务类型（使用固定ID）
INSERT IGNORE INTO task_types (id, name, display_name, description, default_lb_strategy, created_at, updated_at) VALUES
('default-type-001', 'default', '默认任务类型', '系统默认任务类型，适用于通用任务', 'round_robin', NOW(), NOW()),
('batch-type-001', 'batch', '批处理任务', '适用于批量数据处理任务', 'least_loaded', NOW(), NOW()),
('realtime-type-001', 'realtime', '实时任务', '适用于实时性要求高的任务', 'round_robin', NOW(), NOW()),
('scheduled-type-001', 'scheduled', '定时任务', '适用于定时执行的任务', 'sticky', NOW(), NOW()),
('data-sync-type-001', 'data_sync', '数据同步', '适用于数据同步任务', 'weighted', NOW(), NOW());

-- 5. 为现有任务设置默认任务类型
UPDATE tasks 
SET task_type_id = 'default-type-001' 
WHERE task_type_id IS NULL OR task_type_id = '';

-- 6. 为默认任务类型分配所有健康的执行器
INSERT IGNORE INTO task_type_executors (id, task_type_id, executor_id, weight, priority, enabled, created_at, updated_at)
SELECT 
    CONCAT('default-tte-', e.id) as id,
    'default-type-001' as task_type_id,
    e.id as executor_id,
    1 as weight,
    0 as priority,
    TRUE as enabled,
    NOW() as created_at,
    NOW() as updated_at
FROM executors e 
WHERE e.status = 'online' AND e.is_healthy = TRUE;

-- 7. 显示迁移结果
SELECT 'Supplement migration completed' as status,
       (SELECT COUNT(*) FROM task_types) as task_types_count,
       (SELECT COUNT(*) FROM task_type_executors) as assignments_count,
       (SELECT COUNT(*) FROM tasks WHERE task_type_id IS NOT NULL) as tasks_with_type_count;