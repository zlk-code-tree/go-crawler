package service

import "testing"

func TestBuildDouyinProfitSnapshot(t *testing.T) {
	snapshot := buildDouyinProfitSnapshot(douyinProfitOrderTotals{
		TotalOrderCount:     28362,
		ValidOrderCount:     25080,
		InvalidOrderCount:   3282,
		ProductQuantity:     25158,
		ProductCount:        2681,
		SalesAmount:         452848.32,
		CostAmount:          138272.85,
		RefundCostAmount:    1913.20,
		RefundOrderCount:    341,
		ValidOrderCostMatch: 24683,
	}, 272126.46)

	if snapshot.ReserveAmount != 13585.45 || snapshot.ShopServiceFee != 9056.97 {
		t.Fatalf("unexpected percentage costs: %+v", snapshot)
	}
	if snapshot.CommissionDeduction != 2264.24 || snapshot.Profit != 15629.15 {
		t.Fatalf("unexpected canonical profit: %+v", snapshot)
	}
	if snapshot.ValidOrderCount != 25080 || snapshot.ProductQuantity != 25158 {
		t.Fatalf("unexpected canonical quantities: %+v", snapshot)
	}
}
