-- ============================================================
-- 达人订单匹配 - 暂存表（多平台支持）
-- ============================================================

-- 1. 普通订单暂存表
CREATE TABLE IF NOT EXISTS `match_orders` (
  `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
  `platform` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '平台：douyin/kuaishou/shipinhao',
  `order_no` VARCHAR(100) NOT NULL COMMENT '订单号',
  `shop_name` VARCHAR(200) DEFAULT '' COMMENT '店铺名称',
  `product_name` VARCHAR(500) DEFAULT '' COMMENT '商品名称',
  `product_num` INT DEFAULT 0 COMMENT '商品数量',
  `order_pay_amount` DECIMAL(12,2) DEFAULT 0 COMMENT '付款金额',
  `order_status` VARCHAR(50) DEFAULT '' COMMENT '订单状态',
  `talent_id` VARCHAR(100) DEFAULT '' COMMENT '达人ID(可能为空)',
  `talent_name` VARCHAR(100) DEFAULT '' COMMENT '达人昵称(可能为空)',
  `pay_finish_time` VARCHAR(50) DEFAULT '' COMMENT '付款时间',
  `status` TINYINT DEFAULT 0 COMMENT '匹配状态：0=未匹配 1=已匹配',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY `uk_platform_order` (`platform`, `order_no`),
  INDEX `idx_talent_id` (`talent_id`),
  INDEX `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='普通订单暂存表';

-- 2. 精选联盟订单暂存表
CREATE TABLE IF NOT EXISTS `match_jingxuan_orders` (
  `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
  `platform` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '平台：douyin/kuaishou/shipinhao',
  `order_no` VARCHAR(100) NOT NULL COMMENT '精选联盟订单号',
  `talent_id` VARCHAR(100) NOT NULL COMMENT '达人ID',
  `talent_name` VARCHAR(100) DEFAULT '' COMMENT '达人昵称',
  `product_name` VARCHAR(500) DEFAULT '' COMMENT '商品名称',
  `commission_rate` VARCHAR(20) DEFAULT '' COMMENT '佣金比例',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY `uk_platform_order` (`platform`, `order_no`),
  INDEX `idx_talent_id` (`talent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='精选联盟订单暂存表';

-- 3. 匹配结果表
CREATE TABLE IF NOT EXISTS `match_result` (
  `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
  `platform` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '平台',
  `order_no` VARCHAR(100) NOT NULL COMMENT '订单号',
  `talent_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '匹配到达人ID',
  `talent_name` VARCHAR(100) DEFAULT '' COMMENT '匹配到达人昵称',
  `match_status` TINYINT DEFAULT 0 COMMENT '0=已匹配待回填 1=已回填到生产表',
  `matched_table` VARCHAR(50) DEFAULT '' COMMENT '回填的订单表名',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY `uk_platform_order` (`platform`, `order_no`),
  INDEX `idx_status` (`match_status`),
  INDEX `idx_platform` (`platform`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='达人订单匹配结果表';

-- ============================================================
-- 如已有旧表，执行以下 ALTER 语句：
-- ALTER TABLE match_orders ADD COLUMN platform VARCHAR(20) NOT NULL DEFAULT '' COMMENT '平台' AFTER id;
-- ALTER TABLE match_orders DROP INDEX uk_order_no, ADD UNIQUE KEY uk_platform_order (platform, order_no);
-- ALTER TABLE match_orders ADD COLUMN status TINYINT DEFAULT 0 COMMENT '匹配状态：0=未匹配 1=已匹配' AFTER pay_finish_time;
-- ALTER TABLE match_orders ADD INDEX idx_status (status);
-- ALTER TABLE match_jingxuan_orders ADD COLUMN platform VARCHAR(20) NOT NULL DEFAULT '' COMMENT '平台' AFTER id;
-- ALTER TABLE match_jingxuan_orders DROP INDEX uk_order_no, ADD UNIQUE KEY uk_platform_order (platform, order_no);
-- ALTER TABLE match_result ADD COLUMN platform VARCHAR(20) NOT NULL DEFAULT '' COMMENT '平台' AFTER id;
-- ALTER TABLE match_result DROP INDEX uk_order_no, ADD UNIQUE KEY uk_platform_order (platform, order_no);
-- ============================================================
