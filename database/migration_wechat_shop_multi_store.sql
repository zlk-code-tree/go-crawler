-- 微信小店多店铺支持补充迁移
-- 1. 授权表按 store.id 关联
-- 2. 订单表增加 order_item_key，支持多店铺幂等 upsert

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

ALTER TABLE `store`
  ADD COLUMN IF NOT EXISTS `last_poll_time` DATETIME DEFAULT NULL COMMENT '上次成功拉取订单的时间' AFTER `author`;

ALTER TABLE `shipinhao_orders`
  ADD COLUMN IF NOT EXISTS `order_item_key` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '订单明细幂等键' AFTER `order_no`;

SET @db_name = DATABASE();
UPDATE `shipinhao_orders`
SET `order_item_key` = MD5(CONCAT('legacy|', id))
WHERE `order_item_key` = '' OR `order_item_key` IS NULL;

SELECT COUNT(*) INTO @idx_exists FROM information_schema.statistics
WHERE table_schema = @db_name AND table_name = 'shipinhao_orders' AND index_name = 'uk_order_item_key';
SET @sql = IF(@idx_exists = 0, 'ALTER TABLE `shipinhao_orders` ADD UNIQUE INDEX `uk_order_item_key` (`order_item_key`)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
