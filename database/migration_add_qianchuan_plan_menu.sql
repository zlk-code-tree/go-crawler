-- 财务管理 - 千川计划菜单
-- 按权限编码定位父菜单，避免依赖不同环境中的自增 ID。

INSERT INTO `sys_permission` (
    `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`,
    `method`, `api`, `ext_url`, `hide`, `status`, `sort`
)
SELECT
    '千川计划',
    'finance.qianchuan-plan',
    2,
    finance.id,
    '/finance/qianchuan-plan',
    'finance/qianchuan-plan/index',
    'ant-design:fund-outlined',
    '',
    '',
    '',
    1,
    1,
    1
FROM `sys_permission` finance
WHERE finance.code = 'finance'
  AND NOT EXISTS (
      SELECT 1
      FROM `sys_permission`
      WHERE `code` = 'finance.qianchuan-plan'
  )
LIMIT 1;

-- 已存在时同步为当前路由配置，确保迁移可重复执行。
UPDATE `sys_permission` plan
JOIN `sys_permission` finance ON finance.code = 'finance'
SET plan.name = '千川计划',
    plan.type = 2,
    plan.parent_id = finance.id,
    plan.path = '/finance/qianchuan-plan',
    plan.component = 'finance/qianchuan-plan/index',
    plan.icon = 'ant-design:fund-outlined',
    plan.method = '',
    plan.api = '',
    plan.ext_url = '',
    plan.hide = 1,
    plan.status = 1,
    plan.sort = 1
WHERE plan.code = 'finance.qianchuan-plan';

-- 新菜单继承财务管理父菜单的角色授权。
INSERT IGNORE INTO `sys_role_permission` (`role_id`, `permission_id`)
SELECT DISTINCT parent_role.role_id, plan.id
FROM `sys_role_permission` parent_role
JOIN `sys_permission` finance
  ON finance.id = parent_role.permission_id
 AND finance.code = 'finance'
JOIN `sys_permission` plan
  ON plan.code = 'finance.qianchuan-plan';
