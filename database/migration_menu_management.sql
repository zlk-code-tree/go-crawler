-- ============================================
-- 菜单数据补全 & 菜单管理接口支持
-- 在已有 sys_permission 表基础上补充缺失的菜单条目
-- ============================================

-- 补充仪表盘菜单（之前缺失）
INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `status`, `sort`) VALUES
(100, '仪表盘', 'dashboard', 1, 0, '/dashboard', '', 'lucide:layout-dashboard', '', '', 1, 0),
(101, '分析页', 'dashboard:analytics', 1, 100, '/dashboard/analytics', 'dashboard/analytics/index', 'lucide:area-chart', '', '', 1, 1),
(102, '工作台', 'dashboard:workspace', 1, 100, '/dashboard/workspace', 'dashboard/workspace/index', 'carbon:workspace', '', '', 1, 2);

-- 补充系统管理 -> 菜单管理（新增低代码页面）
INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `status`, `sort`) VALUES
(12, '菜单管理', 'system:menu', 1, 1, '/system/menu', 'system/menu/index', 'lucide:menu', '', '', 1, 0);

-- 给超级管理员分配新增菜单的权限
INSERT IGNORE INTO `sys_role_permission` (`role_id`, `permission_id`) VALUES
(1, 100), (1, 101), (1, 102), (1, 12);

-- 更新一级菜单的 component 为空（让前端自动用 BasicLayout）
UPDATE `sys_permission` SET `component` = '' WHERE `parent_id` = 0 AND `type` = 1 AND `component` = 'Layout';
