-- 千川达人维度日报表
-- 存储按达人(aweme_id)拆分的千川广告消耗数据
-- 执行时间：2026-06-30

CREATE TABLE IF NOT EXISTS `qianchuan_author_daily_report` (
    `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `advertiser_id` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '千川广告主ID',
    `account_id` INT NOT NULL DEFAULT 0 COMMENT '内部账号ID（关联account.id）',
    `company_id` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '公司ID',
    `department_id` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '部门ID',
    `trader_id` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '投手ID',
    `aweme_id` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '抖音账号ID（达人UID）',
    `aweme_name` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '抖音账号名称',
    `stat_cost` DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '消耗（元）',
    `pay_order_gmv` DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '成交金额',
    `pay_order_count` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '成交订单数',
    `stat_datetime` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '统计日期',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_adv_aweme_date` (`advertiser_id`, `aweme_id`, `stat_datetime`),
    KEY `idx_aweme_id` (`aweme_id`),
    KEY `idx_stat_datetime` (`stat_datetime`),
    KEY `idx_account_id` (`account_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='千川达人维度日报表';
