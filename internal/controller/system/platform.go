package system

import (
	"context"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/model"
	platformSvc "go-crawler/internal/service/platform"

	"github.com/gogf/gf/v2/errors/gerror"
)

// PlatformController 第三方平台对接控制器
type PlatformController struct{}

func NewPlatformController() *PlatformController {
	return &PlatformController{}
}

// OrderList 获取平台订单列表 GET /api/platform/order-list
func (c *PlatformController) OrderList(ctx context.Context, req *v1.PlatformOrderListReq) (res *v1.PlatformOrderListRes, err error) {
	api, ok := platformSvc.GetPlatform(req.Platform)
	if !ok {
		return nil, gerror.New("不支持的平台: " + req.Platform)
	}

	result, err := api.GetOrderList(ctx, model.PlatformOrderListParams{
		Platform:    req.Platform,
		Page:        req.Page,
		PageSize:    req.PageSize,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		OrderStatus: req.OrderStatus,
		AccountId:   req.AccountId,
		StoreId:     req.StoreId,
	})
	if err != nil {
		return nil, gerror.New("获取订单列表失败: " + err.Error())
	}

	return result, nil
}

// OrderDetail 获取平台订单详情 GET /api/platform/order-detail
func (c *PlatformController) OrderDetail(ctx context.Context, req *v1.PlatformOrderDetailReq) (res *v1.PlatformOrderDetailRes, err error) {
	api, ok := platformSvc.GetPlatform(req.Platform)
	if !ok {
		return nil, gerror.New("不支持的平台: " + req.Platform)
	}

	result, err := api.GetOrderDetail(ctx, model.PlatformOrderDetailParams{
		Platform:  req.Platform,
		OrderNo:   req.OrderNo,
		AccountId: req.AccountId,
		StoreId:   req.StoreId,
	})
	if err != nil {
		return nil, gerror.New("获取订单详情失败: " + err.Error())
	}

	return result, nil
}

// AllianceOrderList 获取精选联盟订单列表 GET /api/platform/alliance-order-list
func (c *PlatformController) AllianceOrderList(ctx context.Context, req *v1.PlatformAllianceOrderListReq) (res *v1.PlatformAllianceOrderListRes, err error) {
	api, ok := platformSvc.GetPlatform(req.Platform)
	if !ok {
		return nil, gerror.New("不支持的平台: " + req.Platform)
	}

	result, err := api.GetAllianceOrderList(ctx, model.PlatformAllianceOrderListParams{
		Platform:  req.Platform,
		Cursor:    req.Cursor,
		Size:      req.Size,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		StoreId:   req.StoreId,
		OrderIds:  req.OrderIds,
	})
	if err != nil {
		return nil, gerror.New("获取精选联盟订单失败: " + err.Error())
	}

	return result, nil
}
