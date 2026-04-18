-- ============================================
-- 迁移脚本：添加 product_id 和 price_range 字段
-- 用途：在 tiktok_ranking_list 表中添加商品 ID 和价格带字段
-- 执行时间：2026-04-15
-- ============================================

USE `easyadmin8`;

-- 添加 product_id 字段（如果不存在）
ALTER TABLE `tiktok_ranking_list`
ADD COLUMN `product_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '商品 ID' AFTER `task_id`;

-- 添加 price_range 字段（如果不存在）
ALTER TABLE `tiktok_ranking_list`
ADD COLUMN `price_range` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '价格带' AFTER `first_list`;

-- 添加索引
ALTER TABLE `tiktok_ranking_list`
ADD KEY `idx_product_id` (`product_id`);

-- 验证
SELECT COLUMN_NAME, DATA_TYPE, COLUMN_DEFAULT, COLUMN_COMMENT
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = 'easyadmin8'
  AND TABLE_NAME = 'tiktok_ranking_list'
  AND COLUMN_NAME IN ('product_id', 'price_range');
