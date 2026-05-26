package cmd

import (
	"context"

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
	system "go-crawler/internal/controller/system"
	ctrlScheduler "go-crawler/internal/controller/scheduler"
	ctrlTask "go-crawler/internal/controller/task"
	ctrlVideo "go-crawler/internal/controller/video"
	ctrlWs "go-crawler/internal/controller/ws"
	"go-crawler/internal/middleware"
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
				system.NewShopController(),
				system.NewOrderController(),
				system.NewKuaishouOrderController(),
				system.NewShipinhaoOrderController(),
				system.NewGoodsController(),
				system.NewOrderAnalysisController(),
				system.NewOrgStructureController(),
				system.NewTraderController(),
				system.NewFinanceController(),
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
			})

			// WebSocket 路由（不经过标准 response 中间件）
			wsCtrl := ctrlWs.New()
			s.Group("/api", func(group *ghttp.RouterGroup) {
				group.GET("/ws/log/{taskId}", wsCtrl.Log)
				group.GET("/ws/crawl-assist/{taskId}", wsCtrl.CrawlAssist)
			})

			// 原有路由 (保持兼容)
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					hello.NewV1(),
					crawler.New(),
				)
			})

			s.Run()
			return nil
		},
	}
)
