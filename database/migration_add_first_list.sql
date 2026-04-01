-- ============================================
-- 迁移脚本：添加 first_list 字段
-- 用途：在现有 tiktok_ranking_list 表中添加首次上榜标识字段
-- 执行时间：2026-03-30
-- ============================================

USE `easyadmin8`;

-- 添加 first_list 字段（如果不存在）
ALTER TABLE `tiktok_ranking_list` 
ADD COLUMN `first_list` TINYINT NOT NULL DEFAULT 0 COMMENT '是否首次上榜：0-否 1-是' AFTER `min_price`;

-- 验证字段是否添加成功
SELECT COLUMN_NAME, DATA_TYPE, COLUMN_DEFAULT, COLUMN_COMMENT 
FROM INFORMATION_SCHEMA.COLUMNS 
WHERE TABLE_SCHEMA = 'easyadmin8' 
  AND TABLE_NAME = 'tiktok_ranking_list' 
  AND COLUMN_NAME = 'first_list';

-- 查询示例：查看所有首次上榜的商品
-- SELECT * FROM tiktok_ranking_list WHERE first_list = 1 ORDER BY created_at DESC;
