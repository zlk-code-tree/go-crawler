-- 达人佣金统计新增类目：给 finance_daily_stats 表增加新字段
-- 与抖音订单列表统计类目对齐

ALTER TABLE `finance_daily_stats`
  ADD COLUMN `total_order_count` int(11) NOT NULL DEFAULT 0 COMMENT '总订单数' AFTER `platform`,
  ADD COLUMN `invalid_order_count` int(11) NOT NULL DEFAULT 0 COMMENT '无效订单数' AFTER `order_count`,
  ADD COLUMN `refund_cost_amount` decimal(12,2) NOT NULL DEFAULT 0.00 COMMENT '发货后退款成本' AFTER `commission`,
  ADD COLUMN `refund_order_count` int(11) NOT NULL DEFAULT 0 COMMENT '退款订单数' AFTER `refund_cost_amount`,
  ADD COLUMN `shop_service_fee` decimal(12,2) NOT NULL DEFAULT 0.00 COMMENT '小店服务费' AFTER `refund_order_count`,
  ADD COLUMN `ad_cost_amount` decimal(12,2) NOT NULL DEFAULT 0.00 COMMENT '广告费消耗' AFTER `shop_service_fee`,
  ADD COLUMN `valid_order_cost_match` int(11) NOT NULL DEFAULT 0 COMMENT '有效订单成本已匹配数' AFTER `ad_cost_amount`,
  ADD COLUMN `commission_order_count` int(11) NOT NULL DEFAULT 0 COMMENT '有佣金的订单数' AFTER `valid_order_cost_match`,
  ADD COLUMN `realtime_profit` decimal(12,2) NOT NULL DEFAULT 0.00 COMMENT '实时利润' AFTER `commission_order_count`;
