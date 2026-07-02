package system

import (
	"context"
	"fmt"
	"time"

	"go-crawler/internal/service/webhook"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// WebhookConfigController webhook推送配置管理
type WebhookConfigController struct{}

func NewWebhookConfigController() *WebhookConfigController {
	return &WebhookConfigController{}
}

// List 配置列表 GET /api/webhook-config/list
func (c *WebhookConfigController) List(r *ghttp.Request) {
	ctx := r.Context()
	rows, err := g.DB().Model("webhook_config").Ctx(ctx).OrderAsc("id").All()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": rows, "msg": "ok"})
}

// Save 新增/更新配置 POST /api/webhook-config/save
func (c *WebhookConfigController) Save(r *ghttp.Request) {
	var req struct {
		Id          int64  `json:"id"`
		Name        string `json:"name" v:"required#配置名称不能为空"`
		Platform    string `json:"platform" v:"required#平台不能为空"`
		WebhookUrl  string `json:"webhookUrl" v:"required#webhook地址不能为空"`
		Secret      string `json:"secret"`
		NodeType    string `json:"nodeType" v:"required#节点类型不能为空"`
		NodeId      int64  `json:"nodeId"`
		NodeName    string `json:"nodeName"`
		PushMorning int    `json:"pushMorning"`
		PushEvening int    `json:"pushEvening"`
		PushPeriods string `json:"pushPeriods"`
		Enabled     int    `json:"enabled"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数错误: " + err.Error()})
		return
	}

	// 校验 nodeType
	validTypes := map[string]bool{"summary": true, "company": true, "department": true, "trader": true}
	if !validTypes[req.NodeType] {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "nodeType 必须是 summary/company/department/trader 之一"})
		return
	}
	// summary 节点 nodeId 必须为 0
	if req.NodeType == "summary" {
		req.NodeId = 0
	}
	// 非 summary 节点 nodeId 必须大于 0
	if req.NodeType != "summary" && req.NodeId <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "非汇总节点 nodeId 必须大于 0"})
		return
	}
	if req.PushPeriods == "" {
		req.PushPeriods = "0,7,15,30"
	}
	if req.PushMorning == 0 && req.PushEvening == 0 {
		req.PushMorning = 1
		req.PushEvening = 1
	}

	data := g.Map{
		"name":         req.Name,
		"platform":     req.Platform,
		"webhook_url":  req.WebhookUrl,
		"secret":       req.Secret,
		"node_type":    req.NodeType,
		"node_id":      req.NodeId,
		"node_name":    req.NodeName,
		"push_morning": req.PushMorning,
		"push_evening": req.PushEvening,
		"push_periods": req.PushPeriods,
		"enabled":      req.Enabled,
		"updated_at":   time.Now(),
	}

	ctx := r.Context()
	if req.Id > 0 {
		_, err := g.DB().Model("webhook_config").Ctx(ctx).Where("id", req.Id).Data(data).Update()
		if err != nil {
			r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "更新失败: " + err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"id": req.Id}, "msg": "更新成功"})
		return
	}

	id, err := g.DB().Model("webhook_config").Ctx(ctx).Data(data).InsertAndGetId()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "新增失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"id": id}, "msg": "新增成功"})
}

// Delete 删除配置 POST /api/webhook-config/delete
func (c *WebhookConfigController) Delete(r *ghttp.Request) {
	var req struct {
		Id int64 `json:"id" v:"required#ID不能为空"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数错误"})
		return
	}
	_, err := g.DB().Model("webhook_config").Ctx(r.Context()).Where("id", req.Id).Delete()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "删除失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "删除成功"})
}

// Toggle 启用/禁用配置 POST /api/webhook-config/toggle
func (c *WebhookConfigController) Toggle(r *ghttp.Request) {
	var req struct {
		Id      int64 `json:"id" v:"required#ID不能为空"`
		Enabled int   `json:"enabled"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数错误"})
		return
	}
	_, err := g.DB().Model("webhook_config").Ctx(r.Context()).Where("id", req.Id).Data(g.Map{
		"enabled":    req.Enabled,
		"updated_at": time.Now(),
	}).Update()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "操作失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "操作成功"})
}

// TestPush 手动触发推送测试 POST /api/webhook-config/test-push
// 参数: id(配置ID)，推送一条测试消息到该配置对应的机器人
func (c *WebhookConfigController) TestPush(r *ghttp.Request) {
	ctx := r.Context()
	id := r.Get("id", 0).Int64()
	if id == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "id不能为空"})
		return
	}

	row, err := g.DB().Model("webhook_config").Ctx(ctx).Where("id", id).One()
	if err != nil || row.IsEmpty() {
		r.Response.WriteJsonExit(g.Map{"code": 404, "data": nil, "msg": "配置不存在"})
		return
	}

	cfg := &webhook.WebhookConfig{
		Id:          row["id"].Int64(),
		Name:        row["name"].String(),
		Platform:    row["platform"].String(),
		WebhookUrl:  row["webhook_url"].String(),
		Secret:      row["secret"].String(),
		NodeType:    row["node_type"].String(),
		NodeId:      row["node_id"].Int64(),
		NodeName:    row["node_name"].String(),
		PushMorning: row["push_morning"].Int(),
		PushEvening: row["push_evening"].Int(),
		PushPeriods: row["push_periods"].String(),
		Enabled:     row["enabled"].Int(),
	}

	svc := webhook.NewWebhookPushService()
	card, err := svc.BuildProfitCardForNode(ctx, cfg)
	if err != nil {
		g.Log().Errorf(ctx, "[WebhookPush] 主动推送构建数据失败 id=%d name=%s node=%s/%d: %v", cfg.Id, cfg.Name, cfg.NodeType, cfg.NodeId, err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "构建数据失败: " + err.Error()})
		return
	}

	if err := pushByPlatform(ctx, cfg, card); err != nil {
		g.Log().Errorf(ctx, "[WebhookPush] 主动推送失败 id=%d name=%s platform=%s node=%s/%d: %v", cfg.Id, cfg.Name, cfg.Platform, cfg.NodeType, cfg.NodeId, err)
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "推送失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": g.Map{"card": card},
		"msg":  fmt.Sprintf("推送成功：%s(%s/%s)", cfg.Name, cfg.Platform, cfg.NodeType),
	})
}

// pushByPlatform 按平台发送（供 controller 调用）
func pushByPlatform(ctx context.Context, cfg *webhook.WebhookConfig, card *webhook.ProfitCardData) error {
	svc := webhook.NewWebhookPushService()
	return svc.SendCard(ctx, cfg, card)
}
