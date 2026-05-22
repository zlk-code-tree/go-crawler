-- =====================================================
-- 投手管理表
-- =====================================================

-- 1. 投手主表
CREATE TABLE IF NOT EXISTS `trader` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `name` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '投手名称',
    `company_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属公司ID（关联sys_company.id）',
    `group_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属部门ID（关联sys_department.id）',
    `role` TINYINT NOT NULL DEFAULT 1 COMMENT '身份：1=投手，2=运营',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0=禁用，1=启用',
    `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    INDEX `idx_company_id` (`company_id`),
    INDEX `idx_group_id` (`group_id`),
    INDEX `idx_role` (`role`),
    INDEX `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='投手管理表';

-- 如果表已存在，执行以下 ALTER 语句添加 company_id 字段：
-- ALTER TABLE `trader` ADD COLUMN `company_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属公司ID（关联sys_company.id）' AFTER `name`;
-- ALTER TABLE `trader` ADD INDEX `idx_company_id` (`company_id`);

-- 2. 投手-店铺关联表（多对多）
CREATE TABLE IF NOT EXISTS `trader_shop_relation` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `trader_id` BIGINT UNSIGNED NOT NULL COMMENT '投手ID',
    `shop_id` INT NOT NULL COMMENT '店铺ID',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE INDEX `uk_trader_shop` (`trader_id`, `shop_id`),
    INDEX `idx_trader_id` (`trader_id`),
    INDEX `idx_shop_id` (`shop_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='投手-店铺关联表';
