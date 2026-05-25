-- ============================================================
-- shop 表迁移脚本：将 platform_account_id → uid
-- ============================================================

-- 步骤1: 添加 uid 列
ALTER TABLE shop ADD COLUMN `uid` VARCHAR(100) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '达人ID（抖音UID/快手号/视频号ID）' AFTER `id`;

-- 步骤2: 数据迁移（将 platform_account_id 的数据拷贝到 uid）
UPDATE shop SET uid = platform_account_id WHERE platform_account_id IS NOT NULL AND platform_account_id != '';

-- 步骤3: 创建唯一索引
ALTER TABLE shop ADD UNIQUE KEY `uk_uid` (`platform`, `uid`);

-- 步骤4: 删除旧列（确认步骤1-3全部正确后，再执行）
-- ALTER TABLE shop DROP COLUMN `platform_account_id`;

-- ============================================================
-- 验证查询
-- ============================================================
-- SELECT COUNT(*) FROM shop WHERE uid = '';
-- SELECT platform, COUNT(*) FROM shop GROUP BY platform;
-- SELECT COUNT(*) FROM shop;
