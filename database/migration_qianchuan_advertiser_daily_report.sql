-- 千川广告主日结报表（账户维度）
-- 替代原有 qianchuan_ad_report（计划维度），数据来自 qianchuan/report/advertiser/get API
CREATE TABLE IF NOT EXISTS `qianchuan_advertiser_daily_report` (
    `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
    `advertiser_id` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '千川广告主ID',
    `stat_datetime` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '统计日期 yyyy-MM-dd',

    -- 核心消耗指标
    `stat_cost` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '消耗（元）',
    `show_cnt` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '展示数',
    `click_cnt` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '点击数',
    `ctr` DECIMAL(10,4) NOT NULL DEFAULT 0.0000 COMMENT '点击率',
    `cpc_platform` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '平均点击单价',
    `cpm_platform` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '千次展示费用',

    -- 转化指标
    `convert_cnt` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '转化数',
    `convert_cost` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '转化成本',
    `convert_rate` DECIMAL(10,4) NOT NULL DEFAULT 0.0000 COMMENT '转化率',
    `deep_convert_cnt` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '深度转化数',
    `deep_convert_cost` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '深度转化成本',

    -- 成交指标
    `pay_order_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '成交金额（元）',
    `pay_order_count` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '成交订单数',
    `pay_order_cost_per_order` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '成交订单成本',
    `pay_order_coupon_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '成交优惠券金额',

    -- 下单指标
    `create_order_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '下单金额（元）',
    `create_order_count` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '下单数',
    `create_order_coupon_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '下单优惠券金额',
    `create_order_roi` DECIMAL(10,4) NOT NULL DEFAULT 0.0000 COMMENT '下单ROI',

    -- 预售指标
    `prepay_order_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '预售金额',
    `prepay_order_count` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '预售订单数',
    `prepay_and_pay_order_roi` DECIMAL(10,4) NOT NULL DEFAULT 0.0000 COMMENT '预售+成交ROI',

    -- 7日/14日归因指标
    `all_order_pay_gmv_7days` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '7日全部订单支付GMV',
    `all_order_pay_count_7days` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '7日全部订单支付数',
    `all_order_pay_roi_7days` DECIMAL(10,4) NOT NULL DEFAULT 0.0000 COMMENT '7日全部订单ROI',
    `all_order_refund_amount_7d` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '7日退款金额',
    `all_order_refund_count_7d` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '7日退款数',
    `all_order_refund_amount_14d` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '14日退款金额',
    `all_order_refund_count_14d` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '14日退款数',

    -- 互动指标
    `dy_follow` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '新增关注数',
    `total_play` BIGINT(20) NOT NULL DEFAULT 0 COMMENT '播放数',

    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_adv_date` (`advertiser_id`, `stat_datetime`),
    KEY `idx_stat_datetime` (`stat_datetime`),
    KEY `idx_advertiser_id` (`advertiser_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='千川广告主日结报表（账户维度）';
