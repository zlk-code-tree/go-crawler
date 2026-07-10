package system

import (
	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type ExternalTalentController struct{}

func NewExternalTalentController() *ExternalTalentController {
	return &ExternalTalentController{}
}

// List 列表 GET /external-talent/list
func (c *ExternalTalentController) List(r *ghttp.Request) {
	var req model.ExternalTalentListReq
	req.Page = r.Get("page", 1).Int()
	req.PageSize = r.Get("pageSize", 10).Int()
	req.Keyword = r.Get("keyword").String()
	req.TeamName = r.Get("teamName").String()
	req.PlatformType = r.Get("platformType").String()
	req.Status = r.Get("status", -1).Int()

	svc := service.NewExternalTalentService()
	items, total, err := svc.List(r.Context(), &req)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code":  0,
		"data":  items,
		"total": total,
		"msg":   "ok",
	})
}

// Stats 统计 GET /external-talent/stats
func (c *ExternalTalentController) Stats(r *ghttp.Request) {
	svc := service.NewExternalTalentService()
	stats, err := svc.Stats(r.Context())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "统计查询失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": stats, "msg": "ok"})
}

// Detail 详情 GET /external-talent/detail
func (c *ExternalTalentController) Detail(r *ghttp.Request) {
	id := r.Get("id").Int()
	if id == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数错误"})
		return
	}

	svc := service.NewExternalTalentService()
	item, err := svc.GetById(r.Context(), id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "data": nil, "msg": err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": item, "msg": "ok"})
}

// Save 保存 POST /external-talent/save
func (c *ExternalTalentController) Save(r *ghttp.Request) {
	var req model.ExternalTalentSaveReq
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误: " + err.Error()})
		return
	}

	svc := service.NewExternalTalentService()
	_, err := svc.Save(r.Context(), &req)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "保存失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}

// Delete 删除 POST /external-talent/delete
func (c *ExternalTalentController) Delete(r *ghttp.Request) {
	id := r.Get("id").Int()
	if id == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数错误"})
		return
	}

	svc := service.NewExternalTalentService()
	_, err := svc.Delete(r.Context(), id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "删除失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}
