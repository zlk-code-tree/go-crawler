-- ============================================
-- 组织架构表迁移脚本
-- 采用单表自关联设计，支持多层级树形结构
-- ============================================

-- 1. 组织架构表（统一的树形结构）
DROP TABLE IF EXISTS `org_structure`;
CREATE TABLE `org_structure` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `parent_id` BIGINT UNSIGNED DEFAULT 0 COMMENT '父节点ID（0表示根节点）',
  `name` VARCHAR(100) NOT NULL COMMENT '节点名称',
  `type` VARCHAR(20) NOT NULL COMMENT '节点类型：company/branch/manager/group/employee/platform/account',
  `code` VARCHAR(50) DEFAULT '' COMMENT '节点编码（可选，用于外部系统关联）',
  `sort` INT DEFAULT 0 COMMENT '排序值',
  `status` TINYINT DEFAULT 1 COMMENT '状态：0禁用 1启用',
  `remark` VARCHAR(500) DEFAULT '' COMMENT '备注',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_parent_id` (`parent_id`),
  KEY `idx_type` (`type`),
  KEY `idx_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='组织架构表';

-- 2. 店铺与组织关联表（店铺关联到账号级别）
DROP TABLE IF EXISTS `shop_org_relation`;
CREATE TABLE `shop_org_relation` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `shop_id` INT NOT NULL COMMENT '店铺ID（关联shop表）',
  `org_id` BIGINT UNSIGNED NOT NULL COMMENT '组织节点ID（关联org_structure表，通常是account级别）',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_shop_org` (`shop_id`, `org_id`),
  KEY `idx_shop_id` (`shop_id`),
  KEY `idx_org_id` (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='店铺组织关联表';

-- ============================================
-- 字段说明
-- ============================================
-- org_structure.type 类型说明：
--   company  - 总公司/集团
--   branch   - 分公司
--   manager  - 组长/经理
--   group    - 投放组/店铺组
--   employee - 成员/员工
--   platform - 平台（抖音/快手/视频号等）
--   account  - 账号（最细粒度，店铺关联到此级别）

-- ============================================
-- 初始化示例数据
-- ============================================
INSERT INTO `org_structure` (`id`, `parent_id`, `name`, `type`, `sort`) VALUES
(1, 0, '总公司', 'company', 1),
(2, 1, '济宁', 'branch', 1),
(3, 1, '济南', 'branch', 2),
(4, 1, '广州', 'branch', 3),
(5, 2, '李艳雨', 'manager', 1),
(6, 5, '投放一组', 'group', 1),
(7, 5, '店铺组', 'group', 2),
(8, 6, '李百川', 'employee', 1),
(9, 6, '刘浩然', 'employee', 2),
(10, 6, '李倩南', 'employee', 3),
(11, 6, '黄子龙', 'employee', 4),
(12, 8, '抖音', 'platform', 1),
(13, 8, '快手', 'platform', 2),
(14, 8, '视频号', 'platform', 3),
(15, 12, '账号1', 'account', 1),
(16, 12, '账号2', 'account', 2),
(17, 12, '账号3', 'account', 3),
(18, 13, '账号4', 'account', 1),
(19, 13, '账号5', 'account', 2),
(20, 13, '账号6', 'account', 3),
(21, 14, '账号7', 'account', 1),
(22, 14, '账号8', 'account', 2),
(23, 14, '账号9', 'account', 3),
(24, 7, '张文蔚/孟令饶', 'employee', 1),
(25, 7, '王春迪', 'employee', 2),
(26, 7, '胡惠惠', 'employee', 3),
(27, 7, '王梅雪', 'employee', 4),
(28, 2, '冯宇', 'manager', 2),
(29, 28, '投放二组', 'group', 1);
