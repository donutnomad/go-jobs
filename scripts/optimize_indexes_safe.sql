-- 优化数据库性能的索引创建脚本（安全版本）
-- 执行命令: mysql -h 127.0.0.1 -P 3306 -u root -p123456 jobs < scripts/optimize_indexes_safe.sql

-- 使用存储过程安全地创建索引
DELIMITER $$

-- 创建辅助存储过程
DROP PROCEDURE IF EXISTS add_index_if_not_exists$$
CREATE PROCEDURE add_index_if_not_exists(
    IN p_table_name VARCHAR(64),
    IN p_index_name VARCHAR(64),
    IN p_columns VARCHAR(255)
)
BEGIN
    DECLARE index_exists INT DEFAULT 0;
    
    SELECT COUNT(*) INTO index_exists
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
        AND table_name = p_table_name
        AND index_name = p_index_name;
    
    IF index_exists = 0 THEN
        SET @sql = CONCAT('ALTER TABLE ', p_table_name, ' ADD INDEX ', p_index_name, ' (', p_columns, ')');
        PREPARE stmt FROM @sql;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;
        SELECT CONCAT('Index ', p_index_name, ' created on table ', p_table_name) AS result;
    ELSE
        SELECT CONCAT('Index ', p_index_name, ' already exists on table ', p_table_name) AS result;
    END IF;
END$$

DELIMITER ;

-- 创建索引
CALL add_index_if_not_exists('task_executions', 'idx_task_executions_status', 'status');
CALL add_index_if_not_exists('task_executions', 'idx_task_executions_task_id', 'task_id');
CALL add_index_if_not_exists('task_executions', 'idx_task_executions_executor_id', 'executor_id');
CALL add_index_if_not_exists('task_executions', 'idx_task_executions_scheduled_time', 'scheduled_time');
CALL add_index_if_not_exists('task_executions', 'idx_task_executions_status_executor', 'status, executor_id');

CALL add_index_if_not_exists('tasks', 'idx_tasks_status', 'status');
CALL add_index_if_not_exists('tasks', 'idx_tasks_name', 'name');
CALL add_index_if_not_exists('tasks', 'idx_tasks_cron', 'cron_expression');

CALL add_index_if_not_exists('executors', 'idx_executors_status', 'status');
CALL add_index_if_not_exists('executors', 'idx_executors_instance_id', 'instance_id');
CALL add_index_if_not_exists('executors', 'idx_executors_health_check', 'status, last_heartbeat');

CALL add_index_if_not_exists('task_executors', 'idx_task_executors_task', 'task_id');
CALL add_index_if_not_exists('task_executors', 'idx_task_executors_executor', 'executor_id');

CALL add_index_if_not_exists('load_balance_state', 'idx_load_balance_task', 'task_id');

CALL add_index_if_not_exists('scheduler_instances', 'idx_scheduler_instances_leader', 'is_leader, last_heartbeat');

-- 清理存储过程
DROP PROCEDURE IF EXISTS add_index_if_not_exists;

-- 分析表以更新统计信息
ANALYZE TABLE task_executions;
ANALYZE TABLE tasks;
ANALYZE TABLE executors;
ANALYZE TABLE task_executors;
ANALYZE TABLE load_balance_state;
ANALYZE TABLE scheduler_instances;