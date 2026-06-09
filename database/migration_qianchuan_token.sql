-- 巨量千川 access_token 存储表
CREATE TABLE IF NOT EXISTS `qianchuan_token` (
    `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `access_token` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'access_token',
    `advertiser_ids` TEXT COMMENT '授权的广告主ID列表（JSON数组）',
    `expires_in` INT(11) NOT NULL DEFAULT 0 COMMENT '过期时间（秒）',
    `expires_at` DATETIME DEFAULT NULL COMMENT 'token过期时间',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='巨量千川access_token';

-- 如果表已存在，添加 expires_at 列
ALTER TABLE `qianchuan_token` ADD COLUMN IF NOT EXISTS `expires_at` DATETIME DEFAULT NULL COMMENT 'token过期时间' AFTER `expires_in`;
