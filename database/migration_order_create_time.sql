-- 为订单表添加 create_time 字段，记录数据首次入库时间
-- 执行时间：2026-06-24

ALTER TABLE order_info ADD COLUMN create_time datetime DEFAULT CURRENT_TIMESTAMP COMMENT '数据入库时间' AFTER expect_arrive_time;
ALTER TABLE kuaishou_orders ADD COLUMN create_time datetime DEFAULT CURRENT_TIMESTAMP COMMENT '数据入库时间';
ALTER TABLE shipinhao_orders ADD COLUMN create_time datetime DEFAULT CURRENT_TIMESTAMP COMMENT '数据入库时间';

-- 回填历史数据（将 order_submit_time 作为近似值）
UPDATE order_info SET create_time = order_submit_time WHERE create_time IS NULL;
UPDATE kuaishou_orders SET create_time = order_submit_time WHERE create_time IS NULL;
UPDATE shipinhao_orders SET create_time = order_submit_time WHERE create_time IS NULL;
