-- account 表 UID 改为可空，允许未填写达人ID的账号重复保存

-- 步骤1: 将 uid 列改为允许 NULL
ALTER TABLE `account`
MODIFY COLUMN `uid` VARCHAR(100) NULL DEFAULT NULL COMMENT '达人ID（抖音UID/快手号/视频号ID）';

-- 步骤2: 将历史空字符串转为 NULL，避免与旧唯一索引冲突
UPDATE `account`
SET `uid` = NULL
WHERE `uid` IS NOT NULL AND TRIM(`uid`) = '';

-- 步骤3: 删除历史唯一索引 uk_uid（若存在）
SET @idx_exists = (
    SELECT COUNT(1)
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'account'
      AND index_name = 'uk_uid'
);
SET @sql = IF(@idx_exists > 0, 'ALTER TABLE `account` DROP INDEX `uk_uid`', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 步骤4: 确保存在普通查询索引 idx_platform_uid（若不存在）
SET @idx_exists = (
    SELECT COUNT(1)
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'account'
      AND index_name = 'idx_platform_uid'
);
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE `account` ADD INDEX `idx_platform_uid` (`platform`, `uid`)', 'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
