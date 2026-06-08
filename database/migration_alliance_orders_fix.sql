-- alliance_orders 字段修正（匹配API实际返回的所有字段）
ALTER TABLE `alliance_orders`
  MODIFY COLUMN `order_status` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '订单状态',
  MODIFY COLUMN `commission_rate` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '佣金比例(字符串)',
  ADD COLUMN IF NOT EXISTS `author_account` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '达人账号' AFTER `commission`,
  ADD COLUMN IF NOT EXISTS `estimated_comission` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '预估佣金' AFTER `author_account`,
  ADD COLUMN IF NOT EXISTS `real_comission` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '实际佣金' AFTER `estimated_comission`,
  ADD COLUMN IF NOT EXISTS `total_pay_amount` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '支付金额' AFTER `real_comission`,
  ADD COLUMN IF NOT EXISTS `shop_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '店铺ID' AFTER `total_pay_amount`,
  ADD COLUMN IF NOT EXISTS `alliance_biz_type` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '联盟业务类型' AFTER `shop_id`,
  ADD COLUMN IF NOT EXISTS `phase_id` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '阶段ID' AFTER `alliance_biz_type`,
  ADD COLUMN IF NOT EXISTS `short_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '短ID' AFTER `phase_id`;
