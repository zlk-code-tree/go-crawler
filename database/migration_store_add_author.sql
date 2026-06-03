-- store 表新增授权字段
ALTER TABLE `store` ADD COLUMN IF NOT EXISTS `author` TINYINT NOT NULL DEFAULT 0 COMMENT '授权状态：0-未授权 1-已授权' AFTER `status`;
-- 已有的抖音店铺默认标记为未授权，需要手动改为已授权
