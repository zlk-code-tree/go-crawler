-- 财务管理 - 千川账户管理
-- 来源：/Users/zhanglikun/Desktop/表格/千川账户.xlsx

CREATE TABLE IF NOT EXISTS `qianchuan_account_owner_mapping` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `advertiser_id` BIGINT(20) NOT NULL COMMENT '千川广告主ID/千川户号',
  `advertiser_name` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '千川户名称',
  `owner_name` VARCHAR(100) NOT NULL DEFAULT '' COMMENT 'Excel中的达人/归属名称',
  `trader_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '匹配到的投手/达人ID，0表示未匹配或备用',
  `company_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属公司ID',
  `department_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属部门ID',
  `remark` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '备注',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0=停用 1=启用',
  `source_file` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '来源文件',
  `source_sheet` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '来源工作表',
  `source_row_no` INT NOT NULL DEFAULT 0 COMMENT '来源行号',
  `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_advertiser_id` (`advertiser_id`),
  KEY `idx_owner_name` (`owner_name`),
  KEY `idx_trader_id` (`trader_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='千川账户管理表';

INSERT INTO `qianchuan_account_owner_mapping`
  (`advertiser_id`, `advertiser_name`, `owner_name`, `trader_id`, `company_id`, `department_id`,
   `remark`, `status`, `source_file`, `source_sheet`, `source_row_no`)
SELECT seed.advertiser_id, seed.advertiser_name, seed.owner_name,
  COALESCE(t.id, 0), COALESCE(t.company_id, 0), COALESCE(t.group_id, 0),
  CASE WHEN seed.owner_name = '备用' THEN 'Excel导入：备用千川户，暂不参与达人归属' ELSE 'Excel导入：千川户达人归属' END,
  1, '千川账户.xlsx', 'Sheet1', seed.source_row_no
