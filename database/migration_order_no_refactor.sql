-- 快手/视频号订单表字段重构：main_order_no + sub_order_no → order_no
-- 执行前请备份数据

-- ============ 快手订单表 ============
ALTER TABLE kuaishou_orders ADD COLUMN order_no VARCHAR(100) DEFAULT '' COMMENT '订单号' AFTER shop_name;
ALTER TABLE kuaishou_orders ADD INDEX idx_order_no (order_no);
UPDATE kuaishou_orders SET order_no = sub_order_no WHERE order_no = '' AND sub_order_no != '';
ALTER TABLE kuaishou_orders DROP INDEX idx_sub_order_no;
ALTER TABLE kuaishou_orders DROP INDEX idx_main_order_no;
ALTER TABLE kuaishou_orders DROP COLUMN main_order_no;
ALTER TABLE kuaishou_orders DROP COLUMN sub_order_no;

-- ============ 视频号订单表 ============
ALTER TABLE shipinhao_orders ADD COLUMN order_no VARCHAR(100) DEFAULT '' COMMENT '订单号' AFTER shop_name;
ALTER TABLE shipinhao_orders ADD INDEX idx_order_no (order_no);
UPDATE shipinhao_orders SET order_no = sub_order_no WHERE order_no = '' AND sub_order_no != '';
ALTER TABLE shipinhao_orders DROP INDEX idx_sub_order_no;
ALTER TABLE shipinhao_orders DROP INDEX idx_main_order_no;
ALTER TABLE shipinhao_orders DROP COLUMN main_order_no;
ALTER TABLE shipinhao_orders DROP COLUMN sub_order_no;
