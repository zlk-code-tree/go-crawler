-- ============================================================
-- 重建 sys_permission 菜单数据，完全对齐前端静态路由
-- code 字段 = 前端路由 name，path/component 需和前端完全一致
-- 执行前请确保已备份数据
-- ============================================================

-- 清空全部权限数据（角色关联表、权限表），完全重建
DELETE FROM `sys_role_permission`;
DELETE FROM `sys_permission`;
-- 重置自增 id（避免 AUTO_INCREMENT 残留导致冲突）
ALTER TABLE `sys_permission` AUTO_INCREMENT = 1;

-- ============================================================
-- 一级父菜单（component = BasicLayout）
-- hide=1 表示在菜单中显示
-- ============================================================
INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `status`, `hide`, `sort`) VALUES
(1,  '仪表板',   'dashboard', 1, 0, '/dashboard', 'BasicLayout', 'lucide:layout-dashboard',     '', '', 1, 1, 0),
(2,  '系统管理', 'system',    1, 0, '/system',    'BasicLayout', 'ant-design:setting-outlined', '', '', 1, 1, 10),
(4,  '抖音盯盘', 'douyin',    1, 0, '/douyin',    'BasicLayout', 'lucide:video',                '', '', 1, 1, 30);

-- ============================================================
-- 仪表板子菜单
-- code = 前端路由 name（dashboard.ts 中定义）
-- ============================================================
INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `status`, `hide`, `sort`) VALUES
(10, '工作台',   'dashboard.workspace', 1, 1, '/dashboard/workspace', 'dashboard/workspace/index', 'carbon:workspace',  '', '', 1, 1, 1),
(11, '数据分析', 'dashboard.analytics', 1, 1, '/dashboard/analytics', 'dashboard/analytics/index', 'lucide:area-chart', '', '', 1, 1, 2);

-- ============================================================
-- 系统管理子菜单（对应前端 system.ts，code 以 system. 开头）
-- ============================================================
INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `status`, `hide`, `sort`) VALUES
(20, '用户管理', 'system.user',          1, 2, '/system/user',          'system/user/index',       'lucide:users',                  '', '', 1, 1, 1),
(21, '角色管理', 'system.role',          1, 2, '/system/role',          'system/role/index',       'lucide:user-cog',               '', '', 1, 1, 2),
(22, '菜单管理', 'system.menu',          1, 2, '/system/menu',          'system/menu/index',       'lucide:menu',                   '', '', 1, 1, 3),
(23, '部门管理', 'system.dept',          1, 2, '/system/dept',          'dept/list/index',         'lucide:network',                '', '', 1, 1, 4),
(24, '权限管理', 'system.permission',    1, 2, '/system/permission',    'permission/list/index',   'ant-design:apartment-outlined', '', '', 1, 1, 5),
(25, '公司管理', 'system.company',       1, 2, '/system/company',       'company/list/index',      'ant-design:home-outlined',      '', '', 1, 1, 6),
(26, '个人设置', 'system.profile',       1, 2, '/system/profile',       'system/profile/index',    'lucide:user',                   '', '', 1, 1, 99),
(27, '操作日志', 'system.log-operation', 1, 2, '/system/log/operation', 'log/operation/index',     'lucide:file-clock',             '', '', 1, 1, 50),
(28, '登录日志', 'system.log-login',     1, 2, '/system/log/login',     'log/login/index',         'lucide:log-in',                 '', '', 1, 1, 51);

-- ============================================================
-- 抖音盯盘子菜单
-- ============================================================
INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `status`, `hide`, `sort`) VALUES
(40, '商品池', 'douyin.product-pool', 1, 4, '/douyin/product-pool', 'douyin/product-pool/index', 'lucide:shopping-bag', '', '', 1, 1, 1),
(41, '视频池', 'douyin.video-pool',   1, 4, '/douyin/video-pool',   'douyin/video-pool/index',   'lucide:film',         '', '', 1, 1, 2);

-- ============================================================
-- 按钮权限（type=2）
-- ============================================================
INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `status`, `hide`, `sort`) VALUES
-- 用户管理按钮
(100, '新增用户', 'user:add',          2, 20, '', '', '', 'POST',   '/api/system/user/save',         1, 0, 1),
(101, '编辑用户', 'user:edit',         2, 20, '', '', '', 'POST',   '/api/system/user/save',         1, 0, 2),
(102, '删除用户', 'user:delete',       2, 20, '', '', '', 'POST',   '/api/system/user/delete',       1, 0, 3),
(103, '分配角色', 'user:assign-roles', 2, 20, '', '', '', 'POST',   '/api/system/user/assign-roles', 1, 0, 4),
-- 角色管理按钮
(110, '新增角色', 'role:add',          2, 21, '', '', '', 'POST',   '/api/system/role/save',         1, 0, 1),
(111, '编辑角色', 'role:edit',         2, 21, '', '', '', 'POST',   '/api/system/role/detail',       1, 0, 2),
(112, '删除角色', 'role:delete',       2, 21, '', '', '', 'POST',   '/api/system/role/delete',       1, 0, 3),
(113, '分配权限', 'role:permission',   2, 21, '', '', '', 'POST',   '/api/system/role/assign-menus', 1, 0, 4),
-- 公司管理按钮
(120, '新增公司', 'company:add',       2, 25, '', '', '', 'POST',   '/api/system/company/save',      1, 0, 1),
(121, '编辑公司', 'company:edit',      2, 25, '', '', '', 'POST',   '/api/system/company/save',      1, 0, 2),
(122, '删除公司', 'company:delete',    2, 25, '', '', '', 'DELETE', '/api/system/company/delete',    1, 0, 3),
-- 菜单管理按钮
(130, '新增菜单', 'menu:add',          2, 22, '', '', '', 'POST',   '/api/system/menu/create',       1, 0, 1),
(131, '编辑菜单', 'menu:edit',         2, 22, '', '', '', 'PUT',    '/api/system/menu/update',       1, 0, 2),
(132, '删除菜单', 'menu:delete',       2, 22, '', '', '', 'DELETE', '/api/system/menu/delete',       1, 0, 3),
-- 权限管理按钮
(140, '新增权限', 'permission:add',    2, 24, '', '', '', 'POST',   '/api/system/permission/create', 1, 0, 1),
(141, '编辑权限', 'permission:edit',   2, 24, '', '', '', 'PUT',    '/api/system/permission/update', 1, 0, 2),
(142, '删除权限', 'permission:delete', 2, 24, '', '', '', 'DELETE', '/api/system/permission/delete', 1, 0, 3);

-- ============================================================
-- 重新给超级管理员角色（id=1）分配所有权限
-- ============================================================
INSERT INTO `sys_role_permission` (`role_id`, `permission_id`)
SELECT 1, id FROM `sys_permission`;
