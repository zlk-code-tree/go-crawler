-- 千川账户临时统计归属规则。账户原绑定关系不变，只改变指定日期范围内的统计负责人。
CREATE TABLE IF NOT EXISTS `qianchuan_advertiser_attribution_rule` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `advertiser_id` BIGINT(20) NOT NULL COMMENT '千川广告主ID',
  `advertiser_name` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '千川账户名称快照',
  `source_account_names` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '命中后改归属的抖音账号名称，逗号分隔；空表示广告主级兜底',
  `source_trader_id` BIGINT UNSIGNED NOT NULL COMMENT '账户原负责人ID',
  `target_trader_id` BIGINT UNSIGNED NOT NULL COMMENT '临时统计负责人ID',
  `effective_start_date` DATE NOT NULL COMMENT '生效开始日期（含）',
  `effective_end_date` DATE DEFAULT NULL COMMENT '生效结束日期（含），空表示持续有效',
  `reason` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '借号原因',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0=停用 1=启用',
  `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_advertiser_start` (`advertiser_id`, `effective_start_date`),
  KEY `idx_effective_rule` (`advertiser_id`, `status`, `effective_start_date`, `effective_end_date`),
  KEY `idx_target_trader` (`target_trader_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='千川账户临时统计归属规则';

ALTER TABLE `qianchuan_advertiser_attribution_rule`
  ADD COLUMN IF NOT EXISTS `source_account_names` VARCHAR(500) NOT NULL DEFAULT ''
  COMMENT '命中后改归属的抖音账号名称，逗号分隔；空表示广告主级兜底'
  AFTER `advertiser_name`;

UPDATE `qianchuan_advertiser_attribution_rule`
SET `source_account_names` = '天书说故事,利云底层逻辑'
WHERE `advertiser_id` IN (1700642741413895, 1774391159738372, 1711882704612356, 1722203379291139)
  AND `reason` = '王梅雪临时借用李百川千川账户';

UPDATE `qianchuan_advertiser_attribution_rule`
SET `effective_start_date` = '2026-07-01'
WHERE `advertiser_id` IN (1700642741413895, 1774391159738372, 1711882704612356, 1722203379291139)
  AND `effective_start_date` = '2026-07-13'
  AND `reason` = '王梅雪临时借用李百川千川账户';

UPDATE `qianchuan_advertiser_attribution_rule`
SET `effective_end_date` = NULL,
    `status` = 1
WHERE `advertiser_id` IN (1700642741413895, 1774391159738372, 1711882704612356, 1722203379291139)
  AND `reason` = '王梅雪临时借用李百川千川账户';

INSERT IGNORE INTO `qianchuan_advertiser_attribution_rule`
  (`advertiser_id`, `advertiser_name`, `source_account_names`, `source_trader_id`, `target_trader_id`,
   `effective_start_date`, `effective_end_date`, `reason`, `status`)
SELECT seed.advertiser_id, seed.advertiser_name, '天书说故事,利云底层逻辑', source.id, target.id,
  '2026-07-01', NULL, '王梅雪临时借用李百川千川账户', 1
FROM (
  SELECT 1700642741413895 AS advertiser_id, '天书万卷-官方号投放户' AS advertiser_name
  UNION ALL SELECT 1774391159738372, '利云图书企业店—店铺组'
  UNION ALL SELECT 1711882704612356, '利云文化进口图书-主账户'
  UNION ALL SELECT 1722203379291139, '闻道清北图书旗舰店主户'
) seed
JOIN trader source ON source.id = (
  SELECT id FROM trader
  WHERE name IN ('李百川', '李百川组') AND status = 1
  ORDER BY (name = '李百川') DESC, id ASC LIMIT 1
)
JOIN trader target ON target.id = (
  SELECT id FROM trader
  WHERE name IN ('王梅雪', '王梅雪组') AND status = 1
  ORDER BY (name = '王梅雪') DESC, id ASC LIMIT 1
)
WHERE NOT EXISTS (
  SELECT 1 FROM qianchuan_advertiser_attribution_rule existing
  WHERE existing.advertiser_id = seed.advertiser_id
);
