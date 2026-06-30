-- 千川计划维度日结报表补充内部组织归属字段
-- 用于支持：组织ID / 内部账号ID / 计划ID / 商品ID / 抖音号 / 消耗 的明细排查与归因。

SET @col_exists = (
  SELECT COUNT(1) FROM information_schema.columns
  WHERE table_schema = DATABASE() AND table_name = 'qianchuan_ad_report' AND column_name = 'organization_id'
);
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `qianchuan_ad_report` ADD COLUMN `organization_id` BIGINT(20) NOT NULL DEFAULT 0 COMMENT ''组织ID（默认公司ID）'' AFTER `id`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(1) FROM information_schema.columns
  WHERE table_schema = DATABASE() AND table_name = 'qianchuan_ad_report' AND column_name = 'account_id'
);
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `qianchuan_ad_report` ADD COLUMN `account_id` INT(11) NOT NULL DEFAULT 0 COMMENT ''内部账号ID，关联account.id'' AFTER `organization_id`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(1) FROM information_schema.columns
  WHERE table_schema = DATABASE() AND table_name = 'qianchuan_ad_report' AND column_name = 'company_id'
);
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `qianchuan_ad_report` ADD COLUMN `company_id` BIGINT(20) NOT NULL DEFAULT 0 COMMENT ''公司ID'' AFTER `account_id`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(1) FROM information_schema.columns
  WHERE table_schema = DATABASE() AND table_name = 'qianchuan_ad_report' AND column_name = 'department_id'
);
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `qianchuan_ad_report` ADD COLUMN `department_id` BIGINT(20) NOT NULL DEFAULT 0 COMMENT ''部门ID'' AFTER `company_id`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(1) FROM information_schema.columns
  WHERE table_schema = DATABASE() AND table_name = 'qianchuan_ad_report' AND column_name = 'trader_id'
);
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `qianchuan_ad_report` ADD COLUMN `trader_id` BIGINT(20) NOT NULL DEFAULT 0 COMMENT ''投手/运营ID'' AFTER `department_id`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(1) FROM information_schema.statistics
  WHERE table_schema = DATABASE() AND table_name = 'qianchuan_ad_report' AND index_name = 'idx_organization_id'
);
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE `qianchuan_ad_report` ADD INDEX `idx_organization_id` (`organization_id`)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(1) FROM information_schema.statistics
  WHERE table_schema = DATABASE() AND table_name = 'qianchuan_ad_report' AND index_name = 'idx_account_id'
);
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE `qianchuan_ad_report` ADD INDEX `idx_account_id` (`account_id`)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(1) FROM information_schema.statistics
  WHERE table_schema = DATABASE() AND table_name = 'qianchuan_ad_report' AND index_name = 'idx_company_id'
);
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE `qianchuan_ad_report` ADD INDEX `idx_company_id` (`company_id`)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(1) FROM information_schema.statistics
  WHERE table_schema = DATABASE() AND table_name = 'qianchuan_ad_report' AND index_name = 'idx_department_id'
);
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE `qianchuan_ad_report` ADD INDEX `idx_department_id` (`department_id`)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @idx_exists = (
  SELECT COUNT(1) FROM information_schema.statistics
  WHERE table_schema = DATABASE() AND table_name = 'qianchuan_ad_report' AND index_name = 'idx_trader_id'
);
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE `qianchuan_ad_report` ADD INDEX `idx_trader_id` (`trader_id`)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
