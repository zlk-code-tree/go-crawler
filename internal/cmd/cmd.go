package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/glog"
	"github.com/gogf/gf/v2/os/gproc"

	"go-crawler/internal/controller/auth"
	ctrlCompany "go-crawler/internal/controller/company"
	"go-crawler/internal/controller/crawler"
	ctrlCrawlerSvc "go-crawler/internal/controller/crawler_service"
	ctrlData "go-crawler/internal/controller/data"
	"go-crawler/internal/controller/hello"
	ctrlMenu "go-crawler/internal/controller/menu"
	permissionCtrl "go-crawler/internal/controller/permission"
	ctrlScheduler "go-crawler/internal/controller/scheduler"
	system "go-crawler/internal/controller/system"
	ctrlTask "go-crawler/internal/controller/task"
	ctrlVideo "go-crawler/internal/controller/video"
	ctrlWs "go-crawler/internal/controller/ws"
	crawlerLogic "go-crawler/internal/logic/crawler"
	"go-crawler/internal/middleware"

	// 注册第三方平台实现（init() 自动注册）
	"go-crawler/internal/service"
	douyinSvc "go-crawler/internal/service/platform/douyin"
	"go-crawler/internal/service/platform/oceanengine"
	"go-crawler/internal/service/platform/wechatshop"
	"go-crawler/internal/service/webhook"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			// 【新增修改点】配置数据库 SQL 日志专门记录到 ./logs/sql 目录下的日期文件中
			sqlLogger := glog.New()
			sqlLogger.SetPath("./logs/sql")
			sqlLogger.SetFile("{Y-m-d}.log")
			sqlLogger.SetLevel(glog.LEVEL_ALL) // 记录所有执行细节
			s := g.Server()
			s.SetClientMaxBodySize(0) // 0表示无限制，解决大文件上传问题

			// 接通榜单 -> 视频池爬取流水线。调度器仍由管理端显式启动。
			crawlerWorker := crawlerLogic.New()
			crawlerScheduler := service.GetScheduler()
			cycleInterval := service.SchedulerCycleInterval(ctx)
			crawlerScheduler.SetCycleInterval(cycleInterval)
			crawlerScheduler.SetTaskInterval(service.TaskTypeProductRank, cycleInterval)
			crawlerScheduler.SetTaskHandler(func(taskCtx context.Context, taskType service.TaskType) error {
				switch taskType {
				case service.TaskTypeProductRank:
					return crawlerWorker.RunProductRankTask(taskCtx)
				case service.TaskTypeVideoPool:
					return crawlerWorker.RunVideoPoolTask(taskCtx)
				default:
					return fmt.Errorf("不支持的爬虫任务类型: %s", taskType)
				}
			})

			addColumnIfMissing := func(table, column, definition string) {
				row, err := g.DB().GetOne(ctx, `
					SELECT COUNT(1) AS cnt
					FROM information_schema.columns
					WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?
				`, table, column)
				if err != nil {
					g.Log().Warningf(ctx, "[DBInit] 检查字段失败 table=%s column=%s err=%v", table, column, err)
					return
				}
				if row["cnt"].Int() > 0 {
					return
				}
				if _, err := g.DB().Exec(ctx, fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN %s", table, definition)); err != nil {
					g.Log().Warningf(ctx, "[DBInit] 添加字段失败 table=%s column=%s err=%v", table, column, err)
				}
			}

			addIndexIfMissing := func(table, indexName, definition string) {
				row, err := g.DB().GetOne(ctx, `
					SELECT COUNT(1) AS cnt
					FROM information_schema.statistics
					WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?
				`, table, indexName)
				if err != nil {
					g.Log().Warningf(ctx, "[DBInit] 检查索引失败 table=%s index=%s err=%v", table, indexName, err)
					return
				}
				if row["cnt"].Int() > 0 {
					return
				}
				if _, err := g.DB().Exec(ctx, fmt.Sprintf("ALTER TABLE `%s` ADD %s", table, definition)); err != nil {
					g.Log().Warningf(ctx, "[DBInit] 添加索引失败 table=%s index=%s err=%v", table, indexName, err)
				}
			}

			dropIndexIfExists := func(table, indexName string) {
				row, err := g.DB().GetOne(ctx, `
					SELECT COUNT(1) AS cnt
					FROM information_schema.statistics
					WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?
				`, table, indexName)
				if err != nil {
					g.Log().Warningf(ctx, "[DBInit] 检查索引失败 table=%s index=%s err=%v", table, indexName, err)
					return
				}
				if row["cnt"].Int() == 0 {
					return
				}
				if _, err := g.DB().Exec(ctx, fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`", table, indexName)); err != nil {
					g.Log().Warningf(ctx, "[DBInit] 删除索引失败 table=%s index=%s err=%v", table, indexName, err)
				}
			}

			modifyColumnIfNeeded := func(table, column, definition string) {
				row, err := g.DB().GetOne(ctx, `
					SELECT IS_NULLABLE AS is_nullable
					FROM information_schema.columns
					WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?
				`, table, column)
				if err != nil {
					g.Log().Warningf(ctx, "[DBInit] 检查字段定义失败 table=%s column=%s err=%v", table, column, err)
					return
				}
				if row.IsEmpty() || strings.EqualFold(row["is_nullable"].String(), "YES") {
					return
				}
				if _, err := g.DB().Exec(ctx, fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN %s", table, definition)); err != nil {
					g.Log().Warningf(ctx, "[DBInit] 修改字段失败 table=%s column=%s err=%v", table, column, err)
				}
			}

			// 自动建表：千川token存储
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS qianchuan_token (
				id BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
				access_token VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'access_token',
				refresh_token VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'refresh_token',
				advertiser_ids TEXT COMMENT '授权的广告主ID列表',
				expires_in INT(11) NOT NULL DEFAULT 0 COMMENT '过期时间（秒）',
				expires_at DATETIME DEFAULT NULL COMMENT 'token过期时间',
				refresh_token_expires_at DATETIME DEFAULT NULL COMMENT 'refresh_token过期时间',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
				PRIMARY KEY (id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
			// 兼容已有表：添加 refresh_token 字段
			addColumnIfMissing("qianchuan_token", "refresh_token", "refresh_token VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'refresh_token' AFTER access_token")
			addColumnIfMissing("qianchuan_token", "refresh_token_expires_at", "refresh_token_expires_at DATETIME DEFAULT NULL COMMENT 'refresh_token过期时间' AFTER expires_at")

			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS store (
				id INT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
				store_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '店铺名称',
				store_id VARCHAR(100) NOT NULL DEFAULT '' COMMENT '店铺ID（外部平台店铺ID）',
				platform VARCHAR(50) NOT NULL DEFAULT 'douyin' COMMENT '平台：douyin/kuaishou/shipinhao',
				status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0-禁用 1-启用',
				blank_order_enabled TINYINT NOT NULL DEFAULT 1 COMMENT '是否参与空白单归属：0=否 1=是',
				blank_order_owner_name VARCHAR(100) NOT NULL DEFAULT '' COMMENT '空白单最终归属人',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
				PRIMARY KEY (id),
				UNIQUE KEY uk_store_id_platform (store_id, platform),
				KEY idx_platform_name (platform, store_name)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='店铺表（外部平台店铺ID）'`)
			addColumnIfMissing("store", "blank_order_enabled", "blank_order_enabled TINYINT NOT NULL DEFAULT 1 COMMENT '是否参与空白单归属：0=否 1=是' AFTER `status`")
			addColumnIfMissing("store", "blank_order_owner_name", "blank_order_owner_name VARCHAR(100) NOT NULL DEFAULT '' COMMENT '空白单最终归属人' AFTER `blank_order_enabled`")
			if _, err := g.DB().Exec(ctx, `
				UPDATE store
				SET blank_order_enabled = 1,
				    blank_order_owner_name = '王梅雪'
				WHERE platform = 'douyin'
				  AND store_name IN ('天书说故事', '利云底层逻辑')
			`); err != nil {
				g.Log().Warningf(ctx, "[DBInit] 初始化特殊店铺空白单归属失败: %v", err)
			}
			if _, err := g.DB().Exec(ctx, `
				UPDATE store
				SET blank_order_owner_name = ''
				WHERE platform = 'douyin'
				  AND store_name = '闻道清北图书旗舰店'
				  AND blank_order_owner_name = '王梅雪'
			`); err != nil {
				g.Log().Warningf(ctx, "[DBInit] 清理闻道清北店铺级空白单归属失败: %v", err)
			}
			addIndexIfMissing("store", "idx_platform_name", "KEY `idx_platform_name` (`platform`, `store_name`)")

			addColumnIfMissing("tiktok_ranking_list", "shop_id", "shop_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '店铺ID' AFTER product_id")
			addColumnIfMissing("tiktok_ranking_list", "product_url", "product_url VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '榜单详情页' AFTER shop_id")
			addColumnIfMissing("tiktok_ranking_list", "image_url", "image_url VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '商品主图地址' AFTER product_name")
			addColumnIfMissing("tiktok_ranking_list", "click_convert_rate", "click_convert_rate VARCHAR(64) NOT NULL DEFAULT '' COMMENT '点击成交转化率' AFTER click_count")
			addColumnIfMissing("tiktok_ranking_list", "product_qr_code_url", "product_qr_code_url VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '商品二维码' AFTER delivery_video_num")
			addColumnIfMissing("tiktok_ranking_list", "shop_logo_url", "shop_logo_url VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '店铺 Logo' AFTER product_qr_code_url")
			addColumnIfMissing("tiktok_ranking_list", "shop_qr_code_url", "shop_qr_code_url VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '店铺二维码' AFTER shop_logo_url")

			// 视频发现、列表和异步下载都依赖此表。部分存量环境只有
			// 榜单表，先完整建表，再由下方的兼容逻辑为旧表补字段。
			if _, err := g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS video_analysis_data (
				id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键 ID',
				task_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '爬虫任务 ID',
				product_id VARCHAR(64) DEFAULT NULL COMMENT '商品 ID',
				product_name VARCHAR(255) DEFAULT NULL COMMENT '商品名称',
				video_id VARCHAR(64) DEFAULT NULL COMMENT '视频 ID',
				video_desc VARCHAR(512) DEFAULT NULL COMMENT '短视频信息',
				author_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '视频作者',
				cover_url VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '视频封面',
				publish_time VARCHAR(64) NOT NULL DEFAULT '' COMMENT '发布时间',
				source_url VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '视频页地址',
				payment_amount VARCHAR(32) DEFAULT NULL COMMENT '用户支付金额',
				expose_user_count VARCHAR(32) DEFAULT NULL COMMENT '商品曝光人数',
				click_user_count VARCHAR(32) DEFAULT NULL COMMENT '商品点击人数',
				convert_user_count VARCHAR(32) DEFAULT NULL COMMENT '商品成交人数',
				click_rate VARCHAR(16) DEFAULT NULL COMMENT '商品点击率',
				click_convert_rate VARCHAR(16) DEFAULT NULL COMMENT '点击成交率',
				expose_convert_rate VARCHAR(16) DEFAULT NULL COMMENT '曝光成交率',
				order_count VARCHAR(32) DEFAULT NULL COMMENT '商品成交订单数',
				sales_volume VARCHAR(32) DEFAULT NULL COMMENT '商品成交件数',
				download_status VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT '下载状态',
				download_url VARCHAR(2048) NOT NULL DEFAULT '' COMMENT '视频下载直链',
				resolve_strategy VARCHAR(64) NOT NULL DEFAULT '' COMMENT '直链解析策略',
				storage_path VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '存储路径',
				file_size BIGINT NOT NULL DEFAULT 0 COMMENT '文件大小',
				last_error TEXT NULL COMMENT '最后错误',
				retry_count INT NOT NULL DEFAULT 0 COMMENT '重试次数',
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
				PRIMARY KEY (id),
				KEY idx_task_id (task_id),
				KEY idx_product_id (product_id),
				KEY idx_video_id (video_id),
				KEY idx_video_download_status (download_status),
				KEY idx_created_at (created_at)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='抖音视频分析数据表'`); err != nil {
				g.Log().Warningf(ctx, "[DBInit] 创建视频分析表失败 err=%v", err)
			}
			addColumnIfMissing("video_analysis_data", "author_name", "author_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '视频作者' AFTER video_desc")
			addColumnIfMissing("video_analysis_data", "cover_url", "cover_url VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '视频封面' AFTER author_name")
			addColumnIfMissing("video_analysis_data", "publish_time", "publish_time VARCHAR(64) NOT NULL DEFAULT '' COMMENT '发布时间' AFTER cover_url")
			addColumnIfMissing("video_analysis_data", "source_url", "source_url VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '视频页地址' AFTER publish_time")
			addColumnIfMissing("video_analysis_data", "download_status", "download_status VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT '下载状态' AFTER sales_volume")
			addColumnIfMissing("video_analysis_data", "download_url", "download_url VARCHAR(2048) NOT NULL DEFAULT '' COMMENT '视频下载直链' AFTER download_status")
			addColumnIfMissing("video_analysis_data", "resolve_strategy", "resolve_strategy VARCHAR(64) NOT NULL DEFAULT '' COMMENT '直链解析策略' AFTER download_url")
			addColumnIfMissing("video_analysis_data", "storage_path", "storage_path VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '存储路径' AFTER download_status")
			addColumnIfMissing("video_analysis_data", "file_size", "file_size BIGINT NOT NULL DEFAULT 0 COMMENT '文件大小' AFTER storage_path")
			addColumnIfMissing("video_analysis_data", "last_error", "last_error TEXT NULL COMMENT '最后错误' AFTER file_size")
			addColumnIfMissing("video_analysis_data", "retry_count", "retry_count INT NOT NULL DEFAULT 0 COMMENT '重试次数' AFTER last_error")
			addIndexIfMissing("video_analysis_data", "idx_video_download_status", "INDEX `idx_video_download_status` (`download_status`)")

			// 自动建表：千川广告主日结报表（账户维度）
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS qianchuan_advertiser_daily_report (
				id BIGINT(20) NOT NULL AUTO_INCREMENT,
				advertiser_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '千川广告主ID',
				stat_datetime VARCHAR(20) NOT NULL DEFAULT '' COMMENT '统计日期',
				stat_cost DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '消耗',
				show_cnt BIGINT(20) NOT NULL DEFAULT 0 COMMENT '展示数',
				click_cnt BIGINT(20) NOT NULL DEFAULT 0 COMMENT '点击数',
				ctr DECIMAL(10,4) NOT NULL DEFAULT 0.0000 COMMENT '点击率',
				cpc_platform DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT 'CPC',
				cpm_platform DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT 'CPM',
				convert_cnt BIGINT(20) NOT NULL DEFAULT 0 COMMENT '转化数',
				convert_cost DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '转化成本',
				convert_rate DECIMAL(10,4) NOT NULL DEFAULT 0.0000 COMMENT '转化率',
				deep_convert_cnt BIGINT(20) NOT NULL DEFAULT 0 COMMENT '深度转化数',
				deep_convert_cost DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '深度转化成本',
				pay_order_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '成交金额',
				pay_order_count BIGINT(20) NOT NULL DEFAULT 0 COMMENT '成交订单数',
				pay_order_cost_per_order DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '成交订单成本',
				pay_order_coupon_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '成交优惠券金额',
				create_order_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '下单金额',
				create_order_count BIGINT(20) NOT NULL DEFAULT 0 COMMENT '下单数',
				create_order_coupon_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '下单优惠券金额',
				create_order_roi DECIMAL(10,4) NOT NULL DEFAULT 0.0000 COMMENT '下单ROI',
				prepay_order_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '预售金额',
				prepay_order_count BIGINT(20) NOT NULL DEFAULT 0 COMMENT '预售订单数',
				prepay_and_pay_order_roi DECIMAL(10,4) NOT NULL DEFAULT 0.0000 COMMENT '预售+成交ROI',
				all_order_pay_gmv_7days DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '7日支付GMV',
				all_order_pay_count_7days BIGINT(20) NOT NULL DEFAULT 0 COMMENT '7日支付数',
				all_order_pay_roi_7days DECIMAL(10,4) NOT NULL DEFAULT 0.0000 COMMENT '7日ROI',
				all_order_refund_amount_7d DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '7日退款金额',
				all_order_refund_count_7d BIGINT(20) NOT NULL DEFAULT 0 COMMENT '7日退款数',
				all_order_refund_amount_14d DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '14日退款金额',
				all_order_refund_count_14d BIGINT(20) NOT NULL DEFAULT 0 COMMENT '14日退款数',
				dy_follow BIGINT(20) NOT NULL DEFAULT 0 COMMENT '新增关注',
				total_play BIGINT(20) NOT NULL DEFAULT 0 COMMENT '播放数',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				PRIMARY KEY (id),
				UNIQUE KEY uk_adv_date (advertiser_id, stat_datetime),
				KEY idx_stat_datetime (stat_datetime)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='千川广告主日结报表'`)
			// 兼容已有表：添加 advertiser_name
			addColumnIfMissing("qianchuan_advertiser_daily_report", "advertiser_name", "advertiser_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '千川账户名称' AFTER advertiser_id")

			// 自动建表：千川计划维度日结报表（含内部组织归属）
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS qianchuan_ad_report (
				id BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
				organization_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '组织ID（默认公司ID）',
				account_id INT(11) NOT NULL DEFAULT 0 COMMENT '内部账号ID，关联account.id',
				company_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '公司ID',
				department_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '部门ID',
				trader_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '投手/运营ID',
				advertiser_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '千川账户ID（广告主ID）',
				advertiser_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '账户名称（广告主名称）',
				ad_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '计划ID',
				ad_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '计划名称',
				product_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '商品ID',
				product_name VARCHAR(512) NOT NULL DEFAULT '' COMMENT '商品名称',
				aweme_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '抖音账号ID',
				aweme_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '抖音账号名称',
				aweme_show_id VARCHAR(100) NOT NULL DEFAULT '' COMMENT '抖音号',
				stat_cost DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '消耗（元）',
				stat_datetime VARCHAR(20) NOT NULL DEFAULT '' COMMENT '统计日期',
				show_cnt BIGINT(20) NOT NULL DEFAULT 0 COMMENT '展示数',
				click_cnt BIGINT(20) NOT NULL DEFAULT 0 COMMENT '点击数',
				ctr DECIMAL(10,4) NOT NULL DEFAULT 0.0000 COMMENT '点击率',
				convert_cnt BIGINT(20) NOT NULL DEFAULT 0 COMMENT '转化数',
				convert_cost DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '转化成本',
				pay_order_count BIGINT(20) NOT NULL DEFAULT 0 COMMENT '成交订单数',
				create_order_count BIGINT(20) NOT NULL DEFAULT 0 COMMENT '下单数',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
				PRIMARY KEY (id),
				UNIQUE KEY uk_ad_date (ad_id, stat_datetime),
				KEY idx_organization_id (organization_id),
				KEY idx_account_id (account_id),
				KEY idx_company_id (company_id),
				KEY idx_department_id (department_id),
				KEY idx_trader_id (trader_id),
				KEY idx_advertiser_id (advertiser_id),
				KEY idx_ad_id (ad_id),
				KEY idx_product_id (product_id),
				KEY idx_aweme_id (aweme_id),
				KEY idx_stat_datetime (stat_datetime)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='巨量千川广告计划日结报表'`)

			// 自动建表：千川达人维度日报表
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS qianchuan_author_daily_report (
				id BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
				advertiser_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '千川广告主ID',
				account_id INT NOT NULL DEFAULT 0 COMMENT '内部账号ID（关联account.id）',
				company_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '公司ID',
				department_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '部门ID',
				trader_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '投手ID',
				aweme_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '抖音账号ID（达人UID）',
				aweme_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '抖音账号名称',
				stat_cost DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '消耗（元）',
				pay_order_gmv DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '成交金额',
				pay_order_count BIGINT(20) NOT NULL DEFAULT 0 COMMENT '成交订单数',
				stat_datetime VARCHAR(20) NOT NULL DEFAULT '' COMMENT '统计日期',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
				PRIMARY KEY (id),
				UNIQUE KEY uk_adv_aweme_date (advertiser_id, aweme_id, stat_datetime),
				KEY idx_aweme_id (aweme_id),
				KEY idx_stat_datetime (stat_datetime),
				KEY idx_account_id (account_id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='千川达人维度日报表'`)

			// 自动建表：千川乘方日报表（账户维度）
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS qianchuan_overall_promotion_daily_report (
				id BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
				account_id INT(11) NOT NULL DEFAULT 0 COMMENT '内部账号ID，关联 account.id',
				company_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '公司ID',
				department_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '部门ID',
				trader_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '投手ID',
				advertiser_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '千川广告主ID',
				stat_datetime VARCHAR(20) NOT NULL DEFAULT '' COMMENT '统计日期',
				adlab_scene VARCHAR(32) NOT NULL DEFAULT 'OVERALL_PROJECT' COMMENT '计划场景',
				data_period VARCHAR(32) NOT NULL DEFAULT 'OVER_ALL_DATA' COMMENT '数据周期',
				marketing_goal VARCHAR(32) NOT NULL DEFAULT 'ALL' COMMENT '营销目标',
				order_platform VARCHAR(32) NOT NULL DEFAULT 'QIANCHUAN' COMMENT '下单平台',
				shop_estimated_comission_cost DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '预估达人佣金成本',
				stat_cost_for_overall_roi2 DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '综合成本',
				stat_cost_for_roi2 DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '整体消耗',
				total_cost_per_pay_order_for_roi2 DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '整体成交订单成本',
				total_cost_per_pay_order_settle_for_overall_roi2_1h DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '综合订单成本',
				total_cost_per_pay_order_settle_for_roi2_1h DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '净成交订单成本',
				total_ecom_platform_subsidy_amount_for_roi2 DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '平台补贴金额',
				total_order_real_settle_amount_for_roi2_1h DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '实际结算金额',
				total_order_settle_amount_for_roi2_1h DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '净成交金额',
				total_order_settle_amount_rate_for_roi2_1h DECIMAL(12,4) NOT NULL DEFAULT 0.0000 COMMENT '净成交金额占比',
				total_order_settle_count_for_roi2_1h BIGINT(20) NOT NULL DEFAULT 0 COMMENT '净成交订单数',
				total_order_settle_count_rate_for_roi2_1h DECIMAL(12,4) NOT NULL DEFAULT 0.0000 COMMENT '净成交订单数占比',
				total_pay_order_count_for_roi2 BIGINT(20) NOT NULL DEFAULT 0 COMMENT '整体成交订单数',
				total_pay_order_coupon_amount_for_roi2 DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '整体成交优惠券金额',
				total_pay_order_gmv_for_roi2 DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '用户实际支付金额',
				total_pay_order_gmv_include_coupon_for_roi2 DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '整体成交金额',
				total_prepay_and_pay_order_roi2 DECIMAL(12,4) NOT NULL DEFAULT 0.0000 COMMENT '整体支付ROI',
				total_prepay_and_pay_settle_overall_roi2_1h DECIMAL(12,4) NOT NULL DEFAULT 0.0000 COMMENT '综合ROI',
				total_prepay_and_pay_settle_roi2_1h DECIMAL(12,4) NOT NULL DEFAULT 0.0000 COMMENT '净成交ROI',
				total_prepay_order_count_for_roi2 BIGINT(20) NOT NULL DEFAULT 0 COMMENT '整体预售订单数',
				total_prepay_order_gmv_for_roi2 DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '整体预售订单金额',
				total_refund_order_count_for_roi2_1h BIGINT(20) NOT NULL DEFAULT 0 COMMENT '退款订单数',
				total_refund_order_gmv_for_roi2_1h_all DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '退款订单金额',
				total_refund_order_gmv_for_roi2_1h_rate DECIMAL(12,4) NOT NULL DEFAULT 0.0000 COMMENT '退款订单金额占比',
				total_unfinished_estimate_order_gmv_for_roi2 DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '未完结预售订单预估金额',
				no_refund_ecom_coupon_amount_for_roi2 DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '不含退款优惠券金额',
				no_refund_ecom_platform_subsidy_amount_for_roi2 DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '不含退款平台补贴金额',
				raw_payload LONGTEXT COMMENT '原始响应快照',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
				PRIMARY KEY (id),
				UNIQUE KEY uk_adv_date_dims (advertiser_id, stat_datetime, adlab_scene, data_period, marketing_goal, order_platform),
				KEY idx_stat_datetime (stat_datetime),
				KEY idx_account_id (account_id),
				KEY idx_company_id (company_id),
				KEY idx_department_id (department_id),
				KEY idx_trader_id (trader_id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='千川乘方日报表'`)

			// 千川账户临时归属规则：账户资产归属不变，仅按生效日期调整统计负责人。
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS qianchuan_advertiser_attribution_rule (
				id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
				advertiser_id BIGINT(20) NOT NULL COMMENT '千川广告主ID',
				advertiser_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '千川账户名称快照',
				source_account_names VARCHAR(500) NOT NULL DEFAULT '' COMMENT '命中后改归属的抖音账号名称，逗号分隔；空表示广告主级兜底',
				source_trader_id BIGINT UNSIGNED NOT NULL COMMENT '账户原负责人ID',
				target_trader_id BIGINT UNSIGNED NOT NULL COMMENT '临时统计负责人ID',
				effective_start_date DATE NOT NULL COMMENT '生效开始日期（含）',
				effective_end_date DATE DEFAULT NULL COMMENT '生效结束日期（含），空表示持续有效',
				reason VARCHAR(500) NOT NULL DEFAULT '' COMMENT '借号原因',
				status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0=停用 1=启用',
				created_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
				updated_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
				PRIMARY KEY (id),
				UNIQUE KEY uk_advertiser_start (advertiser_id, effective_start_date),
				KEY idx_effective_rule (advertiser_id, status, effective_start_date, effective_end_date),
				KEY idx_target_trader (target_trader_id, status)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='千川账户临时统计归属规则'`)
			addColumnIfMissing("qianchuan_advertiser_attribution_rule", "source_account_names", "source_account_names VARCHAR(500) NOT NULL DEFAULT '' COMMENT '命中后改归属的抖音账号名称，逗号分隔；空表示广告主级兜底' AFTER `advertiser_name`")

			g.DB().Exec(ctx, `UPDATE qianchuan_advertiser_attribution_rule
				SET source_account_names = '天书说故事,利云底层逻辑'
				WHERE advertiser_id IN (1700642741413895, 1774391159738372, 1711882704612356, 1722203379291139)
					AND reason = '王梅雪临时借用李百川千川账户'`)
			g.DB().Exec(ctx, `UPDATE qianchuan_advertiser_attribution_rule
				SET effective_start_date = '2026-07-01'
				WHERE advertiser_id IN (1700642741413895, 1774391159738372, 1711882704612356, 1722203379291139)
					AND effective_start_date = '2026-07-13'
					AND reason = '王梅雪临时借用李百川千川账户'`)
			g.DB().Exec(ctx, `UPDATE qianchuan_advertiser_attribution_rule
				SET effective_end_date = NULL,
					status = 1
				WHERE advertiser_id IN (1700642741413895, 1774391159738372, 1711882704612356, 1722203379291139)
					AND reason = '王梅雪临时借用李百川千川账户'`)

			// 一次性初始化当前已确认的借号关系；后续只维护规则表。
			g.DB().Exec(ctx, `INSERT IGNORE INTO qianchuan_advertiser_attribution_rule
				(advertiser_id, advertiser_name, source_account_names, source_trader_id, target_trader_id,
				effective_start_date, effective_end_date, reason, status)
			SELECT seed.advertiser_id, seed.advertiser_name, '天书说故事,利云底层逻辑', source.id, target.id,
				'2026-07-01', NULL, '王梅雪临时借用李百川千川账户', 1
			FROM (
				SELECT 1700642741413895 AS advertiser_id, '天书万卷-官方号投放户' AS advertiser_name
				UNION ALL SELECT 1774391159738372, '利云图书企业店—店铺组'
				UNION ALL SELECT 1711882704612356, '利云文化进口图书-主账户'
				UNION ALL SELECT 1722203379291139, '闻道清北图书旗舰店主户'
			) seed
			JOIN trader source ON source.id = (
				SELECT id FROM trader
				WHERE name IN ('李百川', '李百川组') AND status = 1
				ORDER BY (name = '李百川') DESC, id ASC LIMIT 1
			)
			JOIN trader target ON target.id = (
				SELECT id FROM trader
				WHERE name IN ('王梅雪', '王梅雪组') AND status = 1
				ORDER BY (name = '王梅雪') DESC, id ASC LIMIT 1
			)
			WHERE NOT EXISTS (
				SELECT 1 FROM qianchuan_advertiser_attribution_rule existing
				WHERE existing.advertiser_id = seed.advertiser_id
			)`)
			seedCount, seedErr := g.DB().GetOne(ctx, `SELECT COUNT(DISTINCT advertiser_id) AS cnt
				FROM qianchuan_advertiser_attribution_rule
				WHERE advertiser_id IN (1700642741413895, 1774391159738372, 1711882704612356, 1722203379291139)`)
			if seedErr != nil || seedCount["cnt"].Int() != 4 {
				g.Log().Warningf(ctx, "[DBInit] 千川临时归属初始化未完成，需确认李百川（组）、王梅雪（组）均为启用人员: count=%d err=%v", seedCount["cnt"].Int(), seedErr)
			}
			g.DB().Exec(ctx, `UPDATE qianchuan_advertiser_attribution_rule
				SET advertiser_name = '利云文化图书旗舰店-一组IP L',
					source_account_names = '雷妈说教育,雷老师讲学业规划,汤圆老师说教育.,汤圆老师说教育.有因有果,汤圆老师讲教育有因有果,张可乐校长说学习方法,汤圆教育说-有因有果,汤圆老师数学规划圈,可乐老师来帮忙',
					effective_start_date = '2026-06-11',
					effective_end_date = NULL,
					status = 1
				WHERE advertiser_id = 1805996971742284
					AND reason = '张文蔚/孟令饶组抖音号在李百川千川户消耗归属李百川'`)
			g.DB().Exec(ctx, `INSERT IGNORE INTO qianchuan_advertiser_attribution_rule
				(advertiser_id, advertiser_name, source_account_names, source_trader_id, target_trader_id,
				effective_start_date, effective_end_date, reason, status)
			SELECT 1805996971742284, '利云文化图书旗舰店-一组IP L',
				'雷妈说教育,雷老师讲学业规划,汤圆老师说教育.,汤圆老师说教育.有因有果,汤圆老师讲教育有因有果,张可乐校长说学习方法,汤圆教育说-有因有果,汤圆老师数学规划圈,可乐老师来帮忙',
				source.id, target.id, '2026-06-11', NULL,
				'张文蔚/孟令饶组抖音号在李百川千川户消耗归属李百川', 1
			FROM DUAL
			JOIN trader source ON source.id = (
				SELECT id FROM trader
				WHERE name IN ('张文蔚/孟令饶组', '张文蔚/孟令饶') AND status = 1
				ORDER BY (name = '张文蔚/孟令饶组') DESC, id ASC LIMIT 1
			)
			JOIN trader target ON target.id = (
				SELECT id FROM trader
				WHERE name IN ('李百川', '李百川组') AND status = 1
				ORDER BY (name = '李百川组') DESC, id ASC LIMIT 1
			)
			WHERE NOT EXISTS (
				SELECT 1 FROM qianchuan_advertiser_attribution_rule existing
				WHERE existing.advertiser_id = 1805996971742284
			)`)
			liBaichuanSeedCount, liBaichuanSeedErr := g.DB().GetOne(ctx, `SELECT COUNT(1) AS cnt
				FROM qianchuan_advertiser_attribution_rule
				WHERE advertiser_id = 1805996971742284
					AND target_trader_id = (
						SELECT id FROM trader
						WHERE name IN ('李百川', '李百川组') AND status = 1
						ORDER BY (name = '李百川组') DESC, id ASC LIMIT 1
					)`)
			if liBaichuanSeedErr != nil || liBaichuanSeedCount["cnt"].Int() != 1 {
				g.Log().Warningf(ctx, "[DBInit] 李百川千川借号归属初始化未完成，需确认李百川组、张文蔚/孟令饶组均为启用人员: count=%d err=%v", liBaichuanSeedCount["cnt"].Int(), liBaichuanSeedErr)
			}

			addColumnIfMissing("qianchuan_ad_report", "organization_id", "organization_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '组织ID（默认公司ID）' AFTER `id`")
			addColumnIfMissing("qianchuan_ad_report", "account_id", "account_id INT(11) NOT NULL DEFAULT 0 COMMENT '内部账号ID，关联account.id' AFTER `organization_id`")
			addColumnIfMissing("qianchuan_ad_report", "company_id", "company_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '公司ID' AFTER `account_id`")
			addColumnIfMissing("qianchuan_ad_report", "department_id", "department_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '部门ID' AFTER `company_id`")
			addColumnIfMissing("qianchuan_ad_report", "trader_id", "trader_id BIGINT(20) NOT NULL DEFAULT 0 COMMENT '投手/运营ID' AFTER `department_id`")
			addColumnIfMissing("qianchuan_ad_report", "pay_order_count", "pay_order_count BIGINT(20) NOT NULL DEFAULT 0 COMMENT '成交订单数' AFTER `convert_cost`")
			addColumnIfMissing("qianchuan_ad_report", "create_order_count", "create_order_count BIGINT(20) NOT NULL DEFAULT 0 COMMENT '下单数' AFTER `pay_order_count`")
			addIndexIfMissing("qianchuan_ad_report", "idx_product_stat_advertiser_ad", "KEY `idx_product_stat_advertiser_ad` (`product_id`, `stat_datetime`, `advertiser_id`, `ad_id`)")
			addIndexIfMissing("order_info", "idx_order_submit_time", "KEY `idx_order_submit_time` (`order_submit_time`)")
			addIndexIfMissing("order_info", "idx_product_submit_shop_order", "KEY `idx_product_submit_shop_order` (`product_id`, `order_submit_time`, `shop_name`, `main_order_no`)")
			addIndexIfMissing("order_info", "idx_store_product_pay_time", "KEY `idx_store_product_pay_time` (`store_id`, `product_id`, `pay_finish_time`)")

			go func() {
				syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := oceanengine.SyncQianchuanTemporaryAttribution(syncCtx); err != nil {
					g.Log().Warningf(ctx, "[DBInit] 同步千川临时归属失败: %v", err)
				}
			}()

			// account 表：status 字段，用于支持软删除
			addColumnIfMissing("account", "status", "status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0=已删除 1=正常'")
			addColumnIfMissing("account", "created_at", "created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间'")
			addColumnIfMissing("account", "updated_at", "updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'")
			addColumnIfMissing("account", "bind_status", "bind_status TINYINT NOT NULL DEFAULT 0 COMMENT '绑定状态：0=未绑定，1=被绑定'")
			addColumnIfMissing("account", "blank_order_enabled", "blank_order_enabled TINYINT NOT NULL DEFAULT 0 COMMENT '是否参与空白单归属：0=否 1=是' AFTER `bind_status`")
			addColumnIfMissing("account", "blank_order_owner_name", "blank_order_owner_name VARCHAR(100) NOT NULL DEFAULT '' COMMENT '空白单归属人快照' AFTER `blank_order_enabled`")
			addColumnIfMissing("account", "blank_order_owner_trader_id", "blank_order_owner_trader_id BIGINT NOT NULL DEFAULT 0 COMMENT '空白单归属投手ID' AFTER `blank_order_owner_name`")
			modifyColumnIfNeeded("account", "uid", "`uid` VARCHAR(100) NULL DEFAULT NULL COMMENT '达人ID（抖音UID/快手号/视频号ID）'")
			if _, err := g.DB().Exec(ctx, "UPDATE `account` SET `uid` = NULL WHERE `uid` IS NOT NULL AND TRIM(`uid`) = ''"); err != nil {
				g.Log().Warningf(ctx, "[DBInit] 清理空账号UID失败: %v", err)
			}
			if _, err := g.DB().Exec(ctx, `
					INSERT INTO account (
						platform, uid, account_name, platform_nickname, operator,
						blank_order_enabled, blank_order_owner_name, blank_order_owner_trader_id, status, created_at, updated_at
					) VALUES (
						'douyin', '2986722072732416', '欢喜图书专营店', '70228145728', '张文蔚/孟令饶组',
						1, '张文蔚/孟令饶组', COALESCE((SELECT t.id FROM trader t WHERE t.status = 1 AND t.name = '张文蔚/孟令饶组' ORDER BY t.id ASC LIMIT 1), 0), 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
					)
					ON DUPLICATE KEY UPDATE
						uid = VALUES(uid),
						account_name = VALUES(account_name),
						platform_nickname = VALUES(platform_nickname),
						operator = VALUES(operator),
						blank_order_enabled = VALUES(blank_order_enabled),
						blank_order_owner_name = VALUES(blank_order_owner_name),
						blank_order_owner_trader_id = VALUES(blank_order_owner_trader_id),
						status = VALUES(status),
						updated_at = VALUES(updated_at)`); err != nil {
				g.Log().Warningf(ctx, "[DBInit] 初始化欢喜图书专营店账号失败: %v", err)
			}
			if _, err := g.DB().Exec(ctx, `
						UPDATE account
						SET blank_order_enabled = 1,
					    blank_order_owner_name = CASE
							WHEN account_name IN ('天书说故事', '利云底层逻辑') THEN '王梅雪组'
							ELSE COALESCE(NULLIF(TRIM(operator), ''), blank_order_owner_name)
						END
					WHERE account_name IN (
						'闻道清北图书旗舰店',
						'利云学习文化旗舰店',
						'利云文化官方旗舰店',
						'金抖云图书专营店',
						'利云图书专营店',
						'利云文化千玺进口图书旗舰店',
						'欢喜图书专营店',
						'利云成长课堂',
						'天书说故事',
						'利云底层逻辑',
						'利云文化文创旗舰店',
						'利云文化早教益智旗舰店'
					  )`); err != nil {
				g.Log().Warningf(ctx, "[DBInit] 初始化账号空白单归属标记失败: %v", err)
			}
			if _, err := g.DB().Exec(ctx, `
				UPDATE account a
				SET blank_order_owner_trader_id = COALESCE((
					SELECT t.id
					FROM trader t
					WHERE t.status = 1
					  AND (
						t.name COLLATE utf8mb4_unicode_ci = a.blank_order_owner_name COLLATE utf8mb4_unicode_ci
						OR t.name COLLATE utf8mb4_unicode_ci = CONCAT(a.blank_order_owner_name, '组') COLLATE utf8mb4_unicode_ci
						OR (a.blank_order_owner_name LIKE '%组' AND t.name COLLATE utf8mb4_unicode_ci = TRIM(TRAILING '组' FROM a.blank_order_owner_name) COLLATE utf8mb4_unicode_ci)
					  )
					ORDER BY (t.name COLLATE utf8mb4_unicode_ci = a.blank_order_owner_name COLLATE utf8mb4_unicode_ci) DESC, t.id ASC
					LIMIT 1
				), 0)
				WHERE a.blank_order_enabled = 1
				  AND a.blank_order_owner_trader_id = 0
				  AND TRIM(a.blank_order_owner_name) != ''`); err != nil {
				g.Log().Warningf(ctx, "[DBInit] 回填账号空白单归属投手ID失败: %v", err)
			}
			dropIndexIfExists("account", "uk_uid")

			// account 表：唯一索引，防止重复导入
			addIndexIfMissing("account", "uk_platform_account_name", "UNIQUE KEY `uk_platform_account_name` (`platform`, `account_name`) COMMENT '平台+账号名称唯一'")
			addIndexIfMissing("account", "idx_bind_status", "KEY `idx_bind_status` (`bind_status`)")
			addIndexIfMissing("account", "idx_blank_order_enabled", "KEY `idx_blank_order_enabled` (`blank_order_enabled`)")
			addIndexIfMissing("account", "idx_blank_order_owner_trader_id", "KEY `idx_blank_order_owner_trader_id` (`blank_order_owner_trader_id`)")
			addIndexIfMissing("account", "idx_platform_uid", "KEY `idx_platform_uid` (`platform`, `uid`)")
			addIndexIfMissing("account", "idx_platform_nickname", "KEY `idx_platform_nickname` (`platform`, `platform_nickname`)")
			addIndexIfMissing("alliance_orders", "idx_platform_short_id", "KEY `idx_platform_short_id` (`platform`, `short_id`)")

			// 自动建表：系统消息通知
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS system_notification (
				id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
				title VARCHAR(200) NOT NULL DEFAULT '' COMMENT '消息标题',
				content TEXT COMMENT '消息内容JSON',
				type VARCHAR(30) NOT NULL DEFAULT 'system' COMMENT '消息类型',
				source_type VARCHAR(50) NOT NULL DEFAULT '' COMMENT '来源类型',
				source_id BIGINT NOT NULL DEFAULT 0 COMMENT '来源记录ID',
				level TINYINT NOT NULL DEFAULT 0 COMMENT '消息级别: 0普通 1重要 2紧急',
				is_read TINYINT NOT NULL DEFAULT 0 COMMENT '是否已读',
				read_at DATETIME DEFAULT NULL COMMENT '阅读时间',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
				PRIMARY KEY (id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
			// 自动建表：定时任务管理
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS cron_task (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			name VARCHAR(100) NOT NULL DEFAULT '',
			cron_expr VARCHAR(50) NOT NULL DEFAULT '',
			api_url VARCHAR(300) NOT NULL DEFAULT '',
			api_method VARCHAR(10) NOT NULL DEFAULT 'GET',
			api_params TEXT,
			status TINYINT NOT NULL DEFAULT 1,
			retry_times INT NOT NULL DEFAULT 0,
			retry_interval INT NOT NULL DEFAULT 0,
			next_task VARCHAR(100) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uk_name (name)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)

			// 店铺借用账号归属规则表
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS shop_account_attribution (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			shop_id VARCHAR(100) NOT NULL DEFAULT '' COMMENT '店铺ID（store.store_id，优先用于稳定匹配）',
			shop_name VARCHAR(200) NOT NULL COMMENT '店铺名称',
			source_account_uid VARCHAR(100) NOT NULL COMMENT '借用的达人UID',
			source_account_name VARCHAR(200) DEFAULT '' COMMENT '借用的达人名称',
			target_type VARCHAR(20) NOT NULL DEFAULT 'operator' COMMENT '归属类型：operator=运营组, trader=投手',
			target_value VARCHAR(200) NOT NULL COMMENT '归属目标：运营组名称或投手ID',
			remark VARCHAR(500) DEFAULT '' COMMENT '备注',
			status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0=禁用 1=启用',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uk_shop_account (shop_name, source_account_uid),
			INDEX idx_shop_id (shop_id),
			INDEX idx_shop_name (shop_name),
			INDEX idx_account_uid (source_account_uid),
			INDEX idx_status (status)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='店铺借用账号归属规则表'`)
			addColumnIfMissing("shop_account_attribution", "shop_id", "shop_id VARCHAR(100) NOT NULL DEFAULT '' COMMENT '店铺ID（store.store_id，优先用于稳定匹配）' AFTER id")
			if _, err := g.DB().Exec(ctx, "ALTER TABLE `shop_account_attribution` MODIFY COLUMN `shop_id` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '店铺ID（store.store_id，优先用于稳定匹配）'"); err != nil {
				g.Log().Warningf(ctx, "[DBInit] 修改店铺借号归属店铺ID字段失败 err=%v", err)
			}
			if _, err := g.DB().Exec(ctx, `
				UPDATE shop_account_attribution saa
				INNER JOIN store s
					ON s.platform COLLATE utf8mb4_unicode_ci = 'douyin'
					AND CAST(s.id AS CHAR) COLLATE utf8mb4_unicode_ci = saa.shop_id COLLATE utf8mb4_unicode_ci
					AND s.store_id != ''
				SET saa.shop_id = CONVERT(s.store_id USING utf8mb4) COLLATE utf8mb4_unicode_ci
				WHERE saa.shop_id != ''
				  AND saa.shop_id != '0'
				  AND saa.shop_id COLLATE utf8mb4_unicode_ci != CONVERT(s.store_id USING utf8mb4) COLLATE utf8mb4_unicode_ci
			`); err != nil {
				g.Log().Warningf(ctx, "[DBInit] 同步店铺借号归属平台店铺ID失败 err=%v", err)
			}
			if _, err := g.DB().Exec(ctx, `
				UPDATE shop_account_attribution saa
				INNER JOIN (
					SELECT s.store_name, s.store_id
					FROM store s
					INNER JOIN (
						SELECT MIN(id) AS id
						FROM store
						WHERE platform COLLATE utf8mb4_unicode_ci = 'douyin'
						  AND status = 1
						  AND store_id != ''
						GROUP BY store_name
					) picked ON picked.id = s.id
				) matched_store
					ON matched_store.store_name COLLATE utf8mb4_unicode_ci = saa.shop_name COLLATE utf8mb4_unicode_ci
				SET saa.shop_id = CONVERT(matched_store.store_id USING utf8mb4) COLLATE utf8mb4_unicode_ci
				WHERE COALESCE(NULLIF(NULLIF(TRIM(saa.shop_id), ''), '0'), '') = ''
			`); err != nil {
				g.Log().Warningf(ctx, "[DBInit] 按店铺名称补齐借号归属平台店铺ID失败 err=%v", err)
			}
			if _, err := g.DB().Exec(ctx, "UPDATE shop_account_attribution SET shop_id = '' WHERE shop_id = '0'"); err != nil {
				g.Log().Warningf(ctx, "[DBInit] 清理店铺借号归属空店铺ID失败 err=%v", err)
			}
			if _, err := g.DB().Exec(ctx, `
				INSERT INTO shop_account_attribution (
					shop_id, shop_name, source_account_uid, source_account_name,
					target_type, target_value, remark, status
				)
				SELECT '33415411', '闻道清北图书旗舰店', '__blank__', '空白单',
					'operator', '王梅雪组',
					'空白单借号归属：闻道清北图书旗舰店空白单归属王梅雪组', 1
				WHERE NOT EXISTS (
					SELECT 1
					FROM shop_account_attribution
					WHERE shop_name = '闻道清北图书旗舰店'
					  AND source_account_uid = '__blank__'
				)
			`); err != nil {
				g.Log().Warningf(ctx, "[DBInit] 初始化闻道清北空白单借号归属失败 err=%v", err)
			}
			if _, err := g.DB().Exec(ctx, `
				UPDATE shop_account_attribution
				SET remark = CONCAT(
					'借号订单归属：',
					COALESCE(NULLIF(source_account_name, ''), source_account_uid),
					'（UID ',
					source_account_uid,
					'）在“',
					shop_name,
					'”出单归属',
					target_value
				)
				WHERE TRIM(COALESCE(remark, '')) = ''
				   OR remark LIKE '规则%'
			`); err != nil {
				g.Log().Warningf(ctx, "[DBInit] 补齐借号归属规则备注失败 err=%v", err)
			}
			addIndexIfMissing("shop_account_attribution", "idx_shop_id", "INDEX `idx_shop_id` (`shop_id`)")

			// 抖音小店自营订单的千川归属结果（用于审计和复核）
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS douyin_order_attribution (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			order_info_id BIGINT NOT NULL COMMENT '抖音订单明细ID',
			main_order_no VARCHAR(100) NOT NULL DEFAULT '' COMMENT '主订单号',
			sub_order_no VARCHAR(100) NOT NULL DEFAULT '' COMMENT '子订单号',
			stat_date DATE NOT NULL COMMENT '归属日期',
			shop_name VARCHAR(200) NOT NULL DEFAULT '' COMMENT '店铺名称',
			product_id VARCHAR(100) NOT NULL DEFAULT '' COMMENT '商品ID',
			source VARCHAR(30) NOT NULL DEFAULT 'qianchuan_plan' COMMENT '归属来源',
			allocated_store_name VARCHAR(200) NOT NULL DEFAULT '' COMMENT '千川分配命中的投流店铺',
			blank_order_owner_name VARCHAR(100) NOT NULL DEFAULT '' COMMENT '空白单最终归属人快照',
			account_id BIGINT NOT NULL DEFAULT 0 COMMENT '归属达人账号ID',
			trader_id BIGINT NOT NULL DEFAULT 0 COMMENT '归属投手ID快照',
			talent_uid VARCHAR(100) NOT NULL DEFAULT '' COMMENT '归属达人UID',
			qianchuan_plan_id BIGINT NOT NULL DEFAULT 0 COMMENT '千川计划ID，0表示多计划合并',
			qianchuan_plan_count INT NOT NULL DEFAULT 0 COMMENT '参与归属的计划数',
			qianchuan_stat_cost DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '归属范围内该达人消耗',
			allocation_ratio DECIMAL(12,8) NOT NULL DEFAULT 0.00000000 COMMENT '消耗权重',
			allocation_basis VARCHAR(30) NOT NULL DEFAULT 'stat_cost' COMMENT '分配依据',
			rule_version VARCHAR(40) NOT NULL DEFAULT 'qianchuan_cost_quota_v1' COMMENT '规则版本',
			calculated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '最后计算时间',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uk_order_source (order_info_id, source),
			INDEX idx_stat_shop_product (stat_date, shop_name, product_id),
			INDEX idx_source_account_date (source, account_id, stat_date),
			INDEX idx_account_date (account_id, stat_date),
			INDEX idx_trader_date (trader_id, stat_date),
			INDEX idx_main_order_no (main_order_no)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='抖音小店自营订单千川归属结果'`)
			addColumnIfMissing("douyin_order_attribution", "allocated_store_name", "allocated_store_name VARCHAR(200) NOT NULL DEFAULT '' COMMENT '千川分配命中的投流店铺' AFTER `shop_name`")
			addColumnIfMissing("douyin_order_attribution", "blank_order_owner_name", "blank_order_owner_name VARCHAR(100) NOT NULL DEFAULT '' COMMENT '空白单最终归属人快照' AFTER `allocated_store_name`")
			addIndexIfMissing("douyin_order_attribution", "idx_source_account_date", "KEY `idx_source_account_date` (`source`, `account_id`, `stat_date`)")

			if _, err := g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS blank_order_allocation_exception (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			order_info_id BIGINT NOT NULL DEFAULT 0 COMMENT '抖音订单明细ID',
			main_order_no VARCHAR(100) NOT NULL DEFAULT '' COMMENT '主订单号',
			sub_order_no VARCHAR(100) NOT NULL DEFAULT '' COMMENT '子订单号',
			stat_date DATE DEFAULT NULL COMMENT '归属日期',
			shop_name VARCHAR(200) NOT NULL DEFAULT '' COMMENT '空白单来源店铺',
			product_id VARCHAR(100) NOT NULL DEFAULT '' COMMENT '商品ID',
			exception_type VARCHAR(50) NOT NULL DEFAULT '' COMMENT '异常类型',
			exception_reason VARCHAR(500) NOT NULL DEFAULT '' COMMENT '异常原因',
			qianchuan_plan_snapshot JSON DEFAULT NULL COMMENT '千川计划候选快照',
			shop_owner_snapshot JSON DEFAULT NULL COMMENT '店铺归属快照',
			status VARCHAR(30) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/resolved_auto/resolved_manual/ignored',
			resolved_owner VARCHAR(100) NOT NULL DEFAULT '' COMMENT '人工指定归属人',
			resolved_by VARCHAR(100) NOT NULL DEFAULT '' COMMENT '处理人',
			resolved_at DATETIME DEFAULT NULL COMMENT '处理时间',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uk_order_source (order_info_id, exception_type),
			KEY idx_status_date (status, stat_date),
			KEY idx_shop_product_date (shop_name, product_id, stat_date),
			KEY idx_main_order_no (main_order_no)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='空白单千川归属异常池'`); err != nil {
				g.Log().Warningf(ctx, "[DBInit] 创建空白单千川归属异常池失败: %v", err)
			}

			// 抖店售后/退款订单独立存储表
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS douyin_refund_order (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			platform VARCHAR(30) NOT NULL DEFAULT 'douyin' COMMENT '平台',
			shop_id BIGINT NOT NULL DEFAULT 0 COMMENT '抖店店铺ID',
			shop_name VARCHAR(200) NOT NULL DEFAULT '' COMMENT '店铺名称',
			shop_order_id VARCHAR(100) NOT NULL DEFAULT '' COMMENT '主订单号',
			sku_order_id VARCHAR(100) NOT NULL DEFAULT '' COMMENT '子订单号/商品单号',
			aftersale_id VARCHAR(100) NOT NULL DEFAULT '' COMMENT '售后单号',
			aftersale_order_type INT NOT NULL DEFAULT 0 COMMENT '售后订单类型',
			aftersale_type INT NOT NULL DEFAULT 0 COMMENT '售后类型',
			aftersale_type_text VARCHAR(100) NOT NULL DEFAULT '' COMMENT '售后类型文案',
			aftersale_status INT NOT NULL DEFAULT 0 COMMENT '售后状态',
			aftersale_status_text VARCHAR(100) NOT NULL DEFAULT '' COMMENT '售后状态文案',
			aftersale_refund_type INT NOT NULL DEFAULT 0 COMMENT '售后退款类型',
			aftersale_refund_type_text VARCHAR(100) NOT NULL DEFAULT '' COMMENT '售后退款类型文案',
			refund_type INT NOT NULL DEFAULT 0 COMMENT '退款方式',
			refund_status INT NOT NULL DEFAULT 0 COMMENT '退款状态',
			reason_text VARCHAR(255) NOT NULL DEFAULT '' COMMENT '售后原因文案',
			refund_amount DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '退款金额',
			refund_post_amount DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '退运费金额',
			refund_tax_amount DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '退税费金额',
			return_promotion_amount DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '退优惠金额',
			aftersale_num INT NOT NULL DEFAULT 0 COMMENT '售后数量',
			part_type INT NOT NULL DEFAULT 0 COMMENT '部分退类型',
			apply_time DATETIME DEFAULT NULL COMMENT '申请时间',
			update_time DATETIME DEFAULT NULL COMMENT '售后更新时间',
			aftersale_create_time DATETIME DEFAULT NULL COMMENT '售后单创建时间',
			status_deadline DATETIME DEFAULT NULL COMMENT '当前节点逾期时间',
			aftersale_final_time DATETIME DEFAULT NULL COMMENT '售后完结时间',
			product_id VARCHAR(100) NOT NULL DEFAULT '' COMMENT '商品ID',
			product_name VARCHAR(500) NOT NULL DEFAULT '' COMMENT '商品名称',
			product_image VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '商品图片',
			product_spec VARCHAR(500) NOT NULL DEFAULT '' COMMENT '商品规格',
			shop_sku_code VARCHAR(100) NOT NULL DEFAULT '' COMMENT '商家SKU编码',
			product_num INT NOT NULL DEFAULT 0 COMMENT '售后商品数量',
			product_price DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '商品单价',
			pay_amount DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '原订单支付金额',
			post_amount DECIMAL(14,2) NOT NULL DEFAULT 0.00 COMMENT '原订单运费',
			order_status INT NOT NULL DEFAULT 0 COMMENT '原订单状态',
			order_create_time DATETIME DEFAULT NULL COMMENT '原订单下单时间',
			return_logistics_code VARCHAR(100) NOT NULL DEFAULT '' COMMENT '退货物流单号',
			return_logistics_company_name VARCHAR(200) NOT NULL DEFAULT '' COMMENT '退货物流公司',
			order_flag INT NOT NULL DEFAULT 0 COMMENT '订单插旗',
			risk_decision_code INT NOT NULL DEFAULT 0 COMMENT '风控码',
			risk_decision_reason VARCHAR(255) NOT NULL DEFAULT '' COMMENT '风控理由',
			risk_decision_description VARCHAR(500) NOT NULL DEFAULT '' COMMENT '风控描述',
			arbitrate_status INT NOT NULL DEFAULT 0 COMMENT '仲裁状态',
			remark VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '售后商家备注',
			raw_json JSON DEFAULT NULL COMMENT '原始售后数据',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uk_aftersale_sku (aftersale_id, sku_order_id),
			INDEX idx_shop_update_time (shop_id, update_time),
			INDEX idx_shop_apply_time (shop_id, apply_time),
			INDEX idx_shop_order_id (shop_order_id),
			INDEX idx_sku_order_id (sku_order_id),
			INDEX idx_aftersale_status (aftersale_status),
			INDEX idx_refund_status (refund_status)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='抖店售后退款订单表'`)

			// 外部达人台账表
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS external_talent_registry (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			team_name VARCHAR(100) NOT NULL DEFAULT '' COMMENT '团队名称',
			platform_type VARCHAR(30) NOT NULL DEFAULT '' COMMENT '平台类型：douyin/kuaishou/shipinhao/xiaohongshu/wechat 等',
			shop_name VARCHAR(200) NOT NULL DEFAULT '' COMMENT '店铺名称',
			platform_shop_label VARCHAR(255) NOT NULL DEFAULT '' COMMENT '原始平台列文本，如 抖音-利云文化官方旗舰店',
			uid VARCHAR(100) DEFAULT NULL COMMENT '达人UID',
			account_name VARCHAR(200) DEFAULT '' COMMENT '账号名称原值',
			show_id VARCHAR(200) DEFAULT '' COMMENT '抖音号/展示号原值',
			link_code VARCHAR(255) DEFAULT '' COMMENT '链接/渠道码原值',
			talent_name VARCHAR(200) DEFAULT '' COMMENT '标准达人名称，导入时可取账号名称或抖音号',
			talent_type VARCHAR(20) NOT NULL DEFAULT 'external' COMMENT '达人类型：external',
			product_id VARCHAR(100) DEFAULT '' COMMENT '商品ID',
			goods_code VARCHAR(100) DEFAULT '' COMMENT '商品编码/书号/SKU',
			customer_price DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '客单价',
			online_commission_rate DECIMAL(10,4) NOT NULL DEFAULT 0.0000 COMMENT '线上佣金率',
			offline_commission_rate DECIMAL(10,4) NOT NULL DEFAULT 0.0000 COMMENT '线下佣金率',
			retain_rate DECIMAL(10,4) NOT NULL DEFAULT 0.0000 COMMENT '留存比例',
			remark VARCHAR(500) DEFAULT '' COMMENT '备注',
			status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0=禁用 1=启用',
			source_file VARCHAR(255) DEFAULT '' COMMENT '来源文件名',
			source_sheet VARCHAR(100) DEFAULT '' COMMENT '来源工作表',
			source_row_no INT NOT NULL DEFAULT 0 COMMENT '来源行号',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			INDEX idx_team_name (team_name),
			INDEX idx_platform_shop (platform_type, shop_name),
			INDEX idx_uid (uid),
			INDEX idx_show_id (show_id),
			INDEX idx_account_name (account_name),
			INDEX idx_product_id (product_id),
			INDEX idx_goods_code (goods_code),
			INDEX idx_status (status)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='外部达人台账表'`)

			// 千川广告主映射表
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS account_qianchuan_mapping (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			account_id INT NOT NULL COMMENT '内部账号ID，关联account.id',
			company_id BIGINT NOT NULL DEFAULT 0 COMMENT '所属公司ID，关联sys_company.id',
			qianchuan_advertiser_id BIGINT NOT NULL COMMENT '千川广告主ID',
			qianchuan_advertiser_name VARCHAR(200) DEFAULT '' COMMENT '千川广告主名称（冗余，方便查看）',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uk_account_company (account_id, company_id),
			KEY idx_advertiser_id (qianchuan_advertiser_id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='内部账号与千川广告主映射表'`)

			// 千川账户管理表：千川广告主归属到达人/投手，用于后续消耗统计归属。
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS qianchuan_account_owner_mapping (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			advertiser_id BIGINT(20) NOT NULL COMMENT '千川广告主ID/千川户号',
			advertiser_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT '千川户名称',
			owner_name VARCHAR(100) NOT NULL DEFAULT '' COMMENT 'Excel中的达人/归属名称',
			trader_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '匹配到的投手/达人ID，0表示未匹配或备用',
			company_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属公司ID',
			department_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属部门ID',
			remark VARCHAR(500) NOT NULL DEFAULT '' COMMENT '备注',
			status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：0=停用 1=启用',
			source_file VARCHAR(255) NOT NULL DEFAULT '' COMMENT '来源文件',
			source_sheet VARCHAR(100) NOT NULL DEFAULT '' COMMENT '来源工作表',
			source_row_no INT NOT NULL DEFAULT 0 COMMENT '来源行号',
			created_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
			updated_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY uk_advertiser_id (advertiser_id),
			KEY idx_owner_name (owner_name),
			KEY idx_trader_id (trader_id),
			KEY idx_status (status)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='千川账户管理表'`)

			// Webhook推送配置表（按节点配置不同机器人群）
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS webhook_config (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL DEFAULT '' COMMENT '配置名称',
			platform VARCHAR(20) NOT NULL DEFAULT 'feishu' COMMENT '平台：feishu/wecom',
			webhook_url VARCHAR(500) NOT NULL DEFAULT '' COMMENT '机器人webhook地址',
			secret VARCHAR(200) NOT NULL DEFAULT '' COMMENT '签名密钥(飞书可选)',
			node_type VARCHAR(20) NOT NULL DEFAULT 'summary' COMMENT '节点类型：summary/company/department/trader',
			node_id BIGINT NOT NULL DEFAULT 0 COMMENT '节点ID(summary时为0)',
			node_name VARCHAR(200) NOT NULL DEFAULT '' COMMENT '节点名称(冗余)',
			push_morning TINYINT NOT NULL DEFAULT 1 COMMENT '是否推早报：0=否 1=是',
			push_evening TINYINT NOT NULL DEFAULT 1 COMMENT '是否推晚报：0=否 1=是',
			push_periods VARCHAR(50) NOT NULL DEFAULT '0,7,15,30' COMMENT '推送周期，逗号分隔：0=今日,7,15,30',
			enabled TINYINT NOT NULL DEFAULT 1 COMMENT '是否启用：0=禁用 1=启用',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uk_node (node_type, node_id, platform),
			KEY idx_enabled (enabled)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Webhook推送配置表'`)

			// 应用配置表：存放财务口径开关等全局配置
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS app_setting (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			setting_key VARCHAR(120) NOT NULL DEFAULT '' COMMENT '配置键',
			setting_value VARCHAR(500) NOT NULL DEFAULT '' COMMENT '配置值',
			description VARCHAR(500) NOT NULL DEFAULT '' COMMENT '配置说明',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uk_setting_key (setting_key)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='应用配置表'`)
			g.DB().Exec(ctx, `
			INSERT INTO app_setting (setting_key, setting_value, description)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE
				description = VALUES(description)
			`,
				service.FinanceSettingQianchuanPlanBlankOrderInferenceEnabled,
				"1",
				"财务佣金：达人为空/0的抖音订单是否通过千川计划反推账号归属",
			)

			g.DB().Exec(ctx, `
			INSERT INTO sys_permission (
				name, code, type, parent_id, path, component, icon,
				method, api, ext_url, hide, status, sort
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
			FROM sys_permission finance
			WHERE finance.code = 'finance'
			  AND NOT EXISTS (
			      SELECT 1 FROM sys_permission
			      WHERE code = 'finance.borrowed-account-attribution'
			  )
			LIMIT 1`)
			g.DB().Exec(ctx, `
			UPDATE sys_permission rule_menu
			JOIN sys_permission finance ON finance.code = 'finance'
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
			WHERE rule_menu.code = 'finance.borrowed-account-attribution'`)
			g.DB().Exec(ctx, `
			INSERT IGNORE INTO sys_role_permission (role_id, permission_id)
			SELECT DISTINCT parent_role.role_id, rule_menu.id
			FROM sys_role_permission parent_role
			JOIN sys_permission finance
			  ON finance.id = parent_role.permission_id
			 AND finance.code = 'finance'
			JOIN sys_permission rule_menu
			  ON rule_menu.code = 'finance.borrowed-account-attribution'`)
			g.DB().Exec(ctx, `
			INSERT INTO sys_permission (
				name, code, type, parent_id, path, component, icon,
				method, api, ext_url, hide, status, sort
			)
			SELECT
				'千川归属规则',
				'finance.qianchuan-attribution-rule',
				2,
				finance.id,
				'/finance/qianchuan-attribution-rule',
				'finance/qianchuan-attribution-rule/index',
				'lucide:shuffle',
				'',
				'',
				'',
				1,
				1,
				4
			FROM sys_permission finance
			WHERE finance.code = 'finance'
			  AND NOT EXISTS (
			      SELECT 1 FROM sys_permission
			      WHERE code = 'finance.qianchuan-attribution-rule'
			  )
			LIMIT 1`)
			g.DB().Exec(ctx, `
			UPDATE sys_permission rule_menu
			JOIN sys_permission finance ON finance.code = 'finance'
			SET rule_menu.name = '千川归属规则',
				rule_menu.type = 2,
				rule_menu.parent_id = finance.id,
				rule_menu.path = '/finance/qianchuan-attribution-rule',
				rule_menu.component = 'finance/qianchuan-attribution-rule/index',
				rule_menu.icon = 'lucide:shuffle',
				rule_menu.method = '',
				rule_menu.api = '',
				rule_menu.ext_url = '',
				rule_menu.hide = 1,
				rule_menu.status = 1,
				rule_menu.sort = 4
			WHERE rule_menu.code = 'finance.qianchuan-attribution-rule'`)
			g.DB().Exec(ctx, `
			INSERT IGNORE INTO sys_role_permission (role_id, permission_id)
			SELECT DISTINCT parent_role.role_id, rule_menu.id
			FROM sys_role_permission parent_role
			JOIN sys_permission finance
			  ON finance.id = parent_role.permission_id
			 AND finance.code = 'finance'
			JOIN sys_permission rule_menu
			  ON rule_menu.code = 'finance.qianchuan-attribution-rule'`)

			g.DB().Exec(ctx, `
			INSERT INTO sys_permission (
				name, code, type, parent_id, path, component, icon,
				method, api, ext_url, hide, status, sort
			)
			SELECT
				'千川账户管理',
				'finance.qianchuan-account',
				2,
				finance.id,
				'/finance/qianchuan-account',
				'finance/qianchuan-account/index',
				'lucide:badge-dollar-sign',
				'',
				'',
				'',
				1,
				1,
				5
			FROM sys_permission finance
			WHERE finance.code = 'finance'
			  AND NOT EXISTS (
			      SELECT 1 FROM sys_permission
			      WHERE code = 'finance.qianchuan-account'
			  )
			LIMIT 1`)
			g.DB().Exec(ctx, `
			UPDATE sys_permission account_menu
			JOIN sys_permission finance ON finance.code = 'finance'
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
			WHERE account_menu.code = 'finance.qianchuan-account'`)
			g.DB().Exec(ctx, `
			INSERT IGNORE INTO sys_role_permission (role_id, permission_id)
			SELECT DISTINCT parent_role.role_id, account_menu.id
			FROM sys_role_permission parent_role
			JOIN sys_permission finance
			  ON finance.id = parent_role.permission_id
			 AND finance.code = 'finance'
			JOIN sys_permission account_menu
			  ON account_menu.code = 'finance.qianchuan-account'`)

			// 千川token：调用advertiser/get获取平台账户列表，再解析为千川广告主ID
			// 注意：advertiser/get返回的是巨量引擎平台账户ID（店铺级别），
			// 不能直接用于千川v1.0 API，需要通过 qianchuan/shop/advertiser/list/ 解析
			go func() {
				bgCtx := context.Background()
				time.Sleep(3 * time.Second)
				token, _, err := oceanengine.GetLatestToken(bgCtx)
				if err != nil {
					g.Log().Warningf(bgCtx, "[Qianchuan] 启动同步广告主前获取有效token失败: %v", err)
					return
				}
				row, _ := g.DB().Model("qianchuan_token").Ctx(bgCtx).OrderDesc("id").Limit(1).One()
				if row.IsEmpty() {
					return
				}
				if len(token) < 8 {
					return
				}
				g.Log().Infof(bgCtx, "[Qianchuan] 启动获取advertiser列表, token前8位=%s", token[:8])

				// Step1: 获取平台账户列表（店铺级别）
				shopIDs, err := oceanengine.GetShopIDs(bgCtx, token)
				if err != nil {
					g.Log().Warningf(bgCtx, "[Qianchuan] 获取平台账户列表失败: %v", err)
					return
				}
				g.Log().Infof(bgCtx, "[Qianchuan] 平台账户列表: %v (共%d个)", shopIDs, len(shopIDs))
				if len(shopIDs) == 0 {
					return
				}

				// Step2: 将平台账户ID解析为千川广告主ID
				qianchuanIDs, err := oceanengine.ResolveQianchuanAdvertiserIDs(bgCtx, token, shopIDs)
				if err != nil {
					g.Log().Warningf(bgCtx, "[Qianchuan] 解析千川广告主ID失败: %v", err)
					return
				}
				if len(qianchuanIDs) > 0 {
					idsJSON, _ := json.Marshal(qianchuanIDs)
					g.DB().Model("qianchuan_token").Ctx(bgCtx).Where("id", row["id"]).Data(g.Map{"advertiser_ids": string(idsJSON)}).Update()
					g.Log().Infof(bgCtx, "[Qianchuan] 已更新千川广告主IDs: %v (共%d个)", qianchuanIDs, len(qianchuanIDs))
				} else {
					g.Log().Warningf(bgCtx, "[Qianchuan] 解析千川广告主ID返回空列表")
				}
			}()

			// 全局 CORS 中间件
			s.Use(middleware.CORS)

			// 公开 API 路由 (无需认证)
			s.Group("/api", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					auth.NewPublic(),
					auth.NewFeishuAuthController(),
				)
				// 抖店/巨量千川 OAuth 回调（无需认证）
				// OAuth回调：独立路由，不使用 MiddlewareHandlerResponse（需要302跳转）
				s.BindHandler("GET:/api/douyin/callback", system.NewDouyinCallbackController().Callback)
				group.GET("/qianchuan/auth-url", system.NewQianchuanPollerController().AuthUrl)
				group.GET("/qianchuan/token-status", system.NewQianchuanPollerController().TokenStatus)
			})

			// 需要认证的 API 路由
			s.Group("/api", func(group *ghttp.RouterGroup) {
				group.Middleware(middleware.JwtAuth)
				group.Middleware(middleware.OperationLog)
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					auth.New(),
					ctrlTask.New(),
					ctrlData.New(),
					ctrlMenu.New(),
					ctrlCompany.New(),
					ctrlCrawlerSvc.New(),
					ctrlScheduler.New(),
					system.NewRoleController(),
					system.NewUserController(),
					system.NewMenuController(),
					system.NewDeptController(),
					system.NewProfileController(),
					system.NewOperationLogController(),
					system.NewLoginLogController(),
					system.NewAccountController(),
					system.NewOrderController(),
					system.NewKuaishouOrderController(),
					system.NewShipinhaoOrderController(),
					system.NewGoodsController(),
					system.NewOrderAnalysisController(),
					system.NewOrgStructureController(),
					system.NewTraderController(),
					system.NewFinanceController(),
					system.NewTaskScheduleController(),
					system.NewPlatformController(),
					permissionCtrl.New(),
					ctrlVideo.New(),
				)

				// 达人订单匹配路由（显式绑定，支持多级路径）
				matchCtrl := system.NewOrderMatchController()
				group.POST("/order-match/import-orders", matchCtrl.OrderMatchImportOrders)
				group.POST("/order-match/import-jingxuan", matchCtrl.OrderMatchImportJingxuan)
				group.POST("/order-match/run", matchCtrl.OrderMatchRun)
				group.GET("/order-match/stats", matchCtrl.OrderMatchStats)
				group.GET("/order-match/order-list", matchCtrl.OrderMatchOrderList)
				group.GET("/order-match/jingxuan-list", matchCtrl.OrderMatchJingxuanList)
				group.GET("/order-match/result-list", matchCtrl.OrderMatchResultList)
				// 订单统计
				orderCtrl := system.NewOrderController()
				group.GET("/order-stats", orderCtrl.OrderStats)
				group.POST("/order-stats/backfill", orderCtrl.OrderStatsBackfill)

				// 系统消息
				notifyCtrl := system.NewNotificationController()
				group.GET("/notifications/unread-count", notifyCtrl.UnreadCount)
				group.GET("/notifications/list", notifyCtrl.List)
				group.POST("/notifications/read", notifyCtrl.Read)
				group.POST("/notifications/read-all", notifyCtrl.ReadAll)
				group.GET("/notifications/detail", notifyCtrl.Detail)

				// 精选联盟订单
				allianceCtrl := system.NewAllianceController()
				group.GET("/order/alliance-list", allianceCtrl.AllianceList)
				group.POST("/order/alliance-fill-buyin", allianceCtrl.AllianceFillBuyinId)

				// 店铺管理 CRUD
				storeCtrl := system.NewStoreController()
				group.GET("/store/list", storeCtrl.List)
				group.GET("/store/detail", storeCtrl.Detail)
				group.POST("/store/save", storeCtrl.Save)
				group.POST("/store/delete", storeCtrl.Delete)
				group.GET("/store/stats", storeCtrl.Stats)

				// 店铺借用账号归属规则 CRUD
				saaCtrl := system.NewShopAccountAttributionController()
				group.GET("/shop-account-attribution/list", saaCtrl.List)
				group.GET("/shop-account-attribution/detail", saaCtrl.Detail)
				group.POST("/shop-account-attribution/save", saaCtrl.Save)
				group.POST("/shop-account-attribution/delete", saaCtrl.Delete)
				group.GET("/shop-account-attribution/shops", saaCtrl.Shops)
				group.GET("/shop-account-attribution/operators", saaCtrl.Operators)
				group.GET("/shop-account-attribution/search-accounts", saaCtrl.SearchAccounts)

				// 千川消耗归属规则 CRUD
				qianchuanAttrCtrl := system.NewQianchuanAttributionRuleController()
				group.GET("/qianchuan-attribution-rule/list", qianchuanAttrCtrl.List)
				group.GET("/qianchuan-attribution-rule/detail", qianchuanAttrCtrl.Detail)
				group.POST("/qianchuan-attribution-rule/save", qianchuanAttrCtrl.Save)
				group.POST("/qianchuan-attribution-rule/delete", qianchuanAttrCtrl.Delete)
				group.POST("/qianchuan-attribution-rule/toggle", qianchuanAttrCtrl.Toggle)
				group.GET("/qianchuan-attribution-rule/advertisers", qianchuanAttrCtrl.Advertisers)
				group.GET("/qianchuan-attribution-rule/traders", qianchuanAttrCtrl.Traders)
				group.GET("/qianchuan-attribution-rule/search-accounts", qianchuanAttrCtrl.SearchAccounts)

				qianchuanAccountCtrl := system.NewQianchuanAccountOwnerController()
				group.GET("/qianchuan-account/list", qianchuanAccountCtrl.List)
				group.POST("/qianchuan-account/save", qianchuanAccountCtrl.Save)
				group.POST("/qianchuan-account/delete", qianchuanAccountCtrl.Delete)
				group.POST("/qianchuan-account/toggle", qianchuanAccountCtrl.Toggle)
				group.GET("/qianchuan-account/owners", qianchuanAccountCtrl.Owners)
				group.GET("/qianchuan-account/traders", qianchuanAccountCtrl.Traders)
				group.POST("/qianchuan-account/sync-reports", qianchuanAccountCtrl.SyncReports)

				// 外部达人台账 CRUD
				externalTalentCtrl := system.NewExternalTalentController()
				group.GET("/external-talent/stats", externalTalentCtrl.Stats)
				group.GET("/external-talent/list", externalTalentCtrl.List)
				group.GET("/external-talent/detail", externalTalentCtrl.Detail)
				group.POST("/external-talent/save", externalTalentCtrl.Save)
				group.POST("/external-talent/delete", externalTalentCtrl.Delete)

				// 千川广告主映射管理
				accountCtrl := system.NewAccountController()
				group.GET("/qianchuan-mapping/list", accountCtrl.QianchuanMappingList)
				group.POST("/qianchuan-mapping/bind", accountCtrl.QianchuanMappingBind)
				group.POST("/qianchuan-mapping/unbind", accountCtrl.QianchuanMappingUnbind)
				group.GET("/qianchuan-mapping/advertisers", accountCtrl.QianchuanAdvertisers)
				group.POST("/qianchuan-mapping/auto-bind", accountCtrl.QianchuanMappingAutoBind)
				group.POST("/qianchuan-mapping/import", accountCtrl.QianchuanStoreMappingImport)
				group.POST("/account/sync-org-attribution", accountCtrl.SyncOrgAttribution)
				group.POST("/account/sync-trader-relations", accountCtrl.SyncTraderRelations)

				// 实时盈亏查询（含千川消耗）
				group.GET("/finance/today-cost", system.NewFinanceController().TodayCost)
				group.GET("/finance/cost-ranking", system.NewFinanceController().CostRanking)
				group.GET("/finance/store-cost-stats", system.NewFinanceController().StoreCostStats)
				group.GET("/finance/store-cost-detail", system.NewFinanceController().StoreCostDetail)
				group.GET("/finance/subject-cost-stats", system.NewFinanceController().SubjectCostStats)
				group.GET("/finance/subject-cost-detail", system.NewFinanceController().SubjectCostDetail)
				group.GET("/finance/author-cost-ranking", system.NewFinanceController().AuthorCostRanking)

				// Webhook推送配置管理
				webhookCtrl := system.NewWebhookConfigController()
				group.GET("/webhook-config/list", webhookCtrl.List)
				group.POST("/webhook-config/save", webhookCtrl.Save)
				group.POST("/webhook-config/delete", webhookCtrl.Delete)
				group.POST("/webhook-config/toggle", webhookCtrl.Toggle)
				group.POST("/webhook-config/test-push", webhookCtrl.TestPush)
			})

			// 导出类接口: 需要 JWT 认证，但使用 raw *ghttp.Request 直接处理文件下载
			// （不经过 MiddlewareHandlerResponse，避免文件下载响应被 JSON 包裹）
			s.Group("/api", func(group *ghttp.RouterGroup) {
				group.Middleware(middleware.JwtAuth)
				group.Middleware(middleware.OperationLog)
				oaCtrl := system.NewOrderAnalysisController()
				group.GET("/order-analysis/group-export", oaCtrl.GroupExport)
				group.GET("/order-analysis/export-shop-orders", oaCtrl.ExportShopOrders)
				matchCtrl := system.NewOrderMatchController()
				group.GET("/order-match/export", matchCtrl.OrderMatchExport)
				allianceCtrl := system.NewAllianceController()
				group.GET("/order/alliance-export", allianceCtrl.AllianceExport)
				financeCtrl := system.NewFinanceController()
				group.GET("/finance/trader-export", financeCtrl.TraderExport)
				group.GET("/finance/account-export", financeCtrl.AccountExport)
				group.GET("/finance/blank-order-attributions-export", financeCtrl.BlankOrderAttributionExport)
				group.GET("/finance/valid-order-attributions-export", financeCtrl.ValidOrderAttributionExport)
				group.GET("/finance/cost-ranking-export", financeCtrl.CostRankingExport)
				group.GET("/finance/author-cost-ranking-export", financeCtrl.AuthorCostRankingExport)
				group.GET("/finance/cost-detail-export", financeCtrl.CostDetailExport)
				group.GET("/finance/store-cost-detail-export", financeCtrl.StoreCostDetailExport)
				group.GET("/finance/store-cost-all-detail-export", financeCtrl.StoreCostAllDetailExport)
				group.GET("/finance/subject-cost-detail-export", financeCtrl.SubjectCostDetailExport)
				group.GET("/finance/overall-detail-export", financeCtrl.OverallDetailExport)
				// 抖店轮询器控制
				douyinPollerCtrl := system.NewDouyinPollerController()
				group.POST("/douyin-poller/start", douyinPollerCtrl.Start)
				group.POST("/douyin-poller/stop", douyinPollerCtrl.Stop)
				group.GET("/douyin-poller/status", douyinPollerCtrl.Status)
				group.POST("/douyin-poller/trigger-alliance", douyinPollerCtrl.TriggerAlliance)
				group.GET("/douyin/query-shop-alliance-order", douyinPollerCtrl.QueryShopAllianceOrder)
				group.POST("/douyin-poller/trigger-order", douyinPollerCtrl.TriggerOrder)
				group.POST("/douyin-poller/trigger-refund", douyinPollerCtrl.TriggerRefund)
				wechatShopOrderCtrl := system.NewShipinhaoOrderController()
				group.POST("/wechat-shop-order/trigger-pull", wechatShopOrderCtrl.WechatShopOrderPull)
				// 千川数据拉取控制
				qianchuanPollerCtrl := system.NewQianchuanPollerController()
				group.POST("/qianchuan-poller/start", qianchuanPollerCtrl.Start)
				group.POST("/qianchuan-poller/stop", qianchuanPollerCtrl.Stop)
				group.GET("/qianchuan-poller/status", qianchuanPollerCtrl.Status)
				group.POST("/qianchuan-poller/trigger", qianchuanPollerCtrl.ManualTrigger)
				group.POST("/qianchuan-poller/trigger-overall", qianchuanPollerCtrl.ManualTriggerOverall)
				group.GET("/qianchuan-poller/debug", qianchuanPollerCtrl.DebugToken)
				group.POST("/qianchuan-poller/sync-advertisers", qianchuanPollerCtrl.SyncAdvertisers)
				group.GET("/qianchuan-poller/test-report", qianchuanPollerCtrl.TestReportAPI)
				group.GET("/qianchuan-poller/daily-report", qianchuanPollerCtrl.DailyReportList)
				group.GET("/qianchuan-poller/plan-report", qianchuanPollerCtrl.PlanReportList)
				group.GET("/qianchuan-poller/plan-hierarchy", qianchuanPollerCtrl.PlanHierarchy)
				group.GET("/qianchuan-poller/plan-aweme-export", qianchuanPollerCtrl.PlanAwemeExport)
				group.GET("/qianchuan-poller/overall-report", qianchuanPollerCtrl.OverallReportList)
				group.GET("/qianchuan-poller/overall-report-detail", qianchuanPollerCtrl.OverallReportDetailList)
				group.GET("/qianchuan-poller/daily-report-advertisers", qianchuanPollerCtrl.DailyReportAdvertisers)
				// 店铺管理 导入/导出
				storeFileCtrl := system.NewStoreController()
				group.GET("/store/export", storeFileCtrl.Export)
				group.POST("/store/import", storeFileCtrl.Import)
				group.GET("/store/export-template", storeFileCtrl.ExportTemplate)
			})

			// 视频下载直连入口：通过短期签名 ticket 校验，不走 JWT Header，方便浏览器原生下载。
			s.Group("/api", func(group *ghttp.RouterGroup) {
				videoFileCtrl := ctrlVideo.New()
				group.GET("/videos/{videoId}/file-public", videoFileCtrl.DownloadFilePublic)
			})

			// WebSocket 路由（不经过标准 response 中间件）
			wsCtrl := ctrlWs.New()
			s.Group("/api", func(group *ghttp.RouterGroup) {
				group.GET("/ws/log/{taskId}", wsCtrl.Log)
				group.GET("/ws/crawl-assist/{taskId}", wsCtrl.CrawlAssist)
				group.GET("/ws/notification", wsCtrl.Notification)
			})

			// 原有路由 (保持兼容)
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					hello.NewV1(),
					crawler.New(),
				)
			})

			// 在数据库初始化完成后启动解析服务；恢复和持续补投由管理器负责。
			videoOptions, err := crawlerLogic.VideoDownloadOptionsFromConfig(ctx)
			if err != nil {
				return fmt.Errorf("读取视频解析配置失败: %w", err)
			}
			videoManager := crawlerLogic.GetVideoDownloadManager()
			if err := videoManager.Start(ctx, videoOptions); err != nil {
				return fmt.Errorf("启动视频解析服务失败: %w", err)
			}
			g.Log().Infof(ctx, "[VideoDownload] 解析服务已启动 workers=%d queue_capacity=%d timeout=%s refill_interval=%s",
				videoOptions.Workers, videoOptions.QueueCapacity, videoOptions.TaskTimeout, videoOptions.RefillInterval)
			var videoShutdownOnce sync.Once
			shutdownVideoManager := func() {
				videoShutdownOnce.Do(func() {
					shutdownCtx, cancel := context.WithTimeout(context.Background(), videoOptions.ShutdownTimeout)
					defer cancel()
					if err := videoManager.Shutdown(shutdownCtx); err != nil {
						g.Log().Warningf(shutdownCtx, "[VideoDownload] 关闭解析服务失败 err=%v", err)
					}
					stats := videoManager.Stats()
					g.Log().Infof(context.Background(), "[VideoDownload] 解析服务退出 queued=%d running=%d completed=%d failed=%d retries=%d elapsed=%s",
						stats.Queued, stats.Running, stats.Completed, stats.Failed, stats.Retries, stats.Elapsed)
				})
			}
			defer shutdownVideoManager()
			// GoFrame 在 s.Run 中注册 HTTP 关闭处理器；先注册此处理器以等待解析清理。
			gproc.AddSigHandlerShutdown(func(_ os.Signal) { shutdownVideoManager() })

			// 注册所有内置定时任务（统一由 CronManager 管理）
			// 约定：handler 返回 error 时由 CronManager 统一记录失败日志与通知，handler 内部不应再重复记录
			cronMgr := service.GetCronManager()

			cronMgr.RegisterBuiltin("douyin_order_daily_pull", "0 0 1 * * *", func(ctx context.Context) error {
				yesterday := time.Now().AddDate(0, 0, -1)
				startTime := yesterday.Format("2006-01-02") + " 00:00:00"
				endTime := time.Now().Format("2006-01-02") + " 00:00:00"
				_, err := douyinSvc.NewOrderAllianceSyncService().Sync(ctx, douyinSvc.PollParams{
					StartTime: startTime, EndTime: endTime, Concurrency: 10,
				})
				return err
			})

			cronMgr.RegisterBuiltin("douyin_order_today_pull", "0 0 12,19 * * *", func(ctx context.Context) error {
				now := time.Now()
				startTime := now.Format("2006-01-02") + " 00:00:00"
				endTime := now.Format("2006-01-02 15:04:05")
				_, err := douyinSvc.NewOrderAllianceSyncService().Sync(ctx, douyinSvc.PollParams{
					StartTime: startTime, EndTime: endTime, Concurrency: 10,
				})
				return err
			})

			cronMgr.RegisterBuiltin("douyin_refund_daily_pull", "0 20 1 * * *", func(ctx context.Context) error {
				yesterday := time.Now().AddDate(0, 0, -1)
				startTime := yesterday.Format("2006-01-02") + " 00:00:00"
				endTime := time.Now().Format("2006-01-02") + " 00:00:00"
				_, err := douyinSvc.NewRefundPullService().Pull(ctx, douyinSvc.RefundPullParams{
					StartTime: startTime, EndTime: endTime, Concurrency: 10, UseUpdateTime: true,
				})
				return err
			})

			cronMgr.RegisterBuiltin("douyin_refund_today_pull", "0 20 12,19 * * *", func(ctx context.Context) error {
				now := time.Now()
				startTime := now.Format("2006-01-02") + " 00:00:00"
				endTime := now.Format("2006-01-02 15:04:05")
				_, err := douyinSvc.NewRefundPullService().Pull(ctx, douyinSvc.RefundPullParams{
					StartTime: startTime, EndTime: endTime, Concurrency: 10, UseUpdateTime: true,
				})
				return err
			})

			cronMgr.RegisterBuiltin("wechat_shop_order_daily_pull", "0 40 1 * * *", func(ctx context.Context) error {
				startTime := time.Now().AddDate(0, 0, -2).Format("2006-01-02") + " 00:00:00"
				endTime := time.Now().Format("2006-01-02 15:04:05")
				_, err := wechatshop.NewService().PullOrders(ctx, wechatshop.PullParams{
					StartTime:   startTime,
					EndTime:     endTime,
					TimeType:    "update",
					Concurrency: 5,
				})
				return err
			})

			// 千川昨天数据在 04:37 拉取，财务账号日汇总延后到 06:00，确保广告费已入库。
			cronMgr.RegisterBuiltin("finance_daily_aggregate", "0 0 6 * * *", func(ctx context.Context) error {
				yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
				return service.NewFinanceAggregateService().DailyAggregateJob(ctx, yesterday)
			})

			cronMgr.RegisterBuiltin("order_daily_aggregate", "0 0 3 * * *", func(ctx context.Context) error {
				yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
				return service.NewFinanceAggregateService().OrderDailyAggregateJob(ctx, yesterday)
			})

			// 千川定时拉取：每日只拉昨天，需要回补走手动接口 /api/qianchuan-poller/trigger
			cronMgr.RegisterBuiltin("qianchuan_daily_pull", "0 37 4 * * *", func(ctx context.Context) error {
				yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
				_, err := oceanengine.GetPoller().ManualPullByDate(ctx, yesterday, yesterday)
				return err
			})

			// 千川 access_token 通常按接口返回有效期生效，每6小时检查并在过期前主动续期。
			cronMgr.RegisterBuiltinSilent("qianchuan_token_refresh", "0 17 0,6,12,18 * * *", func(ctx context.Context) error {
				return oceanengine.MaintainAccessToken(ctx)
			})

			// 盈亏早报推送 (09:00) - 只推 push_morning=1 的配置
			cronMgr.RegisterBuiltin("profit_morning_push", "0 0 9 * * *", func(ctx context.Context) error {
				return webhook.NewWebhookPushService().SendReport(ctx, "morning")
			})

			// 盈亏晚报推送 (18:00) - 只推 push_evening=1 的配置
			cronMgr.RegisterBuiltin("profit_evening_push", "0 0 18 * * *", func(ctx context.Context) error {
				return webhook.NewWebhookPushService().SendReport(ctx, "evening")
			})

			// 启动所有定时任务
			cronMgr.StartAll(ctx)

			if allowed, reason := service.SchedulerAutoStartDecision(ctx); allowed {
				crawlerScheduler.Start()
				g.Log().Infof(ctx, "[Scheduler] 采集调度器已按配置自动启动 env=%s", service.AppEnv(ctx))
			} else {
				g.Log().Infof(ctx, "[Scheduler] 跳过采集调度器自动启动 env=%s reason=%s", service.AppEnv(ctx), reason)
			}

			s.Run()
			return nil
		},
	}
)
