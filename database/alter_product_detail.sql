-- ============================================
-- 商品榜单表添加商品详情字段
-- 参考 Python 爬虫提取的字段
-- ============================================

-- 添加运费险字段
ALTER TABLE `tiktok_ranking_list`
ADD COLUMN `has_shipping_insurance` TINYINT DEFAULT NULL COMMENT '是否有运费险：0-无 1-有 NULL-未知' AFTER `min_price`;

-- 添加详情访问状态字段
ALTER TABLE `tiktok_ranking_list`
ADD COLUMN `detail_access_status` VARCHAR(32) DEFAULT 'available' COMMENT '详情访问状态：available-正常 removed-已下架 category_mismatch-类目不匹配' AFTER `has_shipping_insurance`;

-- 添加详情访问备注字段
ALTER TABLE `tiktok_ranking_list`
ADD COLUMN `detail_access_note` VARCHAR(255) DEFAULT '' COMMENT '详情访问备注' AFTER `detail_access_status`;

-- 添加索引
ALTER TABLE `tiktok_ranking_list`
ADD INDEX `idx_product_id` (`product_id`);

-- ============================================
-- 如果表中没有 product_id 字段，需要先添加
-- ============================================
-- ALTER TABLE `tiktok_ranking_list`
-- ADD COLUMN `product_id` VARCHAR(64) DEFAULT '' COMMENT '商品 ID' AFTER `task_id`;

-- ============================================
-- 完整的商品详情字段说明（参考 Python 爬虫）
-- ============================================
-- 1. first_shelf_time (首次上架时间) - 已存在
-- 2. min_price (最低到手价) - 已存在
-- 3. has_shipping_insurance (是否有运费险) - 新增
-- 4. detail_access_status (详情访问状态) - 新增
-- 5. detail_access_note (详情访问备注) - 新增
