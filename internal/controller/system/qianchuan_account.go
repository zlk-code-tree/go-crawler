package system

import (
	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type QianchuanAccountOwnerController struct{}

func NewQianchuanAccountOwnerController() *QianchuanAccountOwnerController {
	return &QianchuanAccountOwnerController{}
}

func (c *QianchuanAccountOwnerController) List(r *ghttp.Request) {
	req := &model.QianchuanAccountOwnerListReq{
		AdvertiserId: r.Get("advertiserId").Int64(),
		Keyword:      r.Get("keyword").String(),
		OwnerName:    r.Get("ownerName").String(),
		Page:         r.Get("page", 1).Int(),
		PageSize:     r.Get("pageSize", 20).Int(),
		Status:       r.Get("status", -1).Int(),
	}
	items, total, err := service.NewQianchuanAccountOwnerService().List(r.Context(), req)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": items, "msg": "ok", "total": total})
}

func (c *QianchuanAccountOwnerController) Save(r *ghttp.Request) {
	var req model.QianchuanAccountOwnerSaveReq
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数错误: " + err.Error()})
		return
	}
	if _, err := service.NewQianchuanAccountOwnerService().Save(r.Context(), &req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "保存失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "保存成功"})
}

func (c *QianchuanAccountOwnerController) Delete(r *ghttp.Request) {
	id := r.Get("id").Int64()
	if id <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "记录ID不能为空"})
		return
	}
	if _, err := service.NewQianchuanAccountOwnerService().Delete(r.Context(), id); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "删除失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "删除成功"})
}

func (c *QianchuanAccountOwnerController) Toggle(r *ghttp.Request) {
	id := r.Get("id").Int64()
	status := r.Get("status", 1).Int()
	if id <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "记录ID不能为空"})
		return
	}
	if _, err := service.NewQianchuanAccountOwnerService().Toggle(r.Context(), id, status); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "状态更新失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "状态更新成功"})
}

func (c *QianchuanAccountOwnerController) Owners(r *ghttp.Request) {
	items, err := service.NewQianchuanAccountOwnerService().GetOwners(r.Context())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": items, "msg": "ok"})
}

func (c *QianchuanAccountOwnerController) Traders(r *ghttp.Request) {
	items, err := service.NewQianchuanAccountOwnerService().GetTraders(r.Context())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": items, "msg": "ok"})
}

func (c *QianchuanAccountOwnerController) SyncReports(r *ghttp.Request) {
	result, err := service.NewQianchuanAccountOwnerService().SyncReportAttribution(r.Context())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "同步失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": result, "msg": "同步成功"})
}
