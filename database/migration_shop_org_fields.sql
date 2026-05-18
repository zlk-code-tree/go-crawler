-- ============================================
-- 店铺组织架构字段迁移脚本
-- 用于支持订单统计分析的多级组织筛选功能
-- ============================================

-- 在 shop 表中增加组织层级字段，关联现有的 RBAC 表
ALTER TABLE shop ADD COLUMN company_id BIGINT UNSIGNED DEFAULT 0 COMMENT '所属公司ID（关联sys_company）';
ALTER TABLE shop ADD COLUMN department_id BIGINT UNSIGNED DEFAULT 0 COMMENT '所属部门ID（关联sys_department）';
ALTER TABLE shop ADD COLUMN group_type VARCHAR(20) DEFAULT '' COMMENT '组类型：invest(投放组)/shop(店铺组)';
ALTER TABLE shop ADD COLUMN member_name VARCHAR(50) DEFAULT '' COMMENT '成员名称';

-- 添加索引
ALTER TABLE shop ADD INDEX idx_company_id (company_id);
ALTER TABLE shop ADD INDEX idx_department_id (department_id);
ALTER TABLE shop ADD INDEX idx_group_type (group_type);
ALTER TABLE shop ADD INDEX idx_operation_leader (operation_leader);

-- 字段说明：
-- company_id: 关联 sys_company.id，用于区分集团/分公司
-- department_id: 关联 sys_department.id，用于区分部门
-- group_type: 区分投放组/店铺组，固定值：invest(投放组) 或 shop(店铺组)
-- member_name: 店铺归属的成员名称
