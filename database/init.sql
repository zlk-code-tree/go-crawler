-- =====================================================
-- 抖音电商罗盘爬虫系统 - 数据库初始化脚本
-- 字符集：utf8mb4_unicode_ci（支持中文和 emoji）
-- =====================================================

-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS `go_crawler` 
DEFAULT CHARACTER SET utf8mb4 
COLLATE utf8mb4_unicode_ci;

USE `go_crawler`;

-- -----------------------------------------------------
-- 表 1: 商品榜单数据表 (tiktok_ranking_list)
-- -----------------------------------------------------
DROP TABLE IF EXISTS `tiktok_ranking_list`;

CREATE TABLE `tiktok_ranking_list` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `task_id` VARCHAR(64) NOT NULL COMMENT '爬虫任务 ID',
  `product_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '商品 ID',
  `shop_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '店铺 ID',
  `product_url` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '商品详情页',
  `rank` INT NOT NULL DEFAULT 0 COMMENT '排名',
  `product_name` VARCHAR(255) NOT NULL COMMENT '商品名称',
  `image_url` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '商品主图地址',
  `shop_name` VARCHAR(255) DEFAULT NULL COMMENT '店铺名称',
  `payment_amount` VARCHAR(32) DEFAULT NULL COMMENT '短视频用户支付金额',
  `click_count` VARCHAR(32) DEFAULT NULL COMMENT '短视频点击次数',
  `click_convert_rate` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '点击成交转化率',
  `sales_volume` VARCHAR(32) DEFAULT NULL COMMENT '短视频成交件数',
  `delivery_video_num` VARCHAR(32) DEFAULT NULL COMMENT '带货视频数',
  `product_qr_code_url` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '商品二维码',
  `shop_logo_url` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '店铺 Logo',
  `shop_qr_code_url` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '店铺二维码',
  `previous_rank` INT NULL COMMENT '上期排名',
  `rank_delta` INT NULL COMMENT '排名变化：正数=上升，负数=下降，0=持平',
  `rank_trend_source` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '排名趋势来源：official/fallback/unknown',
  `first_shelf_time` VARCHAR(32) DEFAULT NULL COMMENT '首次上架时间',
  `min_price` VARCHAR(32) DEFAULT NULL COMMENT '最低到手价',
  `first_list` TINYINT NOT NULL DEFAULT 0 COMMENT '是否首次上榜：0-否 1-是',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_task_id` (`task_id`),
  KEY `idx_product_id` (`product_id`),
  KEY `idx_rank` (`rank`),
  KEY `idx_rank_delta` (`rank_delta`),
  KEY `idx_product_name` (`product_name`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='抖音商品短视频排行榜数据表';

-- -----------------------------------------------------
-- 表 2: 视频分析数据表 (video_analysis_data)
-- -----------------------------------------------------
DROP TABLE IF EXISTS `video_analysis_data`;

CREATE TABLE `video_analysis_data` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `task_id` VARCHAR(64) NOT NULL COMMENT '爬虫任务 ID',
  `product_id` VARCHAR(64) DEFAULT NULL COMMENT '商品 ID',
  `product_name` VARCHAR(255) DEFAULT NULL COMMENT '商品名称',
  `video_id` VARCHAR(64) DEFAULT NULL COMMENT '视频 ID',
  `video_desc` VARCHAR(512) DEFAULT NULL COMMENT '短视频信息',
  `author_name` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '视频作者',
  `cover_url` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '视频封面',
  `publish_time` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '发布时间',
  `source_url` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '视频页地址',
  `payment_amount` VARCHAR(32) DEFAULT NULL COMMENT '用户支付金额',
  `expose_user_count` VARCHAR(32) DEFAULT NULL COMMENT '商品曝光人数',
  `click_user_count` VARCHAR(32) DEFAULT NULL COMMENT '商品点击人数',
  `convert_user_count` VARCHAR(32) DEFAULT NULL COMMENT '商品成交人数',
  `click_rate` VARCHAR(16) DEFAULT NULL COMMENT '商品点击率 (人数)',
  `click_convert_rate` VARCHAR(16) DEFAULT NULL COMMENT '商品点击 - 成交率 (人数)',
  `expose_convert_rate` VARCHAR(16) DEFAULT NULL COMMENT '商品曝光 - 成交率 (人数)',
  `order_count` VARCHAR(32) DEFAULT NULL COMMENT '商品成交订单数',
  `sales_volume` VARCHAR(32) DEFAULT NULL COMMENT '商品成交件数',
  `download_status` VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT '下载状态',
  `download_url` VARCHAR(2048) NOT NULL DEFAULT '' COMMENT '视频下载直链',
  `resolve_strategy` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '直链解析策略',
  `storage_path` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '存储路径',
  `file_size` BIGINT NOT NULL DEFAULT 0 COMMENT '文件大小',
  `last_error` TEXT NULL COMMENT '最后错误',
  `retry_count` INT NOT NULL DEFAULT 0 COMMENT '重试次数',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_task_id` (`task_id`),
  KEY `idx_product_id` (`product_id`),
  KEY `idx_video_id` (`video_id`),
  KEY `idx_video_download_status` (`download_status`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='抖音视频分析数据表';

-- -----------------------------------------------------
-- 表 3: 爬虫任务日志表 (crawler_task_log)
-- -----------------------------------------------------
DROP TABLE IF EXISTS `crawler_task_log`;

CREATE TABLE `crawler_task_log` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `task_id` VARCHAR(64) NOT NULL COMMENT '爬虫任务 ID',
  `status` VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT '任务状态：pending/running/completed/failed',
  `current_step` INT NOT NULL DEFAULT 0 COMMENT '当前步骤',
  `current_page` INT NOT NULL DEFAULT 0 COMMENT '当前页码',
  `total_pages` INT NOT NULL DEFAULT 0 COMMENT '总页数',
  `scraped_count` INT NOT NULL DEFAULT 0 COMMENT '已抓取数量',
  `start_time` TIMESTAMP NULL DEFAULT NULL COMMENT '开始时间',
  `end_time` TIMESTAMP NULL DEFAULT NULL COMMENT '结束时间',
  `error_message` TEXT COMMENT '错误信息',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_task_id` (`task_id`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='爬虫任务日志表';

-- -----------------------------------------------------
-- 插入测试数据（可选）
-- -----------------------------------------------------
-- INSERT INTO `tiktok_ranking_list` 
-- (`task_id`, `rank`, `product_name`, `shop_name`, `payment_amount`, `click_count`, `sales_volume`, `first_list`) 
-- VALUES 
-- ('test_001', 1, '测试商品 1', '测试店铺 1', '¥100', '1000', '500', 0),
-- ('test_001', 2, '测试商品 2', '测试店铺 2', '¥200', '2000', '800', 1);

-- -----------------------------------------------------
-- 查询示例
-- -----------------------------------------------------
-- 查看某个任务的所有商品数据
-- SELECT * FROM tiktok_ranking_list WHERE task_id = 'test_001' ORDER BY rank;

-- 查看首次上榜的商品
-- SELECT * FROM tiktok_ranking_list WHERE first_list = 1 ORDER BY created_at DESC;

-- 统计每个任务的抓取数量
-- SELECT task_id, COUNT(*) as product_count FROM tiktok_ranking_list GROUP BY task_id;
