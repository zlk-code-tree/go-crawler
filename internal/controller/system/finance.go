package system

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
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
		Platform:  req.Platform,
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
		Platform:  req.Platform,
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

// ShopDetail 获取账号明细统计 GET /api/finance/shop-detail
func (c *FinanceController) ShopDetail(ctx context.Context, req *v1.FinanceShopDetailReq) (res *v1.FinanceShopDetailRes, err error) {
	financeService := service.NewFinanceCommissionService()

	detailReq := &model.FinanceShopDetailReq{
		AccountId: req.AccountId,
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
	// 获取当前用户ID
	var userId int64
	if r := g.RequestFromCtx(ctx); r != nil {
		userId = r.GetCtxVar("userId").Int64()
	}

	reqParams, _ := json.Marshal(g.Map{"startDate": req.StartDate, "endDate": req.EndDate})
	logId, _ := service.TaskLogStart(ctx, "达人佣金历史回填", "finance_commission_backfill", "/api/finance/backfill", string(reqParams))

	// 使用独立 context，避免 HTTP 请求超时导致 context canceled
	go func() {
		bgCtx := context.Background()
		overallStart := time.Now()
		err := service.NewFinanceAggregateService().AggregateRange(bgCtx, req.StartDate, req.EndDate)
		durationMs := int(time.Since(overallStart).Milliseconds())

		if err != nil {
			service.TaskLogFailed(bgCtx, logId, err.Error(), durationMs)
			service.CreateNotification(bgCtx, "达人佣金回填失败", fmt.Sprintf(`{"error":"%s","startDate":"%s","endDate":"%s"}`, err.Error(), req.StartDate, req.EndDate), "task_result", "finance_commission_backfill", logId, service.NotifyLevelImportant, userId)
			return
		}
		respBody, _ := json.Marshal(g.Map{
			"startDate":  req.StartDate,
			"endDate":    req.EndDate,
			"durationMs": durationMs,
		})
		service.TaskLogSuccess(bgCtx, logId, "", string(respBody), durationMs)
		service.CreateNotification(bgCtx, "达人佣金回填完成", string(respBody), "task_result", "finance_commission_backfill", logId, service.NotifyLevelNormal, userId)
	}()
	return &v1.FinanceBackfillRes{
		Success: true,
		Message: "任务已启动，后台回填中",
	}, nil
}

// TraderExport 导出投手订单明细 GET /api/finance/trader-export
func (c *FinanceController) TraderExport(r *ghttp.Request) {
	traderId := r.Get("traderId").Int64()
	timeType := r.Get("timeType").String()
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()

	if traderId == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "投手ID不能为空"})
		return
	}

	financeService := service.NewFinanceCommissionService()
	filename, err := financeService.ExportTraderOrders(r.Context(), traderId, timeType, startDate, endDate)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导出失败: " + err.Error()})
		return
	}

	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filename)))
	r.Response.ServeFile(filename)
	defer os.Remove(filename)
}

// AccountOrders 获取账号订单列表 GET /api/finance/account-orders
func (c *FinanceController) AccountOrders(ctx context.Context, req *v1.FinanceAccountOrdersReq) (res *v1.FinanceAccountOrdersRes, err error) {
	financeService := service.NewFinanceCommissionService()
	result, err := financeService.GetAccountOrderList(ctx, &model.FinanceAccountOrdersReq{
		AccountId: req.AccountId,
		TimeType:  req.TimeType,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Page:      req.Page,
		PageSize:  req.PageSize,
	})
	if err != nil {
		return nil, gerror.New("查询失败: " + err.Error())
	}
	return result, nil
}

// AccountExport 导出账号订单明细 GET /api/finance/account-export
func (c *FinanceController) AccountExport(r *ghttp.Request) {
	accountId := r.Get("accountId").Int()
	timeType := r.Get("timeType").String()
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()

	if accountId == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "账号ID不能为空"})
		return
	}

	financeService := service.NewFinanceCommissionService()
	filename, err := financeService.ExportAccountOrders(r.Context(), accountId, timeType, startDate, endDate)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导出失败: " + err.Error()})
		return
	}

	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filename)))
	r.Response.ServeFile(filename)
	defer os.Remove(filename)
}

