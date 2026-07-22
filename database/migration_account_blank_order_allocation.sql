-- 账号表空白单归属准入字段
-- blank_order_enabled=1 的账号才参与官方号/授权号空白单归属。
-- blank_order_owner_trader_id 为最终归属投手ID，blank_order_owner_name 仅作展示/快照。

ALTER TABLE `account`
  ADD COLUMN IF NOT EXISTS `blank_order_enabled` TINYINT NOT NULL DEFAULT 0 COMMENT '是否参与空白单归属：0=否 1=是' AFTER `bind_status`,
  ADD COLUMN IF NOT EXISTS `blank_order_owner_name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '空白单归属人快照' AFTER `blank_order_enabled`,
  ADD COLUMN IF NOT EXISTS `blank_order_owner_trader_id` BIGINT NOT NULL DEFAULT 0 COMMENT '空白单归属投手ID' AFTER `blank_order_owner_name`;

ALTER TABLE `account`
  ADD INDEX IF NOT EXISTS `idx_blank_order_enabled` (`blank_order_enabled`),
  ADD INDEX IF NOT EXISTS `idx_blank_order_owner_trader_id` (`blank_order_owner_trader_id`);

INSERT INTO `account` (
  `platform`, `uid`, `account_name`, `platform_nickname`, `operator`,
  `blank_order_enabled`, `blank_order_owner_name`, `blank_order_owner_trader_id`, `status`, `created_at`, `updated_at`
) VALUES (
  'douyin', '2986722072732416', '欢喜图书专营店', '70228145728', '张文蔚/孟令饶组',
  1, '张文蔚/孟令饶组', COALESCE((SELECT `id` FROM `trader` WHERE `status` = 1 AND `name` = '张文蔚/孟令饶组' ORDER BY `id` ASC LIMIT 1), 0), 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
ON DUPLICATE KEY UPDATE
  `uid` = VALUES(`uid`),
  `account_name` = VALUES(`account_name`),
  `platform_nickname` = VALUES(`platform_nickname`),
  `operator` = VALUES(`operator`),
  `blank_order_enabled` = VALUES(`blank_order_enabled`),
  `blank_order_owner_name` = VALUES(`blank_order_owner_name`),
  `blank_order_owner_trader_id` = VALUES(`blank_order_owner_trader_id`),
  `status` = VALUES(`status`),
  `updated_at` = VALUES(`updated_at`);

UPDATE `account`
SET `blank_order_enabled` = 1,
    `blank_order_owner_name` = CASE
      WHEN `account_name` IN ('天书说故事', '利云底层逻辑') THEN '王梅雪组'
      ELSE COALESCE(NULLIF(TRIM(`operator`), ''), `blank_order_owner_name`)
    END
WHERE `account_name` IN (
    '闻道清北图书旗舰店',
    '利云学习文化旗舰店',
    '利云文化官方旗舰店',
    '金抖云图书专营店',
    '利云图书专营店',
    '利云文化千玺进口图书旗舰店',
    '欢喜图书专营店',
    '利云成长课堂',
    '天书说故事',
    '利云底层逻辑',
    '利云文化文创旗舰店',
    '利云文化早教益智旗舰店'
  );

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
