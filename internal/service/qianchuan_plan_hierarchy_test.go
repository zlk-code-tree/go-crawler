package service

import "testing"

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
			AccountID:       4,
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
			AccountID:       0,
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

func TestQianchuanHierarchyLabels(t *testing.T) {
	if got := storeHierarchySecondaryName(0, 4, 12); got != "4 个千川账户待绑定 · 12 个计划" {
		t.Fatalf("unexpected unmatched label: %s", got)
	}
	if got := storeHierarchySecondaryName(4, 3, 9); got != "3 个千川账户 · 9 个计划" {
		t.Fatalf("unexpected store label: %s", got)
	}
}
