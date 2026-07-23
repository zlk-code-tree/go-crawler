package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/xuri/excelize/v2"
)

// ==================== 实时盈亏查询（含千川消耗） ====================

// RealtimeProfitReq 实时盈亏查询请求
type RealtimeProfitReq struct {
	TraderId     int64  `json:"traderId"`
	DepartmentId int64  `json:"departmentId"`
	CompanyId    int64  `json:"companyId"`
	Date         string `json:"date"`      // 单日期，格式 2006-01-02，空=今天（兼容旧版）
	StartDate    string `json:"startDate"` // 开始日期，格式 2006-01-02
	EndDate      string `json:"endDate"`   // 结束日期，格式 2006-01-02
}

// RealtimeProfitResult 实时盈亏结果
type RealtimeProfitResult struct {
	StatDate          string  `json:"statDate"`
	TotalOrderCount   int64   `json:"totalOrderCount"`
	OrderCount        int64   `json:"orderCount"`
	InvalidOrderCount int64   `json:"invalidOrderCount"`
	ProductNum        int64   `json:"productNum"`
	PayAmount         float64 `json:"payAmount"`
	CostAmount        float64 `json:"costAmount"`
	ReserveAmount     float64 `json:"reserveAmount"`
	ShopServiceFee    float64 `json:"shopServiceFee"`
	Commission        float64 `json:"commission"`
	RefundCost        float64 `json:"refundCost"`
	RefundCount       int64   `json:"refundCount"`
	AdCost            float64 `json:"adCost"`
	NetProfit         float64 `json:"netProfit"`
}

// getValidCond 返回有效/无效订单条件SQL
func getValidCond(alias string) (validCond, invalidCond, refundCond string) {
	return financeValidOrderCondition(alias), financeInvalidOrderCondition(alias), financeRefundOrderCondition(alias)
}

// GetRealtimeProfit 查询实时盈亏（含千川消耗），支持按投手/部门/公司维度
func (s *FinanceCommissionService) GetRealtimeProfit(ctx context.Context, req *RealtimeProfitReq) (*RealtimeProfitResult, error) {
	// 解析日期范围：优先使用 StartDate/EndDate，兼容旧版 Date
	startDate, endDate := req.StartDate, req.EndDate
	if startDate == "" && endDate == "" {
		if req.Date != "" {
			startDate, endDate = req.Date, req.Date
		}
	}
	startDate, endDate = NormalizeFinanceDateRange(startDate, endDate)
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return nil, fmt.Errorf("开始日期格式错误，期望 2006-01-02: %s", startDate)
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		return nil, fmt.Errorf("结束日期格式错误，期望 2006-01-02: %s", endDate)
	}
	var nodeID int64
	var nodeType string
	platform := ""
	switch {
	case req.TraderId > 0:
		nodeID = req.TraderId
		if !model.IsTraderID(nodeID) {
			nodeID = model.EncodeTraderID(nodeID)
		}
		nodeType = "trader"
	case req.DepartmentId > 0:
		nodeID = req.DepartmentId
		if !model.IsDeptID(nodeID) {
			nodeID = model.EncodeDeptID(nodeID)
		}
		nodeType = "department"
	case req.CompanyId > 0:
		nodeID = req.CompanyId
		nodeType = "company"
		platform = "douyin"
	default:
		return nil, fmt.Errorf("请指定投手、部门或公司")
	}

	stats, err := s.GetStats(ctx, &model.FinanceStatsReq{
		NodeId: nodeID, NodeType: nodeType, TimeType: "custom",
		StartDate: startDate, EndDate: endDate, Platform: platform,
	})
	if err != nil {
		return nil, err
	}
	summary := stats.Summary
	return &RealtimeProfitResult{
		StatDate:          startDate + " ~ " + endDate,
		TotalOrderCount:   int64(summary.TotalOrderCount),
		OrderCount:        int64(summary.OrderCount),
		InvalidOrderCount: int64(summary.InvalidOrderCount),
		ProductNum:        int64(summary.ProductNum),
		PayAmount:         summary.PayAmount,
		CostAmount:        summary.CostAmount,
		ReserveAmount:     summary.ReserveAmount,
		ShopServiceFee:    summary.ShopServiceFee,
		Commission:        summary.Commission,
		RefundCost:        summary.RefundCostAmount,
		RefundCount:       int64(summary.RefundOrderCount),
		AdCost:            summary.AdCostAmount,
		NetProfit:         summary.RealtimeProfit,
	}, nil
}

// getAccountIdsByDimension 根据维度获取账号ID列表
func (s *FinanceCommissionService) getAccountIdsByDimension(ctx context.Context, req *RealtimeProfitReq) ([]int, error) {
	var (
		accountIds []int
		err        error
	)

	switch {
	case req.TraderId > 0:
		traderId := req.TraderId
		if !model.IsTraderID(traderId) {
			traderId = model.EncodeTraderID(traderId)
		}
		accountIds, err = s.getAccountIdsByNode(ctx, traderId, "trader", "")
	case req.DepartmentId > 0:
		departmentId := req.DepartmentId
		if !model.IsDeptID(departmentId) {
			departmentId = model.EncodeDeptID(departmentId)
		}
		accountIds, err = s.getAccountIdsByNode(ctx, departmentId, "department", "")
	case req.CompanyId > 0:
		// 公司维度需要汇总当前公司及所有子公司，口径与佣金统计保持一致。
		accountIds, err = s.getAccountIdsByNode(ctx, req.CompanyId, "company", "")
	}
	if err != nil {
		return nil, err
	}

	return uniqueIntSlice(accountIds), nil
}

