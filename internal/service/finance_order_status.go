package service

import (
	"fmt"
	"math"
)

const (
	FinanceReserveRate       = 0.03
	DouyinShopServiceFeeRate = 0.02
)

var (
	validOrderStatuses       = map[string]struct{}{"已发货": {}, "待发货": {}, "已完成": {}, "3": {}, "2": {}, "5": {}}
	cancelledOrderStatuses   = map[string]struct{}{"已关闭": {}, "已取消": {}, "4": {}}
	invalidAfterSaleStatuses = map[string]struct{}{"6": {}, "7": {}, "11": {}, "12": {}, "29": {}}
	closedAfterSaleStatuses  = map[string]struct{}{"27": {}, "28": {}, "51": {}}
)

// IsValidOrderForStats is the single effective-order rule used by all statistics.
func IsValidOrderForStats(orderStatus, afterSaleStatus string) bool {
	if _, ok := validOrderStatuses[orderStatus]; ok {
		_, invalid := invalidAfterSaleStatuses[afterSaleStatus]
		return !invalid
	}
	if _, ok := cancelledOrderStatuses[orderStatus]; ok {
		_, closed := closedAfterSaleStatuses[afterSaleStatus]
		return closed
	}
	return false
}

func IsInvalidOrderForStats(afterSaleStatus string) bool {
	_, ok := invalidAfterSaleStatuses[afterSaleStatus]
	return ok
}

func IsRefundCostOrderForStats(afterSaleStatus, sendTime string) bool {
	return afterSaleStatus == "12" && sendTime != ""
}

func CalculateOrderProfit(salesAmount, costAmount, reserveAmount, refundCostAmount, shopServiceFee, onlineCommission, offlineCommission, adCostAmount float64) float64 {
	return salesAmount - costAmount - reserveAmount - refundCostAmount - shopServiceFee - onlineCommission - offlineCommission - adCostAmount
}

func CalculateReserveAmount(salesAmount float64) float64 {
	return math.Round(salesAmount*FinanceReserveRate*100) / 100
}

func CalculateDouyinShopServiceFee(salesAmount float64) float64 {
	return math.Round(salesAmount*DouyinShopServiceFeeRate*100) / 100
}

func financeOrderColumn(alias, column string) string {
	if alias == "" {
		return column
	}
	return alias + "." + column
}

func financeValidOrderCondition(alias string) string {
	statusCol := financeOrderColumn(alias, "order_status")
	afterSaleCol := financeOrderColumn(alias, "after_sale_status")
	return fmt.Sprintf(
		"((%s IN ('已发货','待发货','已完成','3','2','5') AND (%s NOT IN ('6','7','11','12','29') OR %s IS NULL OR %s = '')) OR (%s IN ('已关闭','已取消','4') AND %s IN ('27','28','51')))",
		statusCol, afterSaleCol, afterSaleCol, afterSaleCol, statusCol, afterSaleCol,
	)
}

func financeInvalidOrderCondition(alias string) string {
	return fmt.Sprintf("%s IN ('6','7','11','12','29')", financeOrderColumn(alias, "after_sale_status"))
}

func financeRefundOrderCondition(alias string) string {
	return fmt.Sprintf(
		"%s = '12' AND %s IS NOT NULL",
		financeOrderColumn(alias, "after_sale_status"),
		financeOrderColumn(alias, "send_time"),
	)
}
