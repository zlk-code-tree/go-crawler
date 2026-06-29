package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// ==================== 实时盈亏查询（含千川消耗） ====================

// RealtimeProfitReq 实时盈亏查询请求
type RealtimeProfitReq struct {
	TraderId     int64  `json:"traderId"`
	DepartmentId int64  `json:"departmentId"`
	CompanyId    int64  `json:"companyId"`
	Date         string `json:"date"` // 日期，格式 2006-01-02，空=今天
}

// RealtimeProfitResult 实时盈亏结果
type RealtimeProfitResult struct {
	StatDate       string  `json:"statDate"`
	OrderCount     int64   `json:"orderCount"`
	PayAmount      float64 `json:"payAmount"`
	CostAmount     float64 `json:"costAmount"`
	ReserveAmount  float64 `json:"reserveAmount"`
	ShopServiceFee float64 `json:"shopServiceFee"`
	Commission     float64 `json:"commission"`
	RefundCost     float64 `json:"refundCost"`
	RefundCount    int64   `json:"refundCount"`
	AdCost         float64 `json:"adCost"`
	NetProfit      float64 `json:"netProfit"`
}

// getValidCond 返回有效/无效订单条件SQL
func getValidCond(alias string) (validCond, invalidCond, refundCond string) {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	validCond = fmt.Sprintf("((%sorder_status IN ('已发货','待发货','已完成') AND %safter_sale_status NOT IN ('6','7','11','12','29')) OR (%sorder_status = '已关闭' AND %safter_sale_status IN ('27','28','51')))", prefix, prefix, prefix, prefix)
	invalidCond = fmt.Sprintf("%safter_sale_status IN ('6','7','11','12','29')", prefix)
	refundCond = fmt.Sprintf("%safter_sale_status IN ('6','7','11','12','29') AND %sorder_status = '已发货'", prefix, prefix)
	return
}

// GetRealtimeProfit 查询实时盈亏（含千川消耗），支持按投手/部门/公司维度
func (s *FinanceCommissionService) GetRealtimeProfit(ctx context.Context, req *RealtimeProfitReq) (*RealtimeProfitResult, error) {
	date := req.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	startTime := date + " 00:00:00"
	endTime := date + " 23:59:59"

	result := &RealtimeProfitResult{StatDate: date}

	// 1. 确定要查询的账号ID列表
	accountIds, err := s.getAccountIdsByDimension(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(accountIds) == 0 {
		return result, nil
	}

	// 2. 查询三平台订单统计
	validCond, invalidCond, refundCond := getValidCond("o")
	s.orderStatsByAccounts(ctx, accountIds, startTime, endTime, validCond, invalidCond, refundCond, result)

	// 3. 查询千川广告消耗
	result.AdCost = s.getAdCostByAccounts(ctx, accountIds, date)

	// 4. 计算净利润
	commissionDeduction := (result.PayAmount - result.ReserveAmount - result.CostAmount) * 0.005
	result.NetProfit = result.PayAmount - result.CostAmount - result.ReserveAmount - result.RefundCost - result.ShopServiceFee - commissionDeduction - result.AdCost
	result.NetProfit = roundFloatFinance(result.NetProfit, 2)
	result.AdCost = roundFloatFinance(result.AdCost, 2)

	return result, nil
}

// getAccountIdsByDimension 根据维度获取账号ID列表
func (s *FinanceCommissionService) getAccountIdsByDimension(ctx context.Context, req *RealtimeProfitReq) ([]int, error) {
	var accountIds []int

	if req.TraderId > 0 {
		rows, err := g.DB().Model("trader_account_relation").Where("trader_id", req.TraderId).Fields("account_id").All()
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			accountIds = append(accountIds, row["account_id"].Int())
		}
	} else if req.DepartmentId > 0 {
		rows, err := g.DB().Model("account").Where("department_id", req.DepartmentId).Fields("id").All()
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			accountIds = append(accountIds, row["id"].Int())
		}
	} else if req.CompanyId > 0 {
		rows, err := g.DB().Model("account").Where("company_id", req.CompanyId).Fields("id").All()
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			accountIds = append(accountIds, row["id"].Int())
		}
	}

	return accountIds, nil
}

