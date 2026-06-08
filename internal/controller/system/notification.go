package system

import (
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// NotificationController 系统消息控制器
type NotificationController struct{}

func NewNotificationController() *NotificationController {
	return &NotificationController{}
}

// UnreadCount 获取未读消息数 GET /api/notifications/unread-count
func (c *NotificationController) UnreadCount(r *ghttp.Request) {
	count, err := service.GetUnreadCount(r.Context())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"count": count}, "msg": "ok"})
}

// List 消息列表 GET /api/notifications/list?type=&page=1&pageSize=10
func (c *NotificationController) List(r *ghttp.Request) {
	notifyType := r.Get("type").String()
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 10).Int()

	list, total, err := service.GetNotificationList(r.Context(), notifyType, page, pageSize)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": g.Map{
			"list":     list,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
		"msg": "ok",
	})
}

// Read 标记单条已读 POST /api/notifications/read
func (c *NotificationController) Read(r *ghttp.Request) {
	id := r.Get("id").Int64()
	if id <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "缺少id参数"})
		return
	}
	if err := service.MarkNotificationRead(r.Context(), id); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "标记失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok"})
}

// ReadAll 全部标记已读 POST /api/notifications/read-all
func (c *NotificationController) ReadAll(r *ghttp.Request) {
	if err := service.MarkAllRead(r.Context()); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "标记失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok"})
}

// Detail 消息详情 GET /api/notifications/detail?id=xxx
func (c *NotificationController) Detail(r *ghttp.Request) {
	id := r.Get("id").Int64()
	if id <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "缺少id参数"})
		return
	}
	detail, err := service.GetNotificationDetail(r.Context(), id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "查询失败: " + err.Error()})
		return
	}
	if detail == nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "msg": "消息不存在"})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": detail, "msg": "ok"})
}
