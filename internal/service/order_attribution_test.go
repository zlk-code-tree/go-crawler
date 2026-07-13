package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

func TestEnrichDouyinOrderRowsStopsBeforeQueriesWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewOrderService().enrichDouyinOrderRows(ctx, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("enrichDouyinOrderRows() error = %v, want context.Canceled", err)
	}
}

func TestOrderTypeStatsSurviveInitialRequestCancellationIntegration(t *testing.T) {
	if os.Getenv("ORDER_TYPE_STATS_DB_INTEGRATION") != "1" {
		t.Skip("set ORDER_TYPE_STATS_DB_INTEGRATION=1 to run against the configured database")
	}

	startDate := os.Getenv("ORDER_TYPE_STATS_TEST_START_DATE")
	endDate := os.Getenv("ORDER_TYPE_STATS_TEST_END_DATE")
	if startDate == "" || endDate == "" {
		t.Fatal("ORDER_TYPE_STATS_TEST_START_DATE and ORDER_TYPE_STATS_TEST_END_DATE are required")
	}

	cacheKey := startDate + "\x00" + endDate
	enrichedDouyinOrderCache.Delete(cacheKey)

	service := NewOrderService()
	canceledCtx, cancelInitial := context.WithCancel(context.Background())
	initialDone := make(chan error, 1)
	go func() {
		_, err := service.Stats(canceledCtx, "", "", "", "", "talent", startDate+" 00:00:00", endDate+" 23:59:59")
		initialDone <- err
	}()
	time.Sleep(5 * time.Millisecond)
	cancelInitial()

	ctx, cancel := context.WithTimeout(context.Background(), enrichedDouyinOrderBuildTimeout+10*time.Second)
	defer cancel()
	stats, err := service.Stats(ctx, "", "", "", "", "self_sale", startDate+" 00:00:00", endDate+" 23:59:59")
	if err != nil {
		t.Fatalf("self-sale stats failed after initial request cancellation: %v", err)
	}
	if stats.TotalOrderCount == 0 {
		t.Fatal("self-sale stats returned no orders")
	}
	refundStats, err := service.Stats(ctx, "", "", "", "", "refund_after_sale", startDate+" 00:00:00", endDate+" 23:59:59")
	if err != nil {
		t.Fatalf("refund after-sale stats failed: %v", err)
	}
	if refundStats.TotalOrderCount == 0 {
		t.Fatal("refund after-sale stats returned no orders")
	}

	select {
	case <-initialDone:
	case <-time.After(time.Second):
		t.Fatal("canceled initial request did not return promptly")
	}
	t.Logf("self-sale orders=%d valid=%d pay=%.2f profit=%.2f", stats.TotalOrderCount, stats.ValidOrderCount, stats.ValidPayAmount, stats.RealtimeProfit)
	t.Logf("refund after-sale orders=%d invalid=%d refund=%d refundCost=%.2f", refundStats.TotalOrderCount, refundStats.InvalidOrderCount, refundStats.RefundOrderCount, refundStats.RefundCostAmount)
}

func TestIsValidOrderForStats(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		afterSale string
		want      bool
	}{
		{name: "normal paid order", status: "已发货", afterSale: "", want: true},
		{name: "numeric normal order", status: "3", afterSale: "", want: true},
		{name: "active after sale", status: "已完成", afterSale: "12", want: false},
		{name: "closed rejected after sale", status: "已关闭", afterSale: "27", want: true},
		{name: "closed refunded order", status: "已关闭", afterSale: "12", want: false},
		{name: "unknown status", status: "未知", afterSale: "", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsValidOrderForStats(test.status, test.afterSale); got != test.want {
				t.Fatalf("IsValidOrderForStats(%q, %q) = %v, want %v", test.status, test.afterSale, got, test.want)
			}
		})
	}
}

func TestOrderListWeightedIndexIsStableAndProportional(t *testing.T) {
	ratios := []float64{0.7, 0.3}
	counts := make([]int, len(ratios))
	const sampleSize = 100000

	for i := 0; i < sampleSize; i++ {
		key := fmt.Sprintf("order-%d", i)
		first := orderListWeightedIndex(key, ratios)
		second := orderListWeightedIndex(key, ratios)
		if first != second {
			t.Fatalf("weighted owner changed for %q: %d != %d", key, first, second)
		}
		if first < 0 || first >= len(counts) {
			t.Fatalf("invalid weighted index %d for %q", first, key)
		}
		counts[first]++
	}

	actual := float64(counts[0]) / sampleSize
	if math.Abs(actual-ratios[0]) > 0.01 {
		t.Fatalf("first owner ratio = %.4f, want approximately %.2f (counts=%v)", actual, ratios[0], counts)
	}
}