// orderStatsByAccounts 查询指定账号集合的订单统计
func (s *FinanceCommissionService) orderStatsByAccounts(ctx context.Context, accountIds []int, startTime, endTime, validCond, invalidCond, refundCond string, result *RealtimeProfitResult) {
	rows, err := g.DB().Model("account").WhereIn("id", accountIds).Fields("uid").All()
	if err != nil {
		g.Log().Warningf(ctx, "[RealtimeProfit] 查询账号UID失败: %v", err)
		return
	}

	uids := make([]string, 0, len(rows))
	for _, row := range rows {
		if uid := row["uid"].String(); uid != "" {
			uids = append(uids, uid)
		}
	}
	if len(uids) == 0 {
		return
	}

	platforms := []struct {
		table      string
		orderNoCol string
		isDouyin   bool
	}{
		{"order_info", "main_order_no", true},
		{"kuaishou_orders", "order_no", false},
		{"shipinhao_orders", "order_no", false},
	}

	for _, p := range platforms {
		var row gdb.Record
		placeholders := strings.TrimRight(strings.Repeat("?,", len(uids)), ",")
		sql := fmt.Sprintf(`
			SELECT
				COUNT(DISTINCT o.%s) AS order_count,
				COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num ELSE 0 END), 0) AS pay_amount,
				COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, o.product_price) * o.product_num ELSE 0 END), 0) AS cost_amount,
				COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, o.product_price) * o.product_num ELSE 0 END), 0) AS refund_cost,
				COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS refund_count
			FROM %s o
			LEFT JOIN goods g ON o.seller_code = g.goods_code
			WHERE o.pay_finish_time >= ? AND o.pay_finish_time <= ?
			  AND o.talent_id IN (%s)
		`, p.orderNoCol, validCond, validCond, refundCond, refundCond, p.orderNoCol, p.table, placeholders)

		params := []interface{}{startTime, endTime}
		for _, uid := range uids {
			params = append(params, uid)
		}

		row, err = g.DB().GetOne(ctx, sql, params...)
		if err != nil {
			g.Log().Warningf(ctx, "[RealtimeProfit] 查询%s订单失败: %v", p.table, err)
			continue
		}

		count := row["order_count"].Int64()
		if count == 0 {
			continue
		}

		result.OrderCount += count
		payAmount := row["pay_amount"].Float64()
		result.PayAmount += payAmount
		result.CostAmount += row["cost_amount"].Float64()
		result.RefundCost += row["refund_cost"].Float64()
		result.RefundCount += row["refund_count"].Int64()

		result.ReserveAmount += payAmount * 0.03
		if p.isDouyin {
			result.ShopServiceFee += payAmount * 0.02
		}
	}

	result.Commission = result.PayAmount - result.ReserveAmount - result.CostAmount - result.ShopServiceFee
	if result.Commission < 0 {
		result.Commission = 0
	}

	result.PayAmount = roundFloatFinance(result.PayAmount, 2)
	result.CostAmount = roundFloatFinance(result.CostAmount, 2)
	result.ReserveAmount = roundFloatFinance(result.ReserveAmount, 2)
	result.ShopServiceFee = roundFloatFinance(result.ShopServiceFee, 2)
	result.Commission = roundFloatFinance(result.Commission, 2)
	result.RefundCost = roundFloatFinance(result.RefundCost, 2)
}

// getAdCostByAccounts 查询指定账号集合的千川广告消耗
func (s *FinanceCommissionService) getAdCostByAccounts(ctx context.Context, accountIds []int, date string) float64 {
	if len(accountIds) == 0 {
		return 0
	}

	row, err := g.DB().Model("account_qianchuan_mapping aqm").
		LeftJoin("qianchuan_advertiser_daily_report qar", "aqm.qianchuan_advertiser_id = qar.advertiser_id AND qar.stat_datetime = ?", date).
		WhereIn("aqm.account_id", accountIds).
		Fields("COALESCE(SUM(qar.stat_cost), 0) AS ad_cost").
		One()
	if err != nil {
		return 0
	}
	return row["ad_cost"].Float64()
}

