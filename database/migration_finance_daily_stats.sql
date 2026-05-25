-- =====================================================
-- 财务管理 - 达人佣金日统计预聚合表
-- 执行时间: 2026-05-25
-- =====================================================

-- 1. 日佣金统计表（预聚合）
CREATE TABLE IF NOT EXISTS `finance_daily_stats` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `stat_date` DATE NOT NULL COMMENT '统计日期',

    -- 组织维度（0表示该维度不参与聚合）
    `company_id` BIGINT UNSIGNED DEFAULT 0 COMMENT '公司ID',
    `department_id` BIGINT UNSIGNED DEFAULT 0 COMMENT '部门ID',
    `trader_id` BIGINT UNSIGNED DEFAULT 0 COMMENT '投手ID',
    `shop_id` INT UNSIGNED DEFAULT 0 COMMENT '店铺ID',

    -- 平台维度
    `platform` VARCHAR(20) DEFAULT '' COMMENT '平台：douyin/kuaishou/shipinhao/空=全部',

    -- 统计指标
    `order_count` INT DEFAULT 0 COMMENT '有效订单数',
    `product_num` INT DEFAULT 0 COMMENT '商品数量',
    `pay_amount` DECIMAL(14,2) DEFAULT 0 COMMENT '订单应付金额',
    `cost_amount` DECIMAL(14,2) DEFAULT 0 COMMENT '货款金额（代发价）',
    `reserve_amount` DECIMAL(14,2) DEFAULT 0 COMMENT '留底金额',
    `commission` DECIMAL(14,2) DEFAULT 0 COMMENT '佣金',

    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

    PRIMARY KEY (`id`),

    -- 唯一索引：防止重复统计
    UNIQUE KEY `uk_stats` (`stat_date`, `company_id`, `department_id`, `trader_id`, `shop_id`, `platform`),

    -- 查询索引
    INDEX `idx_date_company` (`stat_date`, `company_id`),
    INDEX `idx_date_dept` (`stat_date`, `department_id`),
    INDEX `idx_date_trader` (`stat_date`, `trader_id`),
    INDEX `idx_date_shop` (`stat_date`, `shop_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='日佣金统计预聚合表';

-- 2. 订单表索引优化（如果不存在）
-- 抖音订单表
ALTER TABLE `order_info` ADD INDEX IF NOT EXISTS `idx_talent_paytime` (`talent_id`, `pay_finish_time`);
ALTER TABLE `order_info` ADD INDEX IF NOT EXISTS `idx_paytime_status` (`pay_finish_time`, `after_sale_status`);

-- 快手订单表
ALTER TABLE `kuaishou_orders` ADD INDEX IF NOT EXISTS `idx_talent_paytime` (`talent_id`, `pay_finish_time`);
ALTER TABLE `kuaishou_orders` ADD INDEX IF NOT EXISTS `idx_paytime_status` (`pay_finish_time`, `after_sale_status`);

-- 视频号订单表
ALTER TABLE `shipinhao_orders` ADD INDEX IF NOT EXISTS `idx_talent_paytime` (`talent_id`, `pay_finish_time`);
ALTER TABLE `shipinhao_orders` ADD INDEX IF NOT EXISTS `idx_paytime_status` (`pay_finish_time`, `after_sale_status`);

-- 3. 店铺表索引优化
ALTER TABLE `shop` ADD INDEX IF NOT EXISTS `idx_uid` (`uid`);
ALTER TABLE `shop` ADD INDEX IF NOT EXISTS `idx_company_dept` (`company_id`, `department_id`);
