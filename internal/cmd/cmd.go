package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gcron"
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
				douyinCallbackCtrl := system.NewDouyinCallbackController()
				group.GET("/douyin/callback", douyinCallbackCtrl.Callback)
				group.GET("/qianchuan/auth-url", system.NewQianchuanPollerController().AuthUrl)
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

			// 每天凌晨1点拉取昨天抖店订单
			_, _ = gcron.AddSingleton(ctx, "0 1 * * *", func(ctx context.Context) {
				bgCtx := context.Background()
				yesterday := time.Now().AddDate(0, 0, -1)
				startTime := yesterday.Format("2006-01-02") + " 00:00:00"
				endTime := time.Now().Format("2006-01-02") + " 00:00:00"
				g.Log().Infof(bgCtx, "[DouyinPoller] cron触发拉取抖店订单，范围: %s ~ %s", startTime, endTime)

				taskName := fmt.Sprintf("抖店订单拉取-%s", yesterday.Format("2006-01-02"))
				reqParams := fmt.Sprintf(`{"startTime":"%s","endTime":"%s"}`, startTime, endTime)
				logId, _ := service.TaskLogStart(bgCtx, taskName, "douyin_order_pull", "/order/searchList", reqParams)
				t0 := time.Now()
				result, err := douyinSvc.GetPoller().ManualTrigger(bgCtx, douyinSvc.PollParams{
					StartTime: startTime, EndTime: endTime, Concurrency: 5,
				})
				dur := int(time.Since(t0).Milliseconds())
				if err != nil {
					service.TaskLogFailed(bgCtx, logId, err.Error(), dur)
					service.CreateNotification(bgCtx, taskName+"失败", fmt.Sprintf(`{"error":"%s"}`, err.Error()), "task_result", "douyin_order_pull", logId, service.NotifyLevelImportant, 0)
				} else {
					respBody, _ := json.Marshal(g.Map{"totalOrders": result.TotalOrders, "totalSaved": result.TotalSaved, "durationMs": dur})
					service.TaskLogSuccess(bgCtx, logId, "10000", string(respBody), dur)
					service.CreateNotification(bgCtx, taskName+"完成", string(respBody), "task_result", "douyin_order_pull", logId, service.NotifyLevelNormal, 0)
				}
			}, "douyin_order_daily_pull")

			// 每天凌晨1:30拉取昨天精选联盟订单
			_, _ = gcron.AddSingleton(ctx, "30 1 * * *", func(ctx context.Context) {
				bgCtx := context.Background()
				taskName := fmt.Sprintf("精选联盟拉取-%s", time.Now().AddDate(0,0,-1).Format("2006-01-02"))
				logId, _ := service.TaskLogStart(bgCtx, taskName, "alliance_order_pull", "/buyin/queryShopAllianceOrder", "")
				t0 := time.Now()
				err := douyinSvc.NewService().DailyPullAllianceOrders(bgCtx)
				dur := int(time.Since(t0).Milliseconds())
				if err != nil {
					service.TaskLogFailed(bgCtx, logId, err.Error(), dur)
					service.CreateNotification(bgCtx, taskName+"失败", fmt.Sprintf(`{"error":"%s"}`, err.Error()), "task_result", "alliance_order_pull", logId, service.NotifyLevelImportant, 0)
				} else {
					respBody, _ := json.Marshal(g.Map{"durationMs": dur})
					service.TaskLogSuccess(bgCtx, logId, "10000", string(respBody), dur)
					service.CreateNotification(bgCtx, taskName+"完成", string(respBody), "task_result", "alliance_order_pull", logId, service.NotifyLevelNormal, 0)
				}
			}, "alliance_order_daily_pull")

			// 抖店订单轮询器已关闭，需手动启动：POST /api/douyin-poller/start
			// douyinPoller.GetPoller().Start(ctx)

			// 千川数据每日定时拉取（每天凌晨4:37自动执行）
			oceanenginePoller := oceanengine.GetPoller()
			oceanenginePoller.Start(ctx)

			// 每天凌晨2点自动执行达人佣金日聚合
			_, _ = gcron.AddSingleton(ctx, "0 2 * * *", func(ctx context.Context) {
				yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
				g.Log().Infof(ctx, "[FinanceAggregate] cron触发，聚合日期: %s", yesterday)
				svc := service.NewFinanceAggregateService()
				svc.DailyAggregateJob(ctx, yesterday)
			}, "finance_daily_aggregate")

			// 每天凌晨3点自动执行抖音订单日聚合
			_, _ = gcron.AddSingleton(ctx, "0 3 * * *", func(ctx context.Context) {
				yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
				g.Log().Infof(ctx, "[OrderAggregate] cron触发，聚合日期: %s", yesterday)
				service.NewFinanceAggregateService().OrderDailyAggregateJob(ctx, yesterday)
			}, "order_daily_aggregate")

			s.Run()
			return nil
		},
	}
)
