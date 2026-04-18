-- ============================================================
-- 为 sys_permission 表添加 hide 和 ext_url 字段
-- hide: 是否在菜单中隐藏（1-显示，0-隐藏）
-- ext_url: 外链地址（用于外链菜单）
-- ============================================================

-- 添加 hide 字段（默认值为 1，表示显示）
ALTER TABLE `sys_permission`
ADD COLUMN `hide` tinyint DEFAULT 1 COMMENT '是否在菜单中显示（1-显示，0-隐藏）' AFTER `status`;

-- 添加 ext_url 字段（外链地址）
ALTER TABLE `sys_permission`
ADD COLUMN `ext_url` varchar(500) DEFAULT '' COMMENT '外链地址（用于外链菜单）' AFTER `api`;

-- 更新现有数据：按钮权限（type=2）默认隐藏
UPDATE `sys_permission` SET `hide` = 0 WHERE `type` = 2;

-- 更新现有数据：菜单权限（type=1）默认显示
UPDATE `sys_permission` SET `hide` = 1 WHERE `type` = 1;
