-- 定时任务管理表
CREATE TABLE IF NOT EXISTS `cron_task` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    `name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '任务名称',
    `cron_expr` VARCHAR(50) NOT NULL DEFAULT '' COMMENT 'cron表达式',
    `api_url` VARCHAR(300) NOT NULL DEFAULT '' COMMENT 'API地址',
    `api_method` VARCHAR(10) NOT NULL DEFAULT 'GET' COMMENT '请求方式: GET/POST',
    `api_params` TEXT COMMENT '请求参数JSON',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 0-停止 1-运行中',
    `retry_times` INT NOT NULL DEFAULT 0 COMMENT '失败重试次数',
    `retry_interval` INT NOT NULL DEFAULT 0 COMMENT '重试间隔(秒)',
    `next_task` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '完成后触发下一个任务名(任务链)',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='定时任务管理表';
