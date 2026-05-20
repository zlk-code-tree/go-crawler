-- ============================================
-- RBAC 权限管理系统数据库设计
-- 支持多公司、多层级、多角色的权限控制
-- ============================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ============================================
-- 1. 公司表 (sys_company)
-- ============================================
DROP TABLE IF EXISTS `sys_company`;
CREATE TABLE `sys_company` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `name` varchar(100) NOT NULL COMMENT '公司名称',
  `code` varchar(50) NOT NULL COMMENT '公司编码（唯一标识）',
  `parent_id` bigint unsigned DEFAULT '0' COMMENT '父公司 ID（支持集团架构）',
  `level` int DEFAULT '1' COMMENT '公司层级（1-集团，2-子公司，3-分公司）',
  `status` tinyint DEFAULT '1' COMMENT '状态（0-禁用，1-启用）',
  `sort` int DEFAULT '0' COMMENT '排序值',
  `remark` varchar(500) DEFAULT '' COMMENT '备注',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`),
  KEY `idx_parent_id` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='公司表';

-- ============================================
-- 2. 部门表 (sys_department)
-- ============================================
DROP TABLE IF EXISTS `sys_department`;
CREATE TABLE `sys_department` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `company_id` bigint unsigned NOT NULL COMMENT '所属公司 ID',
  `name` varchar(100) NOT NULL COMMENT '部门名称',
  `parent_id` bigint unsigned DEFAULT '0' COMMENT '父部门 ID（支持多级部门）',
  `level` int DEFAULT '1' COMMENT '部门层级',
  `leader_id` bigint unsigned DEFAULT NULL COMMENT '部门负责人 ID',
  `status` tinyint DEFAULT '1' COMMENT '状态（0-禁用，1-启用）',
  `sort` int DEFAULT '0' COMMENT '排序值',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_company_id` (`company_id`),
  KEY `idx_parent_id` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='部门表';

-- ============================================
-- 3. 用户表 (sys_user)
-- ============================================
DROP TABLE IF EXISTS `sys_user`;
CREATE TABLE `sys_user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `username` varchar(50) NOT NULL COMMENT '用户名（登录账号）',
  `password` varchar(255) NOT NULL COMMENT '密码（bcrypt 加密）',
  `real_name` varchar(50) DEFAULT '' COMMENT '真实姓名',
  `email` varchar(100) DEFAULT '' COMMENT '邮箱',
  `phone` varchar(20) DEFAULT '' COMMENT '手机号',
  `avatar` varchar(500) DEFAULT '' COMMENT '头像 URL',
  `gender` tinyint DEFAULT '0' COMMENT '性别（0-未知，1-男，2-女）',
  `status` tinyint DEFAULT '1' COMMENT '状态（0-禁用，1-启用，2-锁定）',
  `last_login_at` datetime DEFAULT NULL COMMENT '最后登录时间',
  `last_login_ip` varchar(50) DEFAULT '' COMMENT '最后登录 IP',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  KEY `idx_phone` (`phone`),
  KEY `idx_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';

-- ============================================
-- 4. 用户 - 公司关联表 (sys_user_company)
-- ============================================
DROP TABLE IF EXISTS `sys_user_company`;
CREATE TABLE `sys_user_company` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户 ID',
  `company_id` bigint unsigned NOT NULL COMMENT '公司 ID',
  `department_id` bigint unsigned DEFAULT NULL COMMENT '部门 ID',
  `is_primary` tinyint DEFAULT '0' COMMENT '是否主公司（0-否，1-是）',
  `joined_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '加入时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_company` (`user_id`, `company_id`),
  KEY `idx_company_id` (`company_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户 - 公司关联表（支持用户在多个公司任职）';

-- ============================================
-- 5. 角色表 (sys_role)
-- ============================================
DROP TABLE IF EXISTS `sys_role`;
CREATE TABLE `sys_role` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `name` varchar(50) NOT NULL COMMENT '角色名称',
  `code` varchar(50) NOT NULL COMMENT '角色编码（唯一标识）',
  `company_id` bigint unsigned DEFAULT NULL COMMENT '所属公司 ID（NULL 表示系统级角色）',
  `type` tinyint DEFAULT '1' COMMENT '角色类型（1-系统角色，2-公司角色，3-自定义角色）',
  `data_scope` tinyint DEFAULT '1' COMMENT '数据范围（1-全部，2-本公司，3-本部门，4-仅本人）',
  `status` tinyint DEFAULT '1' COMMENT '状态（0-禁用，1-启用）',
  `sort` int DEFAULT '0' COMMENT '排序值',
  `remark` varchar(500) DEFAULT '' COMMENT '备注',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`),
  KEY `idx_company_id` (`company_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色表';

-- ============================================
-- 6. 权限表 (sys_permission)
-- ============================================
DROP TABLE IF EXISTS `sys_permission`;
CREATE TABLE `sys_permission` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `name` varchar(100) NOT NULL COMMENT '权限名称',
  `code` varchar(100) NOT NULL COMMENT '权限编码（如：user:add, user:edit）',
  `type` tinyint DEFAULT '1' COMMENT '权限类型（1-菜单，2-按钮，3-接口）',
  `parent_id` bigint unsigned DEFAULT '0' COMMENT '父权限 ID（树形结构）',
  `path` varchar(200) DEFAULT '' COMMENT '前端路由路径',
  `component` varchar(200) DEFAULT '' COMMENT '前端组件路径',
  `icon` varchar(100) DEFAULT '' COMMENT '图标',
  `method` varchar(10) DEFAULT '' COMMENT 'HTTP 方法（GET/POST/PUT/DELETE）',
  `api` varchar(200) DEFAULT '' COMMENT 'API 接口路径',
  `status` tinyint DEFAULT '1' COMMENT '状态（0-禁用，1-启用）',
  `sort` int DEFAULT '0' COMMENT '排序值',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`),
  KEY `idx_parent_id` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='权限表（菜单 + 按钮 + 接口）';