func TestOrderListWeightedIndexRejectsZeroWeights(t *testing.T) {
	if got := orderListWeightedIndex("order-1", []float64{0, 0}); got != -1 {
		t.Fatalf("zero weights returned index %d, want -1", got)
	}
}

func TestOrderListResolveExternalTalentRequiresSameShop(t *testing.T) {
	externalByUID := map[string]externalTalentInfo{
		orderListExternalMatchKey("利云文化图书旗舰店", "972381158384056"): {
			uid:      "972381158384056",
			teamName: "一壶觞",
		},
	}

	_, _, ok := orderListResolveExternalTalent(
		"利云文化官方旗舰店",
		"972381158384056",
		"",
		"",
		externalByUID,
		map[string]externalTalentInfo{},
		map[string]externalTalentInfo{},
	)
	if ok {
		t.Fatal("cross-shop external talent matched, want no match")
	}
}

func TestOrderListResolveExternalTalentPrefersSameShopThenGlobal(t *testing.T) {
	globalInfo := externalTalentInfo{uid: "1", teamName: "全局团队"}
	shopInfo := externalTalentInfo{uid: "1", teamName: "店铺团队"}
	externalByUID := map[string]externalTalentInfo{
		orderListExternalMatchKey("", "1"):          globalInfo,
		orderListExternalMatchKey("利云文化官方旗舰店", "1"): shopInfo,
	}

	got, field, ok := orderListResolveExternalTalent(
		"利云文化官方旗舰店",
		"1",
		"",
		"",
		externalByUID,
		map[string]externalTalentInfo{},
		map[string]externalTalentInfo{},
	)
	if !ok || field != "uid" {
		t.Fatalf("resolveExternalTalent() matched=%v field=%q, want true/uid", ok, field)
	}
	if got.teamName != shopInfo.teamName {
		t.Fatalf("resolveExternalTalent() teamName=%q, want %q", got.teamName, shopInfo.teamName)
	}

	got, field, ok = orderListResolveExternalTalent(
		"不存在的店铺",
		"1",
		"",
		"",
		externalByUID,
		map[string]externalTalentInfo{},
		map[string]externalTalentInfo{},
	)
	if !ok || field != "uid" {
		t.Fatalf("resolveExternalTalent() global fallback matched=%v field=%q, want true/uid", ok, field)
	}
	if got.teamName != globalInfo.teamName {
		t.Fatalf("resolveExternalTalent() fallback teamName=%q, want %q", got.teamName, globalInfo.teamName)
	}
}

func TestOrderListFilterByTypePartitionsOrders(t *testing.T) {
	orders := []g.Map{
		{"id": 1, "orderOwnershipCode": "internal", "isShopSelfSale": false},
		{"id": 2, "orderOwnershipCode": "external", "isShopSelfSale": 0},
		{"id": 3, "orderOwnershipCode": "self_sale", "isShopSelfSale": true},
		{"id": 4, "orderOwnershipCode": "internal", "isShopSelfSale": "1"},
	}
	talentOrders := orderListFilterByType(orders, "talent")
	selfSaleOrders := orderListFilterByType(orders, "self_sale")
	externalOrders := orderListFilterByType(orders, "external")
	if len(talentOrders) != 2 || len(selfSaleOrders) != 2 || len(externalOrders) != 1 {
		t.Fatalf("partition mismatch: talent=%d selfSale=%d external=%d", len(talentOrders), len(selfSaleOrders), len(externalOrders))
	}
	if len(talentOrders)+len(selfSaleOrders) != len(orders) {
		t.Fatalf("partition lost or duplicated orders: all=%d talent=%d selfSale=%d", len(orders), len(talentOrders), len(selfSaleOrders))
	}
	if got := orderListFilterByType(orders, "unknown"); len(got) != len(orders) {
		t.Fatalf("unknown order type filtered rows: got=%d want=%d", len(got), len(orders))
	}
}

func TestOrderListFilterByTypeRefundAfterSale(t *testing.T) {
	orders := []g.Map{
		{"id": 1, "afterSaleStatus": ""},
		{"id": 2, "afterSaleStatus": "0"},
		{"id": 3, "afterSaleStatus": "6"},
		{"id": 4, "afterSaleStatus": "12"},
		{"id": 5, "afterSaleStatus": "27"},
		{"id": 6, "afterSaleStatus": "退款关闭"},
		{"id": 7, "afterSaleStatus": "无"},
	}

	filtered := orderListFilterByType(orders, "refund_after_sale")
	if len(filtered) != 4 {
		t.Fatalf("refund after-sale count = %d, want 4", len(filtered))
	}
	for _, order := range filtered {
		if !orderListHasRefundAfterSale(order) {
			t.Fatalf("non-after-sale order was included: %v", order)
		}
	}
}

