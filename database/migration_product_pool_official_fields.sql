-- 商品池官方字段兼容扩展
-- 用于方案 A：保留当前 top200 抓取链路，官方返回什么就原样落库展示。

ALTER TABLE `tiktok_ranking_list`
  ADD COLUMN `click_convert_rate` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '点击成交转化率' AFTER `click_count`,
  ADD COLUMN `product_qr_code_url` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '商品二维码' AFTER `delivery_video_num`,
  ADD COLUMN `shop_logo_url` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '店铺 Logo' AFTER `product_qr_code_url`,
  ADD COLUMN `shop_qr_code_url` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '店铺二维码' AFTER `shop_logo_url`;
