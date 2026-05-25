-- ============================================================
-- shop 表：uid 改为可空，唯一索引仅对非空 uid 生效
-- ============================================================

-- 步骤1: 将 uid 的空字符串转为 NULL
UPDATE shop SET uid = NULL WHERE uid = '';

-- 步骤2: 修改 uid 列为允许 NULL
ALTER TABLE shop MODIFY COLUMN `uid` VARCHAR(100) COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL COMMENT '达人ID（抖音UID/快手号/视频号ID）';

-- 步骤3: 删除旧唯一索引
ALTER TABLE shop DROP INDEX `uk_uid`;

-- 步骤4: 创建新唯一索引（uid 非空时才唯一）
ALTER TABLE shop ADD UNIQUE KEY `uk_uid` (`platform`, `uid`);
