package system

import (
	douyinSvc "go-crawler/internal/service/platform/douyin"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type DouyinPollerController struct{}

func NewDouyinPollerController() *DouyinPollerController {
	return &DouyinPollerController{}
}

// Start 启动抖店订单轮询 POST /api/douyin-poller/start
func (c *DouyinPollerController) Start(r *ghttp.Request) {
	poller := douyinSvc.GetPoller()
	if poller.IsRunning() {
		r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "轮询器已在运行"})
		return
	}

	if err := poller.Start(r.Context()); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "启动失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "轮询器已启动"})
}

// Stop 停止轮询 POST /api/douyin-poller/stop
func (c *DouyinPollerController) Stop(r *ghttp.Request) {
	poller := douyinSvc.GetPoller()
	if !poller.IsRunning() {
		r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "轮询器未在运行"})
		return
	}
	poller.Stop()
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "轮询器已停止"})
}

// Status 获取轮询器状态 GET /api/douyin-poller/status
func (c *DouyinPollerController) Status(r *ghttp.Request) {
	poller := douyinSvc.GetPoller()
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": poller.Stats(),
		"msg":  "ok",
	})
}
