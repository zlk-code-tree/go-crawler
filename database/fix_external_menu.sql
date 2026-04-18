-- ============================================================
-- 修复外链菜单显示问题
-- ============================================================

-- 1. 查看所有外链菜单的状态
SELECT id, name, code, type, parent_id, path, component, ext_url, hide, status, sort
FROM sys_permission
WHERE ext_url != '' AND ext_url IS NOT NULL
ORDER BY sort ASC;

-- 2. 修复外链菜单：确保 hide=1（显示）且 status=1（启用）
UPDATE sys_permission
SET hide = 1, status = 1
WHERE ext_url != '' AND ext_url IS NOT NULL;

-- 3. 查看修复后的外链菜单
SELECT id, name, code, type, parent_id, path, component, ext_url, hide, status, sort
FROM sys_permission
WHERE ext_url != '' AND ext_url IS NOT NULL
ORDER BY sort ASC;

-- 4. 检查外链菜单是否已分配给超级管理员角色（role_id=1）
SELECT p.id, p.name, p.code, p.ext_url,
       CASE WHEN rp.permission_id IS NULL THEN '未分配' ELSE '已分配' END as assign_status
FROM sys_permission p
LEFT JOIN sys_role_permission rp ON p.id = rp.permission_id AND rp.role_id = 1
WHERE p.ext_url != '' AND p.ext_url IS NOT NULL
ORDER BY p.sort ASC;

-- 5. 给超级管理员分配所有外链菜单权限（如果未分配）
INSERT IGNORE INTO sys_role_permission (role_id, permission_id)
SELECT 1, id FROM sys_permission WHERE ext_url != '' AND ext_url IS NOT NULL;
