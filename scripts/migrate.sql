-- 任务调度器数据库迁移脚本

CREATE DATABASE IF NOT EXISTS jobs CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE jobs;

-- 任务定义表
CREATE TABLE IF NOT EXISTS tasks (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    cron_expression VARCHAR(100) NOT NULL,
    parameters JSON,
    execution_mode ENUM('sequential', 'parallel', 'skip') DEFAULT 'parallel',
    load_balance_strategy ENUM('round_robin', 'weighted_round_robin', 'random', 'sticky', 'least_loaded') DEFAULT 'round_robin',
    max_retry INT DEFAULT 3,
    timeout_seconds INT DEFAULT 300,
    status ENUM('active', 'paused', 'deleted') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 执行器注册表
CREATE TABLE IF NOT EXISTS executors (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    instance_id VARCHAR(255) NOT NULL,
    base_url VARCHAR(500) NOT NULL,
    health_check_url VARCHAR(500),
    status ENUM('online', 'offline', 'maintenance') DEFAULT 'online',
    is_healthy BOOLEAN DEFAULT TRUE,
    last_heartbeat TIMESTAMP NULL,
    last_health_check TIMESTAMP NULL,
    health_check_failures INT DEFAULT 0,
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status_healthy (status, is_healthy),
    INDEX idx_name_instance (name, instance_id),
    UNIQUE KEY uk_instance_id (instance_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 任务与执行器的关系表
CREATE TABLE IF NOT EXISTS task_executors (
    id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL,
    executor_id VARCHAR(64) NOT NULL,
    priority INT DEFAULT 0,
    weight INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (executor_id) REFERENCES executors(id) ON DELETE CASCADE,
    UNIQUE KEY uk_task_executor (task_id, executor_id),
    INDEX idx_task_id (task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 任务执行历史表
CREATE TABLE IF NOT EXISTS task_executions (
    id VARCHAR(64) PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL,
    executor_id VARCHAR(64),
    scheduled_time TIMESTAMP NOT NULL,
    start_time TIMESTAMP NULL,
    end_time TIMESTAMP NULL,
    status ENUM('pending', 'running', 'success', 'failed', 'timeout', 'skipped') DEFAULT 'pending',
    result JSON,
    logs TEXT,
    retry_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (executor_id) REFERENCES executors(id) ON DELETE SET NULL,
    INDEX idx_task_status (task_id, status),
    INDEX idx_scheduled_time (scheduled_time),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 负载均衡状态表
CREATE TABLE IF NOT EXISTS load_balance_state (
    task_id VARCHAR(64) PRIMARY KEY,
    last_executor_id VARCHAR(64),
    round_robin_index INT DEFAULT 0,
    sticky_executor_id VARCHAR(64),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (last_executor_id) REFERENCES executors(id) ON DELETE SET NULL,
    FOREIGN KEY (sticky_executor_id) REFERENCES executors(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 调度器实例表（用于主从选举）
CREATE TABLE IF NOT EXISTS scheduler_instances (
    id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(255) NOT NULL UNIQUE,
    host VARCHAR(255) NOT NULL,
    port INT NOT NULL,
    is_leader BOOLEAN DEFAULT FALSE,
    last_heartbeat TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_is_leader (is_leader),
    INDEX idx_last_heartbeat (last_heartbeat)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;