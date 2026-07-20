package system

import (
	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type ShopAccountAttributionController struct{}

func NewShopAccountAttributionController() *ShopAccountAttributionController {
	return &ShopAccountAttributionController{}
}

// List 列表 GET /system/shop-account-attribution/list
func (c *ShopAccountAttributionController) List(r *ghttp.Request) {
	var req model.ShopAccountAttributionListReq
	req.Page = r.Get("page", 1).Int()
	req.PageSize = r.Get("pageSize", 10).Int()
	req.Keyword = r.Get("keyword").String()
	req.ShopId = r.Get("shopId").String()
	req.ShopName = r.Get("shopName").String()

	svc := service.NewShopAccountAttributionService()
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

// Detail 详情 GET /system/shop-account-attribution/detail
func (c *ShopAccountAttributionController) Detail(r *ghttp.Request) {
	id := r.Get("id").Int()
	if id == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数错误"})
		return
	}

	svc := service.NewShopAccountAttributionService()
	item, err := svc.GetById(r.Context(), id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "data": nil, "msg": err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": item, "msg": "ok"})
}

// Save 保存 POST /system/shop-account-attribution/save
func (c *ShopAccountAttributionController) Save(r *ghttp.Request) {
	var req model.ShopAccountAttributionSaveReq
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误: " + err.Error()})
		return
	}

	svc := service.NewShopAccountAttributionService()
	_, err := svc.Save(r.Context(), &req)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "保存失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}

// Delete 删除 POST /system/shop-account-attribution/delete
func (c *ShopAccountAttributionController) Delete(r *ghttp.Request) {
	id := r.Get("id").Int()
	if id == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数错误"})
		return
	}

	svc := service.NewShopAccountAttributionService()
	_, err := svc.Delete(r.Context(), id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "删除失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}

// Shops 店铺列表 GET /system/shop-account-attribution/shops
func (c *ShopAccountAttributionController) Shops(r *ghttp.Request) {
	svc := service.NewShopAccountAttributionService()
	shops, err := svc.GetShops(r.Context())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败"})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": shops, "msg": "ok"})
}

// Operators 运营者列表 GET /system/shop-account-attribution/operators
func (c *ShopAccountAttributionController) Operators(r *ghttp.Request) {
	svc := service.NewShopAccountAttributionService()
	operators, err := svc.GetOperators(r.Context())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败"})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": operators, "msg": "ok"})
}

// SearchAccounts 搜索账号 GET /system/shop-account-attribution/search-accounts
func (c *ShopAccountAttributionController) SearchAccounts(r *ghttp.Request) {
	keyword := r.Get("keyword").String()
	if keyword == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "关键词不能为空"})
		return
	}

	svc := service.NewShopAccountAttributionService()
	items, err := svc.SearchAccounts(r.Context(), keyword)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败"})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": items, "msg": "ok"})
}
