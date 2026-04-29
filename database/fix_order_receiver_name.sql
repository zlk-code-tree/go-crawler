-- 修复订单表 receiver_name 字段长度不足的问题
-- 执行时间：2026-04-27

-- 查看当前字段定义
-- SHOW CREATE TABLE order_info;

-- 修改 receiver_name 字段长度从默认改为 100
ALTER TABLE order_info 
MODIFY COLUMN receiver_name VARCHAR(100) DEFAULT '' COMMENT '收件人姓名';

-- 同时修改其他可能超长的字段
ALTER TABLE order_info 
MODIFY COLUMN receiver_phone VARCHAR(20) DEFAULT '' COMMENT '收件人手机号';

ALTER TABLE order_info 
MODIFY COLUMN province VARCHAR(50) DEFAULT '' COMMENT '省';

ALTER TABLE order_info 
MODIFY COLUMN city VARCHAR(50) DEFAULT '' COMMENT '市';

ALTER TABLE order_info 
MODIFY COLUMN district VARCHAR(50) DEFAULT '' COMMENT '区';

-- 验证修改结果
-- DESC order_info;
