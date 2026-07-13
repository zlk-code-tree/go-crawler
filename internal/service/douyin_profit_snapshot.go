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
	if startDate == "" && endDate == "" {
		today := time.Now().Format("2006-01-02")
		startDate, endDate = today, today
	} else if startDate == "" {
		startDate = endDate
	} else if endDate == "" {
		endDate = startDate
	}

	start, startErr := time.Parse("2006-01-02", startDate)
	end, endErr := time.Parse("2006-01-02", endDate)
	if startErr != nil || endErr != nil || end.Before(start) {
		return "", "", "", fmt.Errorf("日期格式错误或结束日期早于开始日期")
	}
	return startDate, endDate, end.AddDate(0, 0, 1).Format("2006-01-02"), nil
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
	WHERE o.pay_finish_time >= ? AND o.pay_finish_time < ?`,
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

func isRootFinanceCompany(ctx context.Context, companyID int64) (bool, error) {
	if companyID <= 0 {
		return false, nil
	}
	row, err := g.DB().Model("sys_company").Ctx(ctx).Where("id", companyID).Fields("parent_id").One()
	if err != nil {
		return false, err
	}
	return !row.IsEmpty() && row["parent_id"].Int64() == 0, nil
}
