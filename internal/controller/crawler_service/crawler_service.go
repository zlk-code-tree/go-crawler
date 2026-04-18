package crawler_service

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/grand"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/logic/crawler"
	"go-crawler/internal/service"
)

// Controller is the crawler service controller for workspace widget.
type Controller struct{}

// New returns a Controller.
func New() *Controller { return &Controller{} }

// CrawlerServiceStatus handles GET /api/crawler-service/status
func (c *Controller) CrawlerServiceStatus(ctx context.Context, req *v1.CrawlerServiceStatusReq) (res *v1.CrawlerServiceStatusRes, err error) {
	r := g.RequestFromCtx(ctx)
	mgr := service.GetCrawlerStatusManager()
	status := mgr.GetStatus()

	startTimeStr := ""
	if status.IsOnline {
		startTimeStr = status.StartTime.Format("2006-01-02 15:04:05")
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "ok",
		"data": g.Map{
			"isOnline":  status.IsOnline,
			"taskId":    status.TaskId,
			"loginType": status.LoginType,
			"startTime": startTimeStr,
			"message":   status.Message,
		},
	})
	return
}

// CrawlerServiceLogin handles POST /api/crawler-service/login
func (c *Controller) CrawlerServiceLogin(ctx context.Context, req *v1.CrawlerServiceLoginReq) (res *v1.CrawlerServiceLoginRes, err error) {
	r := g.RequestFromCtx(ctx)
	mgr := service.GetCrawlerStatusManager()
	isRunning := mgr.IsOnline()

	if isRunning {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "爬虫服务已在运行中，请先强制下线", "data": nil})
		return
	}

	loginType := req.LoginType
	if loginType == "" {
		loginType = "qrcode"
	}

	if loginType == "password" && (req.Username == "" || req.Password == "") {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "账号密码登录模式需要提供用户名和密码", "data": nil})
		return
	}

	taskId := grand.S(16)

	_, dbErr := g.DB().Model("crawl_tasks").Insert(g.Map{
		"task_id":    taskId,
		"status":     "running",
		"login_type": loginType,
		"start_time": time.Now(),
		"created_at": time.Now(),
	})
	if dbErr != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": fmt.Sprintf("创建任务失败: %v", dbErr), "data": nil})
		return
	}

	taskCtx, cancelFn := context.WithCancel(context.Background())

	mgr.SetOnline(taskId, loginType, cancelFn)
	mgr.SetMessage("登录中...")

	go func() {
		defer mgr.SetOffline()

		logic := crawler.New()
		crawlReq := &crawler.CrawlReq{
			LoginType: loginType,
			Username:  req.Username,
			Password:  req.Password,
		}

		mgr.SetMessage("爬虫运行中")

		_, runErr := logic.RunBrowserTask(taskCtx, crawlReq)
		if runErr != nil {
			g.Log().Errorf(taskCtx, "[CrawlerService] task %s failed: %v", taskId, runErr)
			_, _ = g.DB().Model("crawl_tasks").Where("task_id", taskId).Update(g.Map{
				"status":   "failed",
				"end_time": time.Now(),
			})
			return
		}

		_, _ = g.DB().Model("crawl_tasks").Where("task_id", taskId).Update(g.Map{
			"status":   "completed",
			"end_time": time.Now(),
		})
		g.Log().Infof(taskCtx, "[CrawlerService] task %s completed", taskId)
	}()

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "爬虫服务已启动",
		"data": g.Map{"taskId": taskId},
	})
	return
}

// CrawlerServiceForceOffline handles POST /api/crawler-service/force-offline
func (c *Controller) CrawlerServiceForceOffline(ctx context.Context, req *v1.CrawlerServiceForceOfflineReq) (res *v1.CrawlerServiceForceOfflineRes, err error) {
	r := g.RequestFromCtx(ctx)
	mgr := service.GetCrawlerStatusManager()
	isRunning := mgr.IsOnline()

	if !isRunning {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "爬虫服务未在运行", "data": nil})
		return
	}

	mgr.ForceOffline()

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "ok",
		"data": nil,
	})
	return
}
