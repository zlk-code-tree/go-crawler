-- 添加操作地点字段到 sys_operation_log 表
ALTER TABLE `sys_operation_log`
ADD COLUMN `location` varchar(200) DEFAULT '' COMMENT '操作地点（根据IP解析）' AFTER `ip`;

-- 添加索引
ALTER TABLE `sys_operation_log`
ADD INDEX `idx_ip` (`ip`);
