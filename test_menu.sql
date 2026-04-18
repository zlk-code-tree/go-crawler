-- 测试外链菜单
-- 先执行迁移脚本添加字段
SOURCE database/migration_add_hide_exturl.sql;

-- 插入一个测试外链菜单
INSERT INTO `sys_permission` (`name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `ext_url`, `status`, `hide`, `sort`) 
VALUES ('百度', 'external.baidu', 1, 2, '/external/baidu', '', 'lucide:external-link', '', '', 'https://www.baidu.com', 1, 1, 100);

-- 给超级管理员分配这个菜单权限
INSERT INTO `sys_role_permission` (`role_id`, `permission_id`) 
SELECT 1, id FROM `sys_permission` WHERE `code` = 'external.baidu';
