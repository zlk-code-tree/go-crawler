-- 迁移：登录日志状态码从 0/1 改为 1/2
-- 旧: 0=失败, 1=成功
-- 新: 1=成功, 2=失败

UPDATE sys_login_log SET status = 2 WHERE status = 0;
-- status=1 的记录保持不变（仍然是成功）
