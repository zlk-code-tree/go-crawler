-- 店铺借用账号归属规则表
-- 用于处理"店铺A借用达人B的账号跑数据，订单归属给指定达人C"的场景
CREATE TABLE IF NOT EXISTS `shop_account_attribution` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `shop_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '店铺ID（store.store_id，优先用于稳定匹配）',
    `shop_name` VARCHAR(200) NOT NULL COMMENT '店铺名称',
    `source_account_uid` VARCHAR(100) NOT NULL COMMENT '借用的达人UID',
    `source_account_name` VARCHAR(200) DEFAULT '' COMMENT '借用的达人名称（便于识别）',
    `target_type` VARCHAR(20) NOT NULL DEFAULT 'operator' COMMENT '归属类型：operator=运营组, trader=投手',
    `target_value` VARCHAR(200) NOT NULL COMMENT '归属目标：运营组名称或投手ID',
    `remark` VARCHAR(500) DEFAULT '' COMMENT '备注',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0=禁用 1=启用',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_shop_account` (`shop_name`, `source_account_uid`),
    INDEX `idx_shop_id` (`shop_id`),
    INDEX `idx_shop_name` (`shop_name`),
    INDEX `idx_account_uid` (`source_account_uid`),
    INDEX `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='店铺借用账号归属规则表';
