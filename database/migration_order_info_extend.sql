-- order_info 补充字段：支持抖店API轮询保存更多数据
ALTER TABLE `order_info`
  ADD COLUMN IF NOT EXISTS `order_create_time` DATETIME DEFAULT NULL COMMENT '订单创建时间',
  ADD COLUMN IF NOT EXISTS `order_finish_time` DATETIME DEFAULT NULL COMMENT '订单完成时间',
  ADD COLUMN IF NOT EXISTS `post_amount` DECIMAL(12,2) DEFAULT 0.00 COMMENT '运费',
  ADD COLUMN IF NOT EXISTS `post_receiver` VARCHAR(100) DEFAULT '' COMMENT '收件人',
  ADD COLUMN IF NOT EXISTS `post_tel` VARCHAR(50) DEFAULT '' COMMENT '收件人电话',
  ADD COLUMN IF NOT EXISTS `post_addr` VARCHAR(500) DEFAULT '' COMMENT '收件地址',
  ADD COLUMN IF NOT EXISTS `promotion_amount` DECIMAL(12,2) DEFAULT 0.00 COMMENT '优惠总金额',
  ADD COLUMN IF NOT EXISTS `buyer_words` VARCHAR(500) DEFAULT '' COMMENT '买家留言',
  ADD COLUMN IF NOT EXISTS `seller_words` VARCHAR(500) DEFAULT '' COMMENT '商家备注',
  ADD COLUMN IF NOT EXISTS `seller_code` VARCHAR(100) DEFAULT '' COMMENT '商家后台商品编码';

-- 添加唯一索引防止重复插入（如果不存在）
ALTER TABLE `order_info`
  ADD UNIQUE INDEX IF NOT EXISTS `uk_platform_order` (`platform`, `main_order_no`, `order_no`);
