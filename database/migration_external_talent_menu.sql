-- 外部账号菜单
-- 适配当前 ly_crawler 库中的 sys_permission 结构
-- 一级直达页风格，和 店铺管理 / 投手管理 / 账号管理 保持一致

INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `hide`, `status`, `sort`)
SELECT 161, '外部账号', 'external-talent', 2, 0, '/external-talent', 'external-talent/index', 'ant-design:team-outlined', '', '', 1, 1, 2
WHERE NOT EXISTS (
    SELECT 1 FROM `sys_permission` WHERE `code` = 'external-talent'
);

INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `hide`, `status`, `sort`)
SELECT 162, '外部账号新增', 'external-talent:add', 3, 161, '', '', '', 'POST', '/api/external-talent/save', 1, 1, 2
WHERE NOT EXISTS (
    SELECT 1 FROM `sys_permission` WHERE `code` = 'external-talent:add'
);

INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `hide`, `status`, `sort`)
SELECT 163, '外部账号编辑', 'external-talent:edit', 3, 161, '', '', '', 'POST', '/api/external-talent/save', 1, 1, 3
WHERE NOT EXISTS (
    SELECT 1 FROM `sys_permission` WHERE `code` = 'external-talent:edit'
);

INSERT INTO `sys_permission` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `method`, `api`, `hide`, `status`, `sort`)
SELECT 164, '外部账号删除', 'external-talent:delete', 3, 161, '', '', '', 'POST', '/api/external-talent/delete', 1, 1, 4
WHERE NOT EXISTS (
    SELECT 1 FROM `sys_permission` WHERE `code` = 'external-talent:delete'
);

INSERT INTO `sys_role_permission` (`role_id`, `permission_id`)
SELECT rp.role_id, p.id
FROM `sys_permission` p
JOIN (
    SELECT 8 AS role_id
    UNION ALL SELECT 9
    UNION ALL SELECT 10
) rp
WHERE p.code IN (
    'external-talent',
    'external-talent:add',
    'external-talent:edit',
    'external-talent:delete'
)
AND NOT EXISTS (
    SELECT 1
    FROM `sys_role_permission` srp
    WHERE srp.role_id = rp.role_id AND srp.permission_id = p.id
);
