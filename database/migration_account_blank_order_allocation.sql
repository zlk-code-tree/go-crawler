-- 账号表空白单归属准入字段
-- blank_order_enabled=1 的账号才参与官方号/授权号空白单归属。
-- blank_order_owner_name 为空时，业务默认使用 account.operator。

ALTER TABLE `account`
  ADD COLUMN IF NOT EXISTS `blank_order_enabled` TINYINT NOT NULL DEFAULT 0 COMMENT '是否参与空白单归属：0=否 1=是' AFTER `bind_status`,
  ADD COLUMN IF NOT EXISTS `blank_order_owner_name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '空白单归属人；为空默认使用运营者' AFTER `blank_order_enabled`;

ALTER TABLE `account`
  ADD INDEX IF NOT EXISTS `idx_blank_order_enabled` (`blank_order_enabled`);

INSERT INTO `account` (
  `platform`, `uid`, `account_name`, `platform_nickname`, `operator`,
  `blank_order_enabled`, `blank_order_owner_name`, `status`, `created_at`, `updated_at`
) VALUES (
  'douyin', '2986722072732416', '欢喜图书专营店', '70228145728', '张文蔚/孟令饶组',
  1, '张文蔚/孟令饶组', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
ON DUPLICATE KEY UPDATE
  `uid` = VALUES(`uid`),
  `account_name` = VALUES(`account_name`),
  `platform_nickname` = VALUES(`platform_nickname`),
  `operator` = VALUES(`operator`),
  `blank_order_enabled` = VALUES(`blank_order_enabled`),
  `blank_order_owner_name` = VALUES(`blank_order_owner_name`),
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
