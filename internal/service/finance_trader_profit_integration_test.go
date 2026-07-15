package service

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"go-crawler/internal/model"
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
	commissionResult, err := NewFinanceCommissionService().GetStats(ctx, &model.FinanceStatsReq{
		NodeId:    model.EncodeTraderID(traderID),
		NodeType:  "trader",
		TimeType:  "custom",
		StartDate: statDate,
		EndDate:   statDate,
		// Today-cost 投手口径包含投手绑定的所有平台账号，这里保持同口径。
		Platform: "",
	})
	if err != nil {
		t.Fatalf("query trader commission stats failed: %v", err)
	}
	if commissionResult.Summary.OrderCount != int(result.OrderCount) || commissionResult.Summary.PayAmount != result.PayAmount || commissionResult.Summary.AdCostAmount != result.AdCost || commissionResult.Summary.RealtimeProfit != result.NetProfit {
		t.Fatalf("commission and today-cost trader results differ: commission=%+v todayCost=%+v", commissionResult.Summary, result)
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

func TestRealtimeProfitCompanyDatabaseIntegration(t *testing.T) {
	if os.Getenv("FINANCE_COMPANY_PROFIT_DB_INTEGRATION") != "1" {
		t.Skip("set FINANCE_COMPANY_PROFIT_DB_INTEGRATION=1 to run against the configured database")
	}
	statDate := os.Getenv("FINANCE_COMPANY_PROFIT_TEST_DATE")
	if statDate == "" {
		t.Fatal("FINANCE_COMPANY_PROFIT_TEST_DATE is required")
	}
	companyID, err := strconv.ParseInt(os.Getenv("FINANCE_COMPANY_PROFIT_TEST_COMPANY_ID"), 10, 64)
	if err != nil || companyID <= 0 {
		t.Fatal("FINANCE_COMPANY_PROFIT_TEST_COMPANY_ID must be a positive integer")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	service := NewFinanceCommissionService()
	useFullSnapshot, err := shouldUseFullDouyinCompanySnapshot(ctx, companyID)
	if err != nil {
		t.Fatalf("query company finance scope failed: %v", err)
	}
	result, err := service.GetRealtimeProfit(ctx, &RealtimeProfitReq{
		CompanyId: companyID,
		StartDate: statDate,
		EndDate:   statDate,
	})
	if err != nil {
		t.Fatalf("query realtime company profit failed: %v", err)
	}
	commissionResult, err := service.GetStats(ctx, &model.FinanceStatsReq{
		NodeId:    companyID,
		NodeType:  "company",
		TimeType:  "custom",
		StartDate: statDate,
		EndDate:   statDate,
		Platform:  "douyin",
	})
	if err != nil {
		t.Fatalf("query company commission stats failed: %v", err)
	}
	if commissionResult.Summary.OrderCount != int(result.OrderCount) || commissionResult.Summary.PayAmount != result.PayAmount || commissionResult.Summary.AdCostAmount != result.AdCost || commissionResult.Summary.RealtimeProfit != result.NetProfit {
		t.Fatalf("commission and today-cost company results differ: commission=%+v todayCost=%+v", commissionResult.Summary, result)
	}
	if result.TotalOrderCount == 0 || result.OrderCount == 0 {
		t.Fatalf("company profit returned no orders: %+v", result)
	}
	if useFullSnapshot {
		snapshot, snapshotErr := GetDouyinProfitSnapshot(ctx, statDate, statDate)
		if snapshotErr != nil {
			t.Fatalf("query canonical snapshot failed: %v", snapshotErr)
		}
		if result.OrderCount != snapshot.ValidOrderCount || result.PayAmount != snapshot.SalesAmount || result.AdCost != snapshot.AdCostAmount || result.NetProfit != snapshot.Profit {
			t.Fatalf("company result differs from canonical snapshot: result=%+v snapshot=%+v", result, snapshot)
		}
	}
	t.Logf("date=%s company=%d fullSnapshot=%v total=%d valid=%d pay=%.2f cost=%.2f adCost=%.2f profit=%.2f",
		statDate, companyID, useFullSnapshot, result.TotalOrderCount, result.OrderCount,
		result.PayAmount, result.CostAmount, result.AdCost, result.NetProfit)
}
