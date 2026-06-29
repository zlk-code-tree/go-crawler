-- =====================================================
-- 修复 qianchuan_token 表：补全 refresh_token 字段
-- 背景：代码 cmd.go 启动时执行 ALTER TABLE ... ADD COLUMN IF NOT EXISTS，
-- 但阿里云 RDS 的 MySQL 版本不支持 IF NOT EXISTS 语法导致列未创建，
-- token 续期逻辑无法运行，token 过期后只能重新 OAuth 授权。
-- 执行方式：mysql -u ly_admin -p ly_crawler < migration_qianchuan_token_refresh_fix.sql
-- =====================================================

ALTER TABLE qianchuan_token
    ADD COLUMN IF NOT EXISTS refresh_token VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'refresh_token' AFTER access_token;

ALTER TABLE qianchuan_token
    ADD COLUMN IF NOT EXISTS refresh_token_expires_at DATETIME DEFAULT NULL COMMENT 'refresh_token过期时间' AFTER expires_at;
