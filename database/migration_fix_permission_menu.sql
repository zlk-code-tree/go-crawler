-- ============================================================
-- 修复 sys_permission 菜单数据，对齐前端 Vben Admin 路由
-- code 字段 = 前端路由 name，path/component 需和前端静态路由完全一致
-- ============================================================

-- 清空旧数据（保留按钮权限，只重建菜单）
DELETE FROM `sys_role_permission` WHERE `permission_id` IN (
  SELECT id FROM `sys_permission` WHERE `type` = 1
);
DELETE FROM `sys_permission` WHERE `type` = 1;

-- ============================================================
-- 一级父菜单（component = BasicLayout，对应前端 layouts/BasicLayout）
-- ============================================================
INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `status`, `sort`) VALUES
(1,  '仪表板',   'Dashboard',  1, 0, '/dashboard', 'BasicLayout', 'lucide:layout-dashboard', '', '', 1, 0),
(2,  '系统管理', 'System',     1, 0, '/system',    'BasicLayout', 'lucide:settings',         '', '', 1, 10),
(3,  '公司管理', 'Company',    1, 0, '/company',   'BasicLayout', 'lucide:building-2',       '', '', 1, 20),
(4,  '爬虫任务', 'Crawler',    1, 0, '/crawler',   'BasicLayout', 'lucide:bot',              '', '', 1, 30),
(5,  '日志管理', 'Log',        1, 0, '/log',       'BasicLayout', 'lucide:file-text',        '', '', 1, 40);

-- ============================================================
-- 仪表板子菜单
-- ============================================================
INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `status`, `sort`) VALUES
(10, '工作台',   'Workspace',  1, 1, 'workspace',  'dashboard/workspace/index',  'carbon:workspace',         '', '', 1, 1),
(11, '数据分析', 'Analytics',  1, 1, 'analytics',  'dashboard/analytics/index',  'lucide:area-chart',        '', '', 1, 2);

-- ============================================================
-- 系统管理子菜单
-- ============================================================
INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `status`, `sort`) VALUES
(20, '用户管理', 'UserManagement', 1, 2, '/system/user',    'system/user/index',    'lucide:users',    '', '', 1, 1),
(21, '角色管理', 'RoleManagement', 1, 2, '/system/role',    'system/role/index',    'lucide:user-cog', '', '', 1, 2),
(22, '菜单管理', 'MenuManagement', 1, 2, '/system/menu',    'system/menu/index',    'lucide:menu',     '', '', 1, 3),
(23, '部门管理', 'DeptManagement', 1, 2, '/system/dept',       'dept/list/index',       'lucide:network',  '', '', 1, 4),
(24, '权限管理', 'PermManagement', 1, 2, '/system/permission', 'permission/list/index', 'lucide:shield',   '', '', 1, 5),
(25, '个人中心', 'SystemProfile',  1, 2, '/system/profile', 'system/profile/index', 'lucide:user',     '', '', 1, 99);

-- ============================================================
-- 公司管理子菜单
-- ============================================================
INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `status`, `sort`) VALUES
(30, '公司列表', 'CompanyList', 1, 3, '/company/list', 'company/list/index', 'lucide:building', '', '', 1, 1);

-- ============================================================
-- 日志管理子菜单
-- ============================================================
INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `status`, `sort`) VALUES
(60, '操作日志', 'OperationLog', 1, 5, '/log/operation', 'log/operation/index', 'lucide:file-clock', '', '', 1, 1),
(61, '登录日志', 'LoginLog',     1, 5, '/log/login',     'log/login/index',    'lucide:log-in',     '', '', 1, 2);

-- ============================================================
-- 按钮权限（type=2，不影响菜单路由）
-- ============================================================
INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `status`, `sort`) VALUES
-- 用户管理按钮
(100, '新增用户', 'user:add',          2, 20, '', '', '', 'POST',   '/api/system/user/save',         1, 1),
(101, '编辑用户', 'user:edit',         2, 20, '', '', '', 'POST',   '/api/system/user/save',         1, 2),
(102, '删除用户', 'user:delete',       2, 20, '', '', '', 'POST',   '/api/system/user/delete',       1, 3),
(103, '分配角色', 'user:assign-roles', 2, 20, '', '', '', 'POST',   '/api/system/user/assign-roles', 1, 4),
-- 角色管理按钮
(110, '新增角色', 'role:add',          2, 21, '', '', '', 'POST',   '/api/system/role/save',         1, 1),
(111, '编辑角色', 'role:edit',         2, 21, '', '', '', 'POST',   '/api/system/role/detail',       1, 2),
(112, '删除角色', 'role:delete',       2, 21, '', '', '', 'POST',   '/api/system/role/delete',       1, 3),
(113, '分配权限', 'role:permission',   2, 21, '', '', '', 'POST',   '/api/system/role/assign-menus', 1, 4),
-- 公司管理按钮
(120, '新增公司', 'company:add',       2, 30, '', '', '', 'POST',   '/api/system/company/save',      1, 1),
(121, '编辑公司', 'company:edit',      2, 30, '', '', '', 'PUT',    '/api/system/company/update',    1, 2),
(122, '删除公司', 'company:delete',    2, 30, '', '', '', 'DELETE', '/api/system/company/delete',    1, 3),
-- 菜单管理按钮
(130, '新增菜单', 'menu:add',          2, 22, '', '', '', 'POST',   '/api/system/menu/create',       1, 1),
(131, '编辑菜单', 'menu:edit',         2, 22, '', '', '', 'PUT',    '/api/system/menu/update',       1, 2),
(132, '删除菜单', 'menu:delete',       2, 22, '', '', '', 'DELETE', '/api/system/menu/delete',       1, 3),
-- 权限管理按钮
(140, '新增权限', 'permission:add',    2, 24, '', '', '', 'POST',   '/api/system/permission/create', 1, 1),
(141, '编辑权限', 'permission:edit',   2, 24, '', '', '', 'PUT',    '/api/system/permission/update', 1, 2),
(142, '删除权限', 'permission:delete', 2, 24, '', '', '', 'DELETE', '/api/system/permission/delete', 1, 3);

-- ============================================================
-- 重新给超级管理员角色分配所有权限
-- ============================================================
INSERT INTO `sys_role_permission` (`role_id`, `permission_id`)
SELECT 1, id FROM `sys_permission`;
