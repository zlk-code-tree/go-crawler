package crawler

import "testing"

func TestRankingDetailUpdatesSkipUncollectedAndUnrelatedProducts(t *testing.T) {
	products := []ProductInfo{{ProductID: "product-1", ProductName: "first"}, {ProductID: "product-2", ProductName: "second"}}
	if updates := rankingDetailUpdates("task-a", products, nil); len(updates) != 0 {
		t.Fatalf("disabled or skipped detail collection would update saved products: %+v", updates)
	}
	updates := rankingDetailUpdates(" task-a ", products, []ProductDetail{
		{ProductID: " product-1 ", FirstShelfTime: "2026-10-01", MinPrice: "19.90"},
		{ProductID: "other-product", MinPrice: "99.00"},
		{ProductName: "missing ID", MinPrice: "49.00"},
	})
	if len(updates) != 1 {
		t.Fatalf("expected only the collected product in this snapshot, got %+v", updates)
	}
	got := updates[0]
	if got.TaskId != "task-a" || got.ProductId != "product-1" || got.ProductName != "first" || got.MinPrice != "19.90" {
		t.Fatalf("details were assigned to the wrong task or product: %+v", got)
	}
}

func TestRankingDetailUpdatesPreserveInsuranceAndAccessState(t *testing.T) {
	insurance := false
	updates := rankingDetailUpdates("task-a", []ProductInfo{{ProductID: "product-1"}}, []ProductDetail{
		{ProductID: "product-1", HasShippingInsurance: &insurance, PriceRange: "10~20",
			DetailAccessStatus: "removed", DetailAccessNote: "product removed"},
	})
	if len(updates) != 1 {
		t.Fatalf("unexpected updates: %+v", updates)
	}
	got := updates[0]
	if got.HasShippingInsurance == nil || *got.HasShippingInsurance != 0 || got.PriceRange != "10~20" ||
		got.DetailAccessStatus != "removed" || got.DetailAccessNote != "product removed" {
		t.Fatalf("detail enrichment dropped known fields: %+v", got)
	}
}
