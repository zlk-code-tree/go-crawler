package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

// DouyinProfitSnapshot is the canonical order and profit summary shared by
// finance and Qianchuan reports. Plan data is only used to allocate this total.
type DouyinProfitSnapshot struct {
	TotalOrderCount      int64
	ValidOrderCount      int64
	InvalidOrderCount    int64
	ProductQuantity      int64
	ProductCount         int64
	SalesAmount          float64
	CostAmount           float64
	ReserveAmount        float64
	RefundCostAmount     float64
	RefundOrderCount     int64
	ShopServiceFee       float64
	Commission           float64
	CommissionDeduction  float64
	AdCostAmount         float64
	Profit               float64
	ValidOrderCostMatch  int64
	CommissionOrderCount int64
}

type douyinProfitOrderTotals struct {
	TotalOrderCount     int64
	ValidOrderCount     int64
	InvalidOrderCount   int64
	ProductQuantity     int64
	ProductCount        int64
	SalesAmount         float64
	CostAmount          float64
	RefundCostAmount    float64
	RefundOrderCount    int64
	ValidOrderCostMatch int64
}

func buildDouyinProfitSnapshot(totals douyinProfitOrderTotals, adCost float64) DouyinProfitSnapshot {
	salesAmount := roundDouyinProfit(totals.SalesAmount)
	costAmount := roundDouyinProfit(totals.CostAmount)
	reserveAmount := CalculateReserveAmount(salesAmount)
	refundCostAmount := roundDouyinProfit(totals.RefundCostAmount)
	shopServiceFee := CalculateDouyinShopServiceFee(salesAmount)
	commission := roundDouyinProfit(salesAmount - costAmount - reserveAmount - shopServiceFee)
	commissionDeduction := roundDouyinProfit(salesAmount * 0.005)
	adCost = roundDouyinProfit(adCost)

	return DouyinProfitSnapshot{
		TotalOrderCount:      totals.TotalOrderCount,
		ValidOrderCount:      totals.ValidOrderCount,
		InvalidOrderCount:    totals.InvalidOrderCount,
		ProductQuantity:      totals.ProductQuantity,
		ProductCount:         totals.ProductCount,
		SalesAmount:          salesAmount,
		CostAmount:           costAmount,
		ReserveAmount:        reserveAmount,
		RefundCostAmount:     refundCostAmount,
		RefundOrderCount:     totals.RefundOrderCount,
		ShopServiceFee:       shopServiceFee,
		Commission:           commission,
		CommissionDeduction:  commissionDeduction,
		AdCostAmount:         adCost,
		Profit:               roundDouyinProfit(CalculateOrderProfit(salesAmount, costAmount, reserveAmount, refundCostAmount, shopServiceFee, commissionDeduction, 0, adCost)),
		ValidOrderCostMatch:  totals.ValidOrderCostMatch,
		CommissionOrderCount: totals.ValidOrderCount,
	}
}

func roundDouyinProfit(value float64) float64 {
	return math.Round(value*100) / 100
}

func normalizeDouyinProfitDates(startDate, endDate string) (string, string, string, error) {
	startDate, endDate = NormalizeFinanceDateRange(startDate, endDate)
	start, startErr := time.Parse(financeDateLayout, startDate)
	end, endErr := time.Parse(financeDateLayout, endDate)
	if startErr != nil || endErr != nil || end.Before(start) {
		return "", "", "", fmt.Errorf("日期格式错误或结束日期早于开始日期")
	}
	return startDate, endDate, end.AddDate(0, 0, 1).Format(financeDateLayout), nil
}

