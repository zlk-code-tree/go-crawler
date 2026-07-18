-- ============================================================
-- 抖音盯盘菜单聚合页
-- 目标：
-- 1. 将原「数据增长中心」父菜单更名为「抖音盯盘」
-- 2. 新增「抖音盯盘」聚合页菜单，指向前端 douyin/monitor/index
-- 3. 隐藏旧的「商品池」「视频池」独立菜单入口，页面仍由聚合页 tab 复用
-- 4. 将新菜单权限分配给已有抖音菜单权限的角色
-- ============================================================

START TRANSACTION;

SET @douyin_id := (
  SELECT id FROM sys_permission WHERE code = 'douyin' LIMIT 1
);

UPDATE sys_permission
SET
  name = '抖音盯盘',
  path = '/douyin',
  component = 'BasicLayout',
  icon = 'lucide:radar',
  status = 1,
  hide = 1
WHERE id = @douyin_id;

SET @monitor_id := (
  SELECT id FROM sys_permission WHERE code = 'douyin.monitor' LIMIT 1
);

SET @monitor_id := IFNULL(
  @monitor_id,
  (SELECT IFNULL(MAX(id), 0) + 1 FROM sys_permission)
);

INSERT INTO sys_permission (
  id,
  name,
  code,
  type,
  parent_id,
  path,
  component,
  icon,
  method,
  api,
  status,
  hide,
  sort
)
VALUES (
  @monitor_id,
  '抖音盯盘',
  'douyin.monitor',
  2,
  @douyin_id,
  '/douyin/monitor',
  'douyin/monitor/index',
  'lucide:radar',
  '',
  '',
  1,
  1,
  1
)
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  type = VALUES(type),
  parent_id = VALUES(parent_id),
  path = VALUES(path),
  component = VALUES(component),
  icon = VALUES(icon),
  status = VALUES(status),
  hide = VALUES(hide),
  sort = VALUES(sort);

UPDATE sys_permission
SET hide = 0
WHERE code IN ('douyin.product-pool', 'douyin.video-pool');

INSERT IGNORE INTO sys_role_permission (role_id, permission_id)
SELECT DISTINCT role_id, @monitor_id
FROM sys_role_permission
WHERE permission_id IN (
  @douyin_id,
  (SELECT id FROM sys_permission WHERE code = 'douyin.product-pool' LIMIT 1),
  (SELECT id FROM sys_permission WHERE code = 'douyin.video-pool' LIMIT 1)
);

INSERT IGNORE INTO sys_role_permission (role_id, permission_id)
VALUES (1, @monitor_id);

COMMIT;
