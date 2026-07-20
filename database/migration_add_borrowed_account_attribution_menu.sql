-- 财务管理 - 借号归属规则菜单
-- 按 code 定位，迁移可重复执行。

INSERT INTO `sys_permission` (
    `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`,
    `method`, `api`, `ext_url`, `hide`, `status`, `sort`
)
SELECT
    '借号归属规则',
    'finance.borrowed-account-attribution',
    2,
    finance.id,
    '/finance/borrowed-account-attribution',
    'finance/borrowed-account-attribution/index',
    'lucide:route',
    '',
    '',
    '',
    1,
    1,
    3
FROM `sys_permission` finance
WHERE finance.code = 'finance'
  AND NOT EXISTS (
      SELECT 1
      FROM `sys_permission`
      WHERE `code` = 'finance.borrowed-account-attribution'
  )
LIMIT 1;

UPDATE `sys_permission` rule_menu
JOIN `sys_permission` finance ON finance.code = 'finance'
SET rule_menu.name = '借号归属规则',
    rule_menu.type = 2,
    rule_menu.parent_id = finance.id,
    rule_menu.path = '/finance/borrowed-account-attribution',
    rule_menu.component = 'finance/borrowed-account-attribution/index',
    rule_menu.icon = 'lucide:route',
    rule_menu.method = '',
    rule_menu.api = '',
    rule_menu.ext_url = '',
    rule_menu.hide = 1,
    rule_menu.status = 1,
    rule_menu.sort = 3
WHERE rule_menu.code = 'finance.borrowed-account-attribution';

-- 新菜单继承财务管理父菜单的角色授权。
INSERT IGNORE INTO `sys_role_permission` (`role_id`, `permission_id`)
SELECT DISTINCT parent_role.role_id, rule_menu.id
FROM `sys_role_permission` parent_role
JOIN `sys_permission` finance
  ON finance.id = parent_role.permission_id
 AND finance.code = 'finance'
JOIN `sys_permission` rule_menu
  ON rule_menu.code = 'finance.borrowed-account-attribution';
