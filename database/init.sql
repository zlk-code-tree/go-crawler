-- 抖音电商罗盘爬虫 - 数据库初始化脚本
-- 数据库：easyadmin8

-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS `easyadmin8` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `easyadmin8`;

-- 商品榜单数据表
DROP TABLE IF EXISTS `tiktok_ranking_list`;
CREATE TABLE `tiktok_ranking_list` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `task_id` varchar(64) NOT NULL DEFAULT '' COMMENT '任务 ID',
  `rank` int(11) NOT NULL DEFAULT '0' COMMENT '排名',
  `product_name` varchar(512) NOT NULL DEFAULT '' COMMENT '商品名称',
  `shop_name` varchar(256) NOT NULL DEFAULT '' COMMENT '店铺名称',
  `payment_amount` varchar(64) NOT NULL DEFAULT '' COMMENT '短视频用户支付金额',
  `click_count` varchar(64) NOT NULL DEFAULT '' COMMENT '短视频点击次数',
  `sales_volume` varchar(64) NOT NULL DEFAULT '' COMMENT '短视频成交件数',
  `delivery_video_num` varchar(64) NOT NULL DEFAULT '' COMMENT '带货视频数',
  `first_shelf_time` varchar(64) NOT NULL DEFAULT '' COMMENT '首次上架时间',
  `min_price` varchar(64) NOT NULL DEFAULT '' COMMENT '最低到手价',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_task_id` (`task_id`),
  KEY `idx_rank` (`rank`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='抖音商品榜单数据表';

-- 视频分析数据表
DROP TABLE IF EXISTS `video_analysis`;
CREATE TABLE `video_analysis` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `task_id` varchar(64) NOT NULL DEFAULT '' COMMENT '任务 ID',
  `product_id` varchar(64) NOT NULL DEFAULT '' COMMENT '商品 ID',
  `product_name` varchar(512) NOT NULL DEFAULT '' COMMENT '商品名称',
  `video_id` varchar(64) NOT NULL DEFAULT '' COMMENT '视频 ID',
  `video_desc` varchar(1024) NOT NULL DEFAULT '' COMMENT '短视频信息',
  `payment_amount` varchar(64) NOT NULL DEFAULT '' COMMENT '用户支付金额',
  `expose_user_count` varchar(64) NOT NULL DEFAULT '' COMMENT '商品曝光人数',
  `click_user_count` varchar(64) NOT NULL DEFAULT '' COMMENT '商品点击人数',
  `convert_user_count` varchar(64) NOT NULL DEFAULT '' COMMENT '商品成交人数',
  `click_rate` varchar(32) NOT NULL DEFAULT '' COMMENT '商品点击率 (人数)',
  `click_convert_rate` varchar(32) NOT NULL DEFAULT '' COMMENT '商品点击 - 成交率 (人数)',
  `expose_convert_rate` varchar(32) NOT NULL DEFAULT '' COMMENT '商品曝光 - 成交率 (人数)',
  `order_count` varchar(64) NOT NULL DEFAULT '' COMMENT '商品成交订单数',
  `sales_volume` varchar(64) NOT NULL DEFAULT '' COMMENT '商品成交件数',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_task_id` (`task_id`),
  KEY `idx_product_id` (`product_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频分析数据表';

-- 爬虫任务记录表
DROP TABLE IF EXISTS `crawl_tasks`;
CREATE TABLE `crawl_tasks` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `task_id` varchar(64) NOT NULL DEFAULT '' COMMENT '任务 ID',
  `status` varchar(32) NOT NULL DEFAULT 'pending' COMMENT '任务状态：pending,running,completed,failed',
  `target_url` varchar(1024) NOT NULL DEFAULT '' COMMENT '目标 URL',
  `login_type` varchar(32) NOT NULL DEFAULT 'qrcode' COMMENT '登录方式：qrcode,password',
  `product_count` int(11) NOT NULL DEFAULT '0' COMMENT '商品数量',
  `video_count` int(11) NOT NULL DEFAULT '0' COMMENT '视频数量',
  `export_files` text COMMENT '导出文件路径（JSON）',
  `error_msg` text COMMENT '错误信息',
  `start_time` datetime DEFAULT NULL COMMENT '开始时间',
  `end_time` datetime DEFAULT NULL COMMENT '结束时间',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_task_id` (`task_id`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='爬虫任务记录表';
