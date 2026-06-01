package service

import (
	"context"
	"fmt"
	"go-crawler/internal/model"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/xuri/excelize/v2"
)

// FinanceCommissionService 财务管理 - 达人佣金统计服务
type FinanceCommissionService struct{}

func NewFinanceCommissionService() *FinanceCommissionService {
	return &FinanceCommissionService{}
}

// ============================================
// 时间处理
// ============================================

// ParseTimeRange 解析时间范围
func (s *FinanceCommissionService) ParseTimeRange(timeType, startDate, endDate string) (start, end string) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	switch timeType {
	case "today":
		start = today.Format("2006-01-02")
		end = today.Format("2006-01-02")
	case "yesterday":
		yesterday := today.AddDate(0, 0, -1)
		start = yesterday.Format("2006-01-02")
		end = yesterday.Format("2006-01-02")
	case "week":
		weekStart := today.AddDate(0, 0, -6)
		start = weekStart.Format("2006-01-02")
		end = today.Format("2006-01-02")
	case "month":
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		start = monthStart.Format("2006-01-02")
		end = today.Format("2006-01-02")
	case "custom":
		start = startDate
		end = endDate
	default:
		start = today.Format("2006-01-02")
		end = today.Format("2006-01-02")
	}
	return
}

// IsDateTime 检查字符串是否包含时间部分
func IsDateTime(s string) bool {
	return len(s) > 10 && s[10] == ' '
}

// CalcEndDateNext 计算结束时间的下一个时间点（用于 < 查询）
func CalcEndDateNext(endDate string) string {
	if IsDateTime(endDate) {
		// 带时间：加1小时
		if t, err := time.Parse("2006-01-02 15:04:05", endDate); err == nil {
			return t.Add(time.Hour).Format("2006-01-02 15:04:05")
		}
	}
	// 纯日期：加1天
	if t, err := time.Parse("2006-01-02", endDate); err == nil {
		return t.Add(24 * time.Hour).Format("2006-01-02")
	}
	return endDate
}

// ============================================
// 统计查询（核心方法）
// ============================================

// GetStats 获取佣金统计
func (s *FinanceCommissionService) GetStats(ctx context.Context, req *model.FinanceStatsReq) (*model.FinanceStatsRes, error) {
	startDate, endDate := s.ParseTimeRange(req.TimeType, req.StartDate, req.EndDate)
	today := time.Now().Format("2006-01-02")
	platform := req.Platform

	// 情况1：查询今日数据（实时计算）
	if startDate >= today {
		return s.getStatsRealtime(ctx, req.NodeId, req.NodeType, startDate, endDate, req.TimeType, platform)
	}

	// 情况2：查询历史数据（先查预聚合表，为空则实时计算）
	if endDate < today {
		result, _ := s.getStatsFromTable(ctx, req.NodeId, req.NodeType, startDate, endDate, req.TimeType, platform)
		if result != nil && result.Summary.OrderCount > 0 {
			return result, nil
		}
		// 预聚合表无数据，实时计算
		return s.getStatsRealtime(ctx, req.NodeId, req.NodeType, startDate, endDate, req.TimeType, platform)
	}

	// 情况3：跨今天（历史 + 今日）
	historyStats, _ := s.getStatsFromTable(ctx, req.NodeId, req.NodeType, startDate, today, req.TimeType, platform)
	todayStats, _ := s.getStatsRealtime(ctx, req.NodeId, req.NodeType, today, endDate, req.TimeType, platform)

	// 如果历史表无数据，用实时计算补充历史部分
	if historyStats == nil || historyStats.Summary.OrderCount == 0 {
		historyStats, _ = s.getStatsRealtime(ctx, req.NodeId, req.NodeType, startDate, today, req.TimeType, platform)
	}

	return s.mergeStats(historyStats, todayStats, req.TimeType, startDate, endDate), nil
}

// getStatsFromTable 从预聚合表查询
func (s *FinanceCommissionService) getStatsFromTable(ctx context.Context, nodeId int64, nodeType, startDate, endDate, timeType, platform string) (*model.FinanceStatsRes, error) {
	// stat_date 是 DATE 类型，截取日期部分
	queryStart := startDate
	queryEnd := endDate
	if len(queryStart) > 10 {
		queryStart = queryStart[:10]
	}
	if len(queryEnd) > 10 {
		queryEnd = queryEnd[:10]
	}

	query := g.DB().Model("finance_daily_stats").
		Where("stat_date >= ?", queryStart).
		Where("stat_date <= ?", queryEnd)
	if platform != "" {
		query = query.Where("platform = ?", platform) // 指定平台
	} else {
		query = query.Where("platform = ''") // 全部平台汇总
	}

	// 根据节点类型筛选（nodeId 为编码后的ID，需要解码）
	// 只查 shop_id > 0 的行（店铺维度），避免公司/部门汇总行重复计算
	switch nodeType {
	case "company":
		query = query.Where("company_id = ? AND shop_id > 0", nodeId)
	case "department":
		realDeptId := model.DecodeDeptID(nodeId)
		query = query.Where("department_id = ? AND shop_id > 0", realDeptId)
	case "trader":
		realTraderId := model.DecodeTraderID(nodeId)
		query = query.Where("trader_id = ? AND shop_id = 0", realTraderId)
	}

	// 聚合统计
	result, err := query.Fields(`
		COALESCE(SUM(order_count), 0) as order_count,
		COALESCE(SUM(product_num), 0) as product_num,
		COALESCE(SUM(pay_amount), 0) as pay_amount,
		COALESCE(SUM(cost_amount), 0) as cost_amount,
		COALESCE(SUM(reserve_amount), 0) as reserve_amount,
		COALESCE(SUM(commission), 0) as commission
	`).One()

	if err != nil {
		return nil, fmt.Errorf("查询预聚合数据失败: %v", err)
	}

	summary := model.FinanceStatsSummary{
		OrderCount:    result["order_count"].Int(),
		ProductNum:    result["product_num"].Int(),
		PayAmount:     result["pay_amount"].Float64(),
		CostAmount:    result["cost_amount"].Float64(),
		ReserveAmount: result["reserve_amount"].Float64(),
		Commission:    result["commission"].Float64(),
	}
	g.Log().Infof(ctx, "[Finance] 预聚合查询: nodeId=%d, nodeType=%s, date=%s~%s, orderCount=%d, commission=%.2f", nodeId, nodeType, startDate, endDate, summary.OrderCount, summary.Commission)

	return &model.FinanceStatsRes{
		TimeType:  timeType,
		StartDate: startDate,
		EndDate:   endDate,
		Summary:   summary,
	}, nil
}

