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
		// 带时间：取整到天，然后加1天
		if t, err := time.Parse("2006-01-02 15:04:05", endDate); err == nil {
			// 取整到当天 00:00:00，然后加1天
			dayStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			return dayStart.Add(24 * time.Hour).Format("2006-01-02")
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

	// 全部使用实时计算
	return s.getStatsRealtime(ctx, req.NodeId, req.NodeType, startDate, endDate, req.TimeType, platform)
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
	// 只查 account_id > 0 的行（账号维度），避免公司/部门汇总行重复计算
	switch nodeType {
	case "company":
		query = query.Where("company_id = ? AND account_id > 0", nodeId)
	case "department":
		realDeptId := model.DecodeDeptID(nodeId)
		query = query.Where("department_id = ? AND account_id > 0", realDeptId)
	case "trader":
		realTraderId := model.DecodeTraderID(nodeId)
		query = query.Where("trader_id = ? AND account_id = 0", realTraderId)
	}

	// 聚合统计
	result, err := query.Fields(`
		COALESCE(SUM(total_order_count), 0) as total_order_count,
		COALESCE(SUM(order_count), 0) as order_count,
		COALESCE(SUM(invalid_order_count), 0) as invalid_order_count,
		COALESCE(SUM(product_num), 0) as product_num,
		COALESCE(SUM(pay_amount), 0) as pay_amount,
		COALESCE(SUM(cost_amount), 0) as cost_amount,
		COALESCE(SUM(reserve_amount), 0) as reserve_amount,
		COALESCE(SUM(commission), 0) as commission,
		COALESCE(SUM(refund_cost_amount), 0) as refund_cost_amount,
		COALESCE(SUM(refund_order_count), 0) as refund_order_count,
		COALESCE(SUM(shop_service_fee), 0) as shop_service_fee,
		COALESCE(SUM(ad_cost_amount), 0) as ad_cost_amount,
		COALESCE(SUM(valid_order_cost_match), 0) as valid_order_cost_match,
		COALESCE(SUM(commission_order_count), 0) as commission_order_count,
		COALESCE(SUM(realtime_profit), 0) as realtime_profit
	`).One()

	if err != nil {
		return nil, fmt.Errorf("查询预聚合数据失败: %v", err)
	}

	summary := model.FinanceStatsSummary{
		TotalOrderCount:      result["total_order_count"].Int(),
		OrderCount:           result["order_count"].Int(),
		InvalidOrderCount:    result["invalid_order_count"].Int(),
		ProductNum:           result["product_num"].Int(),
		PayAmount:            result["pay_amount"].Float64(),
		CostAmount:           result["cost_amount"].Float64(),
		ReserveAmount:        result["reserve_amount"].Float64(),
		Commission:           result["commission"].Float64(),
		RefundCostAmount:     result["refund_cost_amount"].Float64(),
		RefundOrderCount:     result["refund_order_count"].Int(),
		ShopServiceFee:       result["shop_service_fee"].Float64(),
		AdCostAmount:         result["ad_cost_amount"].Float64(),
		ValidOrderCostMatch:  result["valid_order_cost_match"].Int(),
		CommissionOrderCount: result["commission_order_count"].Int(),
		RealtimeProfit:       result["realtime_profit"].Float64(),
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
	// 1. 获取该节点下的所有账号ID
	accountIds, err := s.getAccountIdsByNode(ctx, nodeId, nodeType, platform)
	if err != nil {
		g.Log().Warningf(ctx, "[Finance] getAccountIdsByNode 失败: nodeType=%s, nodeId=%d, err=%v", nodeType, nodeId, err)
		return nil, err
	}

	g.Log().Infof(ctx, "[Finance] 实时统计: nodeType=%s, nodeId=%d, accountIds=%v, date=%s~%s", nodeType, nodeId, accountIds, startDate, endDate)

	if len(accountIds) == 0 {
		g.Log().Warningf(ctx, "[Finance] 节点下无账号: nodeType=%s, nodeId=%d", nodeType, nodeId)
		return &model.FinanceStatsRes{
			TimeType:  timeType,
			StartDate: startDate,
			EndDate:   endDate,
			Summary:   model.FinanceStatsSummary{},
		}, nil
	}

	// 2. 统计这些账号的订单数据（跨三张订单表）
	// 注意：getStatsRealtime 不传 traderName，归属规则在投手级别单独处理
	stats, err := s.statsOrdersByAccountIds(ctx, accountIds, startDate, endDate, "")
	if err != nil {
		g.Log().Warningf(ctx, "[Finance] statsOrdersByAccountIds 失败: accountIds=%v, err=%v", accountIds, err)
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

	hs := history.Summary
	rs := realtime.Summary

	summary := model.FinanceStatsSummary{
		TotalOrderCount:      hs.TotalOrderCount + rs.TotalOrderCount,
		OrderCount:           hs.OrderCount + rs.OrderCount,
		InvalidOrderCount:    hs.InvalidOrderCount + rs.InvalidOrderCount,
		ProductNum:           hs.ProductNum + rs.ProductNum,
		PayAmount:            roundFloatFinance(hs.PayAmount+rs.PayAmount, 2),
		CostAmount:           roundFloatFinance(hs.CostAmount+rs.CostAmount, 2),
		ReserveAmount:        roundFloatFinance(hs.ReserveAmount+rs.ReserveAmount, 2),
		Commission:           roundFloatFinance(hs.Commission+rs.Commission, 2),
		RefundCostAmount:     roundFloatFinance(hs.RefundCostAmount+rs.RefundCostAmount, 2),
		RefundOrderCount:     hs.RefundOrderCount + rs.RefundOrderCount,
		ShopServiceFee:       roundFloatFinance(hs.ShopServiceFee+rs.ShopServiceFee, 2),
		AdCostAmount:         hs.AdCostAmount + rs.AdCostAmount,
		ValidOrderCostMatch:  hs.ValidOrderCostMatch + rs.ValidOrderCostMatch,
		CommissionOrderCount: hs.CommissionOrderCount + rs.CommissionOrderCount,
		RealtimeProfit:       roundFloatFinance(hs.RealtimeProfit+rs.RealtimeProfit, 2),
	}

	return &model.FinanceStatsRes{
		TimeType:  timeType,
		StartDate: startDate,
		EndDate:   endDate,
		Summary:   summary,
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

// getCompanyChildren 获取公司下级节点（批量查询优化）
func (s *FinanceCommissionService) getCompanyChildren(ctx context.Context, companyId int64, startDate, endDate, timeType, platform string) (*model.FinanceChildrenRes, error) {
	// 子节点只查历史数据，避免每个子节点都实时查询今天数据
	childEndDate := endDate
	today := time.Now().Format("2006-01-02")
	if endDate >= today {
		childEndDate = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		if childEndDate < startDate {
			childEndDate = startDate
		}
	}
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

	// 1. 获取子公司
	subCompanies, _ := g.DB().Model("sys_company").
		Where("parent_id", companyId).
		Where("status", 1).
		Order("sort ASC, id ASC").
		All()

	// 2. 获取该公司下的顶级部门
	departments, _ := g.DB().Model("sys_department").
		Where("company_id", companyId).
		Where("parent_id", 0).
		Where("status", 1).
		Order("sort ASC, id ASC").
		All()

	// 3. 批量获取每个子节点的账号ID
	type childNode struct {
		id         int64
		name       string
		nodeType   string
		accountIds []int
	}
	var childNodes []childNode

	for _, sc := range subCompanies {
		cid := sc["id"].Int64()
		accIds, _ := s.getAccountIdsByNode(ctx, cid, "company", platform)
		childNodes = append(childNodes, childNode{id: cid, name: sc["name"].String(), nodeType: "company", accountIds: accIds})
	}
	for _, dept := range departments {
		did := dept["id"].Int64()
		accIds, _ := s.getAccountIdsByNode(ctx, model.EncodeDeptID(did), "department", platform)
		childNodes = append(childNodes, childNode{id: model.EncodeDeptID(did), name: dept["name"].String(), nodeType: "department", accountIds: accIds})
	}

	// 4. 批量查询：按平台聚合所有账号，一次查询每张订单表
	//    同时记录每个 accountId 属于哪些子节点，用于分配结果
	type nodeIdx struct{ idx int }
	accountToNodes := make(map[int][]nodeIdx) // accountId → 所属子节点索引
	// 收集所有不重复的 accountId 及其 platform 信息
	allAccountIdsSet := make(map[int]bool)
	for i, cn := range childNodes {
		for _, aid := range cn.accountIds {
			if !allAccountIdsSet[aid] {
				allAccountIdsSet[aid] = true
			}
			accountToNodes[aid] = append(accountToNodes[aid], nodeIdx{idx: i})
		}
	}

	// 初始化每个子节点的统计结果
	nodeStats := make([]model.FinanceStatsSummary, len(childNodes))

	if len(allAccountIdsSet) > 0 {
		// 收集所有 accountId
		var allAccountIds []int
		for aid := range allAccountIdsSet {
			allAccountIds = append(allAccountIds, aid)
		}

		// 获取账号的 uid、platform 信息
		shops, err := g.DB().Model("account").
			Where("id IN(?)", allAccountIds).
			Fields("id, uid, platform, account_name").
			All()
		if err != nil {
			return nil, err
		}

		// 构建 platform → uids
		platformUids := make(map[string][]string)
		for _, shop := range shops {
			uid := shop["uid"].String()
			p := shop["platform"].String()
			if p != "" && uid != "" {
				platformUids[p] = append(platformUids[p], uid)
			}
		}

		// 按平台分别查询订单表，获取每个 uid 的统计数据
		for platformKey, uids := range platformUids {
			tableName, ok := platformTableMap[platformKey]
			if !ok {
				continue
			}
			orderNoCol := "main_order_no"
			collate := ""
			talentCollate := ""
			if tableName == "kuaishou_orders" || tableName == "shipinhao_orders" {
				orderNoCol = "order_no"
				collate = "COLLATE utf8mb4_unicode_ci"
				talentCollate = "COLLATE utf8mb4_unicode_ci"
			}

			validCond := "(o.order_status IN ('已发货','待发货','已完成') AND o.after_sale_status NOT IN ('6','7','11','12','29')) OR (o.order_status = '已关闭' AND o.after_sale_status IN ('27','28','51'))"
			invalidCond := "o.after_sale_status IN ('6','7','11','12','29')"
			refundCond := "o.after_sale_status IN ('6','7','11','12','29') AND o.order_status = '已发货'"

			// 按 talent_id 分组统计（一次查询获取所有 uid 的统计）
			sql := fmt.Sprintf(`
				SELECT
					o.talent_id AS uid,
					COUNT(DISTINCT o.%s) AS total_order_count,
					COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS order_count,
					COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS invalid_order_count,
					COALESCE(SUM(CASE WHEN %s THEN o.product_num ELSE 0 END), 0) AS product_num,
					COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num ELSE 0 END), 0) AS pay_amount,
					COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS cost_amount,
					COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num * 0.03 ELSE 0 END), 0) AS reserve_amount,
					COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS refund_cost_amount,
					COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS refund_order_count
				FROM %s o
				LEFT JOIN goods g ON o.seller_code %s = g.goods_code %s
				WHERE o.order_submit_time >= ?
				  AND o.order_submit_time < ?
				  AND o.%s IS NOT NULL
				  AND o.%s != ''
				  AND o.talent_id %s IN(?)
				GROUP BY o.talent_id
			`, orderNoCol, validCond, orderNoCol, invalidCond, orderNoCol, validCond, validCond, validCond, validCond, refundCond, refundCond, orderNoCol, tableName, collate, collate, orderNoCol, orderNoCol, talentCollate)

			endDateNext := CalcEndDateNext(childEndDate)
			rows, err := g.DB().GetAll(ctx, sql, startDate, endDateNext, uids)
			if err != nil {
				continue
			}

			// 构建 uid → accountId 列表的映射
			uidToAccountIds := make(map[string][]int)
			for _, shop := range shops {
				uid := shop["uid"].String()
				if uid != "" {
					uidToAccountIds[uid] = append(uidToAccountIds[uid], shop["id"].Int())
				}
			}

			for _, row := range rows {
				uid := row["uid"].String()
				payAmount := row["pay_amount"].Float64()
				costAmount := row["cost_amount"].Float64()
				reserveAmount := row["reserve_amount"].Float64()
				commission := payAmount - costAmount - reserveAmount

				// 小店服务费 = 定价总额 × 2%（仅抖音）
				shopServiceFee := 0.0
				if platformKey == "douyin" {
					shopServiceFee = roundFloatFinance(payAmount*0.02, 2)
				}
				commission = roundFloatFinance(commission-shopServiceFee, 2)
				commissionDeduction := roundFloatFinance(payAmount*0.005, 2)
				realtimeProfit := roundFloatFinance(payAmount-costAmount-reserveAmount-row["refund_cost_amount"].Float64()-shopServiceFee-commissionDeduction, 2)

				stats := model.FinanceStatsSummary{
					TotalOrderCount:     row["total_order_count"].Int(),
					OrderCount:          row["order_count"].Int(),
					InvalidOrderCount:   row["invalid_order_count"].Int(),
					ProductNum:          row["product_num"].Int(),
					PayAmount:           roundFloatFinance(payAmount, 2),
					CostAmount:          roundFloatFinance(costAmount, 2),
					ReserveAmount:       roundFloatFinance(reserveAmount, 2),
					Commission:          commission,
					RefundCostAmount:    roundFloatFinance(row["refund_cost_amount"].Float64(), 2),
					RefundOrderCount:    row["refund_order_count"].Int(),
					ShopServiceFee:      shopServiceFee,
					RealtimeProfit:      realtimeProfit,
				}

				// 将该 uid 的统计数据分配给所有关联的子节点
				for _, aid := range uidToAccountIds[uid] {
					for _, ni := range accountToNodes[aid] {
						nodeStats[ni.idx].TotalOrderCount += stats.TotalOrderCount
						nodeStats[ni.idx].OrderCount += stats.OrderCount
						nodeStats[ni.idx].InvalidOrderCount += stats.InvalidOrderCount
						nodeStats[ni.idx].ProductNum += stats.ProductNum
						nodeStats[ni.idx].PayAmount += stats.PayAmount
						nodeStats[ni.idx].CostAmount += stats.CostAmount
						nodeStats[ni.idx].ReserveAmount += stats.ReserveAmount
						nodeStats[ni.idx].Commission += stats.Commission
						nodeStats[ni.idx].RefundCostAmount += stats.RefundCostAmount
						nodeStats[ni.idx].RefundOrderCount += stats.RefundOrderCount
						nodeStats[ni.idx].ShopServiceFee += stats.ShopServiceFee
						nodeStats[ni.idx].RealtimeProfit += stats.RealtimeProfit
					}
				}
			}
		}
	}

	// 5. 构建结果
	var children []model.NodeStatsItem
	for i, cn := range childNodes {
		st := nodeStats[i]
		st.PayAmount = roundFloatFinance(st.PayAmount, 2)
		st.CostAmount = roundFloatFinance(st.CostAmount, 2)
		st.ReserveAmount = roundFloatFinance(st.ReserveAmount, 2)
		st.Commission = roundFloatFinance(st.Commission, 2)
		st.RefundCostAmount = roundFloatFinance(st.RefundCostAmount, 2)
		st.ShopServiceFee = roundFloatFinance(st.ShopServiceFee, 2)
		st.RealtimeProfit = roundFloatFinance(st.RealtimeProfit, 2)
		children = append(children, model.NodeStatsItem{
			Id:               cn.id,
			Name:             cn.name,
			Type:             cn.nodeType,
			TotalOrderCount:  st.TotalOrderCount,
			OrderCount:       st.OrderCount,
			InvalidOrderCount: st.InvalidOrderCount,
			ProductNum:       st.ProductNum,
			PayAmount:        st.PayAmount,
			CostAmount:       st.CostAmount,
			Commission:       st.Commission,
			RefundCostAmount: st.RefundCostAmount,
			RefundOrderCount: st.RefundOrderCount,
			ShopServiceFee:   st.ShopServiceFee,
			RealtimeProfit:   st.RealtimeProfit,
		})
	}

	return &model.FinanceChildrenRes{
		CurrentNode: currentNode,
		Children:    children,
	}, nil
}

// getDepartmentChildren 获取部门下级节点（批量查询优化）
func (s *FinanceCommissionService) getDepartmentChildren(ctx context.Context, deptId int64, startDate, endDate, timeType, platform string) (*model.FinanceChildrenRes, error) {
	// 子节点只查历史数据，避免每个子节点都实时查询今天数据
	childEndDate := endDate
	today := time.Now().Format("2006-01-02")
	if endDate >= today {
		childEndDate = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		if childEndDate < startDate {
			childEndDate = startDate
		}
	}
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

	// 1. 获取子部门
	subDepts, _ := g.DB().Model("sys_department").
		Where("parent_id", realDeptId).
		Where("status", 1).
		Order("sort ASC, id ASC").
		All()

	// 2. 获取该部门下的投手
	traders, _ := g.DB().Model("trader").
		Where("group_id", realDeptId).
		Where("status", 1).
		Order("id ASC").
		All()

	// 3. 批量获取每个子节点的账号ID
	type childNode struct {
		id         int64
		name       string
		nodeType   string
		accountIds []int
	}
	var childNodes []childNode

	for _, sd := range subDepts {
		sdId := sd["id"].Int64()
		accIds, _ := s.getAccountIdsByNode(ctx, model.EncodeDeptID(sdId), "department", platform)
		childNodes = append(childNodes, childNode{id: model.EncodeDeptID(sdId), name: sd["name"].String(), nodeType: "department", accountIds: accIds})
	}
	for _, trader := range traders {
		tid := trader["id"].Int64()
		accIds, _ := s.getAccountIdsByNode(ctx, model.EncodeTraderID(tid), "trader", platform)
		childNodes = append(childNodes, childNode{id: model.EncodeTraderID(tid), name: trader["name"].String(), nodeType: "trader", accountIds: accIds})
	}

	// 4. 批量查询：按平台聚合所有账号，一次查询每张订单表
	type nodeIdx struct{ idx int }
	accountToNodes := make(map[int][]nodeIdx)
	allAccountIdsSet := make(map[int]bool)
	for i, cn := range childNodes {
		for _, aid := range cn.accountIds {
			allAccountIdsSet[aid] = true
			accountToNodes[aid] = append(accountToNodes[aid], nodeIdx{idx: i})
		}
	}

	nodeStats := make([]model.FinanceStatsSummary, len(childNodes))

	if len(allAccountIdsSet) > 0 {
		var allAccountIds []int
		for aid := range allAccountIdsSet {
			allAccountIds = append(allAccountIds, aid)
		}

		shops, err := g.DB().Model("account").
			Where("id IN(?)", allAccountIds).
			Fields("id, uid, platform, account_name").
			All()
		if err != nil {
			return nil, err
		}

		platformUids := make(map[string][]string)
		for _, shop := range shops {
			uid := shop["uid"].String()
			p := shop["platform"].String()
			if p != "" && uid != "" {
				platformUids[p] = append(platformUids[p], uid)
			}
		}

		// 按平台分别查询订单表，获取每个 uid 的统计数据
		for platformKey, uids := range platformUids {
			tableName, ok := platformTableMap[platformKey]
			if !ok {
				continue
			}
			orderNoCol := "main_order_no"
			collate := ""
			talentCollate := ""
			if tableName == "kuaishou_orders" || tableName == "shipinhao_orders" {
				orderNoCol = "order_no"
				collate = "COLLATE utf8mb4_unicode_ci"
				talentCollate = "COLLATE utf8mb4_unicode_ci"
			}

			validCond := "(o.order_status IN ('已发货','待发货','已完成') AND o.after_sale_status NOT IN ('6','7','11','12','29')) OR (o.order_status = '已关闭' AND o.after_sale_status IN ('27','28','51'))"
			invalidCond := "o.after_sale_status IN ('6','7','11','12','29')"
			refundCond := "o.after_sale_status IN ('6','7','11','12','29') AND o.order_status = '已发货'"

			sql := fmt.Sprintf(`
				SELECT
					o.talent_id AS uid,
					COUNT(DISTINCT o.%s) AS total_order_count,
					COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS order_count,
					COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS invalid_order_count,
					COALESCE(SUM(CASE WHEN %s THEN o.product_num ELSE 0 END), 0) AS product_num,
					COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num ELSE 0 END), 0) AS pay_amount,
					COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS cost_amount,
					COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num * 0.03 ELSE 0 END), 0) AS reserve_amount,
					COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS refund_cost_amount,
					COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS refund_order_count
				FROM %s o
				LEFT JOIN goods g ON o.seller_code %s = g.goods_code %s
				WHERE o.order_submit_time >= ?
				  AND o.order_submit_time < ?
				  AND o.%s IS NOT NULL
				  AND o.%s != ''
				  AND o.talent_id %s IN(?)
				GROUP BY o.talent_id
			`, orderNoCol, validCond, orderNoCol, invalidCond, orderNoCol, validCond, validCond, validCond, validCond, refundCond, refundCond, orderNoCol, tableName, collate, collate, orderNoCol, orderNoCol, talentCollate)

			endDateNext := CalcEndDateNext(childEndDate)
			rows, err := g.DB().GetAll(ctx, sql, startDate, endDateNext, uids)
			if err != nil {
				continue
			}

			uidToAccountIds := make(map[string][]int)
			for _, shop := range shops {
				uid := shop["uid"].String()
				if uid != "" {
					uidToAccountIds[uid] = append(uidToAccountIds[uid], shop["id"].Int())
				}
			}

			for _, row := range rows {
				uid := row["uid"].String()
				payAmount := row["pay_amount"].Float64()
				costAmount := row["cost_amount"].Float64()
				reserveAmount := row["reserve_amount"].Float64()
				commission := payAmount - costAmount - reserveAmount

				shopServiceFee := 0.0
				if platformKey == "douyin" {
					shopServiceFee = roundFloatFinance(payAmount*0.02, 2)
				}
				commission = roundFloatFinance(commission-shopServiceFee, 2)
				commissionDeduction := roundFloatFinance(payAmount*0.005, 2)
				realtimeProfit := roundFloatFinance(payAmount-costAmount-reserveAmount-row["refund_cost_amount"].Float64()-shopServiceFee-commissionDeduction, 2)

				stats := model.FinanceStatsSummary{
					TotalOrderCount:   row["total_order_count"].Int(),
					OrderCount:        row["order_count"].Int(),
					InvalidOrderCount: row["invalid_order_count"].Int(),
					ProductNum:        row["product_num"].Int(),
					PayAmount:         roundFloatFinance(payAmount, 2),
					CostAmount:        roundFloatFinance(costAmount, 2),
					ReserveAmount:     roundFloatFinance(reserveAmount, 2),
					Commission:        commission,
					RefundCostAmount:  roundFloatFinance(row["refund_cost_amount"].Float64(), 2),
					RefundOrderCount:  row["refund_order_count"].Int(),
					ShopServiceFee:    shopServiceFee,
					RealtimeProfit:    realtimeProfit,
				}

				for _, aid := range uidToAccountIds[uid] {
					for _, ni := range accountToNodes[aid] {
						nodeStats[ni.idx].TotalOrderCount += stats.TotalOrderCount
						nodeStats[ni.idx].OrderCount += stats.OrderCount
						nodeStats[ni.idx].InvalidOrderCount += stats.InvalidOrderCount
						nodeStats[ni.idx].ProductNum += stats.ProductNum
						nodeStats[ni.idx].PayAmount += stats.PayAmount
						nodeStats[ni.idx].CostAmount += stats.CostAmount
						nodeStats[ni.idx].ReserveAmount += stats.ReserveAmount
						nodeStats[ni.idx].Commission += stats.Commission
						nodeStats[ni.idx].RefundCostAmount += stats.RefundCostAmount
						nodeStats[ni.idx].RefundOrderCount += stats.RefundOrderCount
						nodeStats[ni.idx].ShopServiceFee += stats.ShopServiceFee
						nodeStats[ni.idx].RealtimeProfit += stats.RealtimeProfit
					}
				}
			}
		}
	}

	// 5. 构建结果
	var children []model.NodeStatsItem
	for i, cn := range childNodes {
		st := nodeStats[i]
		st.PayAmount = roundFloatFinance(st.PayAmount, 2)
		st.CostAmount = roundFloatFinance(st.CostAmount, 2)
		st.ReserveAmount = roundFloatFinance(st.ReserveAmount, 2)
		st.Commission = roundFloatFinance(st.Commission, 2)
		st.RefundCostAmount = roundFloatFinance(st.RefundCostAmount, 2)
		st.ShopServiceFee = roundFloatFinance(st.ShopServiceFee, 2)
		st.RealtimeProfit = roundFloatFinance(st.RealtimeProfit, 2)
		children = append(children, model.NodeStatsItem{
			Id:               cn.id,
			Name:             cn.name,
			Type:             cn.nodeType,
			TotalOrderCount:  st.TotalOrderCount,
			OrderCount:       st.OrderCount,
			InvalidOrderCount: st.InvalidOrderCount,
			ProductNum:       st.ProductNum,
			PayAmount:        st.PayAmount,
			CostAmount:       st.CostAmount,
			Commission:       st.Commission,
			RefundCostAmount: st.RefundCostAmount,
			RefundOrderCount: st.RefundOrderCount,
			ShopServiceFee:   st.ShopServiceFee,
			RealtimeProfit:   st.RealtimeProfit,
		})
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
	shopQuery := g.DB().Model("account a").
		LeftJoin("trader_account_relation tsr", "a.id = tsr.account_id").
		Where("tsr.trader_id", realTraderId)
	if platform != "" {
		shopQuery = shopQuery.Where("a.platform", platform)
	}
	shops, err := shopQuery.Fields("a.id, a.platform, a.uid, a.account_name, a.platform_nickname").
		All()

	var children []model.NodeStatsItem
	if err == nil {
		for _, shop := range shops {
			stats, _ := s.statsOrdersByAccountIds(ctx, []int{shop["id"].Int()}, startDate, endDate, trader["name"].String())
			if stats == nil {
				stats = &model.FinanceStatsSummary{}
			}
			shopName := shop["account_name"].String()
			if shop["platform_nickname"].String() != "" {
				shopName = shop["platform_nickname"].String()
			}
			children = append(children, model.NodeStatsItem{
				Id:               int64(shop["id"].Int()),
				Name:             shopName,
				Type:             "shop",
				DouyinNickname:   shop["account_name"].String(),
				Uid:              shop["uid"].String(),
				TotalOrderCount:  stats.TotalOrderCount,
				OrderCount:       stats.OrderCount,
				InvalidOrderCount: stats.InvalidOrderCount,
				ProductNum:       stats.ProductNum,
				PayAmount:        stats.PayAmount,
				CostAmount:       stats.CostAmount,
				Commission:       stats.Commission,
				RefundCostAmount: stats.RefundCostAmount,
				RefundOrderCount: stats.RefundOrderCount,
				ShopServiceFee:   stats.ShopServiceFee,
				RealtimeProfit:   stats.RealtimeProfit,
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

	// 获取账号明细
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

// getTraderShopStats 获取投手关联账号的统计
func (s *FinanceCommissionService) getTraderShopStats(ctx context.Context, traderId int64, startDate, endDate string) ([]model.ShopStatsItem, error) {
	// 获取投手名称（用于归属规则）
	traderName, _ := s.getTraderName(ctx, traderId)

	// 获取投手关联的店铺
	shops, err := g.DB().Model("account a").
		LeftJoin("trader_account_relation tsr", "a.id = tsr.account_id").
		Where("tsr.trader_id", traderId).
		Fields("a.id, a.platform, a.uid, a.account_name, a.platform_nickname").
		All()
	if err != nil {
		return nil, err
	}

	var result []model.ShopStatsItem
	for _, shop := range shops {
		accountId := shop["id"].Int()
		stats, _ := s.statsOrdersByAccountIds(ctx, []int{accountId}, startDate, endDate, traderName)
		if stats == nil {
			stats = &model.FinanceStatsSummary{}
		}

		shopName := shop["account_name"].String()
		if shop["platform_nickname"].String() != "" {
			shopName = shop["platform_nickname"].String()
		}

		result = append(result, model.ShopStatsItem{
			AccountId:        accountId,
			ShopName:         shopName,
			Uid:              shop["uid"].String(),
			Platform:         shop["platform"].String(),
			TotalOrderCount:  stats.TotalOrderCount,
			OrderCount:       stats.OrderCount,
			InvalidOrderCount: stats.InvalidOrderCount,
			ProductNum:       stats.ProductNum,
			PayAmount:        stats.PayAmount,
			CostAmount:       stats.CostAmount,
			Commission:       stats.Commission,
			RefundCostAmount: stats.RefundCostAmount,
			RefundOrderCount: stats.RefundOrderCount,
			ShopServiceFee:   stats.ShopServiceFee,
			RealtimeProfit:   stats.RealtimeProfit,
			})
	}

	return result, nil
}

// ============================================
// 店铺明细查询
// ============================================

// GetShopDetail 获取账号明细统计
func (s *FinanceCommissionService) GetShopDetail(ctx context.Context, req *model.FinanceShopDetailReq) (*model.FinanceShopDetailRes, error) {
	startDate, endDate := s.ParseTimeRange(req.TimeType, req.StartDate, req.EndDate)

	// 获取账号信息
	shop, err := g.DB().Model("account").Where("id", req.AccountId).One()
	if err != nil || shop.IsEmpty() {
		return nil, fmt.Errorf("账号不存在")
	}

	// 统计数据
	stats, _ := s.statsOrdersByAccountIds(ctx, []int{req.AccountId}, startDate, endDate, "")
	if stats == nil {
		stats = &model.FinanceStatsSummary{}
	}

	shopName := shop["account_name"].String()
	if shop["platform_nickname"].String() != "" {
		shopName = shop["platform_nickname"].String()
	}

	return &model.FinanceShopDetailRes{
		AccountId:    req.AccountId,
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

// getAccountIdsByNode 根据组织节点获取账号ID列表（platform 为空则不过滤）
func (s *FinanceCommissionService) getAccountIdsByNode(ctx context.Context, nodeId int64, nodeType, platform string) ([]int, error) {
	switch nodeType {
	case "company":
		// 获取该公司及所有子公司的ID
		companyIds := s.getCompanyAndChildrenIds(ctx, nodeId)
		// 通过 trader 关联获取账号：公司 → trader(company_id) → trader_account_relation → shop
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
		shopQuery := g.DB().Model("trader_account_relation tsr").
			LeftJoin("account a", "tsr.account_id = a.id").
			Where("tsr.trader_id IN(?)", traderIds)
		if platform != "" {
			shopQuery = shopQuery.Where("a.platform", platform)
		}
		accountIdVars, err := shopQuery.Array("tsr.account_id")
		if err != nil {
			return nil, err
		}
		var accountIds []int
		for _, v := range accountIdVars {
			accountIds = append(accountIds, v.Int())
		}
		return accountIds, nil

	case "department":
		// nodeId 为编码后的ID，需要解码
		realDeptId := model.DecodeDeptID(nodeId)
		deptIds := s.getDeptAndChildrenIds(ctx, realDeptId)
		// 通过 trader 关联获取账号：部门 → trader(group_id) → trader_account_relation → shop
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
		shopQuery := g.DB().Model("trader_account_relation tsr").
			LeftJoin("account a", "tsr.account_id = a.id").
			Where("tsr.trader_id IN(?)", traderIds)
		if platform != "" {
			shopQuery = shopQuery.Where("a.platform", platform)
		}
		accountIdVars, err := shopQuery.Array("tsr.account_id")
		if err != nil {
			return nil, err
		}
		var accountIds []int
		for _, v := range accountIdVars {
			accountIds = append(accountIds, v.Int())
		}
		return accountIds, nil

	case "trader":
		// nodeId 为编码后的ID，需要解码
		realTraderId := model.DecodeTraderID(nodeId)
		shopQuery := g.DB().Model("trader_account_relation tsr").
			LeftJoin("account a", "tsr.account_id = a.id").
			Where("tsr.trader_id", realTraderId)
		if platform != "" {
			shopQuery = shopQuery.Where("a.platform", platform)
		}
		accountIdVars, err := shopQuery.Array("tsr.account_id")
		if err != nil {
			return nil, err
		}
		var accountIds []int
		for _, v := range accountIdVars {
			accountIds = append(accountIds, v.Int())
		}
		return accountIds, nil
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

// ============================================
// 归属规则（shop_account_attribution）
// ============================================

// attributionRule 归属规则缓存项
type attributionRule struct {
	ShopName  string
	TalentId  string
	TraderId  int64
	TraderName string
}

// loadAttributionRules 加载所有有效的特殊归属规则
func (s *FinanceCommissionService) loadAttributionRules(ctx context.Context) ([]attributionRule, error) {
	rows, err := g.DB().Model("shop_account_attribution saa").
		LeftJoin("trader t", "saa.target_value = t.name").
		Where("saa.status", 1).
		Where("saa.target_type", "operator").
		Fields("saa.shop_name, saa.source_account_uid, t.id AS trader_id, saa.target_value AS trader_name").
		All()
	if err != nil {
		return nil, err
	}

	var rules []attributionRule
	for _, row := range rows {
		if row["trader_id"].Int64() > 0 {
			rules = append(rules, attributionRule{
				ShopName:   row["shop_name"].String(),
				TalentId:   row["source_account_uid"].String(),
				TraderId:   row["trader_id"].Int64(),
				TraderName: row["trader_name"].String(),
			})
		}
	}
	return rules, nil
}

// attributionExcludeSQL 生成排除被重定向到其他投手的订单的 SQL 条件
// 返回 NOT EXISTS 子查询片段，用于 WHERE 子句
// 当订单的 (shop_name, talent_id) 命中归属规则 且 目标不是当前投手时，排除该订单
func (s *FinanceCommissionService) attributionExcludeSQL(currentTraderName string) string {
	return fmt.Sprintf(`
		NOT EXISTS (
			SELECT 1 FROM shop_account_attribution saa
			WHERE saa.shop_name COLLATE utf8mb4_unicode_ci = o.shop_name COLLATE utf8mb4_unicode_ci
			  AND saa.source_account_uid = o.talent_id
			  AND saa.status = 1
			  AND saa.target_value != '%s'
		)`, currentTraderName)
}

// attributionRedirectedSQL 生成查找从其他投手重定向到当前投手的订单的 SQL 条件
// 返回 EXISTS 子查询片段，用于 WHERE 子句
func (s *FinanceCommissionService) attributionRedirectedSQL(currentTraderName string) string {
	return fmt.Sprintf(`
		EXISTS (
			SELECT 1 FROM shop_account_attribution saa
			WHERE saa.shop_name COLLATE utf8mb4_unicode_ci = o.shop_name COLLATE utf8mb4_unicode_ci
			  AND saa.source_account_uid = o.talent_id
			  AND saa.status = 1
			  AND saa.target_value = '%s'
		)`, currentTraderName)
}

// getTraderName 获取投手名称
func (s *FinanceCommissionService) getTraderName(ctx context.Context, traderId int64) (string, error) {
	row, err := g.DB().Model("trader").Where("id", traderId).Fields("name").One()
	if err != nil {
		return "", err
	}
	if row.IsEmpty() {
		return "", fmt.Errorf("投手不存在: %d", traderId)
	}
	return row["name"].String(), nil
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

// getAdCostByAccountIds 查询指定账号集合在日期范围内的千川广告消耗
func (s *FinanceCommissionService) getAdCostByAccountIds(ctx context.Context, accountIds []int, startDate, endDate string) float64 {
	if len(accountIds) == 0 || startDate == "" || endDate == "" {
		return 0
	}
	row, err := g.DB().Model("account_qianchuan_mapping aqm").
		LeftJoin("qianchuan_advertiser_daily_report qar", qianchuanAdReportJoinCondRange(startDate, endDate)).
		WhereIn("aqm.account_id", accountIds).
		Fields("COALESCE(SUM(qar.stat_cost), 0) AS ad_cost").
		One()
	if err != nil {
		g.Log().Warningf(ctx, "[Finance] 查询千川广告消耗失败: accountIds=%v, err=%v", accountIds, err)
		return 0
	}
	return roundFloatFinance(row["ad_cost"].Float64(), 2)
}

// extractDateRange 从 startDate/endDate（可能含时间部分）提取纯日期 2006-01-02
func extractDateRange(startDate, endDate string) (string, string) {
	qStart, qEnd := "", ""
	if len(startDate) >= 10 {
		qStart = startDate[:10]
	}
	if len(endDate) >= 10 {
		qEnd = endDate[:10]
	}
	return qStart, qEnd
}

// statsOrdersByAccountIds 统计账号订单数据（根据店铺 platform 只查对应订单表）
// traderName 非空时启用归属规则：排除被重定向的订单 + 纳入从其他投手重定向过来的订单
func (s *FinanceCommissionService) statsOrdersByAccountIds(ctx context.Context, accountIds []int, startDate, endDate string, traderName string) (*model.FinanceStatsSummary, error) {
	if len(accountIds) == 0 && traderName == "" {
		return &model.FinanceStatsSummary{}, nil
	}

	// 获取账号的 uid、platform、account_name
	var shops gdb.Result
	var platformUids map[string][]string
	if len(accountIds) > 0 {
		var err error
		shops, err = g.DB().Model("account").
			Where("id IN(?)", accountIds).
			Fields("uid, platform, account_name, platform_nickname").
			All()
		if err != nil {
			return nil, err
		}
		// 构建 platform→uids（uid 为空时自动通过 account_name 匹配订单表）
		platformUids = s.buildPlatformUids(ctx, shops)
	} else {
		platformUids = make(map[string][]string)
	}

	g.Log().Infof(ctx, "[Finance] statsOrdersByAccountIds: accountIds=%v, traderName=%s, platformUids=%v", accountIds, traderName, platformUids)

	// 生成归属规则排除条件
	attributionExclude := ""
	if traderName != "" {
		attributionExclude = s.attributionExcludeSQL(traderName)
	}

	// 按平台分别统计对应订单表
	totalStats := &model.FinanceStatsSummary{}
	for platform, uids := range platformUids {
		tableName, ok := platformTableMap[platform]
		if !ok {
			continue
		}
		stats, _ := s.statsOrderTable(ctx, tableName, uids, startDate, endDate, attributionExclude)
		if stats != nil {
			// 小店服务费 = 定价总额(payAmount) × 2%（仅抖音、仅有效订单）
			if platform == "douyin" {
				stats.ShopServiceFee = roundFloatFinance(stats.PayAmount*0.02, 2)
			}
			// 佣金 = 定价总额 - 留底 - 货款 - 小店服务费
			stats.Commission = roundFloatFinance(stats.Commission-stats.ShopServiceFee, 2)
			// 实时利润 = 定价总额 - 货款 - 留底 - 退款成本 - 小店服务费 - 佣金 - 广告费
			commissionDeduction := roundFloatFinance(stats.PayAmount*0.005, 2)
				stats.RealtimeProfit = roundFloatFinance(stats.PayAmount-stats.CostAmount-stats.ReserveAmount-stats.RefundCostAmount-stats.ShopServiceFee-commissionDeduction-stats.AdCostAmount, 2)

			totalStats.TotalOrderCount += stats.TotalOrderCount
			totalStats.OrderCount += stats.OrderCount
			totalStats.InvalidOrderCount += stats.InvalidOrderCount
			totalStats.ProductNum += stats.ProductNum
			totalStats.PayAmount += stats.PayAmount
			totalStats.CostAmount += stats.CostAmount
			totalStats.ReserveAmount += stats.ReserveAmount
			totalStats.Commission += stats.Commission
			totalStats.RefundCostAmount += stats.RefundCostAmount
			totalStats.RefundOrderCount += stats.RefundOrderCount
			totalStats.ShopServiceFee += stats.ShopServiceFee
			totalStats.AdCostAmount += stats.AdCostAmount
			totalStats.ValidOrderCostMatch += stats.ValidOrderCostMatch
			totalStats.CommissionOrderCount += stats.CommissionOrderCount
			totalStats.RealtimeProfit += stats.RealtimeProfit
		}
	}

	// 纳入从其他投手重定向到当前投手的订单
	if traderName != "" {
		redirectedStats := s.statsRedirectedOrders(ctx, traderName, startDate, endDate)
		if redirectedStats != nil {
			totalStats.TotalOrderCount += redirectedStats.TotalOrderCount
			totalStats.OrderCount += redirectedStats.OrderCount
			totalStats.InvalidOrderCount += redirectedStats.InvalidOrderCount
			totalStats.ProductNum += redirectedStats.ProductNum
			totalStats.PayAmount += redirectedStats.PayAmount
			totalStats.CostAmount += redirectedStats.CostAmount
			totalStats.ReserveAmount += redirectedStats.ReserveAmount
			totalStats.Commission += redirectedStats.Commission
			totalStats.RefundCostAmount += redirectedStats.RefundCostAmount
			totalStats.RefundOrderCount += redirectedStats.RefundOrderCount
			totalStats.ShopServiceFee += redirectedStats.ShopServiceFee
			totalStats.AdCostAmount += redirectedStats.AdCostAmount
			totalStats.ValidOrderCostMatch += redirectedStats.ValidOrderCostMatch
			totalStats.CommissionOrderCount += redirectedStats.CommissionOrderCount
			totalStats.RealtimeProfit += redirectedStats.RealtimeProfit
		}
	}

	totalStats.PayAmount = roundFloatFinance(totalStats.PayAmount, 2)
	totalStats.CostAmount = roundFloatFinance(totalStats.CostAmount, 2)
	totalStats.ReserveAmount = roundFloatFinance(totalStats.ReserveAmount, 2)
	totalStats.Commission = roundFloatFinance(totalStats.Commission, 2)
	totalStats.RefundCostAmount = roundFloatFinance(totalStats.RefundCostAmount, 2)
	totalStats.ShopServiceFee = roundFloatFinance(totalStats.ShopServiceFee, 2)

	// 查询千川广告消耗（按账号ID和日期范围）
	if len(accountIds) > 0 {
		qStart, qEnd := extractDateRange(startDate, endDate)
		totalStats.AdCostAmount = s.getAdCostByAccountIds(ctx, accountIds, qStart, qEnd)
	}
	// 重新计算实时利润（含广告费）
	commissionDeduction := roundFloatFinance(totalStats.PayAmount*0.005, 2)
	totalStats.RealtimeProfit = roundFloatFinance(totalStats.PayAmount-totalStats.CostAmount-totalStats.ReserveAmount-totalStats.RefundCostAmount-totalStats.ShopServiceFee-commissionDeduction-totalStats.AdCostAmount, 2)

	return totalStats, nil
}

// statsRedirectedOrders 统计从其他投手通过归属规则重定向到当前投手的订单
func (s *FinanceCommissionService) statsRedirectedOrders(ctx context.Context, traderName string, startDate, endDate string) *model.FinanceStatsSummary {
	redirectedCond := s.attributionRedirectedSQL(traderName)
	totalStats := &model.FinanceStatsSummary{}

	for _, tableName := range platformTableMap {
		stats, _ := s.statsOrderTable(ctx, tableName, nil, startDate, endDate, redirectedCond)
		if stats != nil {
			totalStats.TotalOrderCount += stats.TotalOrderCount
			totalStats.OrderCount += stats.OrderCount
			totalStats.InvalidOrderCount += stats.InvalidOrderCount
			totalStats.ProductNum += stats.ProductNum
			totalStats.PayAmount += stats.PayAmount
			totalStats.CostAmount += stats.CostAmount
			totalStats.ReserveAmount += stats.ReserveAmount
			totalStats.Commission += stats.Commission
			totalStats.RefundCostAmount += stats.RefundCostAmount
			totalStats.RefundOrderCount += stats.RefundOrderCount
			totalStats.ShopServiceFee += stats.ShopServiceFee
			totalStats.AdCostAmount += stats.AdCostAmount
			totalStats.ValidOrderCostMatch += stats.ValidOrderCostMatch
			totalStats.CommissionOrderCount += stats.CommissionOrderCount
			totalStats.RealtimeProfit += stats.RealtimeProfit
		}
	}

	// 计算小店服务费和佣金
	if totalStats.PayAmount > 0 {
		totalStats.ShopServiceFee = roundFloatFinance(totalStats.PayAmount*0.02, 2)
		totalStats.Commission = roundFloatFinance(totalStats.Commission-totalStats.ShopServiceFee, 2)
		commissionDeduction := roundFloatFinance(totalStats.PayAmount*0.005, 2)
		totalStats.RealtimeProfit = roundFloatFinance(totalStats.PayAmount-totalStats.CostAmount-totalStats.ReserveAmount-totalStats.RefundCostAmount-totalStats.ShopServiceFee-commissionDeduction-totalStats.AdCostAmount, 2)
	}

	return totalStats
}



// statsOrderTable 统计单张订单表（一次查询计算全部指标）
// 统计规则：
//   有效订单 = (order_status IN ('已发货','待发货','已完成') AND after_sale_status NOT IN ('6','7','11','12','29'))
//            OR (order_status = '已关闭' AND after_sale_status IN ('27','28','51'))
//   无效订单 = after_sale_status IN ('6','7','12')
//   定价总额 = product_price * product_num（仅有效订单）
//   货款     = base_price * product_num（仅有效订单）
//   留底     = product_price * product_num * 0.03
//   退款成本 = product_price * product_num（无效+已发货）
//   佣金     = 定价总额 - 留底 - 货款 - 小店服务费
func (s *FinanceCommissionService) statsOrderTable(ctx context.Context, tableName string, uids []string, startDate, endDate string, attributionWhere string) (*model.FinanceStatsSummary, error) {
	orderNoCol := "main_order_no"
	collateClause := ""
	talentCollate := ""
	if tableName == "kuaishou_orders" || tableName == "shipinhao_orders" {
		orderNoCol = "order_no"
		collateClause = "COLLATE utf8mb4_unicode_ci"
		talentCollate = "COLLATE utf8mb4_unicode_ci"
	}

	// 有效订单条件（两部分用 OR 连接）
	// 无效售后状态：6-售后申请, 7-售后退货中, 11-售后已发货, 12-售后成功, 29-售后退货拒绝
	validCond := fmt.Sprintf(
		"(o.order_status IN ('已发货','待发货','已完成') AND o.after_sale_status NOT IN ('6','7','11','12','29')) OR (o.order_status = '已关闭' AND o.after_sale_status IN ('27','28','51'))",
	)
	// 无效订单条件
	invalidCond := "o.after_sale_status IN ('6','7','11','12','29')"
	// 退款成本条件：无效 且 已发货
	refundCond := "o.after_sale_status IN ('6','7','11','12','29') AND o.order_status = '已发货'"

	// 货款通过 LEFT JOIN goods 用 base_price 计算
	attributionClause := ""
	if attributionWhere != "" {
		attributionClause = " AND " + attributionWhere
	}
	// 当 uids 为空时（重定向订单查询），不按 talent_id 过滤，仅用 attribution 条件
	uidFilter := ""
	if len(uids) > 0 {
		uidFilter = fmt.Sprintf("AND o.talent_id %s IN(?)", talentCollate)
	}
	sql := fmt.Sprintf(`
		SELECT
			COUNT(DISTINCT o.%s) AS total_order_count,
			COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS order_count,
			COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS invalid_order_count,
			COALESCE(SUM(CASE WHEN %s THEN o.product_num ELSE 0 END), 0) AS product_num,
			COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num ELSE 0 END), 0) AS pay_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS cost_amount,
			COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num * 0.03 ELSE 0 END), 0) AS reserve_amount,
			COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS refund_cost_amount,
			COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS refund_order_count,
			COUNT(DISTINCT CASE WHEN %s AND g.base_price > 0 THEN o.%s END) AS valid_order_cost_match
		FROM %s o
		LEFT JOIN goods g ON o.seller_code %s = g.goods_code %s
		WHERE o.order_submit_time >= ?
		  AND o.order_submit_time < ?
		  AND o.%s IS NOT NULL
		  AND o.%s != ''
		  %s
		  %s
	`, orderNoCol, validCond, orderNoCol, invalidCond, orderNoCol, validCond, validCond, validCond, validCond, refundCond, refundCond, orderNoCol, validCond, orderNoCol, tableName, collateClause, collateClause, orderNoCol, orderNoCol, uidFilter, attributionClause)

	endDateNext := CalcEndDateNext(endDate)

	// 根据是否有 uids 决定查询参数（参数顺序需与 SQL 占位符一致）
	var result gdb.Record
	var err error
	if len(uids) > 0 {
		result, err = g.DB().GetOne(ctx, sql, startDate, endDateNext, uids)
	} else {
		result, err = g.DB().GetOne(ctx, sql, startDate, endDateNext)
	}
	if err != nil {
		return nil, err
	}

	payAmount := result["pay_amount"].Float64()
	costAmount := result["cost_amount"].Float64()
	reserveAmount := result["reserve_amount"].Float64()
	g.Log().Debugf(ctx, "[StatsOrderTable] %s: pay=%.2f cost=%.2f reserve=%.2f orderCount=%d", tableName, payAmount, costAmount, reserveAmount, result["order_count"].Int())
	// 佣金 = 定价总额 - 留底 - 货款（小店服务费在 statsOrdersByAccountIds 中扣除）
	commission := payAmount - costAmount - reserveAmount
	orderCount := result["order_count"].Int()

	return &model.FinanceStatsSummary{
		TotalOrderCount:      result["total_order_count"].Int(),
		OrderCount:           orderCount,
		InvalidOrderCount:    result["invalid_order_count"].Int(),
		ProductNum:           result["product_num"].Int(),
		PayAmount:            roundFloatFinance(payAmount, 2),
		CostAmount:           roundFloatFinance(costAmount, 2),
		ReserveAmount:        roundFloatFinance(reserveAmount, 2),
		Commission:           roundFloatFinance(commission, 2),
		RefundCostAmount:     roundFloatFinance(result["refund_cost_amount"].Float64(), 2),
		RefundOrderCount:     result["refund_order_count"].Int(),
		ShopServiceFee:       0,
		AdCostAmount:         0,
		ValidOrderCostMatch:  result["valid_order_cost_match"].Int(),
		CommissionOrderCount: orderCount,
		RealtimeProfit:       0,
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
	shops, err := g.DB().Model("account a").
		LeftJoin("trader_account_relation tsr", "a.id = tsr.account_id").
		Where("tsr.trader_id", realTraderId).
		Fields("a.id, a.uid, a.account_name, a.platform_nickname, a.platform").
		All()
	if err != nil {
		return nil, fmt.Errorf("查询投手账号失败: %v", err)
	}

	// 构建 uid → 账号信息 映射，按 platform 分组 uid
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

	// 有效订单条件：与 statsOrderTable 保持一致
	validFilter := "AND ((o.order_status IN ('已发货','待发货','已完成') AND o.after_sale_status NOT IN ('6','7','11','12','29')) OR (o.order_status = '已关闭' AND o.after_sale_status IN ('27','28','51')))"
	// 归属规则排除：排除被重定向到其他投手的订单
	traderName := trader["name"].String()
	attributionExclude := s.attributionExcludeSQL(traderName)
	validFilter += " AND " + attributionExclude
	// 无效订单条件
	invalidFilter := "AND o.after_sale_status IN ('6','7','11','12','29')"

	// 根据店铺 platform 只查对应订单表
	var totalCount, invalidCount int
	var allOrders []model.FinanceOrderItem
	var totalPay, totalCost, totalReserve float64
	var totalRefundCost float64
	var totalRefundCount int

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

		// 查询该平台的订单数（主订单去重）
		countSQL := fmt.Sprintf(`
			SELECT COUNT(DISTINCT o.%s) AS cnt
			FROM %s o
			WHERE o.talent_id %s IN(?)
			  AND o.order_submit_time >= ?
			  AND o.order_submit_time < ?
			  AND o.%s IS NOT NULL
			  AND o.%s != ''
			  %s
		`, orderNoCol, tableName, talentCollate, orderNoCol, orderNoCol, validFilter)

		countResult, err := g.DB().GetOne(ctx, countSQL, uids, startDate, endDateNext)
		if err != nil {
			continue
		}
		tableCount := countResult["cnt"].Int()
		totalCount += tableCount

		if tableCount == 0 {
			continue
		}

		// 查询该平台的汇总（货款通过 LEFT JOIN goods 用 base_price 计算）
		sumSQL := fmt.Sprintf(`
			SELECT
				COALESCE(SUM(o.product_price * o.product_num), 0) AS pay_amount,
				COALESCE(SUM(COALESCE(g.base_price, 0) * o.product_num), 0) AS cost_amount,
				COALESCE(SUM(o.product_price * o.product_num * 0.03), 0) AS reserve_amount
			FROM %s o
			LEFT JOIN goods g ON o.seller_code %s = g.goods_code %s
			WHERE o.talent_id %s IN(?)
			  AND o.order_submit_time >= ?
			  AND o.order_submit_time < ?
			  AND o.%s IS NOT NULL
			  AND o.%s != ''
			  %s
		`, tableName, collateClause, collateClause, talentCollate, orderNoCol, orderNoCol, validFilter)

		sumResult, err := g.DB().GetOne(ctx, sumSQL, uids, startDate, endDateNext)
		if err == nil {
			pay := sumResult["pay_amount"].Float64()
			cost := sumResult["cost_amount"].Float64()
			reserve := sumResult["reserve_amount"].Float64()
			g.Log().Debugf(ctx, "[TraderOrderList] sumSQL结果: platform=%s pay=%.2f cost=%.2f reserve=%.2f", platform, pay, cost, reserve)
			totalPay += pay
			totalCost += cost
			totalReserve += reserve
		}

		// 查询无效订单数
		invalidSQL := fmt.Sprintf(`SELECT COUNT(DISTINCT o.%s) AS cnt FROM %s o WHERE o.talent_id %s IN(?) AND o.order_submit_time >= ? AND o.order_submit_time < ? AND o.%s IS NOT NULL AND o.%s != '' %s`,
			orderNoCol, tableName, talentCollate, orderNoCol, orderNoCol, invalidFilter)
		if invR, _ := g.DB().GetOne(ctx, invalidSQL, uids, startDate, endDateNext); invR != nil {
			invalidCount += invR["cnt"].Int()
		}

		// 查询退款成本（无效+已发货 的货款）
		refundSQL := fmt.Sprintf(`SELECT COALESCE(SUM(COALESCE(g.base_price,0)*o.product_num),0) AS cost, COUNT(DISTINCT o.%s) AS cnt FROM %s o LEFT JOIN goods g ON o.seller_code %s = g.goods_code %s WHERE o.talent_id %s IN(?) AND o.order_submit_time >= ? AND o.order_submit_time < ? AND o.%s IS NOT NULL AND o.%s != '' AND o.after_sale_status IN ('6','7','11','12','29') AND o.order_status = '已发货'`,
			orderNoCol, tableName, collateClause, collateClause, talentCollate, orderNoCol, orderNoCol)
		if refR, _ := g.DB().GetOne(ctx, refundSQL, uids, startDate, endDateNext); refR != nil {
			totalRefundCost += refR["cost"].Float64()
			totalRefundCount += refR["cnt"].Int()
		}

		// 查询该平台的订单明细（带分页，LEFT JOIN goods 获取 base_price）
		sql := fmt.Sprintf(`
			SELECT
				o.%s AS main_order_no,
				o.product_name,
				o.product_num,
				o.product_price,
				COALESCE(g.base_price, 0) AS base_price,
				o.order_status,
				o.after_sale_status,
				IFNULL(o.order_submit_time, '') AS order_submit_time,
				o.talent_id
			FROM %s o
			LEFT JOIN goods g ON o.seller_code %s = g.goods_code %s
			WHERE o.talent_id %s IN(?)
			  AND o.order_submit_time >= ?
			  AND o.order_submit_time < ?
			  AND o.%s IS NOT NULL
			  AND o.%s != ''
			  %s
			ORDER BY o.order_submit_time DESC
			LIMIT ?, ?
		`, orderNoCol, tableName, collateClause, collateClause, talentCollate, orderNoCol, orderNoCol, validFilter)

		result, err := g.DB().GetAll(ctx, sql, uids, startDate, endDateNext, (page-1)*pageSize, pageSize)
		if err != nil {
			continue
		}

		for _, row := range result {
			talentId := row["talent_id"].String()
			shop := uidShopMap[talentId]
			productNum := row["product_num"].Int()
			productPrice := row["product_price"].Float64()
			basePrice := row["base_price"].Float64()
			payAmount := roundFloatFinance(productPrice*float64(productNum), 2)
			costAmount := basePrice * float64(productNum)
			reserveAmount := roundFloatFinance(productPrice*float64(productNum)*0.03, 2)
			commission := roundFloatFinance(payAmount-costAmount-reserveAmount, 2)

			payTime := row["order_submit_time"].String()
			if len(payTime) >= 10 {
				payTime = payTime[:10]
			}

			allOrders = append(allOrders, model.FinanceOrderItem{
				MainOrderNo:     row["main_order_no"].String(),
				ProductName:     row["product_name"].String(),
				ProductNum:      productNum,
				ProductPrice:    productPrice,
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

	// 纳入从其他投手通过归属规则重定向到当前投手的订单
	redirectedCond := s.attributionRedirectedSQL(traderName)
	redirectedValidFilter := validFilter[:len(validFilter)-len(" AND "+attributionExclude)] + " AND " + redirectedCond
	for platformItem, tableName := range platformTableMap {
		orderNoCol := "main_order_no"
		collateClause := ""
		if tableName == "kuaishou_orders" || tableName == "shipinhao_orders" {
			orderNoCol = "order_no"
			collateClause = "COLLATE utf8mb4_unicode_ci"
		}

		// 查询重定向订单的汇总
		redirectedSumSQL := fmt.Sprintf(`
			SELECT
				COUNT(DISTINCT o.%s) AS cnt,
				COALESCE(SUM(o.product_price * o.product_num), 0) AS pay,
				COALESCE(SUM(COALESCE(g.base_price, 0) * o.product_num), 0) AS cost,
				COALESCE(SUM(o.product_price * o.product_num * 0.03), 0) AS reserve
			FROM %s o
			LEFT JOIN goods g ON o.seller_code %s = g.goods_code %s
			WHERE o.order_submit_time >= ?
			  AND o.order_submit_time < ?
			  AND o.%s IS NOT NULL
			  AND o.%s != ''
			  %s
		`, orderNoCol, tableName, collateClause, collateClause, orderNoCol, orderNoCol, redirectedValidFilter)
		if sumR, _ := g.DB().GetOne(ctx, redirectedSumSQL, startDate, endDateNext); sumR != nil {
			totalCount += sumR["cnt"].Int()
			totalPay += sumR["pay"].Float64()
			totalCost += sumR["cost"].Float64()
			totalReserve += sumR["reserve"].Float64()
		}

		// 查询重定向订单的明细
		redirectedDetailSQL := fmt.Sprintf(`
			SELECT
				o.%s AS main_order_no,
				o.product_name,
				o.product_num,
				o.product_price,
				COALESCE(g.base_price, 0) AS base_price,
				o.order_status,
				o.after_sale_status,
				IFNULL(o.order_submit_time, '') AS order_submit_time,
				o.talent_id
			FROM %s o
			LEFT JOIN goods g ON o.seller_code %s = g.goods_code %s
			WHERE o.order_submit_time >= ?
			  AND o.order_submit_time < ?
			  AND o.%s IS NOT NULL
			  AND o.%s != ''
			  %s
			ORDER BY o.order_submit_time DESC
		`, orderNoCol, tableName, collateClause, collateClause, orderNoCol, orderNoCol, redirectedValidFilter)
		if result, err := g.DB().GetAll(ctx, redirectedDetailSQL, startDate, endDateNext); err == nil {
			for _, row := range result {
				talentId := row["talent_id"].String()
				productNum := row["product_num"].Int()
				productPrice := row["product_price"].Float64()
				basePrice := row["base_price"].Float64()
				payAmount := roundFloatFinance(productPrice*float64(productNum), 2)
				costAmount := basePrice * float64(productNum)
				reserveAmount := roundFloatFinance(productPrice*float64(productNum)*0.03, 2)
				commission := roundFloatFinance(payAmount-costAmount-reserveAmount, 2)

				payTime := row["order_submit_time"].String()
				if len(payTime) >= 10 {
					payTime = payTime[:10]
				}

				// 重定向订单的店铺名从订单表获取
				shopName := talentId
				if info, ok := uidShopMap[talentId]; ok {
					shopName = info.name
				}

				allOrders = append(allOrders, model.FinanceOrderItem{
					MainOrderNo:     row["main_order_no"].String(),
					ProductName:     row["product_name"].String(),
					ProductNum:      productNum,
					ProductPrice:    productPrice,
					OrderPayAmount:  payAmount,
					CostAmount:      roundFloatFinance(costAmount, 2),
					ReserveAmount:   reserveAmount,
					Commission:      commission,
					Platform:        platformItem,
					ShopName:        shopName,
					OrderStatus:     row["order_status"].String(),
					AfterSaleStatus: row["after_sale_status"].String(),
					PayFinishTime:   payTime,
				})
			}
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

	shopFee := roundFloatFinance(totalPay*0.02, 2)
	commission := roundFloatFinance(totalPay-totalCost-totalReserve-shopFee, 2)
	commissionDeduction := roundFloatFinance(totalPay*0.005, 2)
	res.Summary = model.FinanceStatsSummary{
		TotalOrderCount:   totalCount + invalidCount,
		OrderCount:        totalCount,
		InvalidOrderCount: invalidCount,
		PayAmount:         roundFloatFinance(totalPay, 2),
		CostAmount:        roundFloatFinance(totalCost, 2),
		ReserveAmount:     roundFloatFinance(totalReserve, 2),
		ShopServiceFee:    shopFee,
		Commission:        commission,
		RefundCostAmount:  roundFloatFinance(totalRefundCost, 2),
		RefundOrderCount:  totalRefundCount,
		RealtimeProfit:    roundFloatFinance(commission-totalRefundCost-commissionDeduction, 2),
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
	shops, err := g.DB().Model("account a").
		LeftJoin("trader_account_relation tsr", "a.id = tsr.account_id").
		Where("tsr.trader_id", realTraderId).
		Fields("a.id, a.uid, a.platform, a.account_name").
		All()
	if err != nil {
		return nil, fmt.Errorf("查询投手账号失败: %v", err)
	}

	// 按 platform 分组 uid（uid 为空时自动匹配）
	platformUids := s.buildPlatformUids(ctx, shops)

	// 归属规则
	traderName := trader["name"].String()
	attributionExclude := s.attributionExcludeSQL(traderName)

	// 按平台统计
	totalStats := &model.FinanceStatsSummary{}
	var platformStats []model.PlatformStats
	for platform, uids := range platformUids {
		tableName, ok := platformTableMap[platform]
		if !ok {
			continue
		}
		stats, _ := s.statsOrderTable(ctx, tableName, uids, startDate, endDate, attributionExclude)
		if stats != nil {
			// 小店服务费 = 定价总额 × 2%（仅抖音）
			if platform == "douyin" {
				stats.ShopServiceFee = roundFloatFinance(stats.PayAmount*0.02, 2)
			}
			// 佣金 = 定价总额 - 留底 - 货款 - 小店服务费
			stats.Commission = roundFloatFinance(stats.Commission-stats.ShopServiceFee, 2)
			// 实时利润 = 佣金 - 退款成本 - 广告费 - 0.5%扣减（与达人统计一致）
			commissionDeduction := roundFloatFinance(stats.PayAmount*0.005, 2)
			stats.RealtimeProfit = roundFloatFinance(stats.Commission-stats.RefundCostAmount-stats.AdCostAmount-commissionDeduction, 2)

			totalStats.TotalOrderCount += stats.TotalOrderCount
			totalStats.OrderCount += stats.OrderCount
			totalStats.InvalidOrderCount += stats.InvalidOrderCount
			totalStats.ProductNum += stats.ProductNum
			totalStats.PayAmount += stats.PayAmount
			totalStats.CostAmount += stats.CostAmount
			totalStats.ReserveAmount += stats.ReserveAmount
			totalStats.Commission += stats.Commission
			totalStats.RefundCostAmount += stats.RefundCostAmount
			totalStats.RefundOrderCount += stats.RefundOrderCount
			totalStats.ShopServiceFee += stats.ShopServiceFee
			totalStats.AdCostAmount += stats.AdCostAmount
			totalStats.ValidOrderCostMatch += stats.ValidOrderCostMatch
			totalStats.CommissionOrderCount += stats.CommissionOrderCount
			totalStats.RealtimeProfit += stats.RealtimeProfit

			platformStats = append(platformStats, model.PlatformStats{
				Platform:          platform,
				PlatformName:      platformNameMap[platform],
				TotalOrderCount:   stats.TotalOrderCount,
				OrderCount:        stats.OrderCount,
				InvalidOrderCount: stats.InvalidOrderCount,
				ProductNum:        stats.ProductNum,
				PayAmount:         roundFloatFinance(stats.PayAmount, 2),
				CostAmount:        roundFloatFinance(stats.CostAmount, 2),
				ReserveAmount:     roundFloatFinance(stats.ReserveAmount, 2),
				Commission:        roundFloatFinance(stats.Commission, 2),
				RefundCostAmount:  roundFloatFinance(stats.RefundCostAmount, 2),
				RefundOrderCount:  stats.RefundOrderCount,
				ShopServiceFee:    roundFloatFinance(stats.ShopServiceFee, 2),
				RealtimeProfit:    roundFloatFinance(stats.RealtimeProfit, 2),
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
	shops, err := g.DB().Model("account a").
		LeftJoin("trader_account_relation tsr", "a.id = tsr.account_id").
		Where("tsr.trader_id", realTraderId).
		Fields("a.id, a.uid, a.account_name, a.platform_nickname, a.platform").
		All()
	if err != nil {
		return "", fmt.Errorf("查询投手账号失败: %v", err)
	}

	// 按 platform 分组 uid，同时构建 uid→账号名 映射
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
		return "", fmt.Errorf("投手无关联账号")
	}

	// 日期范围
	endDateNext := CalcEndDateNext(endDate)

	validFilter := "AND ((o.order_status IN ('已发货','待发货','已完成') AND o.after_sale_status NOT IN ('6','7','11','12','29')) OR (o.order_status = '已关闭' AND o.after_sale_status IN ('27','28','51')))"
	// 归属规则排除：排除被重定向到其他投手的订单
	traderName := trader["name"].String()
	attributionExclude := s.attributionExcludeSQL(traderName)
	validFilterWithExclude := validFilter + " AND " + attributionExclude
	// 归属规则纳入：查找从其他投手重定向到当前投手的订单
	redirectedCond := s.attributionRedirectedSQL(traderName)
	validFilterWithRedirect := validFilter + " AND " + redirectedCond

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
				o.order_status,
				o.after_sale_status,
				IFNULL(o.order_submit_time, '') AS order_submit_time,
				o.talent_id
			FROM %s o
			LEFT JOIN goods g ON o.seller_code %s = g.goods_code %s
			WHERE o.talent_id %s IN(?)
			  AND o.order_submit_time >= ?
			  AND o.order_submit_time < ?
			  AND o.%s IS NOT NULL
			  AND o.%s != ''
			  %s
			ORDER BY o.order_submit_time DESC
		`, orderNoCol, tableName, collateClause, collateClause, talentCollate, orderNoCol, orderNoCol, validFilterWithExclude)

		result, err := g.DB().GetAll(ctx, sql, uids, startDate, endDateNext)
		if err != nil || len(result) == 0 {
			// 即使直接订单为空，仍需检查重定向订单
			result = nil
		}

		// 查询从其他投手重定向到当前投手的订单
		redirectedSQL := fmt.Sprintf(`
			SELECT
				o.%s AS main_order_no,
				o.product_name,
				o.product_num,
				o.product_price,
				COALESCE(g.base_price, 0) AS base_price,
				o.order_status,
				o.after_sale_status,
				IFNULL(o.order_submit_time, '') AS order_submit_time,
				o.talent_id
			FROM %s o
			LEFT JOIN goods g ON o.seller_code %s = g.goods_code %s
			WHERE o.order_submit_time >= ?
			  AND o.order_submit_time < ?
			  AND o.%s IS NOT NULL
			  AND o.%s != ''
			  %s
			ORDER BY o.order_submit_time DESC
		`, orderNoCol, tableName, collateClause, collateClause, orderNoCol, orderNoCol, validFilterWithRedirect)
		if redirectedResult, err := g.DB().GetAll(ctx, redirectedSQL, startDate, endDateNext); err == nil && len(redirectedResult) > 0 {
			result = append(result, redirectedResult...)
		}

		if len(result) == 0 {
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

		// 流式写入：避免大数据量逐单元格写入导致内存溢出/超时
		sw, swErr := f.NewStreamWriter(sheetName)
		if swErr != nil {
			continue
		}

		// 写表头（带样式）
		headerRow := make([]interface{}, len(headers))
		for i, h := range headers {
			headerRow[i] = h
		}
		_ = sw.SetRow("A1", headerRow, excelize.RowOpts{StyleID: headerStyle})

		// 写数据（逐行，远快于逐单元格）
		for rowIdx, row := range result {
			talentId := row["talent_id"].String()
			productNum := row["product_num"].Int()
			productPrice := row["product_price"].Float64()
			basePrice := row["base_price"].Float64()
			payAmount := roundFloatFinance(productPrice*float64(productNum), 2)
			costAmount := basePrice * float64(productNum)
			reserveAmount := roundFloatFinance(productPrice*float64(productNum)*0.03, 2)
			commission := roundFloatFinance(payAmount-costAmount-reserveAmount, 2)

			payTime := row["order_submit_time"].String()
			if len(payTime) >= 10 {
				payTime = payTime[:10]
			}

			cell, _ := excelize.CoordinatesToCellName(1, rowIdx+2)
			if err := sw.SetRow(cell, []interface{}{
				row["main_order_no"].String(),
				row["product_name"].String(),
				productNum,
				productPrice,
				payAmount,
				roundFloatFinance(costAmount, 2),
				reserveAmount,
				commission,
				uidShopMap[talentId],
				row["order_status"].String(),
				row["after_sale_status"].String(),
				payTime,
			}); err != nil {
				break
			}
		}
		_ = sw.Flush()
	}

	// 保存文件
	exportDir := "./exports"
	os.MkdirAll(exportDir, 0755)
	safeTraderName := sanitizeFileName(trader["name"].String())
	fileName := fmt.Sprintf("投手订单_%s_%s_%s.xlsx", safeTraderName, startDate, endDate)
	filePath := filepath.Join(exportDir, fileName)
	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("保存Excel失败: %v", err)
	}

	return filePath, nil
}

// ExportAccountOrders 导出账号订单明细（按平台分Sheet）
func (s *FinanceCommissionService) ExportAccountOrders(ctx context.Context, accountId int, timeType, startDate, endDate string) (string, error) {
	startDate, endDate = s.ParseTimeRange(timeType, startDate, endDate)

	// 获取账号信息
	shop, err := g.DB().Model("account").Where("id", accountId).One()
	if err != nil || shop.IsEmpty() {
		return "", fmt.Errorf("账号不存在")
	}
	uid := shop["uid"].String()
	if uid == "" {
		return "", fmt.Errorf("账号未关联UID")
	}
	accountName := shop["account_name"].String()
	if shop["platform_nickname"].String() != "" {
		accountName = shop["platform_nickname"].String()
	}

	endDateNext := CalcEndDateNext(endDate)
	validFilter := "AND ((o.order_status IN ('已发货','待发货','已完成') AND o.after_sale_status NOT IN ('6','7','11','12','29')) OR (o.order_status = '已关闭' AND o.after_sale_status IN ('27','28','51')))"

	f := excelize.NewFile()
	defer f.Close()
	firstSheet := true

	headers := []string{"订单号", "商品名称", "数量", "单价", "应付金额", "货款", "留底", "佣金", "订单状态", "售后状态", "付款时间"}
	colWidths := []float64{22, 40, 8, 12, 14, 14, 14, 14, 12, 12, 14}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#CCCCCC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	for platform, tableName := range platformTableMap {
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
				o.order_status,
				o.after_sale_status,
				IFNULL(o.order_submit_time, '') AS order_submit_time
			FROM %s o
			LEFT JOIN goods g ON o.seller_code %s = g.goods_code %s
			WHERE o.talent_id %s = ?
			  AND o.order_submit_time >= ?
			  AND o.order_submit_time < ?
			  AND o.%s IS NOT NULL AND o.%s != ''
			  %s
			ORDER BY o.order_submit_time DESC
		`, orderNoCol, tableName, collateClause, collateClause, talentCollate, orderNoCol, orderNoCol, validFilter)

		result, err := g.DB().GetAll(ctx, sql, uid, startDate, endDateNext)
		if err != nil || len(result) == 0 {
			continue
		}

		sheetName := platformNameMap[platform]
		if sheetName == "" {
			sheetName = platform
		}
		if firstSheet {
			f.SetSheetName("Sheet1", sheetName)
			firstSheet = false
		} else {
			f.NewSheet(sheetName)
		}

		for i, w := range colWidths {
			col, _ := excelize.ColumnNumberToName(i + 1)
			f.SetColWidth(sheetName, col, col, w)
		}

		// 流式写入：避免大数据量逐单元格写入导致内存溢出/超时
		sw, swErr := f.NewStreamWriter(sheetName)
		if swErr != nil {
			continue
		}

		// 写表头（带样式）
		headerRow := make([]interface{}, len(headers))
		for i, h := range headers {
			headerRow[i] = h
		}
		_ = sw.SetRow("A1", headerRow, excelize.RowOpts{StyleID: headerStyle})

		// 写数据（逐行）
		for rowIdx, row := range result {
			productNum := row["product_num"].Int()
			productPrice := row["product_price"].Float64()
			basePrice := row["base_price"].Float64()
			payAmount := roundFloatFinance(productPrice*float64(productNum), 2)
			costAmount := basePrice * float64(productNum)
			reserveAmount := roundFloatFinance(productPrice*float64(productNum)*0.03, 2)
			commission := roundFloatFinance(payAmount-costAmount-reserveAmount, 2)

			payTime := row["order_submit_time"].String()
			if len(payTime) >= 10 {
				payTime = payTime[:10]
			}

			cell, _ := excelize.CoordinatesToCellName(1, rowIdx+2)
			if err := sw.SetRow(cell, []interface{}{
				row["main_order_no"].String(),
				row["product_name"].String(),
				productNum,
				productPrice,
				payAmount,
				roundFloatFinance(costAmount, 2),
				reserveAmount,
				commission,
				row["order_status"].String(),
				row["after_sale_status"].String(),
				payTime,
			}); err != nil {
				break
			}
		}
		_ = sw.Flush()
	}

	exportDir := "./exports"
	os.MkdirAll(exportDir, 0755)
	safeAccountName := sanitizeFileName(accountName)
	fileName := fmt.Sprintf("账号订单_%s_%s_%s.xlsx", safeAccountName, startDate, endDate)
	filePath := filepath.Join(exportDir, fileName)
	if err := f.SaveAs(filePath); err != nil {
		return "", fmt.Errorf("保存Excel失败: %v", err)
	}

	return filePath, nil
}

// GetAccountOrderList 获取账号订单列表
func (s *FinanceCommissionService) GetAccountOrderList(ctx context.Context, req *model.FinanceAccountOrdersReq) (*model.FinanceAccountOrdersRes, error) {
	startDate, endDate := s.ParseTimeRange(req.TimeType, req.StartDate, req.EndDate)

	shop, err := g.DB().Model("account").Where("id", req.AccountId).One()
	if err != nil || shop.IsEmpty() {
		return nil, fmt.Errorf("账号不存在")
	}
	uid := shop["uid"].String()
	if uid == "" {
		return nil, fmt.Errorf("账号未关联UID")
	}
	accountName := shop["account_name"].String()
	if shop["platform_nickname"].String() != "" {
		accountName = shop["platform_nickname"].String()
	}

	page := req.Page
	if page < 1 { page = 1 }
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	endDateNext := CalcEndDateNext(endDate)
	validFilter := "AND ((o.order_status IN ('已发货','待发货','已完成') AND o.after_sale_status NOT IN ('6','7','11','12','29')) OR (o.order_status = '已关闭' AND o.after_sale_status IN ('27','28','51')))"
	invalidFilter := "AND o.after_sale_status IN ('6','7','11','12','29')"

	var totalCount, invalidCount int
	var allOrders []model.FinanceOrderItem
	var totalPay, totalCost, totalReserve float64
	var totalRefundCost float64
	var totalRefundCount int

	// 查各平台订单表（按 uid 筛选）
	uids := []string{uid}
	for platform, tableName := range platformTableMap {
		orderNoCol := "main_order_no"
		collateClause := ""
		talentCollate := ""
		if tableName == "kuaishou_orders" || tableName == "shipinhao_orders" {
			orderNoCol = "order_no"
			collateClause = "COLLATE utf8mb4_unicode_ci"
			talentCollate = "COLLATE utf8mb4_unicode_ci"
		}

		// count（主订单去重）
		countSQL := fmt.Sprintf(`SELECT COUNT(DISTINCT o.%s) AS cnt FROM %s o WHERE o.talent_id %s IN(?) AND o.order_submit_time >= ? AND o.order_submit_time < ? AND o.%s IS NOT NULL AND o.%s != '' %s`, orderNoCol, tableName, talentCollate, orderNoCol, orderNoCol, validFilter)
		cntR, _ := g.DB().GetOne(ctx, countSQL, uids, startDate, endDateNext)
		if cntR != nil { totalCount += cntR["cnt"].Int() }

		// sum（货款通过 LEFT JOIN goods 用 base_price 计算）
		sumSQL := fmt.Sprintf(`SELECT COALESCE(SUM(o.product_price*o.product_num),0) AS pay, COALESCE(SUM(COALESCE(g.base_price,0)*o.product_num),0) AS cost, COALESCE(SUM(o.product_price*o.product_num*0.03),0) AS reserve FROM %s o LEFT JOIN goods g ON o.seller_code %s = g.goods_code %s WHERE o.talent_id %s IN(?) AND o.order_submit_time >= ? AND o.order_submit_time < ? AND o.%s IS NOT NULL AND o.%s != '' %s`, tableName, collateClause, collateClause, talentCollate, orderNoCol, orderNoCol, validFilter)
		sumR, _ := g.DB().GetOne(ctx, sumSQL, uids, startDate, endDateNext)
		if sumR != nil {
			totalPay += sumR["pay"].Float64()
			totalCost += sumR["cost"].Float64()
			totalReserve += sumR["reserve"].Float64()
		}

		// 查询无效订单数
		invalidSQL := fmt.Sprintf(`SELECT COUNT(DISTINCT o.%s) AS cnt FROM %s o WHERE o.talent_id %s IN(?) AND o.order_submit_time >= ? AND o.order_submit_time < ? AND o.%s IS NOT NULL AND o.%s != '' %s`,
			orderNoCol, tableName, talentCollate, orderNoCol, orderNoCol, invalidFilter)
		if invR, _ := g.DB().GetOne(ctx, invalidSQL, uids, startDate, endDateNext); invR != nil {
			invalidCount += invR["cnt"].Int()
		}

		// 查询退款成本
		refundSQL := fmt.Sprintf(`SELECT COALESCE(SUM(COALESCE(g.base_price,0)*o.product_num),0) AS cost, COUNT(DISTINCT o.%s) AS cnt FROM %s o LEFT JOIN goods g ON o.seller_code %s = g.goods_code %s WHERE o.talent_id %s IN(?) AND o.order_submit_time >= ? AND o.order_submit_time < ? AND o.%s IS NOT NULL AND o.%s != '' AND o.after_sale_status IN ('6','7','11','12','29') AND o.order_status = '已发货'`,
			orderNoCol, tableName, collateClause, collateClause, talentCollate, orderNoCol, orderNoCol)
		if refR, _ := g.DB().GetOne(ctx, refundSQL, uids, startDate, endDateNext); refR != nil {
			totalRefundCost += refR["cost"].Float64()
			totalRefundCount += refR["cnt"].Int()
		}

		// orders（LEFT JOIN goods 获取 base_price）
		orderSQL := fmt.Sprintf(`SELECT o.%s AS main_order_no, o.product_name, o.product_num, o.product_price, COALESCE(g.base_price,0) AS base_price, o.order_status, o.after_sale_status, IFNULL(o.order_submit_time,'') AS order_submit_time, o.talent_id FROM %s o LEFT JOIN goods g ON o.seller_code %s = g.goods_code %s WHERE o.talent_id %s IN(?) AND o.order_submit_time >= ? AND o.order_submit_time < ? AND o.%s IS NOT NULL AND o.%s != '' %s ORDER BY o.order_submit_time DESC LIMIT ?,?`, orderNoCol, tableName, collateClause, collateClause, talentCollate, orderNoCol, orderNoCol, validFilter)
		result, _ := g.DB().GetAll(ctx, orderSQL, uids, startDate, endDateNext, (page-1)*pageSize, pageSize)
		for _, row := range result {
			pNum := row["product_num"].Int()
			pPrice := row["product_price"].Float64()
			bPrice := row["base_price"].Float64()
			payAmt := roundFloatFinance(pPrice*float64(pNum), 2)
			costAmt := bPrice * float64(pNum)
			reserveAmt := roundFloatFinance(pPrice*float64(pNum)*0.03, 2)
			comm := roundFloatFinance(payAmt-costAmt-reserveAmt, 2)
			payTime := row["order_submit_time"].String()
			if len(payTime) >= 10 { payTime = payTime[:10] }
			allOrders = append(allOrders, model.FinanceOrderItem{
				MainOrderNo:    row["main_order_no"].String(),
				ProductName:    row["product_name"].String(),
				ProductNum:     pNum,
				ProductPrice:   pPrice,
				OrderPayAmount: payAmt,
				CostAmount:     roundFloatFinance(costAmt, 2),
				ReserveAmount:  reserveAmt,
				Commission:     comm,
				Platform:       platform,
				ShopName:       accountName,
				OrderStatus:    row["order_status"].String(),
				AfterSaleStatus: row["after_sale_status"].String(),
				PayFinishTime:  payTime,
			})
		}
	}

	// 排序截取
	sort.Slice(allOrders, func(i, j int) bool { return allOrders[i].PayFinishTime > allOrders[j].PayFinishTime })
	if len(allOrders) > pageSize { allOrders = allOrders[:pageSize] }

	shopFee := roundFloatFinance(totalPay*0.02, 2)
	comm := totalPay - totalCost - totalReserve - shopFee
	commissionDeduction := roundFloatFinance(totalPay*0.005, 2)
	return &model.FinanceAccountOrdersRes{
		AccountId:   req.AccountId,
		AccountName: accountName,
		TimeType:    req.TimeType,
		StartDate:   startDate,
		EndDate:     endDate,
		Summary: model.FinanceStatsSummary{
			TotalOrderCount: totalCount + invalidCount, OrderCount: totalCount,
			InvalidOrderCount: invalidCount,
			PayAmount: roundFloatFinance(totalPay,2), CostAmount: roundFloatFinance(totalCost,2),
			ReserveAmount: roundFloatFinance(totalReserve,2), ShopServiceFee: shopFee,
			Commission: roundFloatFinance(comm,2),
			RefundCostAmount: roundFloatFinance(totalRefundCost,2),
			RefundOrderCount: totalRefundCount,
			RealtimeProfit: roundFloatFinance(comm-totalRefundCost-commissionDeduction,2),
			CommissionOrderCount: totalCount,
		},
		Total: totalCount, Page: page, PageSize: pageSize,
		Orders: allOrders,
	}, nil
}
