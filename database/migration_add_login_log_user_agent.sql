-- ============================================
-- 登录日志优化迁移：添加 user_agent 字段
-- ============================================

-- 1. 添加 user_agent 字段
ALTER TABLE `sys_login_log`
ADD COLUMN `user_agent` varchar(500) DEFAULT '' COMMENT 'User-Agent' AFTER `msg`;

-- 2. 为已有记录的 IP 补充 location 解析（如果有空值）
UPDATE `sys_login_log` SET `location` = '未知' WHERE `location` IS NULL OR `location` = '';
UPDATE `sys_login_log` SET `browser` = '' WHERE `browser` IS NULL;
UPDATE `sys_login_log` SET `os` = '' WHERE `os` IS NULL;
UPDATE `sys_login_log` SET `user_agent` = '' WHERE `user_agent` IS NULL;

-- 3. 对于已有的内网IP记录，标记 location
UPDATE `sys_login_log`
SET `location` = '内网IP'
WHERE `ip` IS NOT NULL AND `ip` != '' AND `location` = '未知'
AND (
  `ip` = '127.0.0.1' OR `ip` = 'localhost' OR `ip` = '::1'
  OR `ip` LIKE '10.%'
  OR `ip` LIKE '192.168.%'
  OR `ip` LIKE '172.16.%' OR `ip` LIKE '172.17.%' OR `ip` LIKE '172.18.%'
  OR `ip` LIKE '172.19.%' OR `ip` LIKE '172.20.%' OR `ip` LIKE '172.21.%'
  OR `ip` LIKE '172.22.%' OR `ip` LIKE '172.23.%' OR `ip` LIKE '172.24.%'
  OR `ip` LIKE '172.25.%' OR `ip` LIKE '172.26.%' OR `ip` LIKE '172.27.%'
  OR `ip` LIKE '172.28.%' OR `ip` LIKE '172.29.%' OR `ip` LIKE '172.30.%'
  OR `ip` LIKE '172.31.%'
);
