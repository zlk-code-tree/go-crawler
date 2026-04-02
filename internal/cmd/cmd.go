package cmd

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/glog"

	"go-crawler/internal/controller/auth"
	"go-crawler/internal/controller/crawler"
	ctrlData "go-crawler/internal/controller/data"
	"go-crawler/internal/controller/hello"
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
				// 手动注册登录/登出路由
				authCtrl := auth.New()
				group.POST("/auth/login", authCtrl.Login)
				group.POST("/auth/logout", authCtrl.Logout)
			})

			// 需要认证的 API 路由
			s.Group("/api", func(group *ghttp.RouterGroup) {
				group.Middleware(middleware.JwtAuth)
				group.Middleware(ghttp.MiddlewareHandlerResponse)

				// 认证相关
				authCtrl := auth.New()
				group.GET("/user/info", authCtrl.UserInfo)
				group.GET("/auth/codes", authCtrl.AccessCodes)
				group.GET("/menu/all", authCtrl.Menus)

				// 任务管理
				taskCtrl := ctrlTask.New()
				group.GET("/task/list", taskCtrl.List)
				group.GET("/task/detail/{id}", taskCtrl.Detail)
				group.POST("/task/create", taskCtrl.Create)
				group.POST("/task/stop/{id}", taskCtrl.Stop)
				group.DELETE("/task/delete/{id}", taskCtrl.Delete)

				// 数据查询
				dataCtrl := ctrlData.New()
				group.GET("/data/overview", dataCtrl.Overview)
				group.GET("/data/ranking", dataCtrl.RankingList)
				group.GET("/data/ranking/stats", dataCtrl.RankingStats)
				group.GET("/data/video", dataCtrl.VideoList)
				group.GET("/data/video/stats", dataCtrl.VideoStats)
				group.GET("/data/export/{taskId}", dataCtrl.Export)
				group.GET("/data/task_ids", dataCtrl.TaskIds)

				// 系统管理
				sysCtrl := ctrlSystem.New()
				group.GET("/system/user/list", sysCtrl.UserList)
				group.POST("/system/user/create", sysCtrl.UserCreate)
				group.PUT("/system/user/update", sysCtrl.UserUpdate)
				group.DELETE("/system/user/delete", sysCtrl.UserDelete)
				group.PUT("/system/user/reset-password", sysCtrl.UserResetPassword)
				group.POST("/system/user/assign-roles", sysCtrl.UserAssignRoles)
				group.GET("/system/role/list", sysCtrl.RoleList)
				group.POST("/system/role/create", sysCtrl.RoleCreate)
				group.PUT("/system/role/update", sysCtrl.RoleUpdate)
				group.DELETE("/system/role/delete", sysCtrl.RoleDelete)
				group.POST("/system/role/assign-permissions", sysCtrl.RoleAssignPermissions)
				group.GET("/system/permission/tree", sysCtrl.PermissionTree)
				group.GET("/system/log/operation", sysCtrl.OperationLogList)
				group.GET("/system/log/login", sysCtrl.LoginLogList)
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
