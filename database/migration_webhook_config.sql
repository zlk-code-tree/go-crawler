-- =====================================================
-- Webhook 推送配置表：支持按节点(公司/部门/投手)配置不同机器人群
-- 一个机器人 = 一条记录，node_type 决定推送的数据范围
-- 执行方式：mysql -u ly_admin -p ly_crawler < migration_webhook_config.sql
-- =====================================================

CREATE TABLE IF NOT EXISTS webhook_config (
  id            BIGINT AUTO_INCREMENT PRIMARY KEY,
  name          VARCHAR(100)  NOT NULL DEFAULT '' COMMENT '配置名称，如"集团总群"、"利云文化群"',
  platform      VARCHAR(20)   NOT NULL DEFAULT 'feishu' COMMENT '平台：feishu/wecom',
  webhook_url   VARCHAR(500)  NOT NULL DEFAULT '' COMMENT '机器人webhook地址',
  secret        VARCHAR(200)  NOT NULL DEFAULT '' COMMENT '签名密钥(飞书可选)',
  node_type     VARCHAR(20)   NOT NULL DEFAULT 'summary' COMMENT '节点类型：summary=汇总/company/department/trader',
  node_id       BIGINT        NOT NULL DEFAULT 0 COMMENT '节点ID(summary时为0；company=company_id；department=department_id；trader=trader_id)',
  node_name     VARCHAR(200)  NOT NULL DEFAULT '' COMMENT '节点名称(冗余，方便展示)',
  push_morning  TINYINT       NOT NULL DEFAULT 1 COMMENT '是否推早报：0=否 1=是',
  push_evening  TINYINT       NOT NULL DEFAULT 1 COMMENT '是否推晚报：0=否 1=是',
  push_periods  VARCHAR(50)   NOT NULL DEFAULT '0,7,15,30' COMMENT '推送周期，逗号分隔：0=今日,7,15,30',
  enabled       TINYINT       NOT NULL DEFAULT 1 COMMENT '是否启用：0=禁用 1=启用',
  created_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_node (node_type, node_id, platform),
  KEY idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Webhook推送配置表';
