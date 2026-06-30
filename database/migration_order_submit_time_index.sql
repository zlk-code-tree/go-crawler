-- ============================================================
-- 修复：达人佣金统计实时查询超时
-- 根因：order_info/kuaishou_orders/shipinhao_orders 的实时统计
--       按 order_submit_time 过滤，但该字段无索引，导致每个
--       talent_id 都要扫描全部历史订单再内存过滤（今日1条订单耗时3s）。
-- 方案：为三张订单表添加复合索引 (talent_id, order_submit_time)，
--       匹配实时查询的访问模式 (talent_id IN(...) AND order_submit_time BETWEEN)。
--       MySQL 8.0 在线加索引(ALGORITHM=INPLACE, LOCK=NONE)不阻塞读写。
-- 效果：今日1条订单 2.77s -> 0.01s；昨日3.5万条 3.18s -> 0.32s。
-- ============================================================

-- 抖音订单表
ALTER TABLE order_info
  ADD INDEX idx_talent_submit_time (talent_id, order_submit_time),
  ALGORITHM=INPLACE, LOCK=NONE;

-- 快手订单表
ALTER TABLE kuaishou_orders
  ADD INDEX idx_talent_submit_time (talent_id, order_submit_time),
  ALGORITHM=INPLACE, LOCK=NONE;

-- 视频号订单表
ALTER TABLE shipinhao_orders
  ADD INDEX idx_talent_submit_time (talent_id, order_submit_time),
  ALGORITHM=INPLACE, LOCK=NONE;
