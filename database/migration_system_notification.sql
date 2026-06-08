-- 系统消息通知表
CREATE TABLE IF NOT EXISTS `system_notification` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
    `title` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '消息标题',
    `content` TEXT COMMENT '消息内容（JSON格式，前端可解析展示详情）',
    `type` VARCHAR(30) NOT NULL DEFAULT 'system' COMMENT '消息类型: task_result/approval/alert/system',
    `source_type` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '来源类型: alliance_pull/qianchuan_pull/order_import/...',
    `source_id` BIGINT NOT NULL DEFAULT 0 COMMENT '来源记录ID（如task_execution_log.id）',
    `level` TINYINT NOT NULL DEFAULT 0 COMMENT '消息级别: 0-普通 1-重要 2-紧急',
    `is_read` TINYINT NOT NULL DEFAULT 0 COMMENT '是否已读: 0-未读 1-已读',
    `read_at` DATETIME DEFAULT NULL COMMENT '阅读时间',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    INDEX `idx_type` (`type`),
    INDEX `idx_source_type` (`source_type`),
    INDEX `idx_source_id` (`source_id`),
    INDEX `idx_is_read` (`is_read`),
    INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统消息通知表';
