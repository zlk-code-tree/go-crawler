-- 店铺借用账号归属规则增加稳定店铺ID
ALTER TABLE `shop_account_attribution`
  ADD COLUMN IF NOT EXISTS `shop_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '店铺ID（store.store_id，优先用于稳定匹配）' AFTER `id`,
  ADD INDEX IF NOT EXISTS `idx_shop_id` (`shop_id`);

ALTER TABLE `shop_account_attribution`
  MODIFY COLUMN `shop_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '店铺ID（store.store_id，优先用于稳定匹配）';

UPDATE `shop_account_attribution` saa
INNER JOIN `store` s
  ON s.platform COLLATE utf8mb4_unicode_ci = 'douyin'
  AND CAST(s.id AS CHAR) COLLATE utf8mb4_unicode_ci = saa.shop_id COLLATE utf8mb4_unicode_ci
  AND s.store_id != ''
SET saa.shop_id = CONVERT(s.store_id USING utf8mb4) COLLATE utf8mb4_unicode_ci
WHERE saa.shop_id != ''
  AND saa.shop_id != '0'
  AND saa.shop_id COLLATE utf8mb4_unicode_ci != CONVERT(s.store_id USING utf8mb4) COLLATE utf8mb4_unicode_ci;

UPDATE `shop_account_attribution` saa
INNER JOIN (
  SELECT s.store_name, s.store_id
  FROM `store` s
  INNER JOIN (
    SELECT MIN(id) AS id
    FROM `store`
    WHERE platform COLLATE utf8mb4_unicode_ci = 'douyin'
      AND status = 1
      AND store_id != ''
    GROUP BY store_name
  ) picked ON picked.id = s.id
) matched_store
  ON matched_store.store_name COLLATE utf8mb4_unicode_ci = saa.shop_name COLLATE utf8mb4_unicode_ci
SET saa.shop_id = CONVERT(matched_store.store_id USING utf8mb4) COLLATE utf8mb4_unicode_ci
WHERE COALESCE(NULLIF(NULLIF(TRIM(saa.shop_id), ''), '0'), '') = '';

UPDATE `shop_account_attribution`
SET `shop_id` = ''
WHERE `shop_id` = '0';

UPDATE `shop_account_attribution`
SET `remark` = CONCAT(
  '借号订单归属：',
  COALESCE(NULLIF(`source_account_name`, ''), `source_account_uid`),
  '（UID ',
  `source_account_uid`,
  '）在“',
  `shop_name`,
  '”出单归属',
  `target_value`
)
WHERE TRIM(COALESCE(`remark`, '')) = ''
   OR `remark` LIKE '规则%';
