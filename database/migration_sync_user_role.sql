-- 修复存量数据：将 sys_user.role_id 同步到 sys_user_role 关联表
-- 只补充缺失的记录，不影响已有的多角色绑定

INSERT INTO sys_user_role (user_id, role_id)
SELECT u.id, u.role_id
FROM sys_user u
WHERE u.role_id > 0
  AND NOT EXISTS (
    SELECT 1 FROM sys_user_role ur WHERE ur.user_id = u.id
  );
