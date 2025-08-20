-- 安全的任务类型系统数据库迁移脚本
-- 执行命令: /opt/homebrew/opt/mysql-client/bin/mysql -h 127.0.0.1 -P 3306 -u root -p123456 jobs < scripts/safe_migration.sql

-- 检查并删除可能存在的旧外键约束
SET foreign_key_checks = 0;

-- 1. 检查tasks表中是否已存在task_type_id字段
SELECT COUNT(*) as task_type_id_exists FROM information_schema.COLUMNS 
WHERE table_schema = 'jobs' AND table_name = 'tasks' AND column_name = 'task_type_id';

-- 2. 创建任务类型表
CREATE TABLE IF NOT EXISTS task_types (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(200),
    description TEXT,
    parameters_schema JSON,
    default_lb_strategy ENUM('round_robin','weighted','random','sticky','least_loaded') DEFAULT 'round_robin',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 确保name字段有唯一索引
CREATE UNIQUE INDEX IF NOT EXISTS uk_task_types_name ON task_types(name);
CREATE INDEX IF NOT EXISTS idx_task_types_strategy ON task_types(default_lb_strategy);

-- 3. 创建任务类型与执行器关联表
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
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 创建索引和唯一约束
CREATE INDEX IF NOT EXISTS idx_tte_type_executor ON task_type_executors(task_type_id, executor_id);
CREATE INDEX IF NOT EXISTS idx_tte_enabled ON task_type_executors(enabled);
CREATE INDEX IF NOT EXISTS idx_tte_priority ON task_type_executors(priority);
CREATE UNIQUE INDEX IF NOT EXISTS uk_tte_type_executor ON task_type_executors(task_type_id, executor_id);

-- 4. 创建任务类型负载均衡状态表
CREATE TABLE IF NOT EXISTS task_type_load_balance_state (
    task_type_id VARCHAR(64) PRIMARY KEY,
    last_executor_id VARCHAR(64),
    round_robin_index INT DEFAULT 0,
    sticky_executor_id VARCHAR(64),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 5. 安全地添加task_type_id字段到tasks表（如果不存在）
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM information_schema.COLUMNS 
     WHERE table_schema = 'jobs' AND table_name = 'tasks' AND column_name = 'task_type_id') = 0,
    'ALTER TABLE tasks ADD COLUMN task_type_id VARCHAR(64) NULL AFTER name',
    'SELECT "task_type_id column already exists" as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 6. 添加索引（如果不存在）
CREATE INDEX IF NOT EXISTS idx_tasks_task_type ON tasks(task_type_id);

-- 7. 创建一些默认的任务类型（使用固定ID以便后续引用）
INSERT IGNORE INTO task_types (id, name, display_name, description, default_lb_strategy) VALUES
('default-type-001', 'default', '默认任务类型', '系统默认任务类型，适用于通用任务', 'round_robin'),
('batch-type-001', 'batch', '批处理任务', '适用于批量数据处理任务', 'least_loaded'),
('realtime-type-001', 'realtime', '实时任务', '适用于实时性要求高的任务', 'round_robin'),
('scheduled-type-001', 'scheduled', '定时任务', '适用于定时执行的任务', 'sticky'),
('data-sync-type-001', 'data_sync', '数据同步', '适用于数据同步任务', 'weighted');

-- 8. 为所有没有task_type_id的现有任务设置默认类型
UPDATE tasks 
SET task_type_id = 'default-type-001' 
WHERE task_type_id IS NULL OR task_type_id = '';

-- 9. 为默认任务类型分配所有现有的健康执行器（避免重复插入）
INSERT IGNORE INTO task_type_executors (id, task_type_id, executor_id, weight, priority, enabled)
SELECT 
    CONCAT('default-tte-', e.id) as id,
    'default-type-001' as task_type_id,
    e.id as executor_id,
    1 as weight,
    0 as priority,
    TRUE as enabled
FROM executors e 
WHERE e.status = 'online' AND e.is_healthy = TRUE;

-- 重新启用外键检查
SET foreign_key_checks = 1;

-- 10. 验证迁移结果
SELECT 
    COUNT(*) as total_task_types,
    (SELECT COUNT(*) FROM task_type_executors) as total_assignments,
    (SELECT COUNT(*) FROM tasks WHERE task_type_id IS NOT NULL) as tasks_with_type
FROM task_types;

-- 显示结果
SELECT 'Migration completed successfully' as status;