-- 给 qianchuan_token 表添加 refresh_token 支持
-- 执行时间：2026-06-25

ALTER TABLE `qianchuan_token`
  ADD COLUMN `refresh_token` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'refresh_token，用于续期access_token' AFTER `access_token`,
  ADD COLUMN `refresh_token_expires_at` DATETIME DEFAULT NULL COMMENT 'refresh_token过期时间' AFTER `expires_at`;
