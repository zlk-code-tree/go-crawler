-- 查询所有外链菜单的详细信息
SELECT
    id,
    name,
    code,
    type,
    parent_id,
    path,
    component,
    icon,
    ext_url,
    hide,
    status,
    sort
FROM sys_permission
WHERE ext_url != '' AND ext_url IS NOT NULL
ORDER BY sort ASC;

-- 如果外链菜单的 hide=0 或 status=0，执行以下更新
-- UPDATE sys_permission SET hide = 1, status = 1 WHERE ext_url != '' AND ext_url IS NOT NULL;
