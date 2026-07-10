package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"
)

func TestEnrichDouyinOrderRowsStopsBeforeQueriesWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewOrderService().enrichDouyinOrderRows(ctx, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("enrichDouyinOrderRows() error = %v, want context.Canceled", err)
	}
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
