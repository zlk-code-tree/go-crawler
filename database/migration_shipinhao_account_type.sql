-- 视频号订单表新增带货账号类型字段
ALTER TABLE shipinhao_orders ADD COLUMN account_type VARCHAR(50) DEFAULT '' COMMENT '带货账号类型' AFTER talent_name;
