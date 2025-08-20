-- 添加 cancelled 状态到 task_executions 表的 status 枚举
ALTER TABLE task_executions 
MODIFY COLUMN status ENUM('pending','running','success','failed','timeout','skipped','cancelled') DEFAULT 'pending';

-- 如果有需要，可以更新现有的某些记录
-- UPDATE task_executions SET status = 'cancelled' WHERE status = 'failed' AND logs LIKE '%cancelled%';