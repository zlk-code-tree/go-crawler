-- 千川广告报表新增成交金额和成交订单数字段
-- 用于利润核算中的收入侧参考
-- 执行时间：2026-06-26

ALTER TABLE `qianchuan_ad_report`
  ADD COLUMN `pay_order_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '成交金额（元）' AFTER `convert_cost`,
  ADD COLUMN `pay_order_count` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '成交订单数' AFTER `pay_order_amount`;
