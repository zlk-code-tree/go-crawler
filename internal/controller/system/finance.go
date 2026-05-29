package system

import (
	"context"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
)

// FinanceController 财务管理控制器
type FinanceController struct{}

func NewFinanceController() *FinanceController {
	return &FinanceController{}
}

// CommissionStats 获取佣金统计 GET /api/finance/commission-stats
func (c *FinanceController) CommissionStats(ctx context.Context, req *v1.FinanceStatsReq) (res *v1.FinanceStatsRes, err error) {
	financeService := service.NewFinanceCommissionService()

	statsReq := &model.FinanceStatsReq{
		TimeType:  req.TimeType,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		NodeId:    req.NodeId,
		NodeType:  req.NodeType,
	}

	result, err := financeService.GetStats(ctx, statsReq)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	return result, nil
}

// CommissionChildren 获取下级节点统计 GET /api/finance/commission-children
func (c *FinanceController) CommissionChildren(ctx context.Context, req *v1.FinanceChildrenReq) (res *v1.FinanceChildrenRes, err error) {
	financeService := service.NewFinanceCommissionService()

	childrenReq := &model.FinanceChildrenReq{
		NodeId:    req.NodeId,
		NodeType:  req.NodeType,
		TimeType:  req.TimeType,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
	}

	result, err := financeService.GetChildren(ctx, childrenReq)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	return result, nil
}

// TraderDetail 获取投手详情统计 GET /api/finance/trader-detail
func (c *FinanceController) TraderDetail(ctx context.Context, req *v1.FinanceTraderDetailReq) (res *v1.FinanceTraderDetailRes, err error) {
	financeService := service.NewFinanceCommissionService()

	detailReq := &model.FinanceTraderDetailReq{
		TraderId:  req.TraderId,
		TimeType:  req.TimeType,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
	}

	result, err := financeService.GetTraderDetail(ctx, detailReq)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	return result, nil
}

// ShopDetail 获取店铺明细统计 GET /api/finance/shop-detail
func (c *FinanceController) ShopDetail(ctx context.Context, req *v1.FinanceShopDetailReq) (res *v1.FinanceShopDetailRes, err error) {
	financeService := service.NewFinanceCommissionService()

	detailReq := &model.FinanceShopDetailReq{
		ShopId:    req.ShopId,
		TimeType:  req.TimeType,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
	}

	result, err := financeService.GetShopDetail(ctx, detailReq)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	return result, nil
}

// Aggregate 手动触发日聚合任务 POST /api/finance/aggregate
func (c *FinanceController) Aggregate(ctx context.Context, req *v1.FinanceAggregateReq) (res *v1.FinanceAggregateRes, err error) {
	aggregateService := service.NewFinanceAggregateService()

	if err := aggregateService.DailyAggregateJob(ctx, req.Date); err != nil {
		return &v1.FinanceAggregateRes{
			Success: false,
			Message: "聚合失败: " + err.Error(),
		}, nil
	}

	return &v1.FinanceAggregateRes{
		Success: true,
		Message: "聚合成功",
	}, nil
}

// TraderOrders 获取投手订单列表 GET /api/finance/trader-orders
func (c *FinanceController) TraderOrders(ctx context.Context, req *v1.FinanceTraderOrdersReq) (res *v1.FinanceTraderOrdersRes, err error) {
	financeService := service.NewFinanceCommissionService()

	ordersReq := &model.FinanceTraderOrdersReq{
		TraderId:  req.TraderId,
		TimeType:  req.TimeType,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}

	result, err := financeService.GetTraderOrderList(ctx, ordersReq)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	return result, nil
}

// TraderPlatformStats 获取投手平台统计明细 GET /api/finance/trader-platform-stats
func (c *FinanceController) TraderPlatformStats(ctx context.Context, req *v1.FinanceTraderPlatformStatsReq) (res *v1.FinanceTraderPlatformStatsRes, err error) {
	financeService := service.NewFinanceCommissionService()

	statsReq := &model.FinanceTraderPlatformStatsReq{
		TraderId:  req.TraderId,
		TimeType:  req.TimeType,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
	}

	result, err := financeService.GetTraderPlatformStats(ctx, statsReq)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	return result, nil
}

// Backfill 历史数据回填 POST /api/finance/backfill
func (c *FinanceController) Backfill(ctx context.Context, req *v1.FinanceBackfillReq) (res *v1.FinanceBackfillRes, err error) {
	aggregateService := service.NewFinanceAggregateService()

	if err := aggregateService.AggregateRange(ctx, req.StartDate, req.EndDate); err != nil {
		return &v1.FinanceBackfillRes{
			Success: false,
			Message: "回填失败: " + err.Error(),
		}, nil
	}

	return &v1.FinanceBackfillRes{
		Success: true,
		Message: "回填成功",
	}, nil
}
