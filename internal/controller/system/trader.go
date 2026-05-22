package system

import (
	"context"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/gconv"
)

type TraderController struct{}

func NewTraderController() *TraderController {
	return &TraderController{}
}

// GetList 获取投手列表 GET /api/trader/list
func (c *TraderController) GetList(ctx context.Context, req *v1.TraderListReq) (res *v1.TraderListRes, err error) {
	traderService := service.NewTraderService()

	listReq := model.TraderListReq{
		Page:      req.Page,
		PageSize:  req.PageSize,
		Keyword:   req.Keyword,
		Role:      req.Role,
		CompanyId: req.CompanyId,
		GroupId:   req.GroupId,
	}

	list, total, err := traderService.GetList(ctx, listReq)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.TraderListRes{
		List:  list,
		Total: total,
		Page:  req.Page,
	}
	return
}

// GetDetail 获取投手详情 GET /api/trader/detail
func (c *TraderController) GetDetail(ctx context.Context, req *v1.TraderDetailReq) (res *v1.TraderDetailRes, err error) {
	traderService := service.NewTraderService()

	detail, err := traderService.GetDetail(ctx, req.Id)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = detail
	return
}

// Save 保存投手 POST /api/trader/save
func (c *TraderController) Save(ctx context.Context, req *v1.TraderSaveReq) (res *v1.TraderSaveRes, err error) {
	traderService := service.NewTraderService()

	saveReq := model.TraderSaveReq{
		Id:        req.Id,
		Name:      req.Name,
		CompanyId: req.CompanyId,
		GroupId:   req.GroupId,
		Role:      req.Role,
		Remark:    req.Remark,
	}

	id, err := traderService.Save(ctx, saveReq)
	if err != nil {
		return nil, gerror.New("保存失败: " + err.Error())
	}

	res = &v1.TraderSaveRes{
		Id: id,
	}
	return
}

// Delete 删除投手 POST /api/trader/delete
func (c *TraderController) Delete(ctx context.Context, req *v1.TraderDeleteReq) (res *v1.TraderDeleteRes, err error) {
	traderService := service.NewTraderService()

	err = traderService.Delete(ctx, req.Id)
	if err != nil {
		return nil, gerror.New("删除失败: " + err.Error())
	}

	res = &v1.TraderDeleteRes{}
	return
}

// BindShop 批量绑定店铺到投手 POST /api/trader/bind-shop
func (c *TraderController) BindShop(ctx context.Context, req *v1.TraderBindShopReq) (res *v1.TraderBindShopRes, err error) {
	// 手动解析请求体，确保 shopIds 数组正确绑定
	r := ghttp.RequestFromCtx(ctx)
	body := r.GetBody()
	g.Log().Infof(ctx, "[BindShop] 原始body=%s", string(body))
	var bodyMap map[string]interface{}
	if err := gjson.DecodeTo(body, &bodyMap); err == nil {
		g.Log().Infof(ctx, "[BindShop] bodyMap=%v", bodyMap)
		if ids, ok := bodyMap["shopIds"]; ok {
			g.Log().Infof(ctx, "[BindShop] shopIds原始值=%v, 类型=%T", ids, ids)
		}
	}
	g.Log().Infof(ctx, "[BindShop] 结构体绑定: traderId=%d, shopIds=%v, len=%d", req.TraderId, req.ShopIds, len(req.ShopIds))

	// 优先使用手动解析的 shopIds（绕过 GoFrame 结构体绑定问题）
	traderId := req.TraderId
	shopIds := req.ShopIds
	if ids, ok := bodyMap["shopIds"]; ok && len(shopIds) <= 1 {
		if idList, ok := ids.([]interface{}); ok && len(idList) > len(shopIds) {
			shopIds = make([]int64, 0, len(idList))
			for _, id := range idList {
				shopIds = append(shopIds, gconv.Int64(id))
			}
			g.Log().Infof(ctx, "[BindShop] 手动解析shopIds=%v, len=%d", shopIds, len(shopIds))
		}
	}

	traderService := service.NewTraderService()
	boundCount, err := traderService.BindShop(ctx, traderId, shopIds)
	if err != nil {
		return nil, gerror.New("绑定失败: " + err.Error())
	}

	res = &v1.TraderBindShopRes{
		BoundCount: boundCount,
	}
	return
}

// BindableShops 获取可绑定的店铺列表 GET /api/trader/bindable-shops
func (c *TraderController) BindableShops(ctx context.Context, req *v1.TraderBindableShopsReq) (res *v1.TraderBindableShopsRes, err error) {
	traderService := service.NewTraderService()

	shops, err := traderService.GetBindableShops(ctx, req.TraderId, req.Keyword, req.Platform)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.TraderBindableShopsRes{
		Shops: shops,
	}
	return
}

// UnbindShop 解绑店铺 POST /api/trader/unbind-shop
func (c *TraderController) UnbindShop(ctx context.Context, req *v1.TraderUnbindShopReq) (res *v1.TraderUnbindShopRes, err error) {
	traderService := service.NewTraderService()

	err = traderService.UnbindShop(ctx, req.TraderId, req.ShopId)
	if err != nil {
		return nil, gerror.New("解绑失败: " + err.Error())
	}

	res = &v1.TraderUnbindShopRes{}
	return
}

// GetShops 获取投手管理的店铺列表 GET /api/trader/get-shops
func (c *TraderController) GetShops(ctx context.Context, req *v1.TraderGetShopsReq) (res *v1.TraderGetShopsRes, err error) {
	traderService := service.NewTraderService()

	shops, err := traderService.GetShops(ctx, req.TraderId)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.TraderGetShopsRes{
		Shops: shops,
	}
	return
}
