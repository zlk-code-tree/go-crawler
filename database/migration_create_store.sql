-- 创建店铺表（存储外部平台店铺信息）
CREATE TABLE IF NOT EXISTS `store` (
  `id` int NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `store_name` varchar(255) NOT NULL DEFAULT '' COMMENT '店铺名称',
  `store_id` varchar(100) NOT NULL DEFAULT '' COMMENT '店铺ID（外部平台店铺ID）',
  `platform` varchar(50) NOT NULL DEFAULT 'douyin' COMMENT '平台：douyin/kuaishou/shipinhao',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '状态：0-禁用 1-启用',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_store_id_platform` (`store_id`, `platform`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='店铺表（外部平台店铺ID）';