// GetDouyinProfitSnapshot returns the same full Douyin order facts regardless
// of whether the caller is the order, commission, consumption, or plan view.
func GetDouyinProfitSnapshot(ctx context.Context, startDate, endDate string) (*DouyinProfitSnapshot, error) {
	startDate, endDate, endExclusive, err := normalizeDouyinProfitDates(startDate, endDate)
	if err != nil {
		return nil, err
	}

	validCondition := FinanceValidOrderCondition("o")
	invalidCondition := financeInvalidOrderCondition("o")
	refundCondition := financeRefundOrderCondition("o")
	query := fmt.Sprintf(`SELECT
		COUNT(*) AS total_order_count,
		COALESCE(SUM(CASE WHEN %s THEN 1 ELSE 0 END), 0) AS valid_order_count,
		COALESCE(SUM(CASE WHEN %s THEN 1 ELSE 0 END), 0) AS invalid_order_count,
		COALESCE(SUM(CASE WHEN %s THEN o.product_num ELSE 0 END), 0) AS product_quantity,
		COUNT(DISTINCT CASE WHEN %s THEN o.product_id END) AS product_count,
		COALESCE(SUM(CASE WHEN %s THEN o.product_price * o.product_num ELSE 0 END), 0) AS sales_amount,
		COALESCE(SUM(CASE WHEN %s THEN COALESCE(gd.base_price, 0) * o.product_num ELSE 0 END), 0) AS cost_amount,
		COALESCE(SUM(CASE WHEN %s THEN COALESCE(gd.base_price, 0) * o.product_num ELSE 0 END), 0) AS refund_cost_amount,
		COALESCE(SUM(CASE WHEN %s THEN 1 ELSE 0 END), 0) AS refund_order_count,
		COALESCE(SUM(CASE WHEN %s AND COALESCE(gd.base_price, 0) > 0 THEN 1 ELSE 0 END), 0) AS valid_order_cost_match
	FROM order_info o
	LEFT JOIN (
		SELECT goods_code, MAX(base_price) AS base_price
		FROM goods
		GROUP BY goods_code
	) gd ON o.seller_code COLLATE utf8mb4_unicode_ci = gd.goods_code COLLATE utf8mb4_unicode_ci
	WHERE o.order_submit_time >= ? AND o.order_submit_time < ?`,
		validCondition, invalidCondition, validCondition, validCondition,
		validCondition, validCondition, refundCondition, refundCondition, validCondition)
	row, err := g.DB().GetOne(ctx, query, startDate, endExclusive)
	if err != nil {
		return nil, fmt.Errorf("查询抖音统一订单快照失败: %w", err)
	}

	adRow, err := g.DB().Model("qianchuan_advertiser_daily_report").Ctx(ctx).
		Where("LEFT(stat_datetime, 10) >= ?", startDate).
		Where("LEFT(stat_datetime, 10) <= ?", endDate).
		Fields("COALESCE(SUM(stat_cost), 0) AS ad_cost").One()
	if err != nil {
		return nil, fmt.Errorf("查询千川账户实扣消耗失败: %w", err)
	}

	snapshot := buildDouyinProfitSnapshot(douyinProfitOrderTotals{
		TotalOrderCount:     row["total_order_count"].Int64(),
		ValidOrderCount:     row["valid_order_count"].Int64(),
		InvalidOrderCount:   row["invalid_order_count"].Int64(),
		ProductQuantity:     row["product_quantity"].Int64(),
		ProductCount:        row["product_count"].Int64(),
		SalesAmount:         row["sales_amount"].Float64(),
		CostAmount:          row["cost_amount"].Float64(),
		RefundCostAmount:    row["refund_cost_amount"].Float64(),
		RefundOrderCount:    row["refund_order_count"].Int64(),
		ValidOrderCostMatch: row["valid_order_cost_match"].Int64(),
	}, adRow["ad_cost"].Float64())
	return &snapshot, nil
}

func (s DouyinProfitSnapshot) FinanceSummary() model.FinanceStatsSummary {
	return model.FinanceStatsSummary{
		TotalOrderCount:      int(s.TotalOrderCount),
		OrderCount:           int(s.ValidOrderCount),
		InvalidOrderCount:    int(s.InvalidOrderCount),
		ProductNum:           int(s.ProductQuantity),
		PayAmount:            s.SalesAmount,
		CostAmount:           s.CostAmount,
		ReserveAmount:        s.ReserveAmount,
		Commission:           s.Commission,
		RefundCostAmount:     s.RefundCostAmount,
		RefundOrderCount:     int(s.RefundOrderCount),
		ShopServiceFee:       s.ShopServiceFee,
		AdCostAmount:         s.AdCostAmount,
		ValidOrderCostMatch:  int(s.ValidOrderCostMatch),
		CommissionOrderCount: int(s.CommissionOrderCount),
		RealtimeProfit:       s.Profit,
	}
}