func (s *FinanceCommissionService) getDepartmentTreeIds(ctx context.Context, departmentId int64) ([]int64, error) {
	rows, err := g.DB().GetAll(ctx, `
		WITH RECURSIVE dept_tree AS (
			SELECT id FROM sys_department WHERE id = ?
			UNION ALL
			SELECT d.id FROM sys_department d
			INNER JOIN dept_tree dt ON d.parent_id = dt.id
		)
		SELECT id FROM dept_tree
	`, departmentId)
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if id := row["id"].Int64(); id > 0 {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (s *FinanceCommissionService) getCompanyTreeIds(ctx context.Context, companyId int64) ([]int64, error) {
	rows, err := g.DB().GetAll(ctx, `
		WITH RECURSIVE company_tree AS (
			SELECT id FROM sys_company WHERE id = ? AND status = 1
			UNION ALL
			SELECT child.id
			FROM sys_company child
			INNER JOIN company_tree parent ON child.parent_id = parent.id
			WHERE child.status = 1
		)
		SELECT id FROM company_tree
	`, companyId)
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if id := row["id"].Int64(); id > 0 {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func int64Placeholders(count int) string {
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func appendInt64Args(args []interface{}, ids []int64) []interface{} {
	for _, id := range ids {
		args = append(args, id)
	}
	return args
}

func uniqueIntSlice(values []int) []int {
	if len(values) <= 1 {
		return values
	}

	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
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
				COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS order_count,
				COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num ELSE 0 END), 0) AS pay_amount,
				COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS cost_amount,
					COALESCE(SUM(CASE WHEN %s THEN COALESCE(g.base_price, 0) * o.product_num ELSE 0 END), 0) AS refund_cost,
				COUNT(DISTINCT CASE WHEN %s THEN o.%s END) AS refund_count
			FROM %s o
			LEFT JOIN goods g ON o.seller_code COLLATE utf8mb4_unicode_ci = g.goods_code COLLATE utf8mb4_unicode_ci
			WHERE o.pay_finish_time >= ? AND o.pay_finish_time <= ?
			  AND o.talent_id IN (%s)
		`, validCond, p.orderNoCol, validCond, validCond, refundCond, refundCond, p.orderNoCol, p.table, placeholders)

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

		if p.isDouyin {
			result.ShopServiceFee += CalculateDouyinShopServiceFee(payAmount)
		}
	}
	result.ReserveAmount = CalculateReserveAmount(result.PayAmount)

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

// qianchuanAdReportJoinCond 返回千川广告日报表的 JOIN ON 条件。
// GoFrame 的 Join 不支持条件参数占位符，调用方传入的日期已先做格式校验。
func qianchuanAdReportJoinCond(date string) string {
	return fmt.Sprintf("aqm.qianchuan_advertiser_id = qar.advertiser_id AND LEFT(qar.stat_datetime, 10) = '%s'", date)
}

func qianchuanAdReportJoinCondRange(startDate, endDate string) string {
	return fmt.Sprintf("aqm.qianchuan_advertiser_id = qar.advertiser_id AND LEFT(qar.stat_datetime, 10) BETWEEN '%s' AND '%s'", startDate, endDate)
}

// effectiveQianchuanCostCTE 生成账号每日有效消耗：达人日报优先；当天无达人日报时，
// 才使用已准确匹配到内部账号的计划消耗。这样既能覆盖达人接口缺数，也不会重复计算。
const effectiveQianchuanCostCTE = `WITH RECURSIVE
	author_daily AS (
		SELECT
			account_id,
			advertiser_id,
			MAX(aweme_id) AS aweme_id,
			MAX(NULLIF(aweme_name, '')) AS aweme_name,
			company_id,
			department_id,
			trader_id,
			LEFT(stat_datetime, 10) AS stat_date,
			COALESCE(SUM(stat_cost), 0) AS stat_cost,
			COALESCE(SUM(pay_order_gmv), 0) AS pay_order_gmv,
			COALESCE(SUM(pay_order_count), 0) AS pay_order_count
		FROM qianchuan_author_daily_report
		WHERE account_id > 0
		  AND LEFT(stat_datetime, 10) >= ?
		  AND LEFT(stat_datetime, 10) <= ?
		GROUP BY account_id, advertiser_id, company_id, department_id, trader_id, LEFT(stat_datetime, 10)
	),
	plan_daily AS (
		SELECT
			qar.account_id,
			qar.advertiser_id,
			MAX(qar.aweme_id) AS aweme_id,
			MAX(NULLIF(qar.aweme_name, '')) AS aweme_name,
			qar.company_id,
			qar.department_id,
			qar.trader_id,
			LEFT(qar.stat_datetime, 10) AS stat_date,
			COALESCE(SUM(qar.stat_cost), 0) AS stat_cost,
			0 AS pay_order_gmv,
			COALESCE(SUM(qar.pay_order_count), 0) AS pay_order_count
		FROM qianchuan_ad_report qar
		INNER JOIN account matched_account
			ON matched_account.id = qar.account_id
			AND matched_account.platform = 'douyin'
			AND (
				(matched_account.uid REGEXP '^[0-9]+$' AND CAST(matched_account.uid AS UNSIGNED) = qar.aweme_id)
				OR (NULLIF(TRIM(matched_account.platform_nickname), '') IS NOT NULL
					AND LOWER(TRIM(matched_account.platform_nickname)) COLLATE utf8mb4_unicode_ci = LOWER(TRIM(qar.aweme_show_id)) COLLATE utf8mb4_unicode_ci)
				OR (NULLIF(TRIM(matched_account.platform_nickname), '') IS NOT NULL
					AND LOWER(TRIM(matched_account.platform_nickname)) COLLATE utf8mb4_unicode_ci = LOWER(CAST(qar.aweme_id AS CHAR)) COLLATE utf8mb4_unicode_ci)
				OR (NULLIF(TRIM(matched_account.account_name), '') IS NOT NULL
					AND LOWER(TRIM(matched_account.account_name)) COLLATE utf8mb4_unicode_ci = LOWER(TRIM(qar.aweme_name)) COLLATE utf8mb4_unicode_ci)
			)
		WHERE qar.account_id > 0
		  AND LEFT(qar.stat_datetime, 10) >= ?
		  AND LEFT(qar.stat_datetime, 10) <= ?
		GROUP BY qar.account_id, qar.advertiser_id, qar.company_id, qar.department_id, qar.trader_id, LEFT(qar.stat_datetime, 10)
	),
	effective_qianchuan_cost AS (
		SELECT * FROM author_daily
		UNION ALL
		SELECT plan_daily.*
		FROM plan_daily
		WHERE NOT EXISTS (
			SELECT 1 FROM author_daily
			WHERE author_daily.account_id = plan_daily.account_id
			  AND author_daily.advertiser_id = plan_daily.advertiser_id
			  AND author_daily.stat_date = plan_daily.stat_date
		)
	)
`

// attributedQianchuanCostCTE 按计划投放账号解析千川消耗归属。
// 可下钻计划归投放账号所属达人，特殊规则优先；无法下钻及户实扣差额归千川户负责人。
const attributedQianchuanCostCTE = `WITH RECURSIVE
	active_owner AS (
		SELECT owner.*
		FROM qianchuan_account_owner_mapping owner
		WHERE owner.status = 1
		  AND owner.trader_id > 0
		  AND NOT EXISTS (
			SELECT 1
			FROM qianchuan_account_owner_mapping newer
			WHERE newer.advertiser_id = owner.advertiser_id
			  AND newer.status = 1
			  AND newer.trader_id > 0
			  AND newer.id > owner.id
		  )
	),
	plan_all_daily AS (
		SELECT
			advertiser_id,
			LEFT(stat_datetime, 10) AS stat_date,
			COALESCE(SUM(stat_cost), 0) AS stat_cost
		FROM qianchuan_ad_report
		WHERE LEFT(stat_datetime, 10) >= ?
		  AND LEFT(stat_datetime, 10) <= ?
		  AND stat_cost > 0
		GROUP BY advertiser_id, LEFT(stat_datetime, 10)
	),
	plan_account_daily AS (
		SELECT
			qar.account_id,
			qar.advertiser_id,
			MAX(qar.aweme_id) AS aweme_id,
			MAX(NULLIF(qar.aweme_name, '')) AS aweme_name,
			MAX(qar.company_id) AS company_id,
			MAX(qar.department_id) AS department_id,
			MAX(qar.trader_id) AS trader_id,
			LEFT(qar.stat_datetime, 10) AS stat_date,
			COALESCE(SUM(qar.stat_cost), 0) AS stat_cost,
			0 AS pay_order_gmv,
			COALESCE(SUM(qar.pay_order_count), 0) AS pay_order_count
		FROM qianchuan_ad_report qar
		INNER JOIN account matched_account
			ON matched_account.id = qar.account_id
			AND matched_account.platform = 'douyin'
			AND matched_account.status = 1
			AND (
				(matched_account.uid REGEXP '^[0-9]+$' AND CAST(matched_account.uid AS UNSIGNED) = qar.aweme_id)
				OR (NULLIF(TRIM(matched_account.platform_nickname), '') IS NOT NULL
					AND LOWER(TRIM(matched_account.platform_nickname)) COLLATE utf8mb4_unicode_ci = LOWER(TRIM(qar.aweme_show_id)) COLLATE utf8mb4_unicode_ci)
				OR (NULLIF(TRIM(matched_account.platform_nickname), '') IS NOT NULL
					AND LOWER(TRIM(matched_account.platform_nickname)) COLLATE utf8mb4_unicode_ci = LOWER(CAST(qar.aweme_id AS CHAR)) COLLATE utf8mb4_unicode_ci)
				OR (NULLIF(TRIM(matched_account.account_name), '') IS NOT NULL
					AND LOWER(TRIM(matched_account.account_name)) COLLATE utf8mb4_unicode_ci = LOWER(TRIM(qar.aweme_name)) COLLATE utf8mb4_unicode_ci)
			)
		WHERE qar.account_id > 0
		  AND LEFT(qar.stat_datetime, 10) >= ?
		  AND LEFT(qar.stat_datetime, 10) <= ?
		  AND qar.stat_cost > 0
		GROUP BY qar.account_id, qar.advertiser_id, LEFT(qar.stat_datetime, 10)
	),
	advertiser_daily AS (
		SELECT
			advertiser_id,
			MAX(NULLIF(advertiser_name, '')) AS advertiser_name,
			LEFT(stat_datetime, 10) AS stat_date,
			COALESCE(SUM(stat_cost), 0) AS stat_cost
		FROM qianchuan_advertiser_daily_report
		WHERE LEFT(stat_datetime, 10) >= ?
		  AND LEFT(stat_datetime, 10) <= ?
		GROUP BY advertiser_id, LEFT(stat_datetime, 10)
	),
	plan_scaled AS (
		SELECT
			plan.*,
			CASE
				WHEN daily.stat_cost IS NOT NULL
				 AND all_plan.stat_cost > daily.stat_cost
				 AND all_plan.stat_cost > 0
				THEN plan.stat_cost * daily.stat_cost / all_plan.stat_cost
				ELSE plan.stat_cost
			END AS allocated_stat_cost
		FROM plan_account_daily plan
		LEFT JOIN plan_all_daily all_plan
			ON all_plan.advertiser_id = plan.advertiser_id
			AND all_plan.stat_date = plan.stat_date
		LEFT JOIN advertiser_daily daily
			ON daily.advertiser_id = plan.advertiser_id
			AND daily.stat_date = plan.stat_date
	),
	resolved_plan_cost AS (
		SELECT
			plan.account_id,
			plan.advertiser_id,
			plan.aweme_id,
			plan.aweme_name,
			plan.company_id,
			plan.department_id,
			plan.trader_id,
			plan.stat_date,
			plan.allocated_stat_cost AS stat_cost,
			plan.pay_order_gmv,
			plan.pay_order_count,
			COALESCE(
				(
					SELECT rule.target_trader_id
					FROM qianchuan_advertiser_attribution_rule rule
					WHERE rule.advertiser_id = plan.advertiser_id
					  AND rule.status = 1
					  AND plan.stat_date >= rule.effective_start_date
					  AND (rule.effective_end_date IS NULL OR plan.stat_date <= rule.effective_end_date)
					  AND FIND_IN_SET(
						LOWER(TRIM(COALESCE(NULLIF(matched_account.account_name, ''), NULLIF(plan.aweme_name, ''), ''))) COLLATE utf8mb4_unicode_ci,
						LOWER(REPLACE(rule.source_account_names, '，', ',')) COLLATE utf8mb4_unicode_ci
					  ) > 0
					ORDER BY rule.effective_start_date DESC, rule.id DESC
					LIMIT 1
				),
				(
					SELECT MIN(account_owner.trader_id)
					FROM trader_account_relation account_owner
					INNER JOIN trader account_trader
						ON account_trader.id = account_owner.trader_id
						AND account_trader.status = 1
					WHERE account_owner.account_id = plan.account_id
				)
			) AS resolved_trader_id
		FROM plan_scaled plan
		INNER JOIN account matched_account
			ON matched_account.id = plan.account_id
			AND matched_account.platform = 'douyin'
	),
	attributed_plan_cost AS (
		SELECT
			resolved.*,
			'normal' AS cost_type
		FROM resolved_plan_cost resolved
		WHERE resolved.resolved_trader_id > 0
		  AND resolved.stat_cost >= 0.005
	),
	unassigned_plan_cost AS (
		SELECT
			resolved.*,
			'unassigned' AS cost_type
		FROM resolved_plan_cost resolved
		WHERE COALESCE(resolved.resolved_trader_id, 0) <= 0
		  AND resolved.stat_cost >= 0.005
	),
	attributed_plan_by_advertiser AS (
		SELECT advertiser_id, stat_date, COALESCE(SUM(stat_cost), 0) AS stat_cost
		FROM (
			SELECT advertiser_id, stat_date, stat_cost FROM attributed_plan_cost
			UNION ALL
			SELECT advertiser_id, stat_date, stat_cost FROM unassigned_plan_cost
		) effective_plan_cost
		GROUP BY advertiser_id, stat_date
	),
	advertiser_daily_gap AS (
		SELECT
			0 AS account_id,
			daily.advertiser_id,
			0 AS aweme_id,
			CONCAT('千川户差额消耗:', COALESCE(daily.advertiser_name, CAST(daily.advertiser_id AS CHAR))) AS aweme_name,
			COALESCE(owner.company_id, 0) AS company_id,
			COALESCE(owner.department_id, 0) AS department_id,
			COALESCE(owner.trader_id, 0) AS trader_id,
			daily.stat_date,
			daily.stat_cost - COALESCE(attributed.stat_cost, 0) AS stat_cost,
			0 AS pay_order_gmv,
			0 AS pay_order_count,
			COALESCE(owner.trader_id, 0) AS resolved_trader_id,
			CASE WHEN COALESCE(owner.trader_id, 0) > 0 THEN 'gap' ELSE 'unassigned' END AS cost_type
		FROM advertiser_daily daily
		LEFT JOIN active_owner owner
			ON owner.advertiser_id = daily.advertiser_id
		LEFT JOIN attributed_plan_by_advertiser attributed
			ON attributed.advertiser_id = daily.advertiser_id
			AND attributed.stat_date = daily.stat_date
		WHERE daily.stat_cost - COALESCE(attributed.stat_cost, 0) >= 0.005
	),
	attributed_qianchuan_cost AS (
		SELECT * FROM attributed_plan_cost
		UNION ALL
		SELECT * FROM unassigned_plan_cost
		UNION ALL
		SELECT * FROM advertiser_daily_gap
	)
`

type qianchuanCostBreakdown struct {
	Normal float64
	Gap    float64
}

func (b qianchuanCostBreakdown) Total() float64 {
	return b.Normal + b.Gap
}

func qianchuanAttributionDateArgs(startDate, endDate string) []interface{} {
	return []interface{}{startDate, endDate, startDate, endDate, startDate, endDate}
}

func qianchuanCostBreakdownFromRecord(row gdb.Record) qianchuanCostBreakdown {
	return qianchuanCostBreakdown{
		Normal: row["normal_cost"].Float64(),
		Gap:    row["gap_cost"].Float64(),
	}
}

func getAttributedQianchuanCostsByTraderAccountRange(ctx context.Context, traderID int64, startDate, endDate string) (map[int]float64, error) {
	result := make(map[int]float64)
	if traderID <= 0 {
		return result, nil
	}
	startDate, endDate = NormalizeFinanceDateRange(startDate, endDate)
	args := append(qianchuanAttributionDateArgs(startDate, endDate), traderID)
	rows, err := g.DB().GetAll(ctx, attributedQianchuanCostCTE+`
		SELECT account_id, COALESCE(SUM(stat_cost), 0) AS ad_cost
		FROM attributed_qianchuan_cost
		WHERE resolved_trader_id = ?
		GROUP BY account_id`, args...)
	if err != nil {
		return nil, fmt.Errorf("查询达人账号双重校验千川消耗失败: %w", err)
	}
	for _, row := range rows {
		result[row["account_id"].Int()] = row["ad_cost"].Float64()
	}
	return result, nil
}

func getAttributedQianchuanCostByAccountRange(ctx context.Context, accountID int, startDate, endDate string) (float64, error) {
	if accountID <= 0 {
		return 0, nil
	}
	startDate, endDate = NormalizeFinanceDateRange(startDate, endDate)
	args := append(qianchuanAttributionDateArgs(startDate, endDate), accountID)
	row, err := g.DB().GetOne(ctx, attributedQianchuanCostCTE+`
		SELECT COALESCE(SUM(stat_cost), 0) AS ad_cost
		FROM attributed_qianchuan_cost
		WHERE account_id = ?`, args...)
	if err != nil {
		return 0, fmt.Errorf("查询账号双重校验千川消耗失败: %w", err)
	}
	return row["ad_cost"].Float64(), nil
}

// getAdCostByAccounts 查询指定账号集合的单日千川广告消耗（兼容旧版）
func (s *FinanceCommissionService) getAdCostByAccounts(ctx context.Context, accountIds []int, date string) float64 {
	return s.getAdCostByAccountsRange(ctx, accountIds, date, date)
}

// getAdCostByAccountsRange 查询指定账号集合在日期范围内的有效千川消耗。
// 达人日报优先，缺失时才使用已准确匹配账号的计划日报，避免重复计算。
func (s *FinanceCommissionService) getAdCostByAccountsRange(ctx context.Context, accountIds []int, startDate, endDate string) float64 {
	if len(accountIds) == 0 {
		return 0
	}

	startDate, endDate = NormalizeFinanceDateRange(startDate, endDate)
	placeholders := strings.TrimRight(strings.Repeat("?,", len(accountIds)), ",")
	args := []interface{}{startDate, endDate, startDate, endDate}
	for _, accountID := range accountIds {
		args = append(args, accountID)
	}
	row, err := g.DB().GetOne(ctx, effectiveQianchuanCostCTE+fmt.Sprintf(`
		SELECT COALESCE(SUM(stat_cost), 0) AS ad_cost
		FROM effective_qianchuan_cost
		WHERE account_id IN (%s)`, placeholders), args...)
	if err != nil {
		g.Log().Warningf(ctx, "[QianchuanCost] 查询账号有效消耗失败: %v", err)
		return 0
	}
	return row["ad_cost"].Float64()
}

func (s *FinanceCommissionService) getAdCostsByAccountRange(ctx context.Context, accountIds []int, startDate, endDate string) (map[int]float64, error) {
	result := make(map[int]float64, len(accountIds))
	if len(accountIds) == 0 {
		return result, nil
	}
	startDate, endDate = NormalizeFinanceDateRange(startDate, endDate)
	placeholders := strings.TrimRight(strings.Repeat("?,", len(accountIds)), ",")
	args := []interface{}{startDate, endDate, startDate, endDate}
	for _, accountID := range accountIds {
		args = append(args, accountID)
	}
	rows, err := g.DB().GetAll(ctx, effectiveQianchuanCostCTE+fmt.Sprintf(`
		SELECT account_id, COALESCE(SUM(stat_cost), 0) AS ad_cost
		FROM effective_qianchuan_cost
		WHERE account_id IN (%s)
		GROUP BY account_id`, placeholders), args...)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row["account_id"].Int()] = row["ad_cost"].Float64()
	}
	return result, nil
}

// AuthorCostItem 达人消耗明细
type AuthorCostItem struct {
	AccountID      int64   `json:"accountId"`
	TraderID       int64   `json:"traderId"`
	AwemeID        int64   `json:"awemeId"`
	AwemeName      string  `json:"awemeName"`      // 兼容旧字段：抖音昵称
	DouyinNickname string  `json:"douyinNickname"` // 抖音昵称
	AuthorName     string  `json:"authorName"`     // 内部账号表维护的达人名称
	OperatorName   string  `json:"operatorName"`   // 投手名称
	StatCost       float64 `json:"statCost"`
	PayOrderGmv    float64 `json:"payOrderGmv"`
	OrderCount     int64   `json:"orderCount"`
}

// QianchuanCostDetailItem 组织/运营消耗明细
type QianchuanCostDetailItem struct {
	AccountID      int64   `json:"accountId"`
	AwemeID        int64   `json:"awemeId"`
	AwemeName      string  `json:"awemeName"`      // 兼容旧字段：抖音昵称
	DouyinNickname string  `json:"douyinNickname"` // 抖音昵称
	AuthorName     string  `json:"authorName"`     // 内部账号表维护的达人名称
	AdCost         float64 `json:"adCost"`
	PayOrderGmv    float64 `json:"payOrderGmv"`
	OrderCount     int64   `json:"orderCount"`
}

// FinanceOrgScope 统一表示财务看板的组织筛选范围。
// 优先级：trader > department > company。
type FinanceOrgScope struct {
	CompanyId    int64
	DepartmentId int64
	TraderId     int64
}

func (s *FinanceCommissionService) effectiveScopeWhere(ctx context.Context, scope FinanceOrgScope, alias string) (string, []interface{}, error) {
	if alias == "" {
		alias = "effective"
	}
	if scope.TraderId > 0 {
		return fmt.Sprintf(" AND %s.trader_id = ?", alias), []interface{}{scope.TraderId}, nil
	}
	if scope.DepartmentId > 0 {
		deptIds, err := s.getDepartmentTreeIds(ctx, scope.DepartmentId)
		if err != nil {
			return "", nil, err
		}
		if len(deptIds) == 0 {
			return " AND 1=0", nil, nil
		}
		return fmt.Sprintf(" AND %s.department_id IN (%s)", alias, int64Placeholders(len(deptIds))), appendInt64Args(nil, deptIds), nil
	}
	if scope.CompanyId > 0 {
		companyIds, err := s.getCompanyTreeIds(ctx, scope.CompanyId)
		if err != nil {
			return "", nil, err
		}
		if len(companyIds) == 0 {
			return " AND 1=0", nil, nil
		}
		return fmt.Sprintf(" AND %s.company_id IN (%s)", alias, int64Placeholders(len(companyIds))), appendInt64Args(nil, companyIds), nil
	}
	return "", nil, nil
}

func (s *FinanceCommissionService) costRankingBaseWhere(ctx context.Context, scope FinanceOrgScope, dimension, alias string) (string, []interface{}, error) {
	switch dimension {
	case "trader":
		if alias == "" {
			alias = "t"
		}
		if scope.TraderId > 0 {
			return fmt.Sprintf(" AND %s.id = ?", alias), []interface{}{scope.TraderId}, nil
		}
		if scope.DepartmentId > 0 {
			deptIds, err := s.getDepartmentTreeIds(ctx, scope.DepartmentId)
			if err != nil {
				return "", nil, err
			}
			if len(deptIds) == 0 {
				return " AND 1=0", nil, nil
			}
			return fmt.Sprintf(" AND %s.group_id IN (%s)", alias, int64Placeholders(len(deptIds))), appendInt64Args(nil, deptIds), nil
		}
		if scope.CompanyId > 0 {
			companyIds, err := s.getCompanyTreeIds(ctx, scope.CompanyId)
			if err != nil {
				return "", nil, err
			}
			if len(companyIds) == 0 {
				return " AND 1=0", nil, nil
			}
			return fmt.Sprintf(" AND %s.company_id IN (%s)", alias, int64Placeholders(len(companyIds))), appendInt64Args(nil, companyIds), nil
		}
	case "department":
		if alias == "" {
			alias = "d"
		}
		if scope.TraderId > 0 {
			return fmt.Sprintf(" AND %s.id = (SELECT group_id FROM trader WHERE id = ?)", alias), []interface{}{scope.TraderId}, nil
		}
		if scope.DepartmentId > 0 {
			deptIds, err := s.getDepartmentTreeIds(ctx, scope.DepartmentId)
			if err != nil {
				return "", nil, err
			}
			if len(deptIds) == 0 {
				return " AND 1=0", nil, nil
			}
			return fmt.Sprintf(" AND %s.id IN (%s)", alias, int64Placeholders(len(deptIds))), appendInt64Args(nil, deptIds), nil
		}
		if scope.CompanyId > 0 {
			companyIds, err := s.getCompanyTreeIds(ctx, scope.CompanyId)
			if err != nil {
				return "", nil, err
			}
			if len(companyIds) == 0 {
				return " AND 1=0", nil, nil
			}
			return fmt.Sprintf(" AND %s.company_id IN (%s)", alias, int64Placeholders(len(companyIds))), appendInt64Args(nil, companyIds), nil
		}
	case "company":
		if alias == "" {
			alias = "c"
		}
		if scope.TraderId > 0 {
			return fmt.Sprintf(" AND %s.id = (SELECT company_id FROM trader WHERE id = ?)", alias), []interface{}{scope.TraderId}, nil
		}
		if scope.DepartmentId > 0 {
			return fmt.Sprintf(" AND %s.id = (SELECT company_id FROM sys_department WHERE id = ?)", alias), []interface{}{scope.DepartmentId}, nil
		}
		if scope.CompanyId > 0 {
			companyIds, err := s.getCompanyTreeIds(ctx, scope.CompanyId)
			if err != nil {
				return "", nil, err
			}
			if len(companyIds) == 0 {
				return " AND 1=0", nil, nil
			}
			return fmt.Sprintf(" AND %s.id IN (%s)", alias, int64Placeholders(len(companyIds))), appendInt64Args(nil, companyIds), nil
		}
	}
	return "", nil, nil
}

func (s *FinanceCommissionService) storeAdvertiserScopeWhere(ctx context.Context, scope FinanceOrgScope, advertiserAlias string) (string, []interface{}, error) {
	if advertiserAlias == "" {
		advertiserAlias = "adv"
	}
	advertiserColumn := fmt.Sprintf("%s.advertiser_id", advertiserAlias)
	if scope.TraderId > 0 {
		return fmt.Sprintf(`
			AND EXISTS (
				SELECT 1
				FROM trader_account_relation tar_scope
				INNER JOIN account_qianchuan_mapping aqm_scope
					ON aqm_scope.account_id = tar_scope.account_id
				INNER JOIN account a_scope
					ON a_scope.id = tar_scope.account_id
				WHERE tar_scope.trader_id = ?
				  AND a_scope.platform = 'douyin'
				  AND a_scope.status = 1
				  AND aqm_scope.qianchuan_advertiser_id = %s
			)`, advertiserColumn), []interface{}{scope.TraderId}, nil
	}
	if scope.DepartmentId > 0 {
		deptIds, err := s.getDepartmentTreeIds(ctx, scope.DepartmentId)
		if err != nil {
			return "", nil, err
		}
		if len(deptIds) == 0 {
			return " AND 1=0", nil, nil
		}
		return fmt.Sprintf(`
			AND EXISTS (
				SELECT 1
				FROM trader t_scope
				INNER JOIN trader_account_relation tar_scope
					ON tar_scope.trader_id = t_scope.id
				INNER JOIN account_qianchuan_mapping aqm_scope
					ON aqm_scope.account_id = tar_scope.account_id
				WHERE t_scope.status = 1
				  AND t_scope.group_id IN (%s)
				  AND aqm_scope.qianchuan_advertiser_id = %s
			)`, int64Placeholders(len(deptIds)), advertiserColumn), appendInt64Args(nil, deptIds), nil
	}
	if scope.CompanyId > 0 {
		companyIds, err := s.getCompanyTreeIds(ctx, scope.CompanyId)
		if err != nil {
			return "", nil, err
		}
		if len(companyIds) == 0 {
			return " AND 1=0", nil, nil
		}
		return fmt.Sprintf(`
			AND EXISTS (
				SELECT 1
				FROM account_qianchuan_mapping aqm_scope
				WHERE aqm_scope.company_id IN (%s)
				  AND aqm_scope.qianchuan_advertiser_id = %s
			)`, int64Placeholders(len(companyIds)), advertiserColumn), appendInt64Args(nil, companyIds), nil
	}
	return "", nil, nil
}

// QianchuanStoreCostItem 店铺消耗统计。
type QianchuanStoreCostItem struct {
	StoreID      int64  `json:"storeId"`
	StoreName    string `json:"storeName"`
	StoreStatus  int    `json:"storeStatus"`
	MatchedStore bool   `json:"matchedStore"`
	// 以下主体字段保留一段时间，兼容旧接口调用方。
	SubjectID       int64   `json:"subjectId"`
	AdvertiserID    int64   `json:"advertiserId"`
	AdvertiserName  string  `json:"advertiserName"`
	SubjectName     string  `json:"subjectName"`
	AccountName     string  `json:"accountName"`
	Status          string  `json:"status"`
	StatusType      string  `json:"statusType"`
	StatCost        float64 `json:"statCost"`
	PayOrderAmount  float64 `json:"payOrderAmount"`
	AllOrderPayGmv  float64 `json:"allOrderPayGmv"`
	ROI             float64 `json:"roi"`
	Percent         float64 `json:"percent"`
	MappingCount    int64   `json:"mappingCount"`
	AdvertiserCount int64   `json:"advertiserCount"`
	MatchedSubject  bool    `json:"matchedSubject"`
}

// GetStoreCostStats 按店铺汇总千川广告主账户消耗。
func (s *FinanceCommissionService) GetStoreCostStats(ctx context.Context, startDate, endDate string, limit int, scope FinanceOrgScope) ([]QianchuanStoreCostItem, error) {
	startDate, endDate = NormalizeFinanceDateRange(startDate, endDate)
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return nil, fmt.Errorf("开始日期格式错误，期望 2006-01-02: %s", startDate)
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		return nil, fmt.Errorf("结束日期格式错误，期望 2006-01-02: %s", endDate)
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	scopeWhere, scopeArgs, scopeErr := s.storeAdvertiserScopeWhere(ctx, scope, "adv")
	if scopeErr != nil {
		return nil, scopeErr
	}
	args := []interface{}{startDate, endDate}
	args = append(args, scopeArgs...)
	args = append(args, limit)
	rows, err := g.DB().GetAll(ctx, `
		SELECT
			COALESCE(st.id, 0) AS store_id,
			COALESCE(NULLIF(st.store_name, ''), '未绑定店铺') AS store_name,
			COALESCE(st.status, 0) AS store_status,
			COUNT(DISTINCT adv.advertiser_id) AS advertiser_count,
			COALESCE(SUM(adv.stat_cost), 0) AS stat_cost,
			COALESCE(SUM(adv.pay_order_amount), 0) AS pay_order_amount,
			COALESCE(SUM(adv.all_order_pay_gmv), 0) AS all_order_pay_gmv,
			COUNT(DISTINCT CASE WHEN st.id IS NOT NULL THEN adv.advertiser_id END) AS mapping_count,
			MAX(CASE WHEN st.id IS NOT NULL THEN 1 ELSE 0 END) AS matched_store
		FROM (
			SELECT
				advertiser_id,
				COALESCE(SUM(stat_cost), 0) AS stat_cost,
				COALESCE(SUM(pay_order_amount), 0) AS pay_order_amount,
				COALESCE(SUM(all_order_pay_gmv_7days), 0) AS all_order_pay_gmv
			FROM qianchuan_advertiser_daily_report
			WHERE LEFT(stat_datetime, 10) >= ?
			  AND LEFT(stat_datetime, 10) <= ?
			GROUP BY advertiser_id
		) adv
		LEFT JOIN store_qianchuan_mapping sqm
			ON sqm.qianchuan_advertiser_id = adv.advertiser_id
		LEFT JOIN store st ON st.id = sqm.store_id
		WHERE 1=1`+scopeWhere+`
		GROUP BY COALESCE(st.id, 0), COALESCE(NULLIF(st.store_name, ''), '未绑定店铺'), COALESCE(st.status, 0)
		ORDER BY stat_cost DESC
		LIMIT ?
	`, args...)
	if err != nil {
		return nil, err
	}

	result := make([]QianchuanStoreCostItem, len(rows))
	maxCost := 0.0
	for _, row := range rows {
		if cost := row["stat_cost"].Float64(); cost > maxCost {
			maxCost = cost
		}
	}

	for i, row := range rows {
		statCost := row["stat_cost"].Float64()
		payOrderAmount := row["pay_order_amount"].Float64()
		allOrderPayGmv := row["all_order_pay_gmv"].Float64()
		roiAmount := payOrderAmount
		if roiAmount <= 0 {
			roiAmount = allOrderPayGmv
		}
		roi := 0.0
		if statCost > 0 {
			roi = roiAmount / statCost
		}

		matchedStore := row["matched_store"].Int() > 0
		status := "待绑定店铺"
		statusType := "warning"
		if matchedStore && row["store_status"].Int() == 1 {
			status = "已归属"
			statusType = "known"
		} else if matchedStore {
			status = "店铺已停用"
		}

		percent := 0.0
		if maxCost > 0 {
			percent = statCost / maxCost * 100
		}

		storeID := row["store_id"].Int64()
		storeName := row["store_name"].String()
		result[i] = QianchuanStoreCostItem{
			StoreID:         storeID,
			StoreName:       storeName,
			StoreStatus:     row["store_status"].Int(),
			MatchedStore:    matchedStore,
			SubjectID:       storeID,
			AdvertiserID:    storeID,
			AdvertiserName:  storeName,
			SubjectName:     storeName,
			AccountName:     fmt.Sprintf("%d 个千川账户", row["advertiser_count"].Int64()),
			Status:          status,
			StatusType:      statusType,
			StatCost:        roundFloatFinance(statCost, 2),
			PayOrderAmount:  roundFloatFinance(payOrderAmount, 2),
			AllOrderPayGmv:  roundFloatFinance(allOrderPayGmv, 2),
			ROI:             roundFloatFinance(roi, 2),
			Percent:         roundFloatFinance(percent, 2),
			MappingCount:    row["mapping_count"].Int64(),
			AdvertiserCount: row["advertiser_count"].Int64(),
			MatchedSubject:  matchedStore,
		}
	}
	return result, nil
}

// GetSubjectCostStats 兼容旧主体消耗接口，数据已统一为店铺口径。
func (s *FinanceCommissionService) GetSubjectCostStats(ctx context.Context, startDate, endDate string, limit int, scope FinanceOrgScope) ([]QianchuanStoreCostItem, error) {
	return s.GetStoreCostStats(ctx, startDate, endDate, limit, scope)
}

// QianchuanStoreCostDetailItem 店铺消耗明细（按千川广告主账户维度）。
type QianchuanStoreCostDetailItem struct {
	StoreID        int64   `json:"storeId"`
	StoreName      string  `json:"storeName"`
	AdvertiserID   int64   `json:"advertiserId"`
	AdvertiserName string  `json:"advertiserName"`
	AccountID      int64   `json:"accountId"`
	AccountName    string  `json:"accountName"`
	UID            string  `json:"uid"`
	TraderName     string  `json:"traderName"`
	StatCost       float64 `json:"statCost"`
	PayOrderAmount float64 `json:"payOrderAmount"`
	AllOrderPayGmv float64 `json:"allOrderPayGmv"`
	ShowCnt        int64   `json:"showCnt"`
	ClickCnt       int64   `json:"clickCnt"`
	PayOrderCount  int64   `json:"payOrderCount"`
	ROI            float64 `json:"roi"`
}

// GetStoreCostDetail 查询指定店铺下所有千川广告主账户的消耗明细。
// storeId 为 store.id，0 表示未绑定到有效店铺的账户。
func (s *FinanceCommissionService) GetStoreCostDetail(ctx context.Context, startDate, endDate string, storeId int64, page, pageSize int, scope FinanceOrgScope) ([]QianchuanStoreCostDetailItem, int64, error) {
	startDate, endDate = NormalizeFinanceDateRange(startDate, endDate)
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return nil, 0, fmt.Errorf("开始日期格式错误，期望 2006-01-02: %s", startDate)
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		return nil, 0, fmt.Errorf("结束日期格式错误，期望 2006-01-02: %s", endDate)
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 500 {
		pageSize = 500
	}
	offset := (page - 1) * pageSize
	scopeWhere, scopeArgs, scopeErr := s.storeAdvertiserScopeWhere(ctx, scope, "adv")
	if scopeErr != nil {
		return nil, 0, scopeErr
	}

	// 统计总数
	countArgs := []interface{}{startDate, endDate, storeId}
	countArgs = append(countArgs, scopeArgs...)
	countRow, err := g.DB().GetOne(ctx, `
		SELECT COUNT(*) AS total FROM (
			SELECT adv.advertiser_id
			FROM (
				SELECT advertiser_id
				FROM qianchuan_advertiser_daily_report
				WHERE LEFT(stat_datetime, 10) >= ? AND LEFT(stat_datetime, 10) <= ?
				GROUP BY advertiser_id
			) adv
			LEFT JOIN store_qianchuan_mapping sqm ON sqm.qianchuan_advertiser_id = adv.advertiser_id
			LEFT JOIN store st ON st.id = sqm.store_id
			WHERE COALESCE(st.id, 0) = ?
			`+scopeWhere+`
			GROUP BY adv.advertiser_id
		) t
	`, countArgs...)
	if err != nil {
		return nil, 0, err
	}
	total := countRow["total"].Int64()

	listArgs := []interface{}{startDate, endDate, storeId}
	listArgs = append(listArgs, scopeArgs...)
	listArgs = append(listArgs, offset, pageSize)
	rows, err := g.DB().GetAll(ctx, `
		SELECT
			COALESCE(st.id, 0) AS store_id,
			COALESCE(NULLIF(MAX(st.store_name), ''), '未绑定店铺') AS store_name,
			adv.advertiser_id,
			COALESCE(
				NULLIF(MAX(adv.advertiser_name), ''),
				NULLIF(MAX(sqm.qianchuan_advertiser_name), ''),
				NULLIF(MAX(r.advertiser_name), ''),
				CONCAT('千川账户 ', adv.advertiser_id)
			) AS advertiser_name,
			COALESCE(NULLIF(MAX(a.platform_nickname), ''), NULLIF(MAX(a.account_name), ''), '') AS account_name,
			COALESCE(MAX(a.id), 0) AS account_id,
			COALESCE(NULLIF(MAX(a.uid), ''), '') AS uid,
			COALESCE(NULLIF(MAX(t.name), ''), '') AS trader_name,
			MAX(adv.stat_cost) AS stat_cost,
			MAX(adv.pay_order_amount) AS pay_order_amount,
			MAX(adv.all_order_pay_gmv) AS all_order_pay_gmv,
			MAX(adv.show_cnt) AS show_cnt,
			MAX(adv.click_cnt) AS click_cnt,
			MAX(adv.pay_order_count) AS pay_order_count
		FROM (
			SELECT
				advertiser_id,
				MAX(NULLIF(advertiser_name, '')) AS advertiser_name,
				COALESCE(SUM(stat_cost), 0) AS stat_cost,
				COALESCE(SUM(pay_order_amount), 0) AS pay_order_amount,
				COALESCE(SUM(all_order_pay_gmv_7days), 0) AS all_order_pay_gmv,
				COALESCE(SUM(show_cnt), 0) AS show_cnt,
				COALESCE(SUM(click_cnt), 0) AS click_cnt,
				COALESCE(SUM(pay_order_count), 0) AS pay_order_count
			FROM qianchuan_advertiser_daily_report
			WHERE LEFT(stat_datetime, 10) >= ? AND LEFT(stat_datetime, 10) <= ?
			GROUP BY advertiser_id
		) adv
		LEFT JOIN store_qianchuan_mapping sqm ON sqm.qianchuan_advertiser_id = adv.advertiser_id
		LEFT JOIN store st ON st.id = sqm.store_id
		LEFT JOIN account_qianchuan_mapping aqm ON aqm.qianchuan_advertiser_id = adv.advertiser_id
		LEFT JOIN account a ON a.id = aqm.account_id
		LEFT JOIN trader_account_relation tar ON tar.account_id = a.id
		LEFT JOIN trader t ON t.id = tar.trader_id
		LEFT JOIN (
			SELECT advertiser_id, MAX(advertiser_name) AS advertiser_name
			FROM qianchuan_ad_report
			GROUP BY advertiser_id
		) r ON r.advertiser_id = adv.advertiser_id
		WHERE COALESCE(st.id, 0) = ?
		`+scopeWhere+`
		GROUP BY adv.advertiser_id
		ORDER BY stat_cost DESC
		LIMIT ?, ?
	`, listArgs...)
	if err != nil {
		return nil, 0, err
	}

	return buildStoreCostDetailItems(rows), total, nil
}

func buildStoreCostDetailItems(rows gdb.Result) []QianchuanStoreCostDetailItem {
	result := make([]QianchuanStoreCostDetailItem, len(rows))
	for i, row := range rows {
		statCost := row["stat_cost"].Float64()
		payOrderAmount := row["pay_order_amount"].Float64()
		allOrderPayGmv := row["all_order_pay_gmv"].Float64()
		roiAmount := payOrderAmount
		if roiAmount <= 0 {
			roiAmount = allOrderPayGmv
		}
		roi := 0.0
		if statCost > 0 {
			roi = roiAmount / statCost
		}

		result[i] = QianchuanStoreCostDetailItem{
			StoreID:        row["store_id"].Int64(),
			StoreName:      row["store_name"].String(),
			AdvertiserID:   row["advertiser_id"].Int64(),
			AdvertiserName: row["advertiser_name"].String(),
			AccountID:      row["account_id"].Int64(),
			AccountName:    row["account_name"].String(),
			UID:            row["uid"].String(),
			TraderName:     row["trader_name"].String(),
			StatCost:       roundFloatFinance(statCost, 2),
			PayOrderAmount: roundFloatFinance(payOrderAmount, 2),
			AllOrderPayGmv: roundFloatFinance(allOrderPayGmv, 2),
			ShowCnt:        row["show_cnt"].Int64(),
			ClickCnt:       row["click_cnt"].Int64(),
			PayOrderCount:  row["pay_order_count"].Int64(),
			ROI:            roundFloatFinance(roi, 2),
		}
	}
	return result
}

// GetAllStoreCostDetail 查询全部店铺下全部千川广告主账户的消耗明细。
func (s *FinanceCommissionService) GetAllStoreCostDetail(ctx context.Context, startDate, endDate string, scope FinanceOrgScope) ([]QianchuanStoreCostDetailItem, error) {
	startDate, endDate = NormalizeFinanceDateRange(startDate, endDate)
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return nil, fmt.Errorf("开始日期格式错误，期望 2006-01-02: %s", startDate)
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		return nil, fmt.Errorf("结束日期格式错误，期望 2006-01-02: %s", endDate)
	}

	scopeWhere, scopeArgs, scopeErr := s.storeAdvertiserScopeWhere(ctx, scope, "adv")
	if scopeErr != nil {
		return nil, scopeErr
	}
	args := []interface{}{startDate, endDate}
	args = append(args, scopeArgs...)
	rows, err := g.DB().GetAll(ctx, `
		SELECT
			COALESCE(st.id, 0) AS store_id,
			COALESCE(NULLIF(MAX(st.store_name), ''), '未绑定店铺') AS store_name,
			adv.advertiser_id,
			COALESCE(
				NULLIF(MAX(adv.advertiser_name), ''),
				NULLIF(MAX(sqm.qianchuan_advertiser_name), ''),
				NULLIF(MAX(r.advertiser_name), ''),
				CONCAT('千川账户 ', adv.advertiser_id)
			) AS advertiser_name,
			COALESCE(NULLIF(MAX(a.platform_nickname), ''), NULLIF(MAX(a.account_name), ''), '') AS account_name,
			COALESCE(MAX(a.id), 0) AS account_id,
			COALESCE(NULLIF(MAX(a.uid), ''), '') AS uid,
			COALESCE(NULLIF(MAX(t.name), ''), '') AS trader_name,
			MAX(adv.stat_cost) AS stat_cost,
			MAX(adv.pay_order_amount) AS pay_order_amount,
			MAX(adv.all_order_pay_gmv) AS all_order_pay_gmv,
			MAX(adv.show_cnt) AS show_cnt,
			MAX(adv.click_cnt) AS click_cnt,
			MAX(adv.pay_order_count) AS pay_order_count
		FROM (
			SELECT
				advertiser_id,
				MAX(NULLIF(advertiser_name, '')) AS advertiser_name,
				COALESCE(SUM(stat_cost), 0) AS stat_cost,
				COALESCE(SUM(pay_order_amount), 0) AS pay_order_amount,
				COALESCE(SUM(all_order_pay_gmv_7days), 0) AS all_order_pay_gmv,
				COALESCE(SUM(show_cnt), 0) AS show_cnt,
				COALESCE(SUM(click_cnt), 0) AS click_cnt,
				COALESCE(SUM(pay_order_count), 0) AS pay_order_count
			FROM qianchuan_advertiser_daily_report
			WHERE LEFT(stat_datetime, 10) >= ? AND LEFT(stat_datetime, 10) <= ?
			GROUP BY advertiser_id
		) adv
		LEFT JOIN store_qianchuan_mapping sqm ON sqm.qianchuan_advertiser_id = adv.advertiser_id
		LEFT JOIN store st ON st.id = sqm.store_id
		LEFT JOIN account_qianchuan_mapping aqm ON aqm.qianchuan_advertiser_id = adv.advertiser_id
		LEFT JOIN account a ON a.id = aqm.account_id
		LEFT JOIN trader_account_relation tar ON tar.account_id = a.id
		LEFT JOIN trader t ON t.id = tar.trader_id
		LEFT JOIN (
			SELECT advertiser_id, MAX(advertiser_name) AS advertiser_name
			FROM qianchuan_ad_report
			GROUP BY advertiser_id
		) r ON r.advertiser_id = adv.advertiser_id
		WHERE 1=1`+scopeWhere+`
		GROUP BY COALESCE(st.id, 0), adv.advertiser_id
		ORDER BY store_name ASC, stat_cost DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	return buildStoreCostDetailItems(rows), nil
}

// GetSubjectCostDetail 兼容旧主体明细接口，subjectId 参数按 storeId 解释。
func (s *FinanceCommissionService) GetSubjectCostDetail(ctx context.Context, startDate, endDate string, subjectId int64, page, pageSize int, scope FinanceOrgScope) ([]QianchuanStoreCostDetailItem, int64, error) {
	return s.GetStoreCostDetail(ctx, startDate, endDate, subjectId, page, pageSize, scope)
}

// GetAuthorCostRanking 查询达人维度消耗统计
func (s *FinanceCommissionService) GetAuthorCostRanking(ctx context.Context, startDate, endDate string, page, pageSize int, scope FinanceOrgScope) ([]AuthorCostItem, int64, error) {
	startDate, endDate = NormalizeFinanceDateRange(startDate, endDate)
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return nil, 0, fmt.Errorf("开始日期格式错误，期望 2006-01-02: %s", startDate)
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		return nil, 0, fmt.Errorf("结束日期格式错误，期望 2006-01-02: %s", endDate)
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100000 {
		pageSize = 100000
	}
	offset := (page - 1) * pageSize

	scopeWhere, scopeArgs, scopeErr := s.effectiveScopeWhere(ctx, scope, "effective")
	if scopeErr != nil {
		return nil, 0, scopeErr
	}
	totalArgs := []interface{}{startDate, endDate, startDate, endDate}
	totalArgs = append(totalArgs, scopeArgs...)
	totalRow, err := g.DB().GetOne(ctx, effectiveQianchuanCostCTE+`
		SELECT COUNT(1) AS total
		FROM (
			SELECT effective.account_id, effective.trader_id
			FROM effective_qianchuan_cost effective
			WHERE 1=1`+scopeWhere+`
			GROUP BY effective.account_id, effective.trader_id
		) grouped`, totalArgs...)
	if err != nil {
		return nil, 0, err
	}
	total := totalRow["total"].Int64()

	listArgs := []interface{}{startDate, endDate, startDate, endDate}
	listArgs = append(listArgs, scopeArgs...)
	listArgs = append(listArgs, pageSize, offset)
	rows, err := g.DB().GetAll(ctx, effectiveQianchuanCostCTE+`
		SELECT
			effective.account_id,
			effective.trader_id,
			MAX(effective.aweme_id) AS aweme_id,
			COALESCE(MAX(NULLIF(effective.aweme_name, '')), MAX(NULLIF(a.platform_nickname, '')), '') AS douyin_nickname,
			COALESCE(MAX(NULLIF(a.account_name, '')), MAX(NULLIF(effective.aweme_name, '')), '') AS author_name,
			COALESCE(MAX(NULLIF(t.name, '')), MAX(NULLIF(a.operator, '')), '') AS operator_name,
			COALESCE(SUM(effective.stat_cost), 0) AS stat_cost,
			COALESCE(SUM(effective.pay_order_gmv), 0) AS pay_order_gmv,
			COALESCE(SUM(effective.pay_order_count), 0) AS pay_order_count
		FROM effective_qianchuan_cost effective
		LEFT JOIN account a ON a.id = effective.account_id
		LEFT JOIN trader t ON t.id = effective.trader_id
		WHERE 1=1`+scopeWhere+`
		GROUP BY effective.account_id, effective.trader_id
		ORDER BY stat_cost DESC
		LIMIT ? OFFSET ?
	`, listArgs...)
	if err != nil {
		return nil, 0, err
	}

	result := make([]AuthorCostItem, len(rows))
	for i, row := range rows {
		douyinNickname := row["douyin_nickname"].String()
		result[i] = AuthorCostItem{
			AccountID:      row["account_id"].Int64(),
			TraderID:       row["trader_id"].Int64(),
			AwemeID:        row["aweme_id"].Int64(),
			AwemeName:      douyinNickname,
			DouyinNickname: douyinNickname,
			AuthorName:     row["author_name"].String(),
			OperatorName:   row["operator_name"].String(),
			StatCost:       roundFloatFinance(row["stat_cost"].Float64(), 2),
			PayOrderGmv:    roundFloatFinance(row["pay_order_gmv"].Float64(), 2),
			OrderCount:     row["pay_order_count"].Int64(),
		}
	}
	return result, total, nil
}

// GetCostRanking 查询消耗排行
func (s *FinanceCommissionService) GetCostRanking(ctx context.Context, startDate, endDate, dimension string, limit int, scope FinanceOrgScope) ([]g.Map, error) {
	startDate, endDate = NormalizeFinanceDateRange(startDate, endDate)
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return nil, fmt.Errorf("开始日期格式错误，期望 2006-01-02: %s", startDate)
	}
	if _, err := time.Parse("2006-01-02", endDate); err != nil {
		return nil, fmt.Errorf("结束日期格式错误，期望 2006-01-02: %s", endDate)
	}
	if limit <= 0 {
		limit = 20
	}

	switch dimension {
	case "trader":
		return s.costRankingByTrader(ctx, startDate, endDate, limit, scope)
	case "department":
		return s.costRankingByDept(ctx, startDate, endDate, limit, scope)
	case "company":
		return s.costRankingByCompany(ctx, startDate, endDate, limit, scope)
	default:
		return nil, fmt.Errorf("不支持的维度: %s，支持: trader/department/company", dimension)
	}
}

func (s *FinanceCommissionService) costRankingByTrader(ctx context.Context, startDate, endDate string, limit int, scope FinanceOrgScope) ([]g.Map, error) {
	joinScopeWhere, joinScopeArgs, scopeErr := s.effectiveScopeWhere(ctx, scope, "effective")
	if scopeErr != nil {
		return nil, scopeErr
	}
	baseWhere, baseArgs, baseErr := s.costRankingBaseWhere(ctx, scope, "trader", "t")
	if baseErr != nil {
		return nil, baseErr
	}
	args := []interface{}{startDate, endDate, startDate, endDate}
	args = append(args, joinScopeArgs...)
	args = append(args, baseArgs...)
	args = append(args, limit)
	rows, err := g.DB().GetAll(ctx, effectiveQianchuanCostCTE+`
		SELECT t.id, t.name, COALESCE(SUM(effective.stat_cost), 0) AS ad_cost
		FROM trader t
		LEFT JOIN effective_qianchuan_cost effective
			ON effective.trader_id = t.id`+joinScopeWhere+`
		WHERE t.status = 1`+baseWhere+`
		GROUP BY t.id, t.name
		ORDER BY ad_cost DESC, t.id ASC
		LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	result := make([]g.Map, len(rows))
	for i, row := range rows {
		traderID := row["id"].Int64()
		result[i] = g.Map{
			"id":          row["id"],
			"name":        row["name"],
			"adCost":      roundFloatFinance(row["ad_cost"].Float64(), 2),
			"costDetails": s.costDetailsByDimension(ctx, startDate, endDate, "trader", traderID),
		}
	}
	return result, nil
}

func (s *FinanceCommissionService) costRankingByDept(ctx context.Context, startDate, endDate string, limit int, scope FinanceOrgScope) ([]g.Map, error) {
	joinScopeWhere, joinScopeArgs, scopeErr := s.effectiveScopeWhere(ctx, scope, "effective")
	if scopeErr != nil {
		return nil, scopeErr
	}
	baseWhere, baseArgs, baseErr := s.costRankingBaseWhere(ctx, scope, "department", "d")
	if baseErr != nil {
		return nil, baseErr
	}
	args := []interface{}{startDate, endDate, startDate, endDate}
	args = append(args, joinScopeArgs...)
	args = append(args, baseArgs...)
	args = append(args, limit)
	rows, err := g.DB().GetAll(ctx, effectiveQianchuanCostCTE+`
		SELECT d.id, d.name, COALESCE(SUM(effective.stat_cost), 0) AS ad_cost
		FROM sys_department d
		LEFT JOIN effective_qianchuan_cost effective
			ON effective.department_id = d.id`+joinScopeWhere+`
		WHERE d.status = 1`+baseWhere+`
		GROUP BY d.id, d.name
		ORDER BY ad_cost DESC, d.sort ASC, d.id ASC
		LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	result := make([]g.Map, len(rows))
	for i, row := range rows {
		departmentID := row["id"].Int64()
		result[i] = g.Map{
			"id":          row["id"],
			"name":        row["name"],
			"adCost":      roundFloatFinance(row["ad_cost"].Float64(), 2),
			"costDetails": s.costDetailsByDimension(ctx, startDate, endDate, "department", departmentID),
		}
	}
	return result, nil
}

func (s *FinanceCommissionService) costRankingByCompany(ctx context.Context, startDate, endDate string, limit int, scope FinanceOrgScope) ([]g.Map, error) {
	joinScopeWhere, joinScopeArgs, scopeErr := s.effectiveScopeWhere(ctx, scope, "effective")
	if scopeErr != nil {
		return nil, scopeErr
	}
	baseWhere, baseArgs, baseErr := s.costRankingBaseWhere(ctx, scope, "company", "c")
	if baseErr != nil {
		return nil, baseErr
	}
	args := []interface{}{startDate, endDate, startDate, endDate}
	args = append(args, joinScopeArgs...)
	args = append(args, baseArgs...)
	args = append(args, limit)
	rows, err := g.DB().GetAll(ctx, effectiveQianchuanCostCTE+`
		SELECT c.id, c.name, COALESCE(SUM(effective.stat_cost), 0) AS ad_cost
		FROM sys_company c
		LEFT JOIN effective_qianchuan_cost effective
			ON effective.company_id = c.id`+joinScopeWhere+`
		WHERE c.status = 1`+baseWhere+`
		GROUP BY c.id, c.name
		ORDER BY ad_cost DESC, c.sort ASC, c.id ASC
		LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	result := make([]g.Map, len(rows))
	for i, row := range rows {
		companyID := row["id"].Int64()
		result[i] = g.Map{
			"id":          row["id"],
			"name":        row["name"],
			"adCost":      roundFloatFinance(row["ad_cost"].Float64(), 2),
			"costDetails": s.costDetailsByDimension(ctx, startDate, endDate, "company", companyID),
		}
	}
	return result, nil
}

func (s *FinanceCommissionService) costDetailsByDimension(ctx context.Context, startDate, endDate, dimension string, id int64) []QianchuanCostDetailItem {
	whereColumn := map[string]string{
		"trader":     "effective.trader_id",
		"department": "effective.department_id",
		"company":    "effective.company_id",
	}[dimension]
	if whereColumn == "" || id <= 0 {
		return []QianchuanCostDetailItem{}
	}

	rows, err := g.DB().GetAll(ctx, effectiveQianchuanCostCTE+fmt.Sprintf(`
		SELECT
			effective.account_id,
			MAX(effective.aweme_id) AS aweme_id,
			COALESCE(MAX(NULLIF(effective.aweme_name, '')), MAX(NULLIF(a.platform_nickname, '')), '') AS douyin_nickname,
			COALESCE(MAX(NULLIF(a.account_name, '')), MAX(NULLIF(effective.aweme_name, '')), '') AS author_name,
			COALESCE(SUM(effective.stat_cost), 0) AS ad_cost,
			COALESCE(SUM(effective.pay_order_gmv), 0) AS pay_order_gmv,
			COALESCE(SUM(effective.pay_order_count), 0) AS pay_order_count
		FROM effective_qianchuan_cost effective
		LEFT JOIN account a ON a.id = effective.account_id
		WHERE %s = ?
		GROUP BY effective.account_id
		ORDER BY ad_cost DESC
	`, whereColumn), startDate, endDate, startDate, endDate, id)
	if err != nil {
		g.Log().Warningf(ctx, "[QianchuanCostDetail] 查询%s=%d消耗明细失败: %v", dimension, id, err)
		return []QianchuanCostDetailItem{}
	}

	result := make([]QianchuanCostDetailItem, len(rows))
	for i, row := range rows {
		douyinNickname := row["douyin_nickname"].String()
		result[i] = QianchuanCostDetailItem{
			AccountID:      row["account_id"].Int64(),
			AwemeID:        row["aweme_id"].Int64(),
			AwemeName:      douyinNickname,
			DouyinNickname: douyinNickname,
			AuthorName:     row["author_name"].String(),
			AdCost:         roundFloatFinance(row["ad_cost"].Float64(), 2),
			PayOrderGmv:    roundFloatFinance(row["pay_order_gmv"].Float64(), 2),
			OrderCount:     row["pay_order_count"].Int64(),
		}
	}
	return result
}

// ExportAuthorCostRanking 导出达人维度消耗统计全量数据
func (s *FinanceCommissionService) ExportAuthorCostRanking(ctx context.Context, startDate, endDate string, scope FinanceOrgScope) (string, error) {
	list, _, err := s.GetAuthorCostRanking(ctx, startDate, endDate, 1, 1000000, scope)
	if err != nil {
		return "", err
	}
	return writeExcelFile("达人维度消耗统计", []string{
		"排名", "抖音账号昵称", "投手名称", "达人ID", "千川消耗(元)", "成交金额(元)", "成交订单数",
	}, []float64{10, 28, 20, 24, 18, 18, 16}, len(list), func(rowIdx int) []interface{} {
		item := list[rowIdx]
		return []interface{}{
			rowIdx + 1,
			item.DouyinNickname,
			item.OperatorName,
			fmt.Sprintf("%d", item.AwemeID),
			item.StatCost,
			item.PayOrderGmv,
			item.OrderCount,
		}
	})
}

// ExportCostRanking 导出千川消耗统计全量数据
func (s *FinanceCommissionService) ExportCostRanking(ctx context.Context, startDate, endDate, dimension string, scope FinanceOrgScope) (string, error) {
	list, err := s.GetCostRanking(ctx, startDate, endDate, dimension, 1000000, scope)
	if err != nil {
		return "", err
	}
	dimensionName := map[string]string{
		"trader":     "达人统计",
		"department": "部门统计",
		"company":    "公司统计",
	}[dimension]
	if dimensionName == "" {
		dimensionName = dimension
	}
	return writeExcelFile("千川消耗统计", []string{
		"排名", "统计维度", "名称", "千川消耗(元)",
	}, []float64{10, 18, 28, 18}, len(list), func(rowIdx int) []interface{} {
		item := list[rowIdx]
		return []interface{}{
			rowIdx + 1,
			dimensionName,
			gconv.String(item["name"]),
			roundFloatFinance(gconv.Float64(item["adCost"]), 2),
		}
	})
}

// ExportCostDetails 导出某个统计行下的账号消耗明细
func (s *FinanceCommissionService) ExportCostDetails(ctx context.Context, startDate, endDate, dimension string, id int64) (string, error) {
	list := s.costDetailsByDimension(ctx, startDate, endDate, dimension, id)
	return writeExcelFile("账号消耗明细", []string{
		"排名", "抖音账号昵称", "达人名称", "达人ID", "实际消耗(元)", "成交金额(元)", "成交订单数",
	}, []float64{10, 28, 28, 24, 18, 18, 16}, len(list), func(rowIdx int) []interface{} {
		item := list[rowIdx]
		return []interface{}{
			rowIdx + 1,
			item.DouyinNickname,
			item.AuthorName,
			fmt.Sprintf("%d", item.AwemeID),
			item.AdCost,
			item.PayOrderGmv,
			item.OrderCount,
		}
	})
}

// ExportStoreCostDetail 导出店铺消耗明细（按千川广告主账户维度，全量数据）。
func (s *FinanceCommissionService) ExportStoreCostDetail(ctx context.Context, startDate, endDate string, storeId int64, scope FinanceOrgScope) (string, error) {
	list, _, err := s.GetStoreCostDetail(ctx, startDate, endDate, storeId, 1, 1000000, scope)
	if err != nil {
		return "", err
	}
	return writeExcelFile("店铺消耗明细", []string{
		"排名", "千川账户ID", "千川账户名称", "千川消耗(¥)", "成交金额(¥)", "成交订单数", "展示数", "点击数", "ROI",
	}, []float64{8, 18, 24, 16, 16, 12, 12, 12, 10}, len(list), func(rowIdx int) []interface{} {
		item := list[rowIdx]
		return []interface{}{
			rowIdx + 1,
			fmt.Sprintf("%d", item.AdvertiserID),
			item.AdvertiserName,
			item.StatCost,
			item.PayOrderAmount,
			item.PayOrderCount,
			item.ShowCnt,
			item.ClickCnt,
			item.ROI,
		}
	})
}

// ExportAllStoreCostDetail 导出全部店铺下全部千川账户消耗明细。
func (s *FinanceCommissionService) ExportAllStoreCostDetail(ctx context.Context, startDate, endDate string, scope FinanceOrgScope) (string, error) {
	list, err := s.GetAllStoreCostDetail(ctx, startDate, endDate, scope)
	if err != nil {
		return "", err
	}
	return writeExcelFile("全部账户消耗明细", []string{
		"店铺名称", "排名", "千川账户ID", "千川账户名称", "千川消耗(¥)", "成交金额(¥)", "成交订单数", "展示数", "点击数", "ROI",
	}, []float64{28, 8, 18, 24, 16, 16, 12, 12, 12, 10}, len(list), func(rowIdx int) []interface{} {
		item := list[rowIdx]
		return []interface{}{
			item.StoreName,
			rowIdx + 1,
			fmt.Sprintf("%d", item.AdvertiserID),
			item.AdvertiserName,
			item.StatCost,
			item.PayOrderAmount,
			item.PayOrderCount,
			item.ShowCnt,
			item.ClickCnt,
			item.ROI,
		}
	})
}

// ExportSubjectCostDetail 兼容旧主体明细导出接口，subjectId 参数按 storeId 解释。
func (s *FinanceCommissionService) ExportSubjectCostDetail(ctx context.Context, startDate, endDate string, subjectId int64, scope FinanceOrgScope) (string, error) {
	return s.ExportStoreCostDetail(ctx, startDate, endDate, subjectId, scope)
}

// OverallDetailExportItem 乘方明细导出项（供 controller 层填充）
type OverallDetailExportItem struct {
	AnchorName                                 string  `json:"anchorName"`
	AnchorShowID                               string  `json:"anchorShowId"`
	StatCostForOverallRoi2                     float64 `json:"statCostForOverallRoi2"`
	TotalPrepayAndPaySettleOverallRoi21h       float64 `json:"totalPrepayAndPaySettleOverallRoi21h"`
	TotalCostPerPayOrderSettleForOverallRoi21h float64 `json:"totalCostPerPayOrderSettleForOverallRoi21h"`
	ShopEstimatedComissionCost                 float64 `json:"shopEstimatedComissionCost"`
	TotalOrderSettleAmountForRoi21h            float64 `json:"totalOrderSettleAmountForRoi21h"`
	StatCostForRoi2                            float64 `json:"statCostForRoi2"`
	TotalOrderSettleCountForRoi21h             int64   `json:"totalOrderSettleCountForRoi21h"`
	TotalOrderRealSettleAmountForRoi21h        float64 `json:"totalOrderRealSettleAmountForRoi21h"`
}

// ExportOverallDetail 导出乘方抖音号明细（接收已拉取的明细数据列表）
func (s *FinanceCommissionService) ExportOverallDetail(list []OverallDetailExportItem) (string, error) {
	return writeExcelFile("乘方抖音号明细", []string{
		"排名", "抖音号", "抖音号ID", "综合成本(元)", "综合ROI", "综合订单成本", "预估达人佣金(元)",
		"净成交金额(元)", "整体消耗(元)", "净成交订单数", "用户实际支付净成交金额(元)",
	}, []float64{8, 22, 18, 16, 12, 16, 18, 18, 16, 14, 22}, len(list), func(rowIdx int) []interface{} {
		item := list[rowIdx]
		return []interface{}{
			rowIdx + 1,
			item.AnchorName,
			item.AnchorShowID,
			item.StatCostForOverallRoi2,
			item.TotalPrepayAndPaySettleOverallRoi21h,
			item.TotalCostPerPayOrderSettleForOverallRoi21h,
			item.ShopEstimatedComissionCost,
			item.TotalOrderSettleAmountForRoi21h,
			item.StatCostForRoi2,
			item.TotalOrderSettleCountForRoi21h,
			item.TotalOrderRealSettleAmountForRoi21h,
		}
	})
}

func writeExcelFile(sheetName string, headers []string, colWidths []float64, rowCount int, rowData func(rowIdx int) []interface{}) (string, error) {
	f := excelize.NewFile()
	defer f.Close()

	defaultSheet := f.GetSheetName(0)
	if defaultSheet != sheetName {
		f.SetSheetName(defaultSheet, sheetName)
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E8F3FF"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
		if i < len(colWidths) {
			col, _ := excelize.ColumnNumberToName(i + 1)
			f.SetColWidth(sheetName, col, col, colWidths[i])
		}
	}

	for i := 0; i < rowCount; i++ {
		values := rowData(i)
		for colIdx, value := range values {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, i+2)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	if err := os.MkdirAll("./exports", 0755); err != nil {
		return "", err
	}
	fileName := fmt.Sprintf("%s_%s.xlsx", sheetName, time.Now().Format("20060102150405"))
	filePath := filepath.Join("./exports", fileName)
	if err := f.SaveAs(filePath); err != nil {
		return "", err
	}
	return filePath, nil
}
