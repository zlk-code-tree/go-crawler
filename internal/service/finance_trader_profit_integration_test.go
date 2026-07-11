package service

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	_ "go-crawler/internal/packed"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
)

func TestRealtimeProfitTraderDatabaseIntegration(t *testing.T) {
	if os.Getenv("FINANCE_TRADER_PROFIT_DB_INTEGRATION") != "1" {
		t.Skip("set FINANCE_TRADER_PROFIT_DB_INTEGRATION=1 to run against the configured database")
	}
	statDate := os.Getenv("FINANCE_TRADER_PROFIT_TEST_DATE")
	if statDate == "" {
		t.Fatal("FINANCE_TRADER_PROFIT_TEST_DATE is required")
	}
	traderID, err := strconv.ParseInt(os.Getenv("FINANCE_TRADER_PROFIT_TEST_TRADER_ID"), 10, 64)
	if err != nil || traderID <= 0 {
		t.Fatal("FINANCE_TRADER_PROFIT_TEST_TRADER_ID must be a positive integer")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	result, err := NewFinanceCommissionService().GetRealtimeProfit(ctx, &RealtimeProfitReq{
		TraderId:  traderID,
		StartDate: statDate,
		EndDate:   statDate,
	})
	if err != nil {
		t.Fatalf("query realtime trader profit failed: %v", err)
	}
	accountIDs, err := NewFinanceCommissionService().getAccountIdsByDimension(ctx, &RealtimeProfitReq{TraderId: traderID})
	if err != nil {
		t.Fatalf("query trader account ids failed: %v", err)
	}
	traderName, err := NewFinanceCommissionService().getTraderName(ctx, traderID)
	if err != nil {
		t.Fatalf("query trader name failed: %v", err)
	}
	orderStats, err := NewFinanceCommissionService().statsOrdersByAccountIds(ctx, accountIDs, statDate, statDate, traderName)
	if err != nil {
		t.Fatalf("query trader order ownership stats failed: %v", err)
	}
	if minimum := os.Getenv("FINANCE_TRADER_PROFIT_EXPECT_MIN_PAY"); minimum != "" {
		minPay, parseErr := strconv.ParseFloat(minimum, 64)
		if parseErr != nil {
			t.Fatalf("invalid FINANCE_TRADER_PROFIT_EXPECT_MIN_PAY: %v", parseErr)
		}
		if result.PayAmount < minPay {
			t.Fatalf("pay amount %.2f is below expected minimum %.2f", result.PayAmount, minPay)
		}
	}
	t.Logf("date=%s trader=%d orders=%d pay=%.2f cost=%.2f adCost=%.2f commission=%.2f profit=%.2f",
		statDate, traderID, result.OrderCount, result.PayAmount, result.CostAmount, result.AdCost, result.Commission, result.NetProfit)
	t.Logf("ownership total=%d valid=%d invalid=%d refund=%d refundCost=%.2f",
		orderStats.TotalOrderCount, orderStats.OrderCount, orderStats.InvalidOrderCount,
		orderStats.RefundOrderCount, orderStats.RefundCostAmount)
}