// GetCostRanking 查询消耗排行
func (s *FinanceCommissionService) GetCostRanking(ctx context.Context, date string, dimension string, limit int) ([]g.Map, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if limit <= 0 {
		limit = 20
	}

	switch dimension {
	case "trader":
		return s.costRankingByTrader(ctx, date, limit)
	case "department":
		return s.costRankingByDept(ctx, date, limit)
	case "company":
		return s.costRankingByCompany(ctx, date, limit)
	default:
		return nil, fmt.Errorf("不支持的维度: %s，支持: trader/department/company", dimension)
	}
}

func (s *FinanceCommissionService) costRankingByTrader(ctx context.Context, date string, limit int) ([]g.Map, error) {
	rows, err := g.DB().Model("trader t").
		InnerJoin("trader_account_relation tar", "t.id = tar.trader_id").
		InnerJoin("account_qianchuan_mapping aqm", "tar.account_id = aqm.account_id").
		InnerJoin("qianchuan_advertiser_daily_report qar", "aqm.qianchuan_advertiser_id = qar.advertiser_id AND qar.stat_datetime = ?", date).
		Fields("t.id, t.name, COALESCE(SUM(qar.stat_cost), 0) AS ad_cost").
		Group("t.id, t.name").
		OrderDesc("ad_cost").
		Limit(limit).
		All()
	if err != nil {
		return nil, err
	}
	result := make([]g.Map, len(rows))
	for i, row := range rows {
		result[i] = g.Map{
			"id":     row["id"],
			"name":   row["name"],
			"adCost": row["ad_cost"].Float64(),
		}
	}
	return result, nil
}

func (s *FinanceCommissionService) costRankingByDept(ctx context.Context, date string, limit int) ([]g.Map, error) {
	rows, err := g.DB().Model("sys_department d").
		InnerJoin("account a", "a.department_id = d.id").
		InnerJoin("account_qianchuan_mapping aqm", "a.id = aqm.account_id").
		InnerJoin("qianchuan_advertiser_daily_report qar", "aqm.qianchuan_advertiser_id = qar.advertiser_id AND qar.stat_datetime = ?", date).
		Fields("d.id, d.name, COALESCE(SUM(qar.stat_cost), 0) AS ad_cost").
		Group("d.id, d.name").
		OrderDesc("ad_cost").
		Limit(limit).
		All()
	if err != nil {
		return nil, err
	}
	result := make([]g.Map, len(rows))
	for i, row := range rows {
		result[i] = g.Map{
			"id":     row["id"],
			"name":   row["name"],
			"adCost": row["ad_cost"].Float64(),
		}
	}
	return result, nil
}

func (s *FinanceCommissionService) costRankingByCompany(ctx context.Context, date string, limit int) ([]g.Map, error) {
	rows, err := g.DB().Model("sys_company c").
		InnerJoin("account a", "a.company_id = c.id").
		InnerJoin("account_qianchuan_mapping aqm", "a.id = aqm.account_id").
		InnerJoin("qianchuan_advertiser_daily_report qar", "aqm.qianchuan_advertiser_id = qar.advertiser_id AND qar.stat_datetime = ?", date).
		Fields("c.id, c.name, COALESCE(SUM(qar.stat_cost), 0) AS ad_cost").
		Group("c.id, c.name").
		OrderDesc("ad_cost").
		Limit(limit).
		All()
	if err != nil {
		return nil, err
	}
	result := make([]g.Map, len(rows))
	for i, row := range rows {
		result[i] = g.Map{
			"id":     row["id"],
			"name":   row["name"],
			"adCost": row["ad_cost"].Float64(),
		}
	}
	return result, nil
}
