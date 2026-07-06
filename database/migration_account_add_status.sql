-- 为 account 表补充缺失字段和索引
-- 执行时间：2026-07-04

-- 1. status 字段：软删除支持
ALTER TABLE `account` ADD COLUMN `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0=已删除 1=正常';

-- 2. 时间字段
ALTER TABLE `account` ADD COLUMN `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间';
ALTER TABLE `account` ADD COLUMN `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间';

-- 3. bind_status 字段：账号-投手绑定状态
ALTER TABLE `account` ADD COLUMN `bind_status` TINYINT NOT NULL DEFAULT 0 COMMENT '绑定状态：0=未绑定，1=被绑定';
ALTER TABLE `account` ADD KEY `idx_bind_status` (`bind_status`);

-- 4. 唯一索引：防止同平台下账号名称重复导入
-- 如果表中有重复数据，需要先去重后再创建此索引
-- 去重SQL见 scripts/dedup_account.sql
ALTER TABLE `account` ADD UNIQUE KEY `uk_platform_account_name` (`platform`, `account_name`) COMMENT '平台+账号名称唯一';
