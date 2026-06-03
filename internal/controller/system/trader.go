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

// BindShop 批量绑定账号到投手 POST /api/trader/bind-shop
func (c *TraderController) BindShop(ctx context.Context, req *v1.TraderBindShopReq) (res *v1.TraderBindShopRes, err error) {
	// 手动解析请求体，确保 accountIds 数组正确绑定
	r := ghttp.RequestFromCtx(ctx)
	body := r.GetBody()
	g.Log().Infof(ctx, "[BindShop] 原始body=%s", string(body))
	var bodyMap map[string]interface{}
	if err := gjson.DecodeTo(body, &bodyMap); err == nil {
		g.Log().Infof(ctx, "[BindShop] bodyMap=%v", bodyMap)
		if ids, ok := bodyMap["accountIds"]; ok {
			g.Log().Infof(ctx, "[BindShop] accountIds原始值=%v, 类型=%T", ids, ids)
		}
	}
	g.Log().Infof(ctx, "[BindShop] 结构体绑定: traderId=%d, accountIds=%v, len=%d", req.TraderId, req.AccountIds, len(req.AccountIds))

	// 优先使用手动解析的 accountIds（绕过 GoFrame 结构体绑定问题）
	traderId := req.TraderId
	accountIds := req.AccountIds
	if ids, ok := bodyMap["accountIds"]; ok && len(accountIds) <= 1 {
		if idList, ok := ids.([]interface{}); ok && len(idList) > len(accountIds) {
			accountIds = make([]int64, 0, len(idList))
			for _, id := range idList {
				accountIds = append(accountIds, gconv.Int64(id))
			}
			g.Log().Infof(ctx, "[BindShop] 手动解析accountIds=%v, len=%d", accountIds, len(accountIds))
		}
	}

	traderService := service.NewTraderService()
	boundCount, err := traderService.BindShop(ctx, traderId, accountIds)
	if err != nil {
		return nil, gerror.New("绑定失败: " + err.Error())
	}

	res = &v1.TraderBindShopRes{
		BoundCount: boundCount,
	}
	return
}

// BindableShops 获取可绑定的账号列表 GET /api/trader/bindable-shops
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

// UnbindShop 解绑账号 POST /api/trader/unbind-shop
func (c *TraderController) UnbindShop(ctx context.Context, req *v1.TraderUnbindShopReq) (res *v1.TraderUnbindShopRes, err error) {
	traderService := service.NewTraderService()

	err = traderService.UnbindShop(ctx, req.TraderId, req.AccountId)
	if err != nil {
		return nil, gerror.New("解绑失败: " + err.Error())
	}

	res = &v1.TraderUnbindShopRes{}
	return
}

// GetShops 获取投手管理的账号列表 GET /api/trader/get-shops
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
