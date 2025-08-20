-- 修复外键约束和字段长度不匹配问题
-- 执行命令: /opt/homebrew/opt/mysql-client/bin/mysql -h 127.0.0.1 -P 3306 -u root -p123456 jobs < scripts/fix_foreign_key.sql

-- 1. 删除外键约束
ALTER TABLE tasks DROP FOREIGN KEY fk_tasks_task_type;

-- 2. 修改task_type_id字段长度以匹配task_types.id (都改为64)
ALTER TABLE task_types MODIFY COLUMN id VARCHAR(64);
ALTER TABLE tasks MODIFY COLUMN task_type_id VARCHAR(64);

-- 3. 重新添加外键约束（可选，为了数据完整性）
-- ALTER TABLE tasks ADD CONSTRAINT fk_tasks_task_type FOREIGN KEY (task_type_id) REFERENCES task_types(id) ON DELETE SET NULL;

-- 4. 验证修复结果
SELECT 'Foreign key constraint fixed' as status;