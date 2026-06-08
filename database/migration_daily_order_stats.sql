-- 抖音订单每日聚合统计表
CREATE TABLE IF NOT EXISTS `daily_order_stats` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `stat_date` date NOT NULL COMMENT '统计日期',
  `shop_name` varchar(100) NOT NULL DEFAULT '' COMMENT '店铺名称',
  `total_order_count` int NOT NULL DEFAULT 0 COMMENT '总订单数',
  `valid_order_count` int NOT NULL DEFAULT 0 COMMENT '有效订单数',
  `invalid_order_count` int NOT NULL DEFAULT 0 COMMENT '无效订单数',
  `valid_pay_amount` decimal(14,2) NOT NULL DEFAULT 0.00 COMMENT '有效定价总额(product_price*num)',
  `product_cost_amount` decimal(14,2) NOT NULL DEFAULT 0.00 COMMENT '货款(base_price*num)',
  `reserve_amount` decimal(14,2) NOT NULL DEFAULT 0.00 COMMENT '留底(product_price*num*3%%)',
  `refund_cost_amount` decimal(14,2) NOT NULL DEFAULT 0.00 COMMENT '退款成本',
  `refund_order_count` int NOT NULL DEFAULT 0 COMMENT '退款订单数',
  `shop_service_fee` decimal(14,2) NOT NULL DEFAULT 0.00 COMMENT '小店服务费(2%%)',
  `commission` decimal(14,2) NOT NULL DEFAULT 0.00 COMMENT '佣金',
  `valid_order_cost_match` int NOT NULL DEFAULT 0 COMMENT '成本已匹配数',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_date_shop` (`stat_date`, `shop_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='抖音订单每日聚合统计';