FROM (
  SELECT '1865237485681930' AS advertiser_id, '利云文化官方旗舰店-店铺主户' AS advertiser_name, '王梅雪组' AS owner_name, 2 AS source_row_no
  UNION ALL SELECT '1700642741413895', '天书万卷-官方号投放户', '王梅雪组', 3
  UNION ALL SELECT '1774391159738372', '利云图书企业店—店铺组', '王梅雪组', 4
  UNION ALL SELECT '1711882704612356', '利云文化进口图书-主账户', '王梅雪组', 5
  UNION ALL SELECT '1722203379291139', '闻道清北图书旗舰店主户', '王梅雪组', 6
  UNION ALL SELECT '1865237451410756', '利云文化官方旗舰店-增量户', '张文蔚/孟令饶组', 7
  UNION ALL SELECT '1849742187811268', '利云文化官方旗舰店-测试户', '张文蔚/孟令饶组', 8
  UNION ALL SELECT '1849742107401216', '利云文化官方旗舰店-放量户', '张文蔚/孟令饶组', 9
  UNION ALL SELECT '1741941292252167', '利云文化官方旗舰店-乘方', '张文蔚/孟令饶组', 10
  UNION ALL SELECT '1854729743035456', '利云文化图书旗舰店—放量户', '张文蔚/孟令饶组', 11
  UNION ALL SELECT '1858079596355591', '利云文化图书旗舰店-测试户', '张文蔚/孟令饶组', 12
  UNION ALL SELECT '1858003341685828', '利云文化图书旗舰店-增量户', '张文蔚/孟令饶组', 13
  UNION ALL SELECT '1865237466937364', '利云文化官方旗舰店-一组测试1L', '李百川组', 14
  UNION ALL SELECT '1846666008244996', '利云文化官方旗舰店-投放一组1L', '李百川组', 15
  UNION ALL SELECT '1736866640311309', '利云文化官方旗舰店-南山', '李百川组', 16
  UNION ALL SELECT '1744753932115975', '一组备用3-官方旗舰店L', '李百川组', 17
  UNION ALL SELECT '1805996971742284', '利云文化图书旗舰店-一组IP L', '李百川组', 18
  UNION ALL SELECT '1794214697805833', '一组放量-利云文化图书旗舰店L', '李百川组', 19
  UNION ALL SELECT '1779475810782349', '一组放量2-利云文化图书旗舰店L', '李百川组', 20
  UNION ALL SELECT '1768309213112327', '旗舰店-刘浩然一组-ly', '刘浩然组', 21
  UNION ALL SELECT '1787879953311748', '利云文化图书旗舰店-刘浩然', '刘浩然组', 22
  UNION ALL SELECT '1721031121889284', '利云文化官方旗舰店-一组01L', '备用', 23
  UNION ALL SELECT '1700628500072459', '利云文化官方旗舰店-个人户成长课堂', '备用', 24
  UNION ALL SELECT '1761612329327629', '利云官方旗舰-店铺IP', '备用', 25
  UNION ALL SELECT '1733892357563399', '利云文化官方旗舰店-主户备用3', '备用', 26
  UNION ALL SELECT '1865237601252551', '利云文化官方旗舰店-货款15', '备用', 27
  UNION ALL SELECT '1865237517472967', '利云文化官方旗舰店-货款6', '备用', 28
  UNION ALL SELECT '1865237544878088', '利云文化官方旗舰店-货款9', '备用', 29
  UNION ALL SELECT '1865237552392204', '利云文化官方旗舰店-货款10', '备用', 30
  UNION ALL SELECT '1865237559857179', '利云文化官方旗舰店-货款11', '备用', 31
  UNION ALL SELECT '1865237589773322', '利云文化官方旗舰店-货款14', '备用', 32
  UNION ALL SELECT '1865237581955072', '利云文化官方旗舰店-货款13', '备用', 33
  UNION ALL SELECT '1865237529264204', '利云文化官方旗舰店-货款7', '备用', 34
  UNION ALL SELECT '1865237619509321', '利云文化官方旗舰店-冯宇教辅', '备用', 35
  UNION ALL SELECT '1865237627955480', '利云文化官方旗舰店-冯宇社科', '备用', 36
  UNION ALL SELECT '1732618252033038', '利云文化官方旗舰店-一组投放2L', '备用', 37
  UNION ALL SELECT '1768933532047368', '利云文化官方旗舰店-投放一组备用2L', '备用', 38
  UNION ALL SELECT '1865237610782720', '利云文化官方旗舰店-刘浩然', '备用', 39
  UNION ALL SELECT '1730728085825536', '利云个人户一组备用01', '备用', 40
  UNION ALL SELECT '1756537125299208', '一组备用4-官方旗舰店L', '备用', 41
  UNION ALL SELECT '1844038965581719', '利云文化官方旗舰店-投放一组备用1L', '备用', 42
  UNION ALL SELECT '1778805131632644', '利云文化图书旗舰店-主户', '备用', 43
  UNION ALL SELECT '1817876118383747', '利云文化官方旗舰店-张文蔚赵博宁', '备用', 44
  UNION ALL SELECT '1871018034399372', '测试用-利云文化图书旗舰', '备用', 45
  UNION ALL SELECT '1805996964519099', '利云文化图书旗舰店-贾率', '备用', 46
  UNION ALL SELECT '1794214974204260', '利云文化图书备用02一组', '备用', 47
  UNION ALL SELECT '1816950235580419', '欢喜图书-运营一组张春雨', '备用', 48
  UNION ALL SELECT '1816950258443308', '一组童书-利云文化图书旗舰店L', '备用', 49
  UNION ALL SELECT '1776451803016264', '半点读书货款-张锦浩', '备用', 50
  UNION ALL SELECT '1721488611758095', '半点读书-个人户小鱼麻麻', '备用', 51
  UNION ALL SELECT '1722194939401219', '半点读书-备用户2', '备用', 52
  UNION ALL SELECT '1748572735509519', '半点读书-备用户1', '备用', 53
  UNION ALL SELECT '1713106508958728', '店铺备用-天书万卷', '备用', 54
  UNION ALL SELECT '1849724241424489', '天书万卷金乡星图投放L', '备用', 55
  UNION ALL SELECT '1831272838718473', '天书万卷一组-一组L', '备用', 56
  UNION ALL SELECT '1807265413467216', '天书-共享勿动', '备用', 57
  UNION ALL SELECT '1771739390806016', '诚信店铺-利云企业店主户', '备用', 58
  UNION ALL SELECT '1835884088461578', '利云图书企业店-一组黄子龙', '备用', 59
  UNION ALL SELECT '1840511548938883', '利云图书企业店- 刘浩然', '备用', 60
  UNION ALL SELECT '1840511523550346', '利云图书企业店-一组张春雨', '备用', 61
  UNION ALL SELECT '1721210982593543', '利云文化教育旗舰店主户', '备用', 62
  UNION ALL SELECT '1814499085041035', '诚信千玺图书-主户店铺投放', '备用', 63
  UNION ALL SELECT '1863888881374537', '千玺图书专营店-刘浩然', '备用', 64
  UNION ALL SELECT '1864238934562183', '千玺图书专营店-测过审', '备用', 65
  UNION ALL SELECT '1864238941339648', '千玺图书专营店-加素材', '备用', 66
  UNION ALL SELECT '1864238951055562', '千玺图书专营店-测账号', '备用', 67
  UNION ALL SELECT '1864238958800327', '千玺图书专营店-垂直', '备用', 68
  UNION ALL SELECT '1797203882933658', '金抖云学习用品专营店-备用6', '备用', 69
  UNION ALL SELECT '1812622025959513', '新金抖云图书专营店-主户', '备用', 70
  UNION ALL SELECT '1845317128077831', '闻道清北图书旗舰店-一组L', '备用', 71
  UNION ALL SELECT '1786261207015561', '一组低客单-闻道清北L', '备用', 72
  UNION ALL SELECT '1810971634357337', '闻到清北图书旗舰店-刘浩然', '备用', 73
  UNION ALL SELECT '1824290408423424', '金抖云一组备用02', '备用', 74
  UNION ALL SELECT '1824290377173315', '金抖云一组备用01', '备用', 75
  UNION ALL SELECT '1808069159187466', '金斗云个人户一组备用01', '备用', 76
  UNION ALL SELECT '1824290394246484', '金抖云图书专营店-刘浩然', '备用', 77
  UNION ALL SELECT '1827553963543628', '金斗云一组备用03', '备用', 78
  UNION ALL SELECT '1766609268985860', '金抖云个人户一组备用02', '备用', 79
  UNION ALL SELECT '1808074612120656', '张春雨运营1组-老金抖云', '备用', 80
  UNION ALL SELECT '1810971569465610', '一组高客单-闻道清北L', '备用', 81
  UNION ALL SELECT '1845317142848647', '闻道清北图书旗舰店-33', '备用', 82
  UNION ALL SELECT '1845317113521547', '一组中客单-闻道清北L', '备用', 83
) seed
LEFT JOIN trader t
  ON t.status = 1
 AND (
   t.name = seed.owner_name
   OR t.name = CONCAT(seed.owner_name, '组')
   OR (seed.owner_name LIKE '%组' AND t.name = TRIM(TRAILING '组' FROM seed.owner_name))
 )
