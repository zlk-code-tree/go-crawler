-- 第三方API请求日志表
CREATE TABLE IF NOT EXISTS `third_party_api_log` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `platform` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '平台：douyin/kuaishou/shipinhao',
  `api_name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '接口名称：buyin_queryShopAllianceOrder',
  `api_path` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '接口路径：/buyin/queryShopAllianceOrder',
  `request_params` TEXT COMMENT '请求参数JSON',
  `response_code` VARCHAR(20) NOT NULL DEFAULT '' COMMENT 'HTTP状态码/API返回码',
  `response_body` LONGTEXT COMMENT '返回内容JSON',
  `duration_ms` INT NOT NULL DEFAULT 0 COMMENT '耗时(毫秒)',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '0失败 1成功',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `idx_platform_api` (`platform`, `api_name`),
  INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='第三方API请求日志表';