func getNodeQianchuanAdvertiserDailyCost(ctx context.Context, nodeID int64, nodeType, startDate, endDate string) (float64, error) {
	startDate, endDate = NormalizeFinanceDateRange(startDate, endDate)
	if nodeID <= 0 {
		row, err := g.DB().Model("qianchuan_advertiser_daily_report").Ctx(ctx).
			Where("LEFT(stat_datetime, 10) >= ?", startDate).
			Where("LEFT(stat_datetime, 10) <= ?", endDate).
			Fields("COALESCE(SUM(stat_cost), 0) AS ad_cost").One()
		if err != nil {
			return 0, fmt.Errorf("查询千川账户实扣消耗失败: %w", err)
		}
		return row["ad_cost"].Float64(), nil
	}

	switch nodeType {
	case "company":
		return getCompanyQianchuanAdvertiserDailyCost(ctx, nodeID, startDate, endDate)
	case "department":
		departmentID := nodeID
		if model.IsDeptID(nodeID) {
			departmentID = model.DecodeDeptID(nodeID)
		}
		return getDepartmentQianchuanAdvertiserDailyCost(ctx, departmentID, startDate, endDate)
	case "trader":
		traderID := nodeID
		if model.IsTraderID(nodeID) {
			traderID = model.DecodeTraderID(nodeID)
		}
		return getTraderQianchuanAdvertiserDailyCost(ctx, traderID, startDate, endDate)
	default:
		return 0, nil
	}
}

func getCompanyQianchuanAdvertiserDailyCost(ctx context.Context, companyID int64, startDate, endDate string) (float64, error) {
	row, err := g.DB().GetOne(ctx, attributedQianchuanCostCTE+`,
		company_tree AS (
			SELECT id FROM sys_company WHERE id = ? AND status = 1
			UNION ALL
			SELECT child.id
			FROM sys_company child
			INNER JOIN company_tree parent ON child.parent_id = parent.id
			WHERE child.status = 1
		)
		SELECT COALESCE(SUM(cost.stat_cost), 0) AS ad_cost
		FROM attributed_qianchuan_cost cost
		INNER JOIN trader owner ON owner.id = cost.resolved_trader_id AND owner.status = 1
		WHERE owner.company_id IN (SELECT id FROM company_tree)`, startDate, endDate, startDate, endDate, companyID)
	if err != nil {
		return 0, fmt.Errorf("查询公司千川账户实扣消耗失败: %w", err)
	}
	return row["ad_cost"].Float64(), nil
}

func getDepartmentQianchuanAdvertiserDailyCost(ctx context.Context, departmentID int64, startDate, endDate string) (float64, error) {
	row, err := g.DB().GetOne(ctx, attributedQianchuanCostCTE+`,
		dept_tree AS (
			SELECT id FROM sys_department WHERE id = ? AND status = 1
			UNION ALL
			SELECT child.id
			FROM sys_department child
			INNER JOIN dept_tree parent ON child.parent_id = parent.id
			WHERE child.status = 1
		)
		SELECT COALESCE(SUM(cost.stat_cost), 0) AS ad_cost
		FROM attributed_qianchuan_cost cost
		INNER JOIN trader owner ON owner.id = cost.resolved_trader_id AND owner.status = 1
		WHERE owner.group_id IN (SELECT id FROM dept_tree)`, startDate, endDate, startDate, endDate, departmentID)
	if err != nil {
		return 0, fmt.Errorf("查询部门千川账户实扣消耗失败: %w", err)
	}
	return row["ad_cost"].Float64(), nil
}

func getTraderQianchuanAdvertiserDailyCost(ctx context.Context, traderID int64, startDate, endDate string) (float64, error) {
	row, err := g.DB().GetOne(ctx, attributedQianchuanCostCTE+`
		SELECT COALESCE(SUM(stat_cost), 0) AS ad_cost
		FROM attributed_qianchuan_cost
		WHERE resolved_trader_id = ?`, startDate, endDate, startDate, endDate, traderID)
	if err != nil {
		return 0, fmt.Errorf("查询投手千川账户实扣消耗失败: %w", err)
	}
	return row["ad_cost"].Float64(), nil
}