// getStatsRealtime 实时计算统计
func (s *FinanceCommissionService) getStatsRealtime(ctx context.Context, nodeId int64, nodeType, startDate, endDate, timeType, platform string) (*model.FinanceStatsRes, error) {
	// 1. 获取该节点下的所有店铺ID
	shopIds, err := s.getShopIdsByNode(ctx, nodeId, nodeType, platform)
	if err != nil {
		g.Log().Warningf(ctx, "[Finance] getShopIdsByNode 失败: nodeType=%s, nodeId=%d, err=%v", nodeType, nodeId, err)
		return nil, err
	}

	g.Log().Infof(ctx, "[Finance] 实时统计: nodeType=%s, nodeId=%d, shopIds=%v, date=%s~%s", nodeType, nodeId, shopIds, startDate, endDate)

	if len(shopIds) == 0 {
		g.Log().Warningf(ctx, "[Finance] 节点下无店铺: nodeType=%s, nodeId=%d", nodeType, nodeId)
		return &model.FinanceStatsRes{
			TimeType:  timeType,
			StartDate: startDate,
			EndDate:   endDate,
			Summary:   model.FinanceStatsSummary{},
		}, nil
	}

	// 2. 统计这些店铺的订单数据（跨三张订单表）
	stats, err := s.statsOrdersByShopIds(ctx, shopIds, startDate, endDate)
	if err != nil {
		g.Log().Warningf(ctx, "[Finance] statsOrdersByShopIds 失败: shopIds=%v, err=%v", shopIds, err)
		return nil, err
	}

	g.Log().Infof(ctx, "[Finance] 实时统计结果: orderCount=%d, payAmount=%.2f, commission=%.2f", stats.OrderCount, stats.PayAmount, stats.Commission)

	return &model.FinanceStatsRes{
		TimeType:  timeType,
		StartDate: startDate,
		EndDate:   endDate,
		Summary:   *stats,
	}, nil
}

// mergeStats 合并统计数据
func (s *FinanceCommissionService) mergeStats(history, realtime *model.FinanceStatsRes, timeType, startDate, endDate string) *model.FinanceStatsRes {
	if history == nil {
		history = &model.FinanceStatsRes{Summary: model.FinanceStatsSummary{}}
	}
	if realtime == nil {
		realtime = &model.FinanceStatsRes{Summary: model.FinanceStatsSummary{}}
	}

	return &model.FinanceStatsRes{
		TimeType:  timeType,
		StartDate: startDate,
		EndDate:   endDate,
		Summary: model.FinanceStatsSummary{
			OrderCount:    history.Summary.OrderCount + realtime.Summary.OrderCount,
			ProductNum:    history.Summary.ProductNum + realtime.Summary.ProductNum,
			PayAmount:     history.Summary.PayAmount + realtime.Summary.PayAmount,
			CostAmount:    history.Summary.CostAmount + realtime.Summary.CostAmount,
			ReserveAmount: history.Summary.ReserveAmount + realtime.Summary.ReserveAmount,
			Commission:    history.Summary.Commission + realtime.Summary.Commission,
		},
	}
}

// ============================================
// 下级节点查询
// ============================================

// GetChildren 获取下级节点列表
func (s *FinanceCommissionService) GetChildren(ctx context.Context, req *model.FinanceChildrenReq) (*model.FinanceChildrenRes, error) {
	startDate, endDate := s.ParseTimeRange(req.TimeType, req.StartDate, req.EndDate)

	switch req.NodeType {
	case "company":
		return s.getCompanyChildren(ctx, req.NodeId, startDate, endDate, req.TimeType, req.Platform)
	case "department":
		return s.getDepartmentChildren(ctx, req.NodeId, startDate, endDate, req.TimeType, req.Platform)
	case "trader":
		return s.getTraderChildren(ctx, req.NodeId, startDate, endDate, req.TimeType, req.Platform)
	}

	return nil, fmt.Errorf("不支持的节点类型: %s", req.NodeType)
}

// getCompanyChildren 获取公司下级节点
func (s *FinanceCommissionService) getCompanyChildren(ctx context.Context, companyId int64, startDate, endDate, timeType, platform string) (*model.FinanceChildrenRes, error) {
	// 获取当前公司信息
	company, err := g.DB().Model("sys_company").Where("id", companyId).One()
	if err != nil || company.IsEmpty() {
		return nil, fmt.Errorf("公司不存在")
	}

	currentNode := model.NodeInfo{
		Id:   companyId,
		Name: company["name"].String(),
		Type: "company",
	}

	var children []model.NodeStatsItem

	// 1. 获取子公司
	subCompanies, err := g.DB().Model("sys_company").
		Where("parent_id", companyId).
		Where("status", 1).
		Order("sort ASC, id ASC").
		All()
	if err == nil {
		for _, sc := range subCompanies {
			stats, _ := s.GetStats(ctx, &model.FinanceStatsReq{
				NodeId:    sc["id"].Int64(),
				NodeType:  "company",
				StartDate: startDate,
				EndDate:   endDate,
				TimeType:  "custom",
				Platform:  platform,
			})
			children = append(children, model.NodeStatsItem{
				Id:         sc["id"].Int64(),
				Name:       sc["name"].String(),
				Type:       "company",
				OrderCount: stats.Summary.OrderCount,
				ProductNum: stats.Summary.ProductNum,
				PayAmount:  stats.Summary.PayAmount,
				CostAmount: stats.Summary.CostAmount,
				Commission: stats.Summary.Commission,
			})
		}
	}

	// 2. 获取该公司下的部门
	departments, err := g.DB().Model("sys_department").
		Where("company_id", companyId).
		Where("parent_id", 0). // 顶级部门
		Where("status", 1).
		Order("sort ASC, id ASC").
		All()
	if err == nil {
		for _, dept := range departments {
			deptId := dept["id"].Int64()
			stats, _ := s.GetStats(ctx, &model.FinanceStatsReq{
				NodeId:    model.EncodeDeptID(deptId),
				NodeType:  "department",
				StartDate: startDate,
				EndDate:   endDate,
				TimeType:  "custom",
				Platform:  platform,
			})
			children = append(children, model.NodeStatsItem{
				Id:         model.EncodeDeptID(deptId), // 返回编码后的ID
				Name:       dept["name"].String(),
				Type:       "department",
				OrderCount: stats.Summary.OrderCount,
				ProductNum: stats.Summary.ProductNum,
				PayAmount:  stats.Summary.PayAmount,
				CostAmount: stats.Summary.CostAmount,
				Commission: stats.Summary.Commission,
			})
		}
	}

	return &model.FinanceChildrenRes{
		CurrentNode: currentNode,
		Children:    children,
	}, nil
}

