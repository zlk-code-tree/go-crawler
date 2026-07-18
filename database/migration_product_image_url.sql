-- ============================================
-- 商品榜单表添加商品主图字段
-- 罗盘 product_rank/video_bring_good 接口的 product_info.image_url
-- 前端 getProductCover 优先级 product_image_url > product_image > image_url，
-- 此处采用 image_url 与接口字段对齐，控制器透传后前端可直接命中 image_url 分支。
-- ============================================

ALTER TABLE `tiktok_ranking_list`
ADD COLUMN `image_url` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '商品主图地址（罗盘 product_info.image_url）' AFTER `product_name`;

ALTER TABLE `tiktok_ranking_list`
ADD INDEX `idx_image_url` (`image_url`);