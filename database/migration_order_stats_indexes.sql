-- 订单统计性能优化索引
-- 目的：
-- 1. 支持抖音订单统计默认“今日”场景按 order_submit_time 范围快速过滤
-- 2. 支持联盟订单按 shop_name + pay_time 的统计场景

SET @db_name = DATABASE();

SET @sql_order_submit_time = (
  SELECT IF(
    EXISTS(
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = @db_name
        AND table_name = 'order_info'
        AND index_name = 'idx_order_submit_time'
    ),
    'SELECT ''idx_order_submit_time already exists''',
    'ALTER TABLE order_info ADD INDEX idx_order_submit_time (order_submit_time)'
  )
);
PREPARE stmt_order_submit_time FROM @sql_order_submit_time;
EXECUTE stmt_order_submit_time;
DEALLOCATE PREPARE stmt_order_submit_time;

SET @sql_alliance_shop_pay_time = (
  SELECT IF(
    EXISTS(
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = @db_name
        AND table_name = 'alliance_orders'
        AND index_name = 'idx_shop_pay_time'
    ),
    'SELECT ''idx_shop_pay_time already exists''',
    'ALTER TABLE alliance_orders ADD INDEX idx_shop_pay_time (shop_name, pay_time)'
  )
);
PREPARE stmt_alliance_shop_pay_time FROM @sql_alliance_shop_pay_time;
EXECUTE stmt_alliance_shop_pay_time;
DEALLOCATE PREPARE stmt_alliance_shop_pay_time;