// CostRankingExport 导出千川消耗统计 GET /api/finance/cost-ranking-export
func (c *FinanceController) CostRankingExport(r *ghttp.Request) {
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()
	if date := r.Get("date").String(); date != "" && startDate == "" && endDate == "" {
		startDate, endDate = date, date
	}
	dimension := r.Get("dimension", "trader").String()

	filename, err := service.NewFinanceCommissionService().ExportCostRanking(r.Context(), startDate, endDate, dimension)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导出失败: " + err.Error()})
		return
	}
	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filename)))
	r.Response.ServeFile(filename)
	defer os.Remove(filename)
}

// AuthorCostRankingExport 导出达人维度消耗统计 GET /api/finance/author-cost-ranking-export
func (c *FinanceController) AuthorCostRankingExport(r *ghttp.Request) {
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()
	if date := r.Get("date").String(); date != "" && startDate == "" && endDate == "" {
		startDate, endDate = date, date
	}

	filename, err := service.NewFinanceCommissionService().ExportAuthorCostRanking(r.Context(), startDate, endDate)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导出失败: " + err.Error()})
		return
	}
	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filename)))
	r.Response.ServeFile(filename)
	defer os.Remove(filename)
}

// CostDetailExport 导出千川账号消耗明细 GET /api/finance/cost-detail-export
func (c *FinanceController) CostDetailExport(r *ghttp.Request) {
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()
	if date := r.Get("date").String(); date != "" && startDate == "" && endDate == "" {
		startDate, endDate = date, date
	}
	dimension := r.Get("dimension", "trader").String()
	id := r.Get("id").Int64()
	if id <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "ID不能为空"})
		return
	}

	filename, err := service.NewFinanceCommissionService().ExportCostDetails(r.Context(), startDate, endDate, dimension, id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "导出失败: " + err.Error()})
		return
	}
	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filename)))
	r.Response.ServeFile(filename)
	defer os.Remove(filename)
}

// TodayCost 查询今日消耗/佣金/盈亏 GET /api/finance/today-cost
func (c *FinanceController) TodayCost(r *ghttp.Request) {
	// 兼容驼峰(traderId)和下划线(trader_id)两种参数命名
	traderId := r.Get("trader_id", r.Get("traderId", 0).Int64()).Int64()
	departmentId := r.Get("department_id", r.Get("departmentId", 0).Int64()).Int64()
	companyId := r.Get("company_id", r.Get("companyId", 0).Int64()).Int64()
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()
	date := r.Get("date").String()

	if traderId == 0 && departmentId == 0 && companyId == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "请指定 trader_id / department_id / company_id 之一"})
		return
	}

	financeService := service.NewFinanceCommissionService()
	result, err := financeService.GetRealtimeProfit(r.Context(), &service.RealtimeProfitReq{
		TraderId:     traderId,
		DepartmentId: departmentId,
		CompanyId:    companyId,
		Date:         date,
		StartDate:    startDate,
		EndDate:      endDate,
	})
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": result, "msg": "ok"})
}

// CostRanking 消耗排行 GET /api/finance/cost-ranking
func (c *FinanceController) CostRanking(r *ghttp.Request) {
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()
	// 兼容旧版单日期参数
	if date := r.Get("date").String(); date != "" && startDate == "" && endDate == "" {
		startDate, endDate = date, date
	}
	dimension := r.Get("dimension", "trader").String()
	limit := r.Get("limit", 20).Int()

	financeService := service.NewFinanceCommissionService()
	result, err := financeService.GetCostRanking(r.Context(), startDate, endDate, dimension, limit)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": result, "msg": "ok"})
}

// AuthorCostRanking 达人维度消耗统计 GET /api/finance/author-cost-ranking
func (c *FinanceController) AuthorCostRanking(r *ghttp.Request) {
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()
	if date := r.Get("date").String(); date != "" && startDate == "" && endDate == "" {
		startDate, endDate = date, date
	}
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", r.Get("limit", 20).Int()).Int()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	financeService := service.NewFinanceCommissionService()
	list, total, err := financeService.GetAuthorCostRanking(r.Context(), startDate, endDate, page, pageSize)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{
		"list":     list,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	}, "msg": "ok"})
}