ON DUPLICATE KEY UPDATE
  `advertiser_name` = VALUES(`advertiser_name`),
  `owner_name` = VALUES(`owner_name`),
  `trader_id` = VALUES(`trader_id`),
  `company_id` = VALUES(`company_id`),
  `department_id` = VALUES(`department_id`),
  `remark` = VALUES(`remark`),
  `status` = VALUES(`status`),
  `source_file` = VALUES(`source_file`),
  `source_sheet` = VALUES(`source_sheet`),
  `source_row_no` = VALUES(`source_row_no`);

INSERT INTO `sys_permission` (
  `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`,
  `method`, `api`, `ext_url`, `hide`, `status`, `sort`
)
SELECT '千川账户管理', 'finance.qianchuan-account', 2, finance.id,
  '/finance/qianchuan-account', 'finance/qianchuan-account/index',
  'lucide:badge-dollar-sign', '', '', '', 1, 1, 5
FROM `sys_permission` finance
WHERE finance.code = 'finance'
  AND NOT EXISTS (
    SELECT 1 FROM `sys_permission` WHERE `code` = 'finance.qianchuan-account'
  )
LIMIT 1;

UPDATE `sys_permission` account_menu
JOIN `sys_permission` finance ON finance.code = 'finance'
SET account_menu.name = '千川账户管理',
    account_menu.type = 2,
    account_menu.parent_id = finance.id,
    account_menu.path = '/finance/qianchuan-account',
    account_menu.component = 'finance/qianchuan-account/index',
    account_menu.icon = 'lucide:badge-dollar-sign',
    account_menu.method = '',
    account_menu.api = '',
    account_menu.ext_url = '',
    account_menu.hide = 1,
    account_menu.status = 1,
    account_menu.sort = 5
WHERE account_menu.code = 'finance.qianchuan-account';

INSERT IGNORE INTO `sys_role_permission` (`role_id`, `permission_id`)
SELECT DISTINCT parent_role.role_id, account_menu.id
FROM `sys_role_permission` parent_role
JOIN `sys_permission` finance
  ON finance.id = parent_role.permission_id
 AND finance.code = 'finance'
JOIN `sys_permission` account_menu
  ON account_menu.code = 'finance.qianchuan-account';
