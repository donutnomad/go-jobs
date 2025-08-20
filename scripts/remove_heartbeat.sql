-- 移除心跳字段的迁移脚本
-- 执行命令: mysql -h 127.0.0.1 -P 3306 -u root -p123456 jobs < scripts/remove_heartbeat.sql

-- 删除执行器表的 last_heartbeat 字段
ALTER TABLE executors DROP COLUMN last_heartbeat;

-- 删除调度器实例表的 last_heartbeat 字段  
ALTER TABLE scheduler_instances DROP COLUMN last_heartbeat;

-- 添加说明注释
-- 系统现在完全依赖健康检查机制，不再需要心跳功能