package service

import (
	"reflect"
	"testing"
)

func TestNormalizeQianchuanHierarchyDates(t *testing.T) {
	startDate, endDate, endExclusive, err := normalizeQianchuanHierarchyDates("2026-07-08", "2026-07-10")
	if err != nil {
		t.Fatalf("normalize dates failed: %v", err)
	}
	if startDate != "2026-07-08" || endDate != "2026-07-10" || endExclusive != "2026-07-11" {
		t.Fatalf("unexpected normalized dates: %s %s %s", startDate, endDate, endExclusive)
	}

	if _, _, _, err = normalizeQianchuanHierarchyDates("2026-07-10", "2026-07-08"); err == nil {
		t.Fatal("expected reversed date range to fail")
	}
}

func TestSummarizeQianchuanStoreNodes(t *testing.T) {
	summary := summarizeQianchuanStoreNodes([]QianchuanPlanHierarchyNode{
		{
			StoreID:         4,
			AdvertiserCount: 2,
			PlanCount:       5,
			ProductCount:    3,
			StatCost:        100,
			Commission:      180,
			ValidOrderCount: 8,
			CostAmount:      60,
			Profit:          75,
		},
		{
			StoreID:         0,
			AdvertiserCount: 1,
			PlanCount:       2,
			ProductCount:    1,
			StatCost:        20,
			Profit:          -20,
		},
	})

	if summary.StoreCount != 1 || summary.AdvertiserCount != 3 || summary.PlanCount != 7 {
		t.Fatalf("unexpected hierarchy counts: %+v", summary)
	}
	if summary.TotalCost != 120 || summary.TotalProfit != 55 {
		t.Fatalf("unexpected hierarchy totals: %+v", summary)
	}
}

func TestNormalizeQianchuanStoreName(t *testing.T) {
	if got := normalizeQianchuanStoreName("  利云文化 图书旗舰店\t"); got != "利云文化图书旗舰店" {
		t.Fatalf("unexpected normalized store name: %s", got)
	}
}

func TestQianchuanHierarchyLabels(t *testing.T) {
	if got := storeHierarchySecondaryName(0, 4, 12); got != "4 个千川账户待绑定 · 12 个计划" {
		t.Fatalf("unexpected unmatched label: %s", got)
	}
	if got := storeHierarchySecondaryName(4, 3, 9); got != "3 个千川账户 · 9 个计划" {
		t.Fatalf("unexpected store label: %s", got)
	}
	if got := awemeHierarchyFallbackName(123, ""); got != "抖音号 UID 123" {
		t.Fatalf("unexpected aweme fallback name: %s", got)
	}
	if got := awemeHierarchySecondaryName("douyin001", 2, 7); got != "抖音号 douyin001 · 2 个千川账户 · 7 个计划" {
		t.Fatalf("unexpected aweme label: %s", got)
	}
}

func TestSummarizeQianchuanAwemeNodes(t *testing.T) {
	summary := summarizeQianchuanHierarchyNodes([]QianchuanPlanHierarchyNode{
		{StoreCount: 2, AccountCount: 3, AdvertiserCount: 3, PlanCount: 5, ProductCount: 2, StatCost: 80, Profit: 20},
		{StoreCount: 1, AccountCount: 1, AdvertiserCount: 1, PlanCount: 2, ProductCount: 1, StatCost: 30, Profit: -5},
	}, QianchuanHierarchyLevelAweme)

	if summary.AwemeCount != 2 || summary.StoreCount != 3 || summary.AccountCount != 4 || summary.PlanCount != 7 {
		t.Fatalf("unexpected aweme counts: %+v", summary)
	}
	if summary.TotalCost != 110 || summary.TotalProfit != 15 {
		t.Fatalf("unexpected aweme totals: %+v", summary)
	}
}

func TestQianchuanPlanAwemeExportValidation(t *testing.T) {
	if err := validateQianchuanPlanAwemeExportRequest(QianchuanPlanHierarchyRequest{
		Level:   QianchuanHierarchyLevelStore,
		Keyword: "利云文化",
	}); err == nil || err.Error() != "只支持抖音号维度数据导出" {
		t.Fatalf("unexpected level validation error: %v", err)
	}

	if err := validateQianchuanPlanAwemeExportRequest(QianchuanPlanHierarchyRequest{
		Level: QianchuanHierarchyLevelAweme,
	}); err != nil {
		t.Fatalf("empty keyword should export all aweme data: %v", err)
	}
}

func TestQianchuanPlanAwemeExportRow(t *testing.T) {
	wantHeaders := []string{
		"抖音号", "关联千川户", "关联店铺", "计划数", "商品数", "消耗(¥)", "佣金(¥)",
		"有效订单", "成本(¥)", "利润(¥)", "成交订单", "展示", "点击", "点击率",
	}
	if !reflect.DeepEqual(qianchuanPlanAwemeExportHeaders, wantHeaders) {
		t.Fatalf("unexpected export headers: %#v", qianchuanPlanAwemeExportHeaders)
	}

	node := QianchuanPlanHierarchyNode{
		Name:            "利云文化官方旗舰店",
		AwemeShowID:     "liyunwenhua21",
		AdvertiserCount: 2,
		StoreCount:      1,
		PlanCount:       184,
		ProductCount:    32,
		StatCost:        29091.89,
		Commission:      31683.15,
		ValidOrderCount: 2968.8,
		CostAmount:      15914.55,
		Profit:          2340.74,
		PayOrderCount:   3405,
		ShowCnt:         10000,
		ClickCnt:        1234,
		Ctr:             0.1234,
	}
	want := []interface{}{
		"利云文化官方旗舰店（抖音号：liyunwenhua21）",
		int64(2), int64(1), int64(184), int64(32),
		29091.89, 31683.15, 2968.8, 15914.55, 2340.74, 3405.0,
		int64(10000), int64(1234), "12.34%",
	}
	if got := qianchuanPlanAwemeExportRow(node); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected export row:\n got: %#v\nwant: %#v", got, want)
	}
}
