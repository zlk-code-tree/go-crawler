-- 抖音电商罗盘爬虫数据库表结构
-- 创建时间：2026-03-30

-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS `easyadmin8` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `easyadmin8`;

-- ============================================
-- 表 1: 商品榜单数据表
-- ============================================
DROP TABLE IF EXISTS `tiktok_ranking_list`;

CREATE TABLE `tiktok_ranking_list` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `task_id` VARCHAR(64) NOT NULL COMMENT '任务 ID',
  `rank` INT NOT NULL DEFAULT 0 COMMENT '排名',
  `product_name` VARCHAR(255) NOT NULL COMMENT '商品名称',
  `shop_name` VARCHAR(255) DEFAULT '' COMMENT '店铺名称',
  `payment_amount` VARCHAR(64) DEFAULT '' COMMENT '短视频用户支付金额',
  `click_count` VARCHAR(64) DEFAULT '' COMMENT '短视频点击次数',
  `sales_volume` VARCHAR(64) DEFAULT '' COMMENT '短视频成交件数',
  `delivery_video_num` VARCHAR(64) DEFAULT '' COMMENT '带货视频数',
  `first_shelf_time` VARCHAR(64) DEFAULT '' COMMENT '首次上架时间',
  `min_price` VARCHAR(64) DEFAULT '' COMMENT '最低到手价',
  `first_list` TINYINT NOT NULL DEFAULT 0 COMMENT '是否首次上榜：0-否 1-是',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_task_id` (`task_id`),
  KEY `idx_rank` (`rank`),
  KEY `idx_product_name` (`product_name`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='抖音商品榜单数据表';

-- ============================================
-- 表 2: 视频分析数据表
-- ============================================
DROP TABLE IF EXISTS `video_analysis_data`;

CREATE TABLE `video_analysis_data` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `task_id` VARCHAR(64) NOT NULL COMMENT '任务 ID',
  `product_id` VARCHAR(64) DEFAULT '' COMMENT '商品 ID',
  `product_name` VARCHAR(255) NOT NULL COMMENT '商品名称',
  `video_id` VARCHAR(64) DEFAULT '' COMMENT '视频 ID',
  `video_desc` TEXT COMMENT '短视频信息',
  `payment_amount` VARCHAR(64) DEFAULT '' COMMENT '用户支付金额',
  `expose_user_count` VARCHAR(64) DEFAULT '' COMMENT '商品曝光人数',
  `click_user_count` VARCHAR(64) DEFAULT '' COMMENT '商品点击人数',
  `convert_user_count` VARCHAR(64) DEFAULT '' COMMENT '商品成交人数',
  `click_rate` VARCHAR(32) DEFAULT '' COMMENT '商品点击率 (人数)',
  `click_convert_rate` VARCHAR(32) DEFAULT '' COMMENT '商品点击 - 成交率 (人数)',
  `expose_convert_rate` VARCHAR(32) DEFAULT '' COMMENT '商品曝光 - 成交率 (人数)',
  `order_count` VARCHAR(64) DEFAULT '' COMMENT '商品成交订单数',
  `sales_volume` VARCHAR(64) DEFAULT '' COMMENT '商品成交件数',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_task_id` (`task_id`),
  KEY `idx_product_id` (`product_id`),
  KEY `idx_video_id` (`video_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='视频分析数据表';

-- ============================================
-- 表 3: 爬虫任务记录表
-- ============================================
DROP TABLE IF EXISTS `crawler_task_log`;

CREATE TABLE `crawler_task_log` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `task_id` VARCHAR(64) NOT NULL COMMENT '任务 ID',
  `target_url` VARCHAR(512) NOT NULL COMMENT '目标 URL',
  `login_type` VARCHAR(32) DEFAULT 'qrcode' COMMENT '登录方式：qrcode/password',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '任务状态：0-进行中 1-成功 2-失败',
  `product_count` INT DEFAULT 0 COMMENT '抓取商品数量',
  `video_count` INT DEFAULT 0 COMMENT '抓取视频数量',
  `error_message` TEXT COMMENT '错误信息',
  `start_time` DATETIME DEFAULT NULL COMMENT '开始时间',
  `end_time` DATETIME DEFAULT NULL COMMENT '结束时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_task_id` (`task_id`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='爬虫任务日志表';

-- ============================================
-- 初始化数据
-- ============================================
INSERT INTO `crawler_task_log` (`task_id`, `target_url`, `login_type`, `status`) 
VALUES ('init', 'https://compass.jinritemai.com/shop', 'qrcode', 1)
ON DUPLICATE KEY UPDATE `created_at` = CURRENT_TIMESTAMP;

-- ============================================
-- 查询示例
-- ============================================
-- 查询最新的商品榜单数据
-- SELECT * FROM tiktok_ranking_list ORDER BY created_at DESC LIMIT 200;

-- 查询某个任务的所有商品数据
-- SELECT * FROM tiktok_ranking_list WHERE task_id = 'your_task_id' ORDER BY rank;

-- 查询某个商品的所有视频分析数据
-- SELECT * FROM video_analysis_data WHERE product_name = '商品名称' ORDER BY created_at DESC;

-- 统计每日抓取任务
-- SELECT DATE(created_at) as date, COUNT(*) as task_count FROM crawler_task_log GROUP BY DATE(created_at) ORDER BY date DESC;
