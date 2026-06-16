package cmd

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/glog"

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
	"go-crawler/internal/middleware"

	// 注册第三方平台实现（init() 自动注册）
	"go-crawler/internal/service"
	douyinSvc "go-crawler/internal/service/platform/douyin"
	"go-crawler/internal/service/platform/oceanengine"
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

			// 自动建表：千川token存储
			g.DB().Exec(ctx, `CREATE TABLE IF NOT EXISTS qianchuan_token (
				id BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
				access_token VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'access_token',
				advertiser_ids TEXT COMMENT '授权的广告主ID列表',
				expires_in INT(11) NOT NULL DEFAULT 0 COMMENT '过期时间（秒）',
				expires_at DATETIME DEFAULT NULL COMMENT 'token过期时间',
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
				PRIMARY KEY (id)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)

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


			// 千川token：调用advertiser/get API获取正确的广告主列表
			go func() {
				bgCtx := context.Background()
				time.Sleep(3 * time.Second)
				row, _ := g.DB().Model("qianchuan_token").Ctx(bgCtx).OrderDesc("id").Limit(1).One()
				if row.IsEmpty() || (row["advertiser_ids"].String() != "" && row["advertiser_ids"].String() != "[]") { return }
				token := row["access_token"].String()
				g.Log().Infof(bgCtx, "[Qianchuan] 启动获取advertiser列表, token前8位=%s", token[:8])
				client := g.Client()
				resp, err := client.Get(bgCtx, "https://api.oceanengine.com/open_api/oauth2/advertiser/get/", g.Map{"access_token": token})
				if err != nil { g.Log().Errorf(bgCtx, "[Qianchuan] 获取失败: %v", err); return }
				defer resp.Close()
				body := resp.ReadAllString()
				g.Log().Infof(bgCtx, "[Qianchuan] advertiser响应: %s", body)
				var advResp struct {
					Code int64 `json:"code"`
					Data struct {
						List []struct {
							AdvertiserId   int64  `json:"advertiser_id"`
							AdvertiserName string `json:"advertiser_name"`
						} `json:"list"`
					} `json:"data"`
				}
				json.Unmarshal([]byte(body), &advResp)
				if advResp.Code != 0 { return }
				var ids []int64
				for _, item := range advResp.Data.List {
					ids = append(ids, item.AdvertiserId)
					g.Log().Infof(bgCtx, "[Qianchuan] 广告主: id=%d name=%s", item.AdvertiserId, item.AdvertiserName)
				}
				if len(ids) > 0 {
					idsJSON, _ := json.Marshal(ids)
					g.DB().Model("qianchuan_token").Ctx(bgCtx).Where("id", row["id"]).Data(g.Map{"advertiser_ids": string(idsJSON)}).Update()
					g.Log().Infof(bgCtx, "[Qianchuan] 已更新advertiser_ids: %s", string(idsJSON))
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
				// 抖店轮询器控制
				douyinPollerCtrl := system.NewDouyinPollerController()
				group.POST("/douyin-poller/start", douyinPollerCtrl.Start)
				group.POST("/douyin-poller/stop", douyinPollerCtrl.Stop)
				group.GET("/douyin-poller/status", douyinPollerCtrl.Status)
				group.POST("/douyin-poller/trigger-alliance", douyinPollerCtrl.TriggerAlliance)
				group.GET("/douyin/query-shop-alliance-order", douyinPollerCtrl.QueryShopAllianceOrder)
				group.POST("/douyin-poller/trigger-order", douyinPollerCtrl.TriggerOrder)
				// 千川数据拉取控制
				qianchuanPollerCtrl := system.NewQianchuanPollerController()
				group.POST("/qianchuan-poller/start", qianchuanPollerCtrl.Start)
				group.POST("/qianchuan-poller/stop", qianchuanPollerCtrl.Stop)
				group.GET("/qianchuan-poller/status", qianchuanPollerCtrl.Status)
				group.POST("/qianchuan-poller/trigger", qianchuanPollerCtrl.ManualTrigger)
				// 店铺管理 导入/导出
				storeFileCtrl := system.NewStoreController()
				group.GET("/store/export", storeFileCtrl.Export)
				group.POST("/store/import", storeFileCtrl.Import)
				group.GET("/store/export-template", storeFileCtrl.ExportTemplate)
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

			// 注册所有内置定时任务（统一由 CronManager 管理）
			cronMgr := service.GetCronManager()

			cronMgr.RegisterBuiltin("douyin_order_daily_pull", "0 1 * * *", func(ctx context.Context) {
				yesterday := time.Now().AddDate(0, 0, -1)
				startTime := yesterday.Format("2006-01-02") + " 00:00:00"
				endTime := time.Now().Format("2006-01-02") + " 00:00:00"
				douyinSvc.GetPoller().ManualTrigger(ctx, douyinSvc.PollParams{
					StartTime: startTime, EndTime: endTime, Concurrency: 10,
				})
			})

			cronMgr.RegisterBuiltin("alliance_order_daily_pull", "30 1 * * *", func(ctx context.Context) {
				douyinSvc.NewService().DailyPullAllianceOrders(ctx)
			})

			cronMgr.RegisterBuiltin("finance_daily_aggregate", "0 2 * * *", func(ctx context.Context) {
				yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
				service.NewFinanceAggregateService().DailyAggregateJob(ctx, yesterday)
			})

			cronMgr.RegisterBuiltin("order_daily_aggregate", "0 3 * * *", func(ctx context.Context) {
				yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
				service.NewFinanceAggregateService().OrderDailyAggregateJob(ctx, yesterday)
			})

			// 千川定时拉取
			cronMgr.RegisterBuiltin("qianchuan_daily_pull", "37 4 * * *", func(ctx context.Context) {
				oceanengine.GetPoller().ManualPull(ctx, 1)
			})

			// 启动所有定时任务
			cronMgr.StartAll(ctx)

			s.Run()
			return nil
		},
	}
)
