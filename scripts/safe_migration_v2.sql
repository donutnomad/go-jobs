-- 兼容性更好的任务类型系统数据库迁移脚本
-- 执行命令: /opt/homebrew/opt/mysql-client/bin/mysql -h 127.0.0.1 -P 3306 -u root -p123456 jobs < scripts/safe_migration_v2.sql

-- 禁用外键检查
SET foreign_key_checks = 0;

-- 1. 创建任务类型表
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

-- 2. 安全地添加索引（检查是否存在）
-- 添加唯一索引
SET @index_exists = (SELECT COUNT(1) FROM information_schema.statistics 
                     WHERE table_schema = 'jobs' AND table_name = 'task_types' 
                     AND index_name = 'uk_task_types_name');
SET @sql = IF(@index_exists = 0, 'CREATE UNIQUE INDEX uk_task_types_name ON task_types(name)', 'SELECT "Index already exists"');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加策略索引
SET @index_exists = (SELECT COUNT(1) FROM information_schema.statistics 
                     WHERE table_schema = 'jobs' AND table_name = 'task_types' 
                     AND index_name = 'idx_task_types_strategy');
SET @sql = IF(@index_exists = 0, 'CREATE INDEX idx_task_types_strategy ON task_types(default_lb_strategy)', 'SELECT "Index already exists"');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

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

-- 4. 为task_type_executors添加索引
SET @index_exists = (SELECT COUNT(1) FROM information_schema.statistics 
                     WHERE table_schema = 'jobs' AND table_name = 'task_type_executors' 
                     AND index_name = 'idx_tte_type_executor');
SET @sql = IF(@index_exists = 0, 'CREATE INDEX idx_tte_type_executor ON task_type_executors(task_type_id, executor_id)', 'SELECT "Index already exists"');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @index_exists = (SELECT COUNT(1) FROM information_schema.statistics 
                     WHERE table_schema = 'jobs' AND table_name = 'task_type_executors' 
                     AND index_name = 'uk_tte_type_executor');
SET @sql = IF(@index_exists = 0, 'CREATE UNIQUE INDEX uk_tte_type_executor ON task_type_executors(task_type_id, executor_id)', 'SELECT "Index already exists"');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 5. 创建任务类型负载均衡状态表
CREATE TABLE IF NOT EXISTS task_type_load_balance_state (
    task_type_id VARCHAR(64) PRIMARY KEY,
    last_executor_id VARCHAR(64),
    round_robin_index INT DEFAULT 0,
    sticky_executor_id VARCHAR(64),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 6. 安全地添加task_type_id字段到tasks表（如果不存在）
SET @column_exists = (SELECT COUNT(*) FROM information_schema.COLUMNS 
                      WHERE table_schema = 'jobs' AND table_name = 'tasks' 
                      AND column_name = 'task_type_id');
SET @sql = IF(@column_exists = 0, 
              'ALTER TABLE tasks ADD COLUMN task_type_id VARCHAR(64) NULL AFTER name', 
              'SELECT "task_type_id column already exists"');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 7. 为tasks表的task_type_id添加索引
SET @index_exists = (SELECT COUNT(1) FROM information_schema.statistics 
                     WHERE table_schema = 'jobs' AND table_name = 'tasks' 
                     AND index_name = 'idx_tasks_task_type');
SET @sql = IF(@index_exists = 0, 'CREATE INDEX idx_tasks_task_type ON tasks(task_type_id)', 'SELECT "Index already exists"');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 8. 创建默认的任务类型
INSERT IGNORE INTO task_types (id, name, display_name, description, default_lb_strategy) VALUES
('default-type-001', 'default', '默认任务类型', '系统默认任务类型，适用于通用任务', 'round_robin'),
('batch-type-001', 'batch', '批处理任务', '适用于批量数据处理任务', 'least_loaded'),
('realtime-type-001', 'realtime', '实时任务', '适用于实时性要求高的任务', 'round_robin'),
('scheduled-type-001', 'scheduled', '定时任务', '适用于定时执行的任务', 'sticky'),
('data-sync-type-001', 'data_sync', '数据同步', '适用于数据同步任务', 'weighted');

-- 9. 为现有任务设置默认类型
UPDATE tasks 
SET task_type_id = 'default-type-001' 
WHERE task_type_id IS NULL OR task_type_id = '';

-- 10. 为默认任务类型分配所有健康的执行器
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

-- 验证迁移结果
SELECT 'Migration completed successfully' as status,
       (SELECT COUNT(*) FROM task_types) as task_types_count,
       (SELECT COUNT(*) FROM task_type_executors) as assignments_count,
       (SELECT COUNT(*) FROM tasks WHERE task_type_id IS NOT NULL) as tasks_with_type_count;