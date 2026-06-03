-- 任务执行日志表（统一监控所有调度任务）
CREATE TABLE IF NOT EXISTS `task_execution_log` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `task_name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '任务名称：抖店订单轮询/商品榜单抓取/视频分析抓取',
  `task_type` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '任务类型：douyin_poller/crawl_product/crawl_video',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '执行状态：0-运行中 1-成功 2-失败 3-跳过',
  `start_time` DATETIME NOT NULL COMMENT '开始时间',
  `end_time` DATETIME DEFAULT NULL COMMENT '结束时间',
  `duration_ms` INT NOT NULL DEFAULT 0 COMMENT '耗时（毫秒）',
  `api_url` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '调用的第三方接口URL',
  `request_params` TEXT COMMENT '请求参数JSON',
  `response_code` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '接口返回码',
  `response_body` LONGTEXT COMMENT '接口返回内容JSON（7天后清除）',
  `error_msg` TEXT COMMENT '错误信息',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  INDEX `idx_task_type` (`task_type`),
  INDEX `idx_status` (`status`),
  INDEX `idx_start_time` (`start_time`),
  INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务执行日志表';

-- 统计表（保留超过7天的聚合数据）
CREATE TABLE IF NOT EXISTS `task_execution_stats` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `task_type` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '任务类型',
  `stat_date` DATE NOT NULL COMMENT '统计日期',
  `total_count` INT NOT NULL DEFAULT 0 COMMENT '当日总次数',
  `success_count` INT NOT NULL DEFAULT 0 COMMENT '成功次数',
  `fail_count` INT NOT NULL DEFAULT 0 COMMENT '失败次数',
  `skip_count` INT NOT NULL DEFAULT 0 COMMENT '跳过次数',
  `total_duration_ms` BIGINT NOT NULL DEFAULT 0 COMMENT '总耗时(毫秒)',
  `avg_duration_ms` INT NOT NULL DEFAULT 0 COMMENT '平均耗时(毫秒)',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_task_date` (`task_type`, `stat_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='任务执行统计表（聚合数据）';
