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
	ctrlSystem "go-crawler/internal/controller/system"
	ctrlTask "go-crawler/internal/controller/task"
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

			// 全局 CORS 中间件
			s.Use(middleware.CORS)

			// 公开 API 路由 (无需认证)
			s.Group("/api", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					auth.NewPublic(),
				)
			})

			// 需要认证的 API 路由
			s.Group("/api", func(group *ghttp.RouterGroup) {
				group.Middleware(middleware.JwtAuth)
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					auth.New(),
					ctrlTask.New(),
					ctrlData.New(),
					ctrlSystem.New(),
					ctrlMenu.New(),
					ctrlCompany.New(),
					ctrlCrawlerSvc.New(),
				)
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
