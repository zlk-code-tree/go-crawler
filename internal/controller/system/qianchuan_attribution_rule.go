package system

import (
	"go-crawler/internal/model"
	"go-crawler/internal/service"
	"go-crawler/internal/service/platform/oceanengine"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type QianchuanAttributionRuleController struct{}

func NewQianchuanAttributionRuleController() *QianchuanAttributionRuleController {
	return &QianchuanAttributionRuleController{}
}

func (c *QianchuanAttributionRuleController) List(r *ghttp.Request) {
	req := &model.QianchuanAttributionRuleListReq{
		Page:         r.Get("page", 1).Int(),
		PageSize:     r.Get("pageSize", 20).Int(),
		Keyword:      r.Get("keyword").String(),
		AdvertiserId: r.Get("advertiserId").Int64(),
		Status:       r.Get("status", -1).Int(),
	}
	items, total, err := service.NewQianchuanAttributionRuleService().List(r.Context(), req)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": items, "total": total, "msg": "ok"})
}

func (c *QianchuanAttributionRuleController) Detail(r *ghttp.Request) {
	id := r.Get("id").Int64()
	if id <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数错误"})
		return
	}
	item, err := service.NewQianchuanAttributionRuleService().GetById(r.Context(), id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "data": nil, "msg": err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": item, "msg": "ok"})
}

func (c *QianchuanAttributionRuleController) Save(r *ghttp.Request) {
	var req model.QianchuanAttributionRuleSaveReq
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误: " + err.Error()})
		return
	}
	if _, err := service.NewQianchuanAttributionRuleService().Save(r.Context(), &req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "保存失败: " + err.Error()})
		return
	}
	if err := oceanengine.SyncQianchuanTemporaryAttribution(r.Context()); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "规则已保存，但同步千川归属失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}

func (c *QianchuanAttributionRuleController) Delete(r *ghttp.Request) {
	id := r.Get("id").Int64()
	if id <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数错误"})
		return
	}
	if _, err := service.NewQianchuanAttributionRuleService().Delete(r.Context(), id); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "删除失败: " + err.Error()})
		return
	}
	if err := oceanengine.SyncQianchuanTemporaryAttribution(r.Context()); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "规则已删除，但同步千川归属失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}

func (c *QianchuanAttributionRuleController) Toggle(r *ghttp.Request) {
	id := r.Get("id").Int64()
	status := r.Get("status", 1).Int()
	if id <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数错误"})
		return
	}
	if _, err := service.NewQianchuanAttributionRuleService().Toggle(r.Context(), id, status); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "更新状态失败: " + err.Error()})
		return
	}
	if err := oceanengine.SyncQianchuanTemporaryAttribution(r.Context()); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "状态已更新，但同步千川归属失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}

func (c *QianchuanAttributionRuleController) Advertisers(r *ghttp.Request) {
	items, err := service.NewQianchuanAttributionRuleService().GetAdvertisers(r.Context(), r.Get("keyword").String())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": items, "msg": "ok"})
}

func (c *QianchuanAttributionRuleController) Traders(r *ghttp.Request) {
	items, err := service.NewQianchuanAttributionRuleService().GetTraders(r.Context())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": items, "msg": "ok"})
}

func (c *QianchuanAttributionRuleController) SearchAccounts(r *ghttp.Request) {
	keyword := r.Get("keyword").String()
	if keyword == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "关键词不能为空"})
		return
	}
	items, err := service.NewQianchuanAttributionRuleService().SearchAccounts(r.Context(), keyword)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败"})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": items, "msg": "ok"})
}
