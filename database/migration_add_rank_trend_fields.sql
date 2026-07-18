-- 迁移脚本：商品榜单排名趋势字段
-- 用途：优先保存罗盘官方返回的上期排名/排名变化；官方未返回时由后端补算后入库。

ALTER TABLE `tiktok_ranking_list`
  ADD COLUMN `previous_rank` INT NULL COMMENT '上期排名' AFTER `delivery_video_num`,
  ADD COLUMN `rank_delta` INT NULL COMMENT '排名变化：正数=上升，负数=下降，0=持平' AFTER `previous_rank`,
  ADD COLUMN `rank_trend_source` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '排名趋势来源：official/fallback/unknown' AFTER `rank_delta`;

CREATE INDEX `idx_rank_delta` ON `tiktok_ranking_list` (`rank_delta`);
