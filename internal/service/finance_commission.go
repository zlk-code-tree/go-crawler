package service

import (
	"context"
	"fmt"
	"go-crawler/internal/model"
	"time"

	"github.com/gogf/gf/v2/frame/g"
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

// ============================================
// 统计查询（核心方法）
// ============================================

// GetStats 获取佣金统计
func (s *FinanceCommissionService) GetStats(ctx context.Context, req *model.FinanceStatsReq) (*model.FinanceStatsRes, error) {
	startDate, endDate := s.ParseTimeRange(req.TimeType, req.StartDate, req.EndDate)
	today := time.Now().Format("2006-01-02")

	// 情况1：查询历史数据（直接查预聚合表）
	if endDate < today {
		return s.getStatsFromTable(ctx, req.NodeId, req.NodeType, startDate, endDate, req.TimeType)
	}

	// 情况2：查询今日数据（实时计算）
	if startDate >= today {
		return s.getStatsRealtime(ctx, req.NodeId, req.NodeType, startDate, endDate, req.TimeType)
	}

	// 情况3：跨今天（历史 + 今日）
	historyStats, _ := s.getStatsFromTable(ctx, req.NodeId, req.NodeType, startDate, today, req.TimeType)
	todayStats, _ := s.getStatsRealtime(ctx, req.NodeId, req.NodeType, today, endDate, req.TimeType)

	return s.mergeStats(historyStats, todayStats, req.TimeType, startDate, endDate), nil
}

// getStatsFromTable 从预聚合表查询
func (s *FinanceCommissionService) getStatsFromTable(ctx context.Context, nodeId int64, nodeType, startDate, endDate, timeType string) (*model.FinanceStatsRes, error) {
	query := g.DB().Model("finance_daily_stats").
		Where("stat_date >= ?", startDate).
		Where("stat_date <= ?", endDate).
		Where("platform = ''") // 只查汇总数据

	// 根据节点类型筛选（需要解码ID）
	switch nodeType {
	case "company":
		query = query.Where("company_id = ? AND department_id = 0 AND trader_id = 0 AND shop_id = 0", nodeId)
	case "department":
		realDeptId := model.DecodeDeptID(nodeId)
		query = query.Where("department_id = ? AND trader_id = 0 AND shop_id = 0", realDeptId)
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

	return &model.FinanceStatsRes{
		TimeType:  timeType,
		StartDate: startDate,
		EndDate:   endDate,
		Summary: model.FinanceStatsSummary{
			OrderCount:    result["order_count"].Int(),
			ProductNum:    result["product_num"].Int(),
			PayAmount:     result["pay_amount"].Float64(),
			CostAmount:    result["cost_amount"].Float64(),
			ReserveAmount: result["reserve_amount"].Float64(),
			Commission:    result["commission"].Float64(),
		},
	}, nil
}

// getStatsRealtime 实时计算统计
func (s *FinanceCommissionService) getStatsRealtime(ctx context.Context, nodeId int64, nodeType, startDate, endDate, timeType string) (*model.FinanceStatsRes, error) {
	// 1. 获取该节点下的所有店铺ID
	shopIds, err := s.getShopIdsByNode(ctx, nodeId, nodeType)
	if err != nil {
		return nil, err
	}

	if len(shopIds) == 0 {
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
		return nil, err
	}

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
		return s.getCompanyChildren(ctx, req.NodeId, startDate, endDate, req.TimeType)
	case "department":
		return s.getDepartmentChildren(ctx, req.NodeId, startDate, endDate, req.TimeType)
	case "trader":
		return s.getTraderChildren(ctx, req.NodeId, startDate, endDate, req.TimeType)
	}

	return nil, fmt.Errorf("不支持的节点类型: %s", req.NodeType)
}

// getCompanyChildren 获取公司下级节点
func (s *FinanceCommissionService) getCompanyChildren(ctx context.Context, companyId int64, startDate, endDate, timeType string) (*model.FinanceChildrenRes, error) {
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
		Where("parent_id = companyId").
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
				NodeId:    deptId,
				NodeType:  "department",
				StartDate: startDate,
				EndDate:   endDate,
				TimeType:  "custom",
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
func (s *FinanceCommissionService) getDepartmentChildren(ctx context.Context, deptId int64, startDate, endDate, timeType string) (*model.FinanceChildrenRes, error) {
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
				NodeId:    sdId,
				NodeType:  "department",
				StartDate: startDate,
				EndDate:   endDate,
				TimeType:  "custom",
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
				NodeId:    traderId,
				NodeType:  "trader",
				StartDate: startDate,
				EndDate:   endDate,
				TimeType:  "custom",
			})
			children = append(children, model.NodeStatsItem{
				Id:         model.EncodeTraderID(traderId), // 返回编码后的ID
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
func (s *FinanceCommissionService) getTraderChildren(ctx context.Context, traderId int64, startDate, endDate, timeType string) (*model.FinanceChildrenRes, error) {
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
	shops, err := g.DB().Model("shop s").
		LeftJoin("trader_shop_relation tsr", "s.id = tsr.shop_id").
		Where("tsr.trader_id", realTraderId).
		Where("s.status", 1).
		Fields("s.id, s.platform, s.uid, s.account_name, s.platform_nickname").
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
		Where("s.status", 1).
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

// getShopIdsByNode 根据组织节点获取店铺ID列表
func (s *FinanceCommissionService) getShopIdsByNode(ctx context.Context, nodeId int64, nodeType string) ([]int, error) {
	switch nodeType {
	case "company":
		// 获取该公司及所有子公司的ID
		companyIds := s.getCompanyAndChildrenIds(ctx, nodeId)
		// 查询这些公司下的店铺
		var shopIds []int
		err := g.DB().Model("shop").
			Where("company_id IN(?)", companyIds).
			Where("status", 1).
			Fields("id").
			Scan(&shopIds)
		if err != nil {
			return nil, err
		}
		return shopIds, nil

	case "department":
		// 解码部门ID
		realDeptId := model.DecodeDeptID(nodeId)
		// 获取该部门及所有子部门的ID
		deptIds := s.getDeptAndChildrenIds(ctx, realDeptId)
		// 通过 trader 关联获取店铺：部门 → trader(group_id) → trader_shop_relation → shop
		var traderIds []int64
		err := g.DB().Model("trader").
			Where("group_id IN(?)", deptIds).
			Where("status", 1).
			Fields("id").
			Scan(&traderIds)
		if err != nil {
			return nil, err
		}
		if len(traderIds) == 0 {
			return nil, nil
		}
		var shopIds []int
		err = g.DB().Model("trader_shop_relation").
			Where("trader_id IN(?)", traderIds).
			Fields("shop_id").
			Scan(&shopIds)
		if err != nil {
			return nil, err
		}
		return shopIds, nil

	case "trader":
		// 解码投手ID
		realTraderId := model.DecodeTraderID(nodeId)
		// 查询投手关联的店铺
		var shopIds []int
		err := g.DB().Model("trader_shop_relation").
			Where("trader_id", realTraderId).
			Fields("shop_id").
			Scan(&shopIds)
		if err != nil {
			return nil, err
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

// statsOrdersByShopIds 统计店铺订单数据（跨三张订单表）
func (s *FinanceCommissionService) statsOrdersByShopIds(ctx context.Context, shopIds []int, startDate, endDate string) (*model.FinanceStatsSummary, error) {
	if len(shopIds) == 0 {
		return &model.FinanceStatsSummary{}, nil
	}

	// 获取店铺的 uid 列表
	shops, err := g.DB().Model("shop").
		Where("id IN(?)", shopIds).
		Fields("uid").
		All()
	if err != nil {
		return nil, err
	}

	var uids []string
	for _, shop := range shops {
		uid := shop["uid"].String()
		if uid != "" {
			uids = append(uids, uid)
		}
	}

	if len(uids) == 0 {
		return &model.FinanceStatsSummary{}, nil
	}

	// 统计三张订单表
	totalStats := &model.FinanceStatsSummary{}

	// 抖音订单
	douyinStats, _ := s.statsOrderTable(ctx, "order_info", uids, startDate, endDate)
	if douyinStats != nil {
		totalStats.OrderCount += douyinStats.OrderCount
		totalStats.ProductNum += douyinStats.ProductNum
		totalStats.PayAmount += douyinStats.PayAmount
		totalStats.CostAmount += douyinStats.CostAmount
		totalStats.ReserveAmount += douyinStats.ReserveAmount
		totalStats.Commission += douyinStats.Commission
	}

	// 快手订单
	kuaishouStats, _ := s.statsOrderTable(ctx, "kuaishou_orders", uids, startDate, endDate)
	if kuaishouStats != nil {
		totalStats.OrderCount += kuaishouStats.OrderCount
		totalStats.ProductNum += kuaishouStats.ProductNum
		totalStats.PayAmount += kuaishouStats.PayAmount
		totalStats.CostAmount += kuaishouStats.CostAmount
		totalStats.ReserveAmount += kuaishouStats.ReserveAmount
		totalStats.Commission += kuaishouStats.Commission
	}

	// 视频号订单
	shipinhaoStats, _ := s.statsOrderTable(ctx, "shipinhao_orders", uids, startDate, endDate)
	if shipinhaoStats != nil {
		totalStats.OrderCount += shipinhaoStats.OrderCount
		totalStats.ProductNum += shipinhaoStats.ProductNum
		totalStats.PayAmount += shipinhaoStats.PayAmount
		totalStats.CostAmount += shipinhaoStats.CostAmount
		totalStats.ReserveAmount += shipinhaoStats.ReserveAmount
		totalStats.Commission += shipinhaoStats.Commission
	}

	return totalStats, nil
}

// statsOrderTable 统计单张订单表
func (s *FinanceCommissionService) statsOrderTable(ctx context.Context, tableName string, uids []string, startDate, endDate string) (*model.FinanceStatsSummary, error) {
	// 构建有效订单过滤条件
	validFilter := "AND after_sale_status NOT IN ('退款成功', '已全额退款', '待退货') AND order_status != '已关闭'"

	sql := fmt.Sprintf(`
		SELECT
			COUNT(DISTINCT main_order_no) AS order_count,
			COALESCE(SUM(product_num), 0) AS product_num,
			COALESCE(SUM(order_pay_amount), 0) AS pay_amount,
			COALESCE(SUM(product_price * product_num), 0) AS cost_amount
		FROM %s
		WHERE talent_id IN(?)
		  AND pay_finish_time >= ?
		  AND pay_finish_time < ?
		  AND main_order_no IS NOT NULL
		  AND main_order_no != ''
		  %s
	`, tableName, validFilter)

	endDateNext := endDate + " 00:00:00"
	if t, err := time.Parse("2006-01-02", endDate); err == nil {
		endDateNext = t.Add(24 * time.Hour).Format("2006-01-02")
	}

	result, err := g.DB().GetOne(ctx, sql, uids, startDate, endDateNext)
	if err != nil {
		return nil, err
	}

	payAmount := result["pay_amount"].Float64()
	costAmount := result["cost_amount"].Float64()
	reserveAmount := payAmount * 0.05 // 留底5%
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


// roundFloatFinance 四舍五入（财务专用）
func roundFloatFinance(f float64, n int) float64 {
	pow := 1.0
	for i := 0; i < n; i++ {
		pow *= 10
	}
	return float64(int(f*pow+0.5)) / pow
}
