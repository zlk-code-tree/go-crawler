-- 内部账号与千川广告主映射表
-- 映射关系：(account_id, company_id) → qianchuan_advertiser_id
-- 抖音规则：单一主体下 1抖音号 ↔ 1千川账户，多主体下一对多
-- 执行时间：2026-06-25

CREATE TABLE IF NOT EXISTS account_qianchuan_mapping (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  account_id INT NOT NULL COMMENT '内部账号ID，关联account.id',
  company_id BIGINT NOT NULL DEFAULT 0 COMMENT '所属公司ID，关联sys_company.id',
  qianchuan_advertiser_id BIGINT NOT NULL COMMENT '千川广告主ID',
  qianchuan_advertiser_name VARCHAR(200) DEFAULT '' COMMENT '千川广告主名称（冗余，方便查看）',
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_account_company (account_id, company_id),
  KEY idx_advertiser_id (qianchuan_advertiser_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='内部账号与千川广告主映射表';