// getDepartmentChildren 获取部门下级节点
func (s *FinanceCommissionService) getDepartmentChildren(ctx context.Context, deptId int64, startDate, endDate, timeType, platform string) (*model.FinanceChildrenRes, error) {
	// 解码部门ID：前端传来的是编码后的ID（1000000 + 原始ID），需要解码
	realDeptId := model.DecodeDeptID(deptId)

	// 获取当前部门信息
	dept, err := g.DB().Model("sys_department").Where("id", realDeptId).One()
	if err != nil || dept.IsEmpty() {
		return nil, fmt.Errorf("部门不存在")
	}

	currentNode := model.NodeInfo{
		Id:   deptId, // 返回编码后的ID给前端
		Name: dept["name"].String(),
		Type: "department",
	}

	var children []model.NodeStatsItem

	// 1. 获取子部门
	subDepts, err := g.DB().Model("sys_department").
		Where("parent_id", realDeptId).
		Where("status", 1).
		Order("sort ASC, id ASC").
		All()
	if err == nil {
		for _, sd := range subDepts {
			sdId := sd["id"].Int64()
			stats, _ := s.GetStats(ctx, &model.FinanceStatsReq{
				NodeId:    model.EncodeDeptID(sdId),
				NodeType:  "department",
				StartDate: startDate,
				EndDate:   endDate,
				TimeType:  "custom",
				Platform:  platform,
			})
			children = append(children, model.NodeStatsItem{
				Id:         model.EncodeDeptID(sdId), // 返回编码后的ID
				Name:       sd["name"].String(),
				Type:       "department",
				OrderCount: stats.Summary.OrderCount,
				ProductNum: stats.Summary.ProductNum,
				PayAmount:  stats.Summary.PayAmount,
				CostAmount: stats.Summary.CostAmount,
				Commission: stats.Summary.Commission,
			})
		}
	}

	// 2. 获取该部门下的投手
	traders, err := g.DB().Model("trader").
		Where("group_id", realDeptId).
		Where("status", 1).
		Order("id ASC").
		All()
	if err == nil {
		for _, trader := range traders {
			traderId := trader["id"].Int64()
			stats, _ := s.GetStats(ctx, &model.FinanceStatsReq{
				NodeId:    model.EncodeTraderID(traderId),
				NodeType:  "trader",
				StartDate: startDate,
				EndDate:   endDate,
				TimeType:  "custom",
				Platform:  platform,
			})
			children = append(children, model.NodeStatsItem{
				Id:         model.EncodeTraderID(traderId),
				Name:       trader["name"].String(),
				Type:       "trader",
				OrderCount: stats.Summary.OrderCount,
				ProductNum: stats.Summary.ProductNum,
				PayAmount:  stats.Summary.PayAmount,
				CostAmount: stats.Summary.CostAmount,
				Commission: stats.Summary.Commission,
			})
		}
	}

	return &model.FinanceChildrenRes{
		CurrentNode: currentNode,
		Children:    children,
	}, nil
}

// getTraderChildren 获取投手下级节点（关联的店铺）
func (s *FinanceCommissionService) getTraderChildren(ctx context.Context, traderId int64, startDate, endDate, timeType, platform string) (*model.FinanceChildrenRes, error) {
	// 解码投手ID
	realTraderId := model.DecodeTraderID(traderId)

	// 获取当前投手信息
	trader, err := g.DB().Model("trader").Where("id", realTraderId).One()
	if err != nil || trader.IsEmpty() {
		return nil, fmt.Errorf("投手不存在")
	}

	currentNode := model.NodeInfo{
		Id:   traderId, // 返回编码后的ID给前端
		Name: trader["name"].String(),
		Type: "trader",
	}

	// 获取投手关联的店铺
	shopQuery := g.DB().Model("shop s").
		LeftJoin("trader_shop_relation tsr", "s.id = tsr.shop_id").
		Where("tsr.trader_id", realTraderId)
	if platform != "" {
		shopQuery = shopQuery.Where("s.platform", platform)
	}
	shops, err := shopQuery.Fields("s.id, s.platform, s.uid, s.account_name, s.platform_nickname").
		All()

	var children []model.NodeStatsItem
	if err == nil {
		for _, shop := range shops {
			stats, _ := s.statsOrdersByShopIds(ctx, []int{shop["id"].Int()}, startDate, endDate)
			if stats == nil {
				stats = &model.FinanceStatsSummary{}
			}
			shopName := shop["account_name"].String()
			if shop["platform_nickname"].String() != "" {
				shopName = shop["platform_nickname"].String()
			}
			children = append(children, model.NodeStatsItem{
				Id:         int64(shop["id"].Int()),
				Name:       shopName,
				Type:       "shop",
				OrderCount: stats.OrderCount,
				ProductNum: stats.ProductNum,
				PayAmount:  stats.PayAmount,
				CostAmount: stats.CostAmount,
				Commission: stats.Commission,
			})
		}
	}

	return &model.FinanceChildrenRes{
		CurrentNode: currentNode,
		Children:    children,
	}, nil
}

// ============================================
// 投手详情查询
// ============================================

