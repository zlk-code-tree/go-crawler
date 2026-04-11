-- 为用户表添加缺失的字段
-- 执行此脚本前请确保数据库已存在

USE `easyadmin8`;

-- 为用户表添加缺失的字段
ALTER TABLE `sys_user` 
ADD COLUMN `nickname` VARCHAR(100) DEFAULT '' COMMENT '用户昵称' AFTER `real_name`,
ADD COLUMN `dept_id` BIGINT UNSIGNED DEFAULT 0 COMMENT '部门ID' AFTER `status`,
ADD COLUMN `position` VARCHAR(100) DEFAULT '' COMMENT '职位' AFTER `dept_id`,
ADD COLUMN `role_id` BIGINT UNSIGNED DEFAULT 0 COMMENT '角色ID' AFTER `position`,
ADD COLUMN `remark` VARCHAR(500) DEFAULT '' COMMENT '备注' AFTER `role_id`;

-- 为用户表添加索引
ALTER TABLE `sys_user` 
ADD INDEX `idx_dept_id` (`dept_id`),
ADD INDEX `idx_role_id` (`role_id`);

-- 为部门表添加缺失的字段（如果不存在）
ALTER TABLE `sys_department` 
ADD COLUMN `remark` VARCHAR(500) DEFAULT '' COMMENT '备注' AFTER `sort`;

-- 显示修改后的表结构
DESC `sys_user`;
DESC `sys_department`;