func TestOrderListOfflineCommissionAmountOnlyCountsValidExternalRate(t *testing.T) {
	order := g.Map{
		"orderOwnershipCode":            "external",
		"productPrice":                  100,
		"productNum":                    2,
		"externalOfflineCommissionRate": 0.25,
	}
	if got := orderListOfflineCommissionAmount(order); got != 50 {
		t.Fatalf("offline commission = %.2f, want 50.00", got)
	}

	internal := g.Map{
		"orderOwnershipCode":            "internal",
		"productPrice":                  100,
		"productNum":                    2,
		"externalOfflineCommissionRate": 0.25,
	}
	if got := orderListOfflineCommissionAmount(internal); got != 0 {
		t.Fatalf("non-external offline commission = %.2f, want 0", got)
	}
}

func TestStatsFromEnrichedOrdersIncludesOfflineCommissionForExternalOrders(t *testing.T) {
	orders := []g.Map{
		{
			"rawOrderStatus":                "已发货",
			"afterSaleStatus":               "",
			"productPrice":                  100,
			"productNum":                    2,
			"productCost":                   60,
			"commission":                    10,
			"orderOwnershipCode":            "external",
			"externalOfflineCommissionRate": 0.25,
		},
		{
			"rawOrderStatus":                "已发货",
			"afterSaleStatus":               "",
			"productPrice":                  100,
			"productNum":                    1,
			"productCost":                   50,
			"commission":                    5,
			"orderOwnershipCode":            "internal",
			"externalOfflineCommissionRate": 0.30,
		},
		{
			"rawOrderStatus":                "已关闭",
			"afterSaleStatus":               "12",
			"productPrice":                  100,
			"productNum":                    3,
			"productCost":                   90,
			"commission":                    0,
			"orderOwnershipCode":            "external",
			"externalOfflineCommissionRate": 0.40,
			"sendTime":                      "2026-07-09 10:00:00",
		},
	}

	stats := NewOrderService().statsFromEnrichedOrders(context.Background(), orders, "", "", "2026-07-11 00:00:00", "2026-07-11 23:59:59")
	if got := math.Round(stats.OfflineCommission*100) / 100; got != 50 {
		t.Fatalf("stats offline commission = %.2f, want 50.00", got)
	}
}

func TestOrderListApplyCommissionScope(t *testing.T) {
	tests := []struct {
		name            string
		orderType       string
		wantOnline      float64
		wantOfflineKeep float64
	}{
		{name: "external only shows offline commission", orderType: "external", wantOnline: 0, wantOfflineKeep: 50},
		{name: "combined talent view shows both commissions", orderType: "talent", wantOnline: 120, wantOfflineKeep: 50},
		{name: "self sale does not show online commission", orderType: "self_sale", wantOnline: 0, wantOfflineKeep: 50},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stats := &OrderStats{OnlineCommission: 120, OfflineCommission: 50}
			orderListApplyCommissionScope(stats, test.orderType)
			if stats.OnlineCommission != test.wantOnline {
				t.Fatalf("online commission = %.2f, want %.2f", stats.OnlineCommission, test.wantOnline)
			}
			if stats.OfflineCommission != test.wantOfflineKeep {
				t.Fatalf("offline commission = %.2f, want %.2f", stats.OfflineCommission, test.wantOfflineKeep)
			}
		})
	}
}

func TestCalculateOrderProfit(t *testing.T) {
	salesAmount := 434873.85
	reserveAmount := CalculateReserveAmount(salesAmount)
	shopServiceFee := CalculateDouyinShopServiceFee(salesAmount)
	profit := CalculateOrderProfit(
		salesAmount,
		123481.55,
		reserveAmount,
		82.40,
		shopServiceFee,
		289648.60,
		0,
		419398.98,
	)
	if got := math.Round(profit*100) / 100; got != -419481.38 {
		t.Fatalf("profit = %.2f, want -419481.38", got)
	}
}

func TestRefundCostOnlyForCompletedPostShipmentRefund(t *testing.T) {
	tests := []struct {
		afterSale string
		sendTime  string
		want      bool
	}{
		{afterSale: "12", sendTime: "2026-07-09 10:00:00", want: true},
		{afterSale: "12", sendTime: "", want: false},
		{afterSale: "6", sendTime: "2026-07-09 10:00:00", want: false},
		{afterSale: "11", sendTime: "2026-07-09 10:00:00", want: false},
		{afterSale: "29", sendTime: "2026-07-09 10:00:00", want: false},
	}
	for _, test := range tests {
		if got := IsRefundCostOrderForStats(test.afterSale, test.sendTime); got != test.want {
			t.Fatalf("IsRefundCostOrderForStats(%q, %q) = %v, want %v", test.afterSale, test.sendTime, got, test.want)
		}
	}
}
