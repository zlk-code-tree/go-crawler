-- 空白单归属统计卡片查询索引。
ALTER TABLE `douyin_order_attribution`
  ADD INDEX IF NOT EXISTS `idx_source_account_date` (`source`, `account_id`, `stat_date`);
