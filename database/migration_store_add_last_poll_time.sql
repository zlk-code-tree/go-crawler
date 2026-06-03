-- store 表新增上次拉取时间字段（用于任务中断恢复）
ALTER TABLE `store` ADD COLUMN IF NOT EXISTS `last_poll_time` DATETIME DEFAULT NULL COMMENT '上次成功拉取订单的时间' AFTER `author`;
