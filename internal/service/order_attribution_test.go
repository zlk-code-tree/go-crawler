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
	"github.com/gogf/gf/v2/util/gconv"
)

func TestEnrichDouyinOrderRowsStopsBeforeQueriesWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewOrderService().enrichDouyinOrderRows(ctx, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("enrichDouyinOrderRows() error = %v, want context.Canceled", err)
	}
}

func TestQianchuanPlanBlankOrderInferenceContextOverride(t *testing.T) {
	if IsQianchuanPlanBlankOrderInferenceEnabled(withQianchuanPlanBlankOrderInferenceSetting(context.Background(), false)) {
		t.Fatal("context override false was ignored")
	}
	if !IsQianchuanPlanBlankOrderInferenceEnabled(withQianchuanPlanBlankOrderInferenceSetting(context.Background(), true)) {
		t.Fatal("context override true was ignored")
	}
}

func TestDouyinOrderMatchesBoundUIDUsesOriginalUIDOrExplicitQianchuanAttribution(t *testing.T) {
	selectedUIDs := map[string]struct{}{"972381158384056": {}}
	tests := []struct {
		name  string
		order g.Map
		want  bool
	}{
		{
			name: "original UID is bound",
			order: g.Map{
				"originalTalentId": "972381158384056",
				"talentId":         "another-display-uid",
				"teamOwner":        "其他组",
			},
			want: true,
		},
		{
			name: "qianchuan inferred UID can claim a self-sale order",
			order: g.Map{
				"originalTalentId":        "",
				"talentId":                "972381158384056",
				"profitAttributionSource": "qianchuan_plan",
				"teamOwner":               "张文蔚/孟令饶组",
			},
			want: true,
		},
		{
			name: "shop-name inference cannot claim an empty original UID",
			order: g.Map{
				"originalTalentId":        "",
				"talentId":                "972381158384056",
				"profitAttributionSource": "shop_account",
				"teamOwner":               "张文蔚/孟令饶组",
			},
			want: false,
		},
		{
			name: "team owner cannot override a different original UID",
			order: g.Map{
				"originalTalentId": "111111111111111",
				"talentId":         "972381158384056",
				"teamOwner":        "张文蔚/孟令饶组",
			},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := douyinOrderMatchesBoundUID(test.order, selectedUIDs); got != test.want {
				t.Fatalf("douyinOrderMatchesBoundUID() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestOrderListExactWeightedAssignmentsUsesIntegerQuotas(t *testing.T) {
	orderKeys := []string{"o1", "o2", "o3", "o4", "o5", "o6", "o7", "o8", "o9", "o10"}
	assignments := orderListExactWeightedAssignments("2026-07-13\x00测试店\x00p1", orderKeys, []float64{60, 30, 10})
	counts := make([]int, 3)
	for _, ownerIndex := range assignments {
		counts[ownerIndex]++
	}
	if counts[0] != 6 || counts[1] != 3 || counts[2] != 1 {
		t.Fatalf("allocation counts = %v, want [6 3 1]", counts)
	}
	if len(assignments) != len(orderKeys) {
		t.Fatalf("assigned orders = %d, want %d", len(assignments), len(orderKeys))
	}
}

func TestOrderListExactWeightedAssignmentsIsStableAndDeduplicatesOrders(t *testing.T) {
	forward := []string{"o1", "o2", "o3", "o4", "o5", "o5"}
	reverse := []string{"o5", "o4", "o3", "o2", "o1"}
	first := orderListExactWeightedAssignments("scope", forward, []float64{2, 1})
	second := orderListExactWeightedAssignments("scope", reverse, []float64{2, 1})
	if fmt.Sprint(first) != fmt.Sprint(second) {
		t.Fatalf("allocation changed with input order: first=%v second=%v", first, second)
	}
	if len(first) != 5 {
		t.Fatalf("assigned orders = %d, want 5 unique orders", len(first))
	}
}

func TestOrderListExactWeightedAssignmentsRejectsZeroWeights(t *testing.T) {
	if got := orderListExactWeightedAssignments("scope", []string{"o1"}, []float64{0, 0}); len(got) != 0 {
		t.Fatalf("zero weights assigned orders: %v", got)
	}
}

func TestOrderListExactWeightedAssignmentsWithCapsLimitsOverAllocation(t *testing.T) {
	orderKeys := []string{"o1", "o2", "o3", "o4", "o5", "o6", "o7", "o8", "o9", "o10"}
	assignments := orderListExactWeightedAssignmentsWithCaps("scope", orderKeys, []float64{7, 3}, []int{3, 2})
	counts := make([]int, 2)
	for _, ownerIndex := range assignments {
		counts[ownerIndex]++
	}
	if len(assignments) != 5 {
		t.Fatalf("assigned orders = %d, want capped 5", len(assignments))
	}
	if counts[0] > 3 || counts[1] > 2 {
		t.Fatalf("allocation exceeded caps: counts=%v caps=[3 2]", counts)
	}
}

func TestOrderListExactWeightedAssignmentsWithCapsRedistributesRemainder(t *testing.T) {
	orderKeys := []string{"o1", "o2", "o3", "o4", "o5"}
	assignments := orderListExactWeightedAssignmentsWithCaps("scope", orderKeys, []float64{9, 1}, []int{1, 5})
	counts := make([]int, 2)
	for _, ownerIndex := range assignments {
		counts[ownerIndex]++
	}
	if len(assignments) != 5 {
		t.Fatalf("assigned orders = %d, want all 5 assigned within total cap", len(assignments))
	}
	if counts[0] != 1 || counts[1] != 4 {
		t.Fatalf("allocation counts = %v, want [1 4]", counts)
	}
}

func TestDouyinQianchuanSelfSaleAttributionDatabaseIntegration(t *testing.T) {
	if os.Getenv("DOUYIN_SELF_SALE_ATTRIBUTION_DB_INTEGRATION") != "1" {
		t.Skip("set DOUYIN_SELF_SALE_ATTRIBUTION_DB_INTEGRATION=1 to run against the configured database")
	}
	statDate := os.Getenv("DOUYIN_SELF_SALE_ATTRIBUTION_TEST_DATE")
	if statDate == "" {
		t.Fatal("DOUYIN_SELF_SALE_ATTRIBUTION_TEST_DATE is required")
	}

	ClearEnrichedDouyinOrderCache()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	orders, err := NewOrderService().loadEnrichedDouyinOrders(ctx, statDate, statDate)
	if err != nil {
		t.Fatalf("load enriched Douyin orders failed: %v", err)
	}

	type attributionSummary struct {
		orders map[string]struct{}
		valid  map[string]struct{}
		pay    float64
		cost   float64
	}
	summaries := make(map[int64]*attributionSummary)
	ownersByOrder := make(map[string]map[string]struct{})
	for _, order := range orders {
		if gconv.String(order["profitAttributionSource"]) != "qianchuan_plan" {
			continue
		}
		if orderListNormalizedTalentID(gconv.String(order["originalTalentId"])) != "" {
			t.Fatalf("Qianchuan attribution overwrote non-empty original UID: order=%s originalUID=%s",
				gconv.String(order["mainOrderNo"]), gconv.String(order["originalTalentId"]))
		}
		accountID := gconv.Int64(order["attributedAccountId"])
		traderID := gconv.Int64(order["attributedTraderId"])
		talentUID := orderListNormalizedTalentID(gconv.String(order["talentId"]))
		if accountID <= 0 || traderID <= 0 || talentUID == "" {
			t.Fatalf("incomplete Qianchuan attribution: order=%s account=%d trader=%d uid=%s",
				gconv.String(order["mainOrderNo"]), accountID, traderID, talentUID)
		}

		orderKey := orderListFirstNonEmpty(gconv.String(order["mainOrderNo"]), gconv.String(order["subOrderNo"]), gconv.String(order["id"]))
		if _, ok := ownersByOrder[orderKey]; !ok {
			ownersByOrder[orderKey] = make(map[string]struct{})
		}
		ownersByOrder[orderKey][talentUID] = struct{}{}
		summary := summaries[traderID]
		if summary == nil {
			summary = &attributionSummary{orders: make(map[string]struct{}), valid: make(map[string]struct{})}
			summaries[traderID] = summary
		}
		summary.orders[orderKey] = struct{}{}
		if gconv.Bool(order["isValidOrder"]) {
			summary.valid[orderKey] = struct{}{}
			summary.pay += gconv.Float64(order["orderPayAmount"])
			summary.cost += gconv.Float64(order["productCost"])
		}
	}

	for orderKey, owners := range ownersByOrder {
		if len(owners) > 1 {
			t.Fatalf("main order was assigned to multiple talents: order=%s owners=%v", orderKey, owners)
		}
	}
	if len(ownersByOrder) == 0 {
		t.Fatal("no self-sale orders matched a Qianchuan plan")
	}
	for traderID, summary := range summaries {
		t.Logf("self-sale attribution date=%s trader=%d total=%d valid=%d pay=%.2f cost=%.2f",
			statDate, traderID, len(summary.orders), len(summary.valid), summary.pay, summary.cost)
	}
	t.Logf("self-sale attribution date=%s assignedMainOrders=%d traders=%d", statDate, len(ownersByOrder), len(summaries))
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

	ClearEnrichedDouyinOrderCache()

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
		{name: "closed initialized after sale", status: "已关闭", afterSale: "0", want: false},
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

func TestStatsFromEnrichedOrdersUsesLinkRefundRateForAllocatedBlankOrders(t *testing.T) {
	orders := []g.Map{
		{
			"rawOrderStatus":          "已发货",
			"afterSaleStatus":         "",
			"productPrice":            100,
			"productNum":              1,
			"productCost":             40,
			"profitAttributionSource": "qianchuan_plan",
			"originalTalentId":        "",
			"linkRefundCostRate":      0.25,
		},
		{
			"rawOrderStatus":          "已关闭",
			"afterSaleStatus":         "12",
			"productPrice":            100,
			"productNum":              1,
			"productCost":             80,
			"profitAttributionSource": "qianchuan_plan",
			"originalTalentId":        "",
			"sendTime":                "2026-07-09 10:00:00",
		},
	}

	stats := NewOrderService().statsFromEnrichedOrders(context.Background(), orders, "", "", "2026-07-11 00:00:00", "2026-07-11 23:59:59")
	if got := math.Round(stats.RefundCostAmount*100) / 100; got != 10 {
		t.Fatalf("allocated blank refund cost = %.2f, want 10.00", got)
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
