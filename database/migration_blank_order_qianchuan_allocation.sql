-- 空白单千川归属口径：店铺是否参与分配、最终归属人、异常池。

ALTER TABLE `store`
  ADD COLUMN `blank_order_enabled` TINYINT NOT NULL DEFAULT 1 COMMENT '是否参与空白单归属：0=否 1=是' AFTER `status`,
  ADD COLUMN `blank_order_owner_name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '空白单最终归属人；为空时需补配置' AFTER `blank_order_enabled`;

UPDATE `store`
SET `blank_order_enabled` = 1,
    `blank_order_owner_name` = '王梅雪'
WHERE `platform` = 'douyin'
  AND `store_name` IN ('天书说故事', '利云底层逻辑');

ALTER TABLE `douyin_order_attribution`
  ADD COLUMN `allocated_store_name` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '千川分配命中的投流店铺' AFTER `shop_name`,
  ADD COLUMN `blank_order_owner_name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '空白单最终归属人快照' AFTER `allocated_store_name`;

CREATE TABLE IF NOT EXISTS `blank_order_allocation_exception` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `order_info_id` BIGINT NOT NULL DEFAULT 0 COMMENT '抖音订单明细ID',
  `main_order_no` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '主订单号',
  `sub_order_no` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '子订单号',
  `stat_date` DATE DEFAULT NULL COMMENT '归属日期',
  `shop_name` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '空白单来源店铺',
  `product_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '商品ID',
  `exception_type` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '异常类型',
  `exception_reason` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '异常原因',
  `qianchuan_plan_snapshot` JSON DEFAULT NULL COMMENT '千川计划候选快照',
  `shop_owner_snapshot` JSON DEFAULT NULL COMMENT '店铺归属快照',
  `status` VARCHAR(30) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/resolved_auto/resolved_manual/ignored',
  `resolved_owner` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '人工指定归属人',
  `resolved_by` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '处理人',
  `resolved_at` DATETIME DEFAULT NULL COMMENT '处理时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_source` (`order_info_id`, `exception_type`),
  KEY `idx_status_date` (`status`, `stat_date`),
  KEY `idx_shop_product_date` (`shop_name`, `product_id`, `stat_date`),
  KEY `idx_main_order_no` (`main_order_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='空白单千川归属异常池';
