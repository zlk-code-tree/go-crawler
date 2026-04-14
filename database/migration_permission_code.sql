-- 添加permission_code字段到sys_operation_log表
ALTER TABLE `sys_operation_log` ADD COLUMN `permission_code` varchar(100) DEFAULT '' COMMENT '权限编码' AFTER `duration`;