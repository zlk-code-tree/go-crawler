-- 微信小店订单 API 拉取支持
-- 注意：按本次需求，首次接入前清空当前视频号订单表。
-- 执行本脚本前请确认 shipinhao_orders 旧数据已不再需要。

CREATE TABLE IF NOT EXISTS `wechat_shop_auth` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `store_id` INT NOT NULL COMMENT '关联 store.id',
  `appid` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '微信小店/服务商调用 appid',
  `app_secret` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '微信小店/服务商调用 appsecret',
  `access_token` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '接口调用凭证',
  `expires_at` DATETIME DEFAULT NULL COMMENT 'access_token 过期时间',
  `last_pull_time` DATETIME DEFAULT NULL COMMENT '最近成功拉取时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_wechat_shop_auth_store` (`store_id`),
  KEY `idx_wechat_shop_auth_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='微信小店接口授权配置';

TRUNCATE TABLE `shipinhao_orders`;

SET @db_name = DATABASE();

SELECT COUNT(*) INTO @col_exists FROM information_schema.columns
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND column_name = 'sku_id';
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD COLUMN `sku_id` VARCHAR(100) NOT NULL DEFAULT '''' COMMENT ''微信小店SKU ID'' AFTER `product_id`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @col_exists FROM information_schema.columns
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND column_name = 'out_product_id';
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD COLUMN `out_product_id` VARCHAR(100) NOT NULL DEFAULT '''' COMMENT ''外部商品ID'' AFTER `sku_id`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @col_exists FROM information_schema.columns
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND column_name = 'out_sku_id';
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD COLUMN `out_sku_id` VARCHAR(100) NOT NULL DEFAULT '''' COMMENT ''外部SKU ID'' AFTER `out_product_id`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @col_exists FROM information_schema.columns
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND column_name = 'openid';
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD COLUMN `openid` VARCHAR(128) NOT NULL DEFAULT '''' COMMENT ''买家openid'' AFTER `receiver_phone`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @col_exists FROM information_schema.columns
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND column_name = 'wechat_order_status';
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD COLUMN `wechat_order_status` INT NOT NULL DEFAULT 0 COMMENT ''微信小店订单状态码'' AFTER `order_status`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @col_exists FROM information_schema.columns
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND column_name = 'wechat_order_status_desc';
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD COLUMN `wechat_order_status_desc` VARCHAR(50) NOT NULL DEFAULT '''' COMMENT ''微信小店订单状态描述'' AFTER `wechat_order_status`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @col_exists FROM information_schema.columns
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND column_name = 'order_update_time';
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD COLUMN `order_update_time` DATETIME DEFAULT NULL COMMENT ''订单更新时间'' AFTER `order_submit_time`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @col_exists FROM information_schema.columns
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND column_name = 'source_info';
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD COLUMN `source_info` JSON DEFAULT NULL COMMENT ''订单来源信息'' AFTER `flow_channel`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @col_exists FROM information_schema.columns
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND column_name = 'commission_info';
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD COLUMN `commission_info` JSON DEFAULT NULL COMMENT ''分账/佣金信息'' AFTER `source_info`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @col_exists FROM information_schema.columns
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND column_name = 'raw_json';
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD COLUMN `raw_json` JSON DEFAULT NULL COMMENT ''微信小店订单原始详情JSON'' AFTER `expect_arrive_time`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @col_exists FROM information_schema.columns
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND column_name = 'api_sync_time';
SET @sql = IF(@col_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD COLUMN `api_sync_time` DATETIME DEFAULT NULL COMMENT ''API同步时间'' AFTER `raw_json`', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @idx_exists FROM information_schema.statistics
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND index_name = 'idx_sku_id';
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD INDEX `idx_sku_id` (`sku_id`)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @idx_exists FROM information_schema.statistics
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND index_name = 'idx_wechat_order_status';
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD INDEX `idx_wechat_order_status` (`wechat_order_status`)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @idx_exists FROM information_schema.statistics
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND index_name = 'idx_api_sync_time';
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD INDEX `idx_api_sync_time` (`api_sync_time`)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
