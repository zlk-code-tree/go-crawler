package system

import (
	"context"

	"go-crawler/api/hello/v1"
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
)

type OrderAnalysisController struct{}

func NewOrderAnalysisController() *OrderAnalysisController {
	return &OrderAnalysisController{}
}

// GroupStats 小组维度统计 GET /api/order-analysis/group-stats
func (c *OrderAnalysisController) GroupStats(ctx context.Context, req *v1.SystemOrderAnalysisGroupStatsReq) (res *v1.SystemOrderAnalysisGroupStatsRes, err error) {
	analysisService := service.NewOrderAnalysisService()
	
	// 使用 ParseTimeRange 支持所有时间类型（today/yesterday/last7/last30/thisMonth/all）
	startDate, endDate := analysisService.ParseTimeRange(req.TimeType, "", "")

	stats, err := analysisService.GroupStats(ctx, startDate, endDate)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.SystemOrderAnalysisGroupStatsRes{
		TimeType:  req.TimeType,
		StartDate: startDate,
		EndDate:   endDate,
		Stats:     stats,
	}
	return
}

// ShopStats 店铺维度统计 GET /api/order-analysis/shop-stats
func (c *OrderAnalysisController) ShopStats(ctx context.Context, req *v1.SystemOrderAnalysisShopStatsReq) (res *v1.SystemOrderAnalysisShopStatsRes, err error) {
	analysisService := service.NewOrderAnalysisService()
	
	// 使用 ParseTimeRange 支持所有时间类型（today/yesterday/last7/last30/thisMonth/all）
	startDate, endDate := analysisService.ParseTimeRange(req.TimeType, "", "")

	stats, err := analysisService.ShopStats(ctx, startDate, endDate, req.GroupId)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.SystemOrderAnalysisShopStatsRes{
		TimeType:  req.TimeType,
		StartDate: startDate,
		EndDate:   endDate,
		Stats:     stats,
	}
	return
}

// GroupList 获取所有小组列表（用于下拉筛选） GET /api/order-analysis/groups
func (c *OrderAnalysisController) GroupList(ctx context.Context, req *v1.SystemOrderAnalysisGroupListReq) (res *v1.SystemOrderAnalysisGroupListRes, err error) {
	analysisService := service.NewOrderAnalysisService()
	groups, err := analysisService.GetGroupList(ctx)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.SystemOrderAnalysisGroupListRes{
		Data: groups,
	}
	return
}

// ShopList 获取所有店铺列表（用于下拉筛选） GET /api/order-analysis/shops
func (c *OrderAnalysisController) ShopList(ctx context.Context, req *v1.SystemOrderAnalysisShopListReq) (res *v1.SystemOrderAnalysisShopListRes, err error) {
	analysisService := service.NewOrderAnalysisService()
	shops, err := analysisService.GetShopList(ctx, req.GroupName)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.SystemOrderAnalysisShopListRes{
		Data: shops,
	}
	return
}

// Overview 统计概览 GET /api/order-analysis/overview
func (c *OrderAnalysisController) Overview(ctx context.Context, req *v1.SystemOrderAnalysisOverviewReq) (res *v1.SystemOrderAnalysisOverviewRes, err error) {
	analysisService := service.NewOrderAnalysisService()

	startDate, endDate := analysisService.ParseTimeRange(req.Preset, req.StartDate, req.EndDate)
	period, total, err := analysisService.Overview(ctx, startDate, endDate)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.SystemOrderAnalysisOverviewRes{}
	res.Period.OrderCount = period["orderCount"].(int)
	res.Period.PayAmount = period["payAmount"].(float64)
	res.Period.CostAmount = period["costAmount"].(float64)
	res.Period.Commission = period["commission"].(float64)
	res.Period.RefundCount = period["refundCount"].(int)
	res.Period.RefundAmount = period["refundAmount"].(float64)

	res.Total.OrderCount = total["orderCount"].(int)
	res.Total.PayAmount = total["payAmount"].(float64)
	res.Total.Commission = total["commission"].(float64)
	return
}

// SalesTrend 销售额趋势 GET /api/order-analysis/sales-trend
func (c *OrderAnalysisController) SalesTrend(ctx context.Context, req *v1.SystemOrderAnalysisSalesTrendReq) (res *v1.SystemOrderAnalysisSalesTrendRes, err error) {
	analysisService := service.NewOrderAnalysisService()

	startDate, endDate := analysisService.ParseTimeRange(req.Preset, req.StartDate, req.EndDate)
	current, compare, err := analysisService.SalesTrend(ctx, startDate, endDate)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.SystemOrderAnalysisSalesTrendRes{
		Current:   current,
		Compare:   compare,
		StartDate: startDate,
		EndDate:   endDate,
	}
	return
}

// Rank 排名榜单 GET /api/order-analysis/rank
func (c *OrderAnalysisController) Rank(ctx context.Context, req *v1.SystemOrderAnalysisRankReq) (res *v1.SystemOrderAnalysisRankRes, err error) {
	analysisService := service.NewOrderAnalysisService()

	startDate, endDate := analysisService.ParseTimeRange(req.Preset, req.StartDate, req.EndDate)
	groupRank, shopRank, err := analysisService.Rank(ctx, startDate, endDate)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.SystemOrderAnalysisRankRes{
		GroupRank: groupRank,
		ShopRank:  shopRank,
		StartDate: startDate,
		EndDate:   endDate,
	}
	return
}

// AnalysisResult 获取完整分析结果 GET /api/order-analysis/result
func (c *OrderAnalysisController) AnalysisResult(ctx context.Context, req *v1.SystemOrderAnalysisResultReq) (res *v1.SystemOrderAnalysisResultRes, err error) {
	analysisService := service.NewOrderAnalysisService()
	
	// 使用 ParseTimeRange 支持所有时间类型（today/yesterday/last7/last30/thisMonth/all）
	startDate, endDate := analysisService.ParseTimeRange(req.TimeType, "", "")

	// 获取小组统计
	groupStats, err := analysisService.GroupStats(ctx, startDate, endDate)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	// 获取店铺统计
	shopStats, err := analysisService.ShopStats(ctx, startDate, endDate, req.GroupId)
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}

	res = &v1.SystemOrderAnalysisResultRes{
		TimeType:   req.TimeType,
		StartDate:  startDate,
		EndDate:    endDate,
		GroupStats: groupStats,
		ShopStats:  shopStats,
	}
	return
}
