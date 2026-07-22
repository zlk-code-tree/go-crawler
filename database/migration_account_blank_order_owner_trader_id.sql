-- 空白单归属改为使用 trader_id，避免按归属人名称或来源账号统计时串组。

ALTER TABLE `account`
  ADD COLUMN IF NOT EXISTS `blank_order_owner_trader_id` BIGINT NOT NULL DEFAULT 0 COMMENT '空白单归属投手ID' AFTER `blank_order_owner_name`;

ALTER TABLE `account`
  ADD INDEX IF NOT EXISTS `idx_blank_order_owner_trader_id` (`blank_order_owner_trader_id`);

UPDATE `account` a
SET `blank_order_owner_trader_id` = COALESCE((
  SELECT t.`id`
  FROM `trader` t
  WHERE t.`status` = 1
    AND (
      t.`name` COLLATE utf8mb4_unicode_ci = a.`blank_order_owner_name` COLLATE utf8mb4_unicode_ci
      OR t.`name` COLLATE utf8mb4_unicode_ci = CONCAT(a.`blank_order_owner_name`, '组') COLLATE utf8mb4_unicode_ci
      OR (a.`blank_order_owner_name` LIKE '%组' AND t.`name` COLLATE utf8mb4_unicode_ci = TRIM(TRAILING '组' FROM a.`blank_order_owner_name`) COLLATE utf8mb4_unicode_ci)
    )
  ORDER BY (t.`name` COLLATE utf8mb4_unicode_ci = a.`blank_order_owner_name` COLLATE utf8mb4_unicode_ci) DESC, t.`id` ASC
  LIMIT 1
), 0)
WHERE a.`blank_order_enabled` = 1
  AND a.`blank_order_owner_trader_id` = 0
  AND TRIM(a.`blank_order_owner_name`) != '';
