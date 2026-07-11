-- 千川账户与店铺归属关系：一个店铺可拥有多个千川账户，一个千川账户只归属一个店铺。
CREATE TABLE IF NOT EXISTS `store_qianchuan_mapping` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `store_id` INT NOT NULL COMMENT '关联 store.id',
  `qianchuan_advertiser_id` BIGINT NOT NULL COMMENT '千川账户ID（广告主ID）',
  `qianchuan_advertiser_name` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '千川账户名称',
  `source_store_name` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '导入文件中的店铺名称',
  `source` VARCHAR(30) NOT NULL DEFAULT 'manual' COMMENT '来源：manual/excel',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_qianchuan_advertiser` (`qianchuan_advertiser_id`),
  KEY `idx_store_id` (`store_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='店铺与千川账户归属关系表';
