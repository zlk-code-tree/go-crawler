-- shop 表新增绑定状态字段
ALTER TABLE `shop` ADD COLUMN `bind_status` TINYINT NOT NULL DEFAULT 0 COMMENT '绑定状态：0=未绑定，1=被绑定' AFTER `member_name`;
ALTER TABLE `shop` ADD INDEX `idx_bind_status` (`bind_status`);

-- 初始化已有绑定数据
UPDATE `shop` SET `bind_status` = 1 WHERE `id` IN (SELECT `shop_id` FROM `trader_shop_relation`);
