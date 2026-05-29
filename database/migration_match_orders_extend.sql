-- match_orders 表新增字段，支持快手/视频号平台专属匹配条件
ALTER TABLE match_orders ADD COLUMN after_sale_status VARCHAR(20) DEFAULT '' COMMENT '售后状态' AFTER order_status;
ALTER TABLE match_orders ADD COLUMN account_type VARCHAR(50) DEFAULT '' COMMENT '带货账号类型' AFTER talent_name;