// GetTraderDetail 获取投手详情统计
func (s *FinanceCommissionService) GetTraderDetail(ctx context.Context, req *model.FinanceTraderDetailReq) (*model.FinanceTraderDetailRes, error) {
	startDate, endDate := s.ParseTimeRange(req.TimeType, req.StartDate, req.EndDate)

	// 获取投手信息
	trader, err := g.DB().Model("trader t").
		LeftJoin("sys_company c", "t.company_id = c.id").
		LeftJoin("sys_department d", "t.group_id = d.id").
		Where("t.id", req.TraderId).
		Fields("t.*, c.name as company_name, d.name as dept_name").
		One()
	if err != nil || trader.IsEmpty() {
		return nil, fmt.Errorf("投手不存在")
	}

	// 获取汇总统计
	stats, err := s.GetStats(ctx, &model.FinanceStatsReq{
		NodeId:    req.TraderId,
		NodeType:  "trader",
		StartDate: startDate,
		EndDate:   endDate,
		TimeType:  "custom",
	})
	if err != nil {
		return nil, err
	}

	// 获取店铺明细
	shopStats, err := s.getTraderShopStats(ctx, req.TraderId, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return &model.FinanceTraderDetailRes{
		TraderId:    req.TraderId,
		TraderName:  trader["name"].String(),
		CompanyName: trader["company_name"].String(),
		DeptName:    trader["dept_name"].String(),
		TimeType:    req.TimeType,
		StartDate:   startDate,
		EndDate:     endDate,
		Summary:     stats.Summary,
		ShopStats:   shopStats,
	}, nil
}

// getTraderShopStats 获取投手关联店铺的统计
func (s *FinanceCommissionService) getTraderShopStats(ctx context.Context, traderId int64, startDate, endDate string) ([]model.ShopStatsItem, error) {
	// 获取投手关联的店铺
	shops, err := g.DB().Model("shop s").
		LeftJoin("trader_shop_relation tsr", "s.id = tsr.shop_id").
		Where("tsr.trader_id", traderId).
		Fields("s.id, s.platform, s.uid, s.account_name, s.platform_nickname").
		All()
	if err != nil {
		return nil, err
	}

	var result []model.ShopStatsItem
	for _, shop := range shops {
		shopId := shop["id"].Int()
		stats, _ := s.statsOrdersByShopIds(ctx, []int{shopId}, startDate, endDate)
		if stats == nil {
			stats = &model.FinanceStatsSummary{}
		}

		shopName := shop["account_name"].String()
		if shop["platform_nickname"].String() != "" {
			shopName = shop["platform_nickname"].String()
		}

		result = append(result, model.ShopStatsItem{
			ShopId:     shopId,
			ShopName:   shopName,
			Uid:        shop["uid"].String(),
			Platform:   shop["platform"].String(),
			OrderCount: stats.OrderCount,
			ProductNum: stats.ProductNum,
			PayAmount:  stats.PayAmount,
			CostAmount: stats.CostAmount,
			Commission: stats.Commission,
		})
	}

	return result, nil
}

// ============================================
// 店铺明细查询
// ============================================

// GetShopDetail 获取店铺明细统计
func (s *FinanceCommissionService) GetShopDetail(ctx context.Context, req *model.FinanceShopDetailReq) (*model.FinanceShopDetailRes, error) {
	startDate, endDate := s.ParseTimeRange(req.TimeType, req.StartDate, req.EndDate)

	// 获取店铺信息
	shop, err := g.DB().Model("shop").Where("id", req.ShopId).One()
	if err != nil || shop.IsEmpty() {
		return nil, fmt.Errorf("店铺不存在")
	}

	// 统计数据
	stats, _ := s.statsOrdersByShopIds(ctx, []int{req.ShopId}, startDate, endDate)
	if stats == nil {
		stats = &model.FinanceStatsSummary{}
	}

	shopName := shop["account_name"].String()
	if shop["platform_nickname"].String() != "" {
		shopName = shop["platform_nickname"].String()
	}

	return &model.FinanceShopDetailRes{
		ShopId:    req.ShopId,
		ShopName:  shopName,
		Uid:       shop["uid"].String(),
		Platform:  shop["platform"].String(),
		TimeType:  req.TimeType,
		StartDate: startDate,
		EndDate:   endDate,
		Summary:   *stats,
	}, nil
}

// ============================================
// 辅助方法
// ============================================

// getShopIdsByNode 根据组织节点获取店铺ID列表（platform 为空则不过滤）
func (s *FinanceCommissionService) getShopIdsByNode(ctx context.Context, nodeId int64, nodeType, platform string) ([]int, error) {
	switch nodeType {
	case "company":
		// 获取该公司及所有子公司的ID
		companyIds := s.getCompanyAndChildrenIds(ctx, nodeId)
		// 通过 trader 关联获取店铺：公司 → trader(company_id) → trader_shop_relation → shop
		traderQuery := g.DB().Model("trader").
			Where("company_id IN(?)", companyIds).
			Where("status", 1)
		traderIdVars, err := traderQuery.Array("id")
		if err != nil {
			return nil, err
		}
		var traderIds []int64
		for _, v := range traderIdVars {
			traderIds = append(traderIds, v.Int64())
		}
		if len(traderIds) == 0 {
			return nil, nil
		}
		shopQuery := g.DB().Model("trader_shop_relation tsr").
			LeftJoin("shop s", "tsr.shop_id = s.id").
			Where("tsr.trader_id IN(?)", traderIds)
		if platform != "" {
			shopQuery = shopQuery.Where("s.platform", platform)
		}
		shopIdVars, err := shopQuery.Array("tsr.shop_id")
		if err != nil {
			return nil, err
		}
		var shopIds []int
		for _, v := range shopIdVars {
			shopIds = append(shopIds, v.Int())
		}
		return shopIds, nil

	case "department":
		// nodeId 为编码后的ID，需要解码
		realDeptId := model.DecodeDeptID(nodeId)
		deptIds := s.getDeptAndChildrenIds(ctx, realDeptId)
		// 通过 trader 关联获取店铺：部门 → trader(group_id) → trader_shop_relation → shop
		traderQuery := g.DB().Model("trader").
			Where("group_id IN(?)", deptIds).
			Where("status", 1)
		traderIdVars, err := traderQuery.Array("id")
		if err != nil {
			return nil, err
		}
		var traderIds []int64
		for _, v := range traderIdVars {
			traderIds = append(traderIds, v.Int64())
		}
		if len(traderIds) == 0 {
			return nil, nil
		}
		shopQuery := g.DB().Model("trader_shop_relation tsr").
			LeftJoin("shop s", "tsr.shop_id = s.id").
			Where("tsr.trader_id IN(?)", traderIds)
		if platform != "" {
			shopQuery = shopQuery.Where("s.platform", platform)
		}
		shopIdVars, err := shopQuery.Array("tsr.shop_id")
		if err != nil {
			return nil, err
		}
		var shopIds []int
		for _, v := range shopIdVars {
			shopIds = append(shopIds, v.Int())
		}
		return shopIds, nil

	case "trader":
		// nodeId 为编码后的ID，需要解码
		realTraderId := model.DecodeTraderID(nodeId)
		shopQuery := g.DB().Model("trader_shop_relation tsr").
			LeftJoin("shop s", "tsr.shop_id = s.id").
			Where("tsr.trader_id", realTraderId)
		if platform != "" {
			shopQuery = shopQuery.Where("s.platform", platform)
		}
		shopIdVars, err := shopQuery.Array("tsr.shop_id")
		if err != nil {
			return nil, err
		}
		var shopIds []int
		for _, v := range shopIdVars {
			shopIds = append(shopIds, v.Int())
		}
		return shopIds, nil
	}

	return nil, nil
}

// getCompanyAndChildrenIds 获取公司及所有子公司ID
func (s *FinanceCommissionService) getCompanyAndChildrenIds(ctx context.Context, companyId int64) []int64 {
	ids := []int64{companyId}

	// 查询子公司
	children, err := g.DB().Model("sys_company").
		Where("parent_id", companyId).
		Where("status", 1).
		Array("id")
	if err != nil {
		return ids
	}

	for _, child := range children {
		childId := child.Int64()
		ids = append(ids, s.getCompanyAndChildrenIds(ctx, childId)...)
	}

	return ids
}

// getDeptAndChildrenIds 获取部门及所有子部门ID
func (s *FinanceCommissionService) getDeptAndChildrenIds(ctx context.Context, deptId int64) []int64 {
	ids := []int64{deptId}

	// 查询子部门
	children, err := g.DB().Model("sys_department").
		Where("parent_id", deptId).
		Where("status", 1).
		Array("id")
	if err != nil {
		return ids
	}

	for _, child := range children {
		childId := child.Int64()
		ids = append(ids, s.getDeptAndChildrenIds(ctx, childId)...)
	}

	return ids
}

// platformTableMap 平台→订单表映射
var platformTableMap = map[string]string{
	"douyin":     "order_info",
	"kuaishou":   "kuaishou_orders",
	"shipinhao":  "shipinhao_orders",
}

// buildPlatformUids 构建 platform→uids 映射，uid 为空时通过 account_name 匹配订单表 talent_name 获取 talent_id
func (s *FinanceCommissionService) buildPlatformUids(ctx context.Context, shops gdb.Result) map[string][]string {
	platformUids := make(map[string][]string)
	var emptyUidShops []gdb.Record

	for _, shop := range shops {
		uid := shop["uid"].String()
		platform := shop["platform"].String()
		if platform == "" {
			continue
		}
		if uid != "" {
			platformUids[platform] = append(platformUids[platform], uid)
		} else {
			emptyUidShops = append(emptyUidShops, shop)
		}
	}

	// uid 为空的店铺，通过 account_name 匹配订单表的 talent_name 获取 talent_id
	if len(emptyUidShops) > 0 {
		var emptyNames []string
		for _, shop := range emptyUidShops {
			name := shop["account_name"].String()
			if name != "" {
				emptyNames = append(emptyNames, name)
			}
		}
		if len(emptyNames) > 0 {
			for _, tableName := range platformTableMap {
				collate := ""
				if tableName == "kuaishou_orders" || tableName == "shipinhao_orders" {
					collate = "COLLATE utf8mb4_unicode_ci"
				}
				sql := fmt.Sprintf(`
					SELECT DISTINCT talent_id, talent_name
					FROM %s
					WHERE talent_name %s IN(?)
					  AND talent_id IS NOT NULL AND talent_id != ''
				`, tableName, collate)
				rows, _ := g.DB().GetAll(ctx, sql, emptyNames)
				nameTidMap := make(map[string]string)
				for _, row := range rows {
					tid := row["talent_id"].String()
					tname := row["talent_name"].String()
					if tid != "" && tname != "" {
						nameTidMap[tname] = tid
					}
				}
				for _, shop := range emptyUidShops {
					name := shop["account_name"].String()
					platform := shop["platform"].String()
					if tid, ok := nameTidMap[name]; ok {
						platformUids[platform] = append(platformUids[platform], tid)
					}
				}
			}
		}
	}

	return platformUids
}

// statsOrdersByShopIds 统计店铺订单数据（根据店铺 platform 只查对应订单表）
func (s *FinanceCommissionService) statsOrdersByShopIds(ctx context.Context, shopIds []int, startDate, endDate string) (*model.FinanceStatsSummary, error) {
	if len(shopIds) == 0 {
		return &model.FinanceStatsSummary{}, nil
	}

	// 获取店铺的 uid、platform、account_name
	shops, err := g.DB().Model("shop").
		Where("id IN(?)", shopIds).
		Fields("uid, platform, account_name").
		All()
	if err != nil {
		return nil, err
	}

	// 构建 platform→uids（uid 为空时自动通过 account_name 匹配订单表）
	platformUids := s.buildPlatformUids(ctx, shops)

	g.Log().Infof(ctx, "[Finance] statsOrdersByShopIds: shopIds=%v, platformUids=%v", shopIds, platformUids)

	if len(platformUids) == 0 {
		g.Log().Warningf(ctx, "[Finance] 店铺uid为空: shopIds=%v, shops=%v", shopIds, shops)
		return &model.FinanceStatsSummary{}, nil
	}

	// 按平台分别统计对应订单表
	totalStats := &model.FinanceStatsSummary{}
	for platform, uids := range platformUids {
		tableName, ok := platformTableMap[platform]
		if !ok {
			continue
		}
		stats, _ := s.statsOrderTable(ctx, tableName, uids, startDate, endDate)
		if stats != nil {
			totalStats.OrderCount += stats.OrderCount
			totalStats.ProductNum += stats.ProductNum
			totalStats.PayAmount += stats.PayAmount
			totalStats.CostAmount += stats.CostAmount
			totalStats.ReserveAmount += stats.ReserveAmount
			totalStats.Commission += stats.Commission
		}
	}

	return totalStats, nil
}

// statsOrderTable 统计单张订单表
func (s *FinanceCommissionService) statsOrderTable(ctx context.Context, tableName string, uids []string, startDate, endDate string) (*model.FinanceStatsSummary, error) {
	// 构建有效订单过滤条件
	validFilter := "AND o.after_sale_status NOT IN ('退款成功', '已全额退款', '待退货') AND o.order_status != '已关闭'"

	// kuaishou/shipinhao 表用 order_no，douyin 用 main_order_no
	orderNoCol := "main_order_no"
	collateClause := ""
	talentCollate := ""
	if tableName == "kuaishou_orders" || tableName == "shipinhao_orders" {
		orderNoCol = "order_no"
		collateClause = "COLLATE utf8mb4_unicode_ci"
		talentCollate = "COLLATE utf8mb4_unicode_ci"
	}

	sql := fmt.Sprintf(`
		SELECT
			COUNT(DISTINCT o.%s) AS order_count,
			COALESCE(SUM(o.product_num), 0) AS product_num,
			COALESCE(SUM(o.order_pay_amount + COALESCE(o.platform_discount, 0) + COALESCE(o.talent_discount, 0)), 0) AS pay_amount,
			COALESCE(SUM(COALESCE(g.base_price, 0) * o.product_num), 0) AS cost_amount,
			COALESCE(SUM(COALESCE(o.product_price, 0) * o.product_num * 0.05), 0) AS reserve_amount
		FROM %s o
		LEFT JOIN goods g ON o.seller_code %s = g.goods_code
		WHERE o.talent_id %s IN(?)
		  AND o.pay_finish_time >= ?
		  AND o.pay_finish_time < ?
		  AND o.%s IS NOT NULL
		  AND o.%s != ''
		  %s
	`, orderNoCol, tableName, talentCollate, collateClause, orderNoCol, orderNoCol, validFilter)

	endDateNext := CalcEndDateNext(endDate)

	result, err := g.DB().GetOne(ctx, sql, uids, startDate, endDateNext)
	if err != nil {
		return nil, err
	}

	payAmount := result["pay_amount"].Float64()
	costAmount := result["cost_amount"].Float64()
	reserveAmount := result["reserve_amount"].Float64()
	commission := payAmount - costAmount - reserveAmount

	return &model.FinanceStatsSummary{
		OrderCount:    result["order_count"].Int(),
		ProductNum:    result["product_num"].Int(),
		PayAmount:     roundFloatFinance(payAmount, 2),
		CostAmount:    roundFloatFinance(costAmount, 2),
		ReserveAmount: roundFloatFinance(reserveAmount, 2),
		Commission:    roundFloatFinance(commission, 2),
	}, nil
}


// ============================================
// 投手订单列表
// ============================================

// GetTraderOrderList 获取投手关联的订单列表
func (s *FinanceCommissionService) GetTraderOrderList(ctx context.Context, req *model.FinanceTraderOrdersReq) (*model.FinanceTraderOrdersRes, error) {
	startDate, endDate := s.ParseTimeRange(req.TimeType, req.StartDate, req.EndDate)
	realTraderId := model.DecodeTraderID(req.TraderId)

	// 获取投手信息
	trader, err := g.DB().Model("trader").Where("id", realTraderId).One()
	if err != nil || trader.IsEmpty() {
		return nil, fmt.Errorf("投手不存在")
	}

	// 获取投手关联的店铺
	shops, err := g.DB().Model("shop s").
		LeftJoin("trader_shop_relation tsr", "s.id = tsr.shop_id").
		Where("tsr.trader_id", realTraderId).
		Fields("s.id, s.uid, s.account_name, s.platform_nickname, s.platform").
		All()
	if err != nil {
		return nil, fmt.Errorf("查询投手店铺失败: %v", err)
	}

	// 构建 uid → 店铺信息 映射，按 platform 分组 uid
	type shopInfo struct {
		name     string
		platform string
	}
	uidShopMap := make(map[string]shopInfo)
	platformUids := s.buildPlatformUids(ctx, shops)
	for _, shop := range shops {
		uid := shop["uid"].String()
		platform := shop["platform"].String()
		if uid == "" || platform == "" {
			continue
		}
		name := shop["account_name"].String()
		if shop["platform_nickname"].String() != "" {
			name = shop["platform_nickname"].String()
		}
		uidShopMap[uid] = shopInfo{name: name, platform: platform}
	}
	// 为通过 account_name 匹配到的 uid 也建立 shopInfo 映射
	for platform, uids := range platformUids {
		for _, uid := range uids {
			if _, exists := uidShopMap[uid]; !exists {
				uidShopMap[uid] = shopInfo{name: uid, platform: platform}
			}
		}
	}

	res := &model.FinanceTraderOrdersRes{
		TraderId:   req.TraderId,
		TraderName: trader["name"].String(),
		TimeType:   req.TimeType,
		StartDate:  startDate,
		EndDate:    endDate,
		Page:       req.Page,
		PageSize:   req.PageSize,
		Orders:     []model.FinanceOrderItem{},
	}

	if len(uidShopMap) == 0 {
		return res, nil
	}

	// 分页参数
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 日期范围
	endDateNext := CalcEndDateNext(endDate)

	validFilter := "AND after_sale_status NOT IN ('退款成功', '已全额退款', '待退货') AND order_status != '已关闭'"

	// 根据店铺 platform 只查对应订单表
	var totalCount int
	var allOrders []model.FinanceOrderItem
	var totalPay, totalCost, totalReserve float64

	for platform, uids := range platformUids {
		tableName, ok := platformTableMap[platform]
		if !ok {
			continue
		}

		orderNoCol := "main_order_no"
		collateClause := ""
		talentCollate := ""
		if tableName == "kuaishou_orders" || tableName == "shipinhao_orders" {
			orderNoCol = "order_no"
			collateClause = "COLLATE utf8mb4_unicode_ci"
			talentCollate = "COLLATE utf8mb4_unicode_ci"
		}

		// 查询该平台的订单数
		countSQL := fmt.Sprintf(`
			SELECT COUNT(*) AS cnt
			FROM %s o
			WHERE o.talent_id %s IN(?)
			  AND o.pay_finish_time >= ?
			  AND o.pay_finish_time < ?
			  AND o.%s IS NOT NULL
			  AND o.%s != ''
			  %s
		`, tableName, talentCollate, orderNoCol, orderNoCol, validFilter)

		countResult, err := g.DB().GetOne(ctx, countSQL, uids, startDate, endDateNext)
		if err != nil {
			continue
		}
		tableCount := countResult["cnt"].Int()
		totalCount += tableCount

		if tableCount == 0 {
			continue
		}

		// 查询该平台的汇总
		sumSQL := fmt.Sprintf(`
			SELECT
				COALESCE(SUM(o.order_pay_amount + COALESCE(o.platform_discount, 0) + COALESCE(o.talent_discount, 0)), 0) AS pay_amount,
				COALESCE(SUM(COALESCE(g.base_price, 0) * o.product_num), 0) AS cost_amount,
				COALESCE(SUM(COALESCE(o.product_price, 0) * o.product_num * 0.05), 0) AS reserve_amount
			FROM %s o
			LEFT JOIN goods g ON o.seller_code %s = g.goods_code
			WHERE o.talent_id %s IN(?)
			  AND o.pay_finish_time >= ?
			  AND o.pay_finish_time < ?
			  AND o.%s IS NOT NULL
			  AND o.%s != ''
			  %s
		`, tableName, collateClause, talentCollate, orderNoCol, orderNoCol, validFilter)

		sumResult, err := g.DB().GetOne(ctx, sumSQL, uids, startDate, endDateNext)
		if err == nil {
			totalPay += sumResult["pay_amount"].Float64()
			totalCost += sumResult["cost_amount"].Float64()
			totalReserve += sumResult["reserve_amount"].Float64()
		}

		// 查询该平台的订单明细（带分页）
		sql := fmt.Sprintf(`
			SELECT
				o.%s AS main_order_no,
				o.product_name,
				o.product_num,
				o.product_price,
				COALESCE(g.base_price, 0) AS base_price,
				o.order_pay_amount,
				COALESCE(o.platform_discount, 0) AS platform_discount,
				COALESCE(o.talent_discount, 0) AS talent_discount,
				o.order_status,
				o.after_sale_status,
				IFNULL(o.pay_finish_time, '') AS pay_finish_time,
				o.talent_id
			FROM %s o
			LEFT JOIN goods g ON o.seller_code %s = g.goods_code
			WHERE o.talent_id %s IN(?)
			  AND o.pay_finish_time >= ?
			  AND o.pay_finish_time < ?
			  AND o.%s IS NOT NULL
			  AND o.%s != ''
			  %s
			ORDER BY o.pay_finish_time DESC
			LIMIT ?, ?
		`, orderNoCol, tableName, collateClause, talentCollate, orderNoCol, orderNoCol, validFilter)

		result, err := g.DB().GetAll(ctx, sql, uids, startDate, endDateNext, (page-1)*pageSize, pageSize)
		if err != nil {
			continue
		}

		for _, row := range result {
			talentId := row["talent_id"].String()
			shop := uidShopMap[talentId]
			productNum := row["product_num"].Int()
			productPrice := row["product_price"].Float64()
			payAmount := row["order_pay_amount"].Float64() + row["platform_discount"].Float64() + row["talent_discount"].Float64()
			basePrice := row["base_price"].Float64()
			costAmount := basePrice * float64(productNum)
			reserveAmount := roundFloatFinance(productPrice*float64(productNum)*0.05, 2)
			commission := roundFloatFinance(payAmount-costAmount-reserveAmount, 2)

			payTime := row["pay_finish_time"].String()
			if len(payTime) >= 10 {
				payTime = payTime[:10]
			}

			allOrders = append(allOrders, model.FinanceOrderItem{
				MainOrderNo:     row["main_order_no"].String(),
				ProductName:     row["product_name"].String(),
				ProductNum:      row["product_num"].Int(),
				ProductPrice:    basePrice,
				OrderPayAmount:  payAmount,
				CostAmount:      roundFloatFinance(costAmount, 2),
				ReserveAmount:   reserveAmount,
				Commission:      commission,
				Platform:        shop.platform,
				ShopName:        shop.name,
				OrderStatus:     row["order_status"].String(),
				AfterSaleStatus: row["after_sale_status"].String(),
				PayFinishTime:   payTime,
			})
		}
	}

	// 按付款时间倒序排序
	sort.Slice(allOrders, func(i, j int) bool {
		return allOrders[i].PayFinishTime > allOrders[j].PayFinishTime
	})

	// 截取当前页
	if len(allOrders) > pageSize {
		allOrders = allOrders[:pageSize]
	}

	res.Summary = model.FinanceStatsSummary{
		OrderCount:    totalCount,
		PayAmount:     roundFloatFinance(totalPay, 2),
		CostAmount:    roundFloatFinance(totalCost, 2),
		ReserveAmount: roundFloatFinance(totalReserve, 2),
		Commission:    roundFloatFinance(totalPay-totalCost-totalReserve, 2),
	}
	res.Total = totalCount
	res.Orders = allOrders

	return res, nil
}

// roundFloatFinance 四舍五入（财务专用）
func roundFloatFinance(f float64, n int) float64 {
	pow := 1.0
	for i := 0; i < n; i++ {
		pow *= 10
	}
	return float64(int(f*pow+0.5)) / pow
}

// platformNameMap 平台标识→中文名称
var platformNameMap = map[string]string{
	"douyin":    "抖音",
	"kuaishou":  "快手",
	"shipinhao": "视频号",
}

// GetTraderPlatformStats 获取投手各平台统计明细
func (s *FinanceCommissionService) GetTraderPlatformStats(ctx context.Context, req *model.FinanceTraderPlatformStatsReq) (*model.FinanceTraderPlatformStatsRes, error) {
	startDate, endDate := s.ParseTimeRange(req.TimeType, req.StartDate, req.EndDate)
	realTraderId := model.DecodeTraderID(req.TraderId)

	// 获取投手信息
	trader, err := g.DB().Model("trader").Where("id", realTraderId).One()
	if err != nil || trader.IsEmpty() {
		return nil, fmt.Errorf("投手不存在")
	}

	// 获取投手关联的店铺
	shops, err := g.DB().Model("shop s").
		LeftJoin("trader_shop_relation tsr", "s.id = tsr.shop_id").
		Where("tsr.trader_id", realTraderId).
		Fields("s.id, s.uid, s.platform, s.account_name").
		All()
	if err != nil {
		return nil, fmt.Errorf("查询投手店铺失败: %v", err)
	}

	// 按 platform 分组 uid（uid 为空时自动匹配）
	platformUids := s.buildPlatformUids(ctx, shops)

	// 按平台统计
	totalStats := &model.FinanceStatsSummary{}
	var platformStats []model.PlatformStats
	for platform, uids := range platformUids {
		tableName, ok := platformTableMap[platform]
		if !ok {
			continue
		}
		stats, _ := s.statsOrderTable(ctx, tableName, uids, startDate, endDate)
		if stats != nil {
			totalStats.OrderCount += stats.OrderCount
			totalStats.ProductNum += stats.ProductNum
			totalStats.PayAmount += stats.PayAmount
			totalStats.CostAmount += stats.CostAmount
			totalStats.ReserveAmount += stats.ReserveAmount
			totalStats.Commission += stats.Commission

			platformStats = append(platformStats, model.PlatformStats{
				Platform:      platform,
				PlatformName:  platformNameMap[platform],
				OrderCount:    stats.OrderCount,
				ProductNum:    stats.ProductNum,
				PayAmount:     roundFloatFinance(stats.PayAmount, 2),
				CostAmount:    roundFloatFinance(stats.CostAmount, 2),
				ReserveAmount: roundFloatFinance(stats.ReserveAmount, 2),
				Commission:    roundFloatFinance(stats.Commission, 2),
			})
		}
	}

	// 按订单数降序排序
	sort.Slice(platformStats, func(i, j int) bool {
		return platformStats[i].OrderCount > platformStats[j].OrderCount
	})

	return &model.FinanceTraderPlatformStatsRes{
		TraderId:      req.TraderId,
		TraderName:    trader["name"].String(),
		TimeType:      req.TimeType,
		StartDate:     startDate,
		EndDate:       endDate,
		Summary:       *totalStats,
		PlatformStats: platformStats,
	}, nil
}

// ExportTraderOrders 导出投手订单明细（按平台分Sheet）
func (s *FinanceCommissionService) ExportTraderOrders(ctx context.Context, traderId int64, timeType, startDate, endDate string) (string, error) {
	startDate, endDate = s.ParseTimeRange(timeType, startDate, endDate)
	realTraderId := model.DecodeTraderID(traderId)

	// 获取投手信息
	trader, err := g.DB().Model("trader").Where("id", realTraderId).One()
	if err != nil || trader.IsEmpty() {
		return "", fmt.Errorf("投手不存在")
	}

	// 获取投手关联的店铺
	shops, err := g.DB().Model("shop s").
		LeftJoin("trader_shop_relation tsr", "s.id = tsr.shop_id").
		Where("tsr.trader_id", realTraderId).
		Fields("s.id, s.uid, s.account_name, s.platform_nickname, s.platform").
		All()
	if err != nil {
		return "", fmt.Errorf("查询投手店铺失败: %v", err)
	}

	// 按 platform 分组 uid，同时构建 uid→店铺名 映射
	platformUids := s.buildPlatformUids(ctx, shops)
	uidShopMap := make(map[string]string)
	for _, shop := range shops {
		uid := shop["uid"].String()
		if uid == "" {
			continue
		}
		name := shop["account_name"].String()
		if shop["platform_nickname"].String() != "" {
			name = shop["platform_nickname"].String()
		}
		uidShopMap[uid] = name
	}
	// 为通过 account_name 匹配到的 uid 也建立映射
	for _, uids := range platformUids {
		for _, uid := range uids {
			if _, exists := uidShopMap[uid]; !exists {
				uidShopMap[uid] = uid
			}
		}
	}

	if len(platformUids) == 0 {
		return "", fmt.Errorf("投手无关联店铺")
	}

	// 日期范围
	endDateNext := CalcEndDateNext(endDate)

	validFilter := "AND o.after_sale_status NOT IN ('退款成功', '已全额退款', '待退货') AND o.order_status != '已关闭'"

	// 创建 Excel
	f := excelize.NewFile()
	defer f.Close()
	firstSheet := true

	headers := []string{"订单号", "商品名称", "数量", "单价", "应付金额", "货款", "留底", "佣金", "店铺", "订单状态", "售后状态", "付款时间"}
	colWidths := []float64{22, 40, 8, 12, 14, 14, 14, 14, 20, 12, 12, 14}

	// 表头样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	for platform, uids := range platformUids {
		tableName, ok := platformTableMap[platform]
		if !ok {
			continue
		}

		orderNoCol := "main_order_no"
		collateClause := ""
		talentCollate := ""
		if tableName == "kuaishou_orders" || tableName == "shipinhao_orders" {
			orderNoCol = "order_no"
			collateClause = "COLLATE utf8mb4_unicode_ci"
			talentCollate = "COLLATE utf8mb4_unicode_ci"
		}

		sql := fmt.Sprintf(`
			SELECT
				o.%s AS main_order_no,
				o.product_name,
				o.product_num,
				o.product_price,
				COALESCE(g.base_price, 0) AS base_price,
				o.order_pay_amount,
				COALESCE(o.platform_discount, 0) AS platform_discount,
				COALESCE(o.talent_discount, 0) AS talent_discount,
				o.order_status,
				o.after_sale_status,
				IFNULL(o.pay_finish_time, '') AS pay_finish_time,
				o.talent_id
			FROM %s o
			LEFT JOIN goods g ON o.seller_code %s = g.goods_code
			WHERE o.talent_id %s IN(?)
			  AND o.pay_finish_time >= ?
			  AND o.pay_finish_time < ?
			  AND o.%s IS NOT NULL
			  AND o.%s != ''
			  %s
			ORDER BY o.pay_finish_time DESC
		`, orderNoCol, tableName, collateClause, talentCollate, orderNoCol, orderNoCol, validFilter)

		result, err := g.DB().GetAll(ctx, sql, uids, startDate, endDateNext)
		if err != nil || len(result) == 0 {
			continue
		}

		// 创建 sheet
		sheetName := platformNameMap[platform]
		if sheetName == "" {
			sheetName = platform
		}
		if firstSheet {
			// 重命名默认 sheet
			f.SetSheetName("Sheet1", sheetName)
			firstSheet = false
		} else {
			f.NewSheet(sheetName)
		}

		// 设置列宽
		for i, w := range colWidths {
			col, _ := excelize.ColumnNumberToName(i + 1)
			f.SetColWidth(sheetName, col, col, w)
		}

		// 写表头
		for i, h := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheetName, cell, h)
		}
		f.SetRowStyle(sheetName, 1, 1, headerStyle)

		// 写数据
		for rowIdx, row := range result {
			talentId := row["talent_id"].String()
			productNum := row["product_num"].Int()
			productPrice := row["product_price"].Float64()
			payAmount := row["order_pay_amount"].Float64() + row["platform_discount"].Float64() + row["talent_discount"].Float64()
			basePrice := row["base_price"].Float64()
			costAmount := basePrice * float64(productNum)
			reserveAmount := roundFloatFinance(productPrice*float64(productNum)*0.05, 2)
			commission := roundFloatFinance(payAmount-costAmount-reserveAmount, 2)

			payTime := row["pay_finish_time"].String()
			if len(payTime) >= 10 {
				payTime = payTime[:10]
			}

			dataRow := []interface{}{
				row["main_order_no"].String(),
				row["product_name"].String(),
				productNum,
				basePrice,
				payAmount,
				roundFloatFinance(costAmount, 2),
				reserveAmount,
				commission,
				uidShopMap[talentId],
				row["order_status"].String(),
				row["after_sale_status"].String(),
				payTime,
			}
			for colIdx, val := range dataRow {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
				f.SetCellValue(sheetName, cell, val)
			}
		}
	}

	// 保存文件
	exportDir := "./exports"
	os.MkdirAll(exportDir, 0755)
	fileName := fmt.Sprintf("投手订单_%s_%s_%s.xlsx", trader["name"].String(), startDate, endDate)
	filePath := filepath.Join(exportDir, fileName)
	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("保存Excel失败: %v", err)
	}

	return filePath, nil
}