-- ============================================
-- 7. 角色 - 权限关联表 (sys_role_permission)
-- ============================================
DROP TABLE IF EXISTS `sys_role_permission`;
CREATE TABLE `sys_role_permission` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `role_id` bigint unsigned NOT NULL COMMENT '角色 ID',
  `permission_id` bigint unsigned NOT NULL COMMENT '权限 ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_permission` (`role_id`, `permission_id`),
  KEY `idx_permission_id` (`permission_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色 - 权限关联表';

-- ============================================
-- 8. 用户 - 角色关联表 (sys_user_role)
-- ============================================
DROP TABLE IF EXISTS `sys_user_role`;
CREATE TABLE `sys_user_role` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户 ID',
  `role_id` bigint unsigned NOT NULL COMMENT '角色 ID',
  `company_id` bigint unsigned DEFAULT NULL COMMENT '公司 ID（NULL 表示全局角色）',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_role_company` (`user_id`, `role_id`, `company_id`),
  KEY `idx_role_id` (`role_id`),
  KEY `idx_company_id` (`company_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户 - 角色关联表（支持用户在不同公司有不同角色）';

-- ============================================
-- 9. 操作日志表 (sys_operation_log)
-- ============================================
DROP TABLE IF EXISTS `sys_operation_log`;
CREATE TABLE `sys_operation_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `user_id` bigint unsigned DEFAULT NULL COMMENT '用户 ID',
  `username` varchar(50) DEFAULT '' COMMENT '用户名',
  `module` varchar(50) DEFAULT '' COMMENT '操作模块',
  `action` varchar(50) DEFAULT '' COMMENT '操作类型',
  `method` varchar(10) DEFAULT '' COMMENT '请求方法',
  `url` varchar(500) DEFAULT '' COMMENT '请求 URL',
  `params` text COMMENT '请求参数（JSON）',
  `ip` varchar(50) DEFAULT '' COMMENT 'IP 地址',
  `user_agent` varchar(500) DEFAULT '' COMMENT 'User-Agent',
  `status` tinyint DEFAULT '1' COMMENT '执行状态（0-失败，1-成功）',
  `error_msg` varchar(1000) DEFAULT '' COMMENT '错误信息',
  `duration` int DEFAULT '0' COMMENT '执行时长（ms）',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_module` (`module`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='操作日志表';

-- ============================================
-- 10. 登录日志表 (sys_login_log)
-- ============================================
DROP TABLE IF EXISTS `sys_login_log`;
CREATE TABLE `sys_login_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
  `user_id` bigint unsigned DEFAULT NULL COMMENT '用户 ID',
  `username` varchar(50) DEFAULT '' COMMENT '用户名',
  `ip` varchar(50) DEFAULT '' COMMENT 'IP 地址',
  `location` varchar(200) DEFAULT '' COMMENT '登录地点',
  `browser` varchar(100) DEFAULT '' COMMENT '浏览器',
  `os` varchar(100) DEFAULT '' COMMENT '操作系统',
  `user_agent` varchar(500) DEFAULT '' COMMENT 'User-Agent原始值',
  `status` tinyint DEFAULT '1' COMMENT '登录状态（1-成功，2-失败）',
  `msg` varchar(500) DEFAULT '' COMMENT '提示信息',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='登录日志表';

SET FOREIGN_KEY_CHECKS = 1;

-- ============================================
-- 初始化基础数据
-- ============================================

-- 1. 初始化公司数据
INSERT INTO `sys_company` (`id`, `name`, `code`, `parent_id`, `level`, `status`, `sort`) VALUES
(1, '集团公司', 'GROUP', 0, 1, 1, 1),
(2, '北京分公司', 'BJ_BRANCH', 1, 2, 1, 1),
(3, '上海分公司', 'SH_BRANCH', 1, 2, 1, 2),
(4, '深圳分公司', 'SZ_BRANCH', 1, 2, 1, 3);

-- 2. 初始化部门数据
INSERT INTO `sys_department` (`id`, `company_id`, `name`, `parent_id`, `level`, `status`, `sort`) VALUES
(1, 1, '总经办', 0, 1, 1, 1),
(2, 1, '技术部', 0, 1, 1, 2),
(3, 1, '产品部', 0, 1, 1, 3),
(4, 2, '技术部', 0, 1, 1, 1),
(5, 2, '销售部', 0, 1, 1, 2),
(6, 3, '技术部', 0, 1, 1, 1),
(7, 4, '技术部', 0, 1, 1, 1);

-- 3. 初始化角色数据
INSERT INTO `sys_role` (`id`, `name`, `code`, `company_id`, `type`, `data_scope`, `status`, `sort`) VALUES
(1, '超级管理员', 'SUPER_ADMIN', NULL, 1, 1, 1, 1),
(2, '系统管理员', 'SYSTEM_ADMIN', NULL, 1, 1, 1, 2),
(3, '公司董事长', 'CHAIRMAN', 1, 2, 1, 1, 1),
(4, '分公司总经理', 'BRANCH_MANAGER', 2, 2, 2, 1, 1),
(5, '部门经理', 'DEPT_MANAGER', NULL, 2, 3, 1, 1),
(6, '普通员工', 'EMPLOYEE', NULL, 2, 4, 1, 6);

-- 4. 初始化权限数据（菜单 + 按钮）
INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `status`, `sort`) VALUES
-- 一级菜单
(1, '系统管理', 'system', 1, 0, '/system', 'Layout', 'setting', '', '', 1, 1),
(2, '用户管理', 'user', 1, 0, '/user', 'Layout', 'user', '', '', 1, 2),
(3, '公司管理', 'company', 1, 0, '/company', 'Layout', 'home', '', '', 1, 3),
(4, '角色管理', 'role', 1, 0, '/role', 'Layout', 'peoples', '', '', 1, 4),
(5, '权限管理', 'permission', 1, 0, '/permission', 'Layout', 'tree', '', '', 1, 5),
(6, '日志管理', 'log', 1, 0, '/log', 'Layout', 'documentation', '', '', 1, 6),

-- 系统管理子菜单
(10, '个人设置', 'system:profile', 1, 1, '/system/profile', 'system/profile/index', 'user', '', '', 1, 1),
(11, '修改密码', 'system:password', 2, 1, '', '', '', 'PUT', '/api/system/password', 1, 2),

-- 用户管理子菜单和按钮
(20, '用户列表', 'user:list', 1, 2, '/user/list', 'user/list/index', 'list', '', '', 1, 1),
(21, '用户新增', 'user:add', 2, 2, '', '', '', 'POST', '/api/user/add', 1, 2),
(22, '用户编辑', 'user:edit', 2, 2, '', '', '', 'PUT', '/api/user/edit', 1, 3),
(23, '用户删除', 'user:delete', 2, 2, '', '', '', 'DELETE', '/api/user/delete', 1, 4),
(24, '分配角色', 'user:role', 2, 2, '', '', '', 'POST', '/api/user/role', 1, 5),

-- 公司管理子菜单和按钮
(30, '公司列表', 'company:list', 1, 3, '/company/list', 'company/list/index', 'list', '', '', 1, 1),
(31, '公司新增', 'company:add', 2, 3, '', '', '', 'POST', '/api/company/add', 1, 2),
(32, '公司编辑', 'company:edit', 2, 3, '', '', '', 'PUT', '/api/company/edit', 1, 3),
(33, '公司删除', 'company:delete', 2, 3, '', '', '', 'DELETE', '/api/company/delete', 1, 4),

-- 角色管理子菜单和按钮
(40, '角色列表', 'role:list', 1, 4, '/role/list', 'role/list/index', 'list', '', '', 1, 1),
(41, '角色新增', 'role:add', 2, 4, '', '', '', 'POST', '/api/role/add', 1, 2),
(42, '角色编辑', 'role:edit', 2, 4, '', '', '', 'PUT', '/api/role/edit', 1, 3),
(43, '角色删除', 'role:delete', 2, 4, '', '', '', 'DELETE', '/api/role/delete', 1, 4),
(44, '分配权限', 'role:permission', 2, 4, '', '', '', 'POST', '/api/role/permission', 1, 5),

-- 权限管理子菜单和按钮
(50, '权限列表', 'permission:list', 1, 5, '/permission/list', 'permission/list/index', 'tree-table', '', '', 1, 1),
(51, '权限新增', 'permission:add', 2, 5, '', '', '', 'POST', '/api/permission/add', 1, 2),
(52, '权限编辑', 'permission:edit', 2, 5, '', '', '', 'PUT', '/api/permission/edit', 1, 3),
(53, '权限删除', 'permission:delete', 2, 5, '', '', '', 'DELETE', '/api/permission/delete', 1, 4),

-- 日志管理子菜单
(60, '操作日志', 'log:operation', 1, 6, '/log/operation', 'log/operation/index', 'documentation', '', '', 1, 1),
(61, '登录日志', 'log:login', 1, 6, '/log/login', 'log/login/index', 'documentation', '', '', 1, 2);

-- 5. 给超级管理员分配所有权限
INSERT INTO `sys_role_permission` (`role_id`, `permission_id`)
SELECT 1, id FROM `sys_permission`;

-- 6. 初始化默认用户（密码：admin123，使用 bcrypt 加密）
-- 注意：实际密码需要使用 bcrypt 加密，这里使用占位符，首次启动时需要更新
INSERT INTO `sys_user` (`id`, `username`, `password`, `real_name`, `email`, `phone`, `status`) VALUES
(1, 'admin', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iKTVKIUi', '张立昆', 'admin@example.com', '13800138000', 1),
(2, 'beijing_mgr', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iKTVKIUi', '北京经理', 'beijing@example.com', '13800138001', 1),
(3, 'shanghai_mgr', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iKTVKIUi', '上海经理', 'shanghai@example.com', '13800138002', 1),
(4, 'employee1', '$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8iAt6Z5EHsM8lE9lBOsl7iKTVKIUi', '员工 1', 'emp1@example.com', '13800138003', 1);

-- 7. 分配用户到公司
INSERT INTO `sys_user_company` (`user_id`, `company_id`, `department_id`, `is_primary`) VALUES
(1, 1, 1, 1),  -- admin 在集团公司总经办
(2, 2, 4, 1),  -- 北京经理在北京分公司技术部
(3, 3, 6, 1),  -- 上海经理在上海分公司技术部
(4, 2, 4, 1);  -- 员工 1 在北京分公司技术部

-- 8. 分配用户角色
INSERT INTO `sys_user_role` (`user_id`, `role_id`, `company_id`) VALUES
(1, 1, NULL),      -- admin 是超级管理员（全局）
(2, 4, 2),         -- 北京经理是北京分公司总经理
(3, 4, 3),         -- 上海经理是上海分公司总经理
(4, 6, 2);         -- 员工 1 是北京分公司普通员工

-- ============================================
-- 视图：用户完整信息（包含公司、部门、角色）
-- ============================================
DROP VIEW IF EXISTS `v_user_full`;
CREATE VIEW `v_user_full` AS
SELECT 
    u.id,
    u.username,
    u.real_name,
    u.email,
    u.phone,
    u.avatar,
    u.status as user_status,
    uc.company_id,
    c.name as company_name,
    c.code as company_code,
    uc.department_id,
    d.name as department_name,
    uc.is_primary,
    r.id as role_id,
    r.name as role_name,
    r.code as role_code
FROM sys_user u
LEFT JOIN sys_user_company uc ON u.id = uc.user_id
LEFT JOIN sys_company c ON uc.company_id = c.id
LEFT JOIN sys_department d ON uc.department_id = d.id
LEFT JOIN sys_user_role ur ON u.id = ur.user_id AND (ur.company_id = uc.company_id OR ur.company_id IS NULL)
LEFT JOIN sys_role r ON ur.role_id = r.id
WHERE u.status = 1;

-- ============================================
-- 视图：用户权限列表
-- ============================================
DROP VIEW IF EXISTS `v_user_permissions`;
CREATE VIEW `v_user_permissions` AS
SELECT DISTINCT
    u.id as user_id,
    u.username,
    p.id as permission_id,
    p.code as permission_code,
    p.name as permission_name,
    p.type as permission_type,
    p.path,
    p.component,
    p.method,
    p.api
FROM sys_user u
INNER JOIN sys_user_role ur ON u.id = ur.user_id
INNER JOIN sys_role_permission rp ON ur.role_id = rp.role_id
INNER JOIN sys_permission p ON rp.permission_id = p.id
WHERE u.status = 1 AND p.status = 1;
