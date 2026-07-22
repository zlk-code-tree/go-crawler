-- 千川计划财务统计查询索引。
-- 计划与抖店订单按“统计日/下单日 + 商品ID”匹配归因。

SET @db = DATABASE();

SELECT COUNT(*) INTO @idx_exists
FROM information_schema.statistics
WHERE table_schema = @db AND table_name = 'qianchuan_ad_report' AND index_name = 'idx_account_product_stat_date';
SET @sql = IF(@idx_exists = 0,
  'ALTER TABLE `qianchuan_ad_report` ADD INDEX `idx_account_product_stat_date` (`account_id`, `product_id`, `stat_datetime`)',
  'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @idx_exists
FROM information_schema.statistics
WHERE table_schema = @db AND table_name = 'qianchuan_ad_report' AND index_name = 'idx_product_stat_advertiser_ad';
SET @sql = IF(@idx_exists = 0,
  'ALTER TABLE `qianchuan_ad_report` ADD INDEX `idx_product_stat_advertiser_ad` (`product_id`, `stat_datetime`, `advertiser_id`, `ad_id`)',
  'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @idx_exists
FROM information_schema.statistics
WHERE table_schema = @db AND table_name = 'qianchuan_ad_report' AND index_name = 'idx_account_stat_advertiser';
SET @sql = IF(@idx_exists = 0,
  'ALTER TABLE `qianchuan_ad_report` ADD INDEX `idx_account_stat_advertiser` (`account_id`, `stat_datetime`, `advertiser_id`, `ad_id`)',
  'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @idx_exists
FROM information_schema.statistics
WHERE table_schema = @db AND table_name = 'qianchuan_ad_report' AND index_name = 'idx_advertiser_stat_ad';
SET @sql = IF(@idx_exists = 0,
  'ALTER TABLE `qianchuan_ad_report` ADD INDEX `idx_advertiser_stat_ad` (`advertiser_id`, `stat_datetime`, `ad_id`)',
  'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @idx_exists
FROM information_schema.statistics
WHERE table_schema = @db AND table_name = 'order_info' AND index_name = 'idx_shop_product_pay_time';
SET @sql = IF(@idx_exists = 0,
  'ALTER TABLE `order_info` ADD INDEX `idx_shop_product_pay_time` (`shop_name`, `product_id`, `pay_finish_time`)',
  'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SELECT COUNT(*) INTO @idx_exists
FROM information_schema.statistics
WHERE table_schema = @db AND table_name = 'order_info' AND index_name = 'idx_product_submit_shop_order';
SET @sql = IF(@idx_exists = 0,
  'ALTER TABLE `order_info` ADD INDEX `idx_product_submit_shop_order` (`product_id`, `order_submit_time`, `shop_name`, `main_order_no`)',
  'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
