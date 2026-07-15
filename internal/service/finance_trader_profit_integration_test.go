package service

import (
	"context"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"go-crawler/internal/model"
	_ "go-crawler/internal/packed"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	"github.com/gogf/gf/v2/frame/g"
)

func TestFinanceSummarySubtractAndMergeIsConservative(t *testing.T) {
	total := model.FinanceStatsSummary{
		TotalOrderCount: 100, OrderCount: 90, InvalidOrderCount: 10, ProductNum: 95,
		PayAmount: 1000, CostAmount: 300, ReserveAmount: 30, Commission: 650,
		RefundCostAmount: 20, RefundOrderCount: 2, ShopServiceFee: 20,
		AdCostAmount: 400, ValidOrderCostMatch: 80, CommissionOrderCount: 90, RealtimeProfit: 225,
	}
	assigned := model.FinanceStatsSummary{
		TotalOrderCount: 75, OrderCount: 68, InvalidOrderCount: 7, ProductNum: 70,
		PayAmount: 800, CostAmount: 240, ReserveAmount: 24, Commission: 520,
		RefundCostAmount: 15, RefundOrderCount: 1, ShopServiceFee: 16,
		AdCostAmount: 310, ValidOrderCostMatch: 65, CommissionOrderCount: 68, RealtimeProfit: 191,
	}
	residual := subtractFinanceStatsSummary(total, assigned)
	mergeFinanceStatsSummary(&assigned, &residual)
	if !reflect.DeepEqual(assigned, total) {
		t.Fatalf("assigned + residual = %+v, want %+v", assigned, total)
	}
}

func TestFinanceProfitFormulaIncludesAdvertisingCost(t *testing.T) {
	profit := roundFloatFinance(CalculateOrderProfit(
		515853.50,
		146184.40,
		15475.60,
		1219.60,
		10287.73,
		2579.27,
		0,
		321627.75,
	), 2)
	if profit != 18479.15 {
		t.Fatalf("profit = %.2f, want 18479.15; advertising cost must be deducted", profit)
	}
}

func TestFinanceDailyAggregateDatabaseIntegration(t *testing.T) {
	if os.Getenv("FINANCE_DAILY_AGGREGATE_DB_INTEGRATION") != "1" {
		t.Skip("set FINANCE_DAILY_AGGREGATE_DB_INTEGRATION=1 to rebuild configured database dates")
	}
	startDate := os.Getenv("FINANCE_DAILY_AGGREGATE_START_DATE")
	endDate := os.Getenv("FINANCE_DAILY_AGGREGATE_END_DATE")
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		t.Fatalf("invalid start date: %v", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil || end.Before(start) {
		t.Fatalf("invalid end date: %s", endDate)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	aggregate := NewFinanceAggregateService()
	for date := start; !date.After(end); date = date.AddDate(0, 0, 1) {
		statDate := date.Format("2006-01-02")
		startedAt := time.Now()
		if err := aggregate.DailyAggregateJob(ctx, statDate); err != nil {
			t.Fatalf("aggregate %s failed: %v", statDate, err)
		}
		row, err := g.DB().Model("finance_daily_stats").Ctx(ctx).
			Where("stat_date", statDate).
			Fields("COUNT(*) AS rows_count, SUM(account_id > 0 AND platform <> '') AS leaf_count, SUM(company_id <> 0 OR department_id <> 0 OR trader_id <> 0) AS legacy_count").
			One()
		if err != nil {
			t.Fatalf("verify %s failed: %v", statDate, err)
		}
		if row["rows_count"].Int() == 0 || row["rows_count"].Int() != row["leaf_count"].Int() || row["legacy_count"].Int() != 0 {
			t.Fatalf("invalid leaf aggregate %s: %+v", statDate, row.Map())
		}
		t.Logf("aggregated date=%s rows=%d duration=%s", statDate, row["rows_count"].Int(), time.Since(startedAt).Round(time.Millisecond))
	}
}

func TestFinanceDailyAggregateMatchesRealtimeDatabaseIntegration(t *testing.T) {
	if os.Getenv("FINANCE_DAILY_COMPARE_DB_INTEGRATION") != "1" {
		t.Skip("set FINANCE_DAILY_COMPARE_DB_INTEGRATION=1 to compare aggregate and realtime data")
	}
	statDate := os.Getenv("FINANCE_DAILY_COMPARE_DATE")
	companyID, err := strconv.ParseInt(os.Getenv("FINANCE_DAILY_COMPARE_COMPANY_ID"), 10, 64)
	if statDate == "" || err != nil || companyID <= 0 {
		t.Fatal("compare date and company id are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	service := NewFinanceCommissionService()

	compareNode := func(nodeID int64, nodeType string) {
		t.Helper()
		daily, err := service.getStatsFromTable(ctx, nodeID, nodeType, statDate, statDate, "custom", "douyin")
		if err != nil {
			t.Fatalf("query aggregate %s/%d failed: %v", nodeType, nodeID, err)
		}
		realtime, err := service.getStatsRealtime(ctx, nodeID, nodeType, statDate, statDate, "custom", "douyin")
		if err != nil {
			t.Fatalf("query realtime %s/%d failed: %v", nodeType, nodeID, err)
		}
		if !reflect.DeepEqual(daily.Summary, realtime.Summary) {
			t.Fatalf("aggregate mismatch %s/%d: daily=%+v realtime=%+v", nodeType, nodeID, daily.Summary, realtime.Summary)
		}
		t.Logf("aggregate matches realtime node=%s/%d date=%s orders=%d pay=%.2f ad=%.2f profit=%.2f",
			nodeType, nodeID, statDate, daily.Summary.OrderCount, daily.Summary.PayAmount, daily.Summary.AdCostAmount, daily.Summary.RealtimeProfit)
	}

	compareNode(companyID, "company")
	if traderID, parseErr := strconv.ParseInt(os.Getenv("FINANCE_DAILY_COMPARE_TRADER_ID"), 10, 64); parseErr == nil && traderID > 0 {
		compareNode(model.EncodeTraderID(traderID), "trader")
	}
}

func TestFinanceCompanyChildrenRollupDatabaseIntegration(t *testing.T) {
	if os.Getenv("FINANCE_HIERARCHY_DB_INTEGRATION") != "1" {
		t.Skip("set FINANCE_HIERARCHY_DB_INTEGRATION=1 to run against the configured database")
	}
	statDate := os.Getenv("FINANCE_HIERARCHY_TEST_DATE")
	companyID, err := strconv.ParseInt(os.Getenv("FINANCE_HIERARCHY_TEST_COMPANY_ID"), 10, 64)
	if statDate == "" || err != nil || companyID <= 0 {
		t.Fatal("FINANCE_HIERARCHY_TEST_DATE and FINANCE_HIERARCHY_TEST_COMPANY_ID are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	service := NewFinanceCommissionService()
	parent, err := service.GetStats(ctx, &model.FinanceStatsReq{
		NodeId: companyID, NodeType: "company", TimeType: "custom",
		StartDate: statDate, EndDate: statDate, Platform: "douyin",
	})
	if err != nil {
		t.Fatalf("query company stats failed: %v", err)
	}
	children, err := service.GetChildren(ctx, &model.FinanceChildrenReq{
		NodeId: companyID, NodeType: "company", TimeType: "custom",
		StartDate: statDate, EndDate: statDate, Platform: "douyin",
	})
	if err != nil {
		t.Fatalf("query company children failed: %v", err)
	}
	rolledUp := model.FinanceStatsSummary{}
	for _, child := range children.Children {
		childStats := financeSummaryFromNodeStats(child)
		mergeFinanceStatsSummary(&rolledUp, &childStats)
	}
	want := parent.Summary
	// 子节点模型不展示成本命中数和佣金订单数，不纳入展示层守恒校验。
	want.ValidOrderCostMatch = 0
	want.CommissionOrderCount = 0
	rolledUp.ValidOrderCostMatch = 0
	rolledUp.CommissionOrderCount = 0
	if !reflect.DeepEqual(rolledUp, want) {
		t.Fatalf("company children do not roll up to parent: children=%+v parent=%+v", rolledUp, want)
	}
	t.Logf("hierarchy rollup date=%s company=%d children=%d orders=%d pay=%.2f ad=%.2f profit=%.2f",
		statDate, companyID, len(children.Children), rolledUp.OrderCount, rolledUp.PayAmount, rolledUp.AdCostAmount, rolledUp.RealtimeProfit)
}

func TestFinanceTraderAccountRollupDatabaseIntegration(t *testing.T) {
	if os.Getenv("FINANCE_HIERARCHY_DB_INTEGRATION") != "1" {
		t.Skip("set FINANCE_HIERARCHY_DB_INTEGRATION=1 to run against the configured database")
	}
	statDate := os.Getenv("FINANCE_HIERARCHY_TEST_DATE")
	traderID, err := strconv.ParseInt(os.Getenv("FINANCE_HIERARCHY_TEST_TRADER_ID"), 10, 64)
	if statDate == "" || err != nil || traderID <= 0 {
		t.Skip("FINANCE_HIERARCHY_TEST_TRADER_ID is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	service := NewFinanceCommissionService()
	encodedTraderID := model.EncodeTraderID(traderID)
	parent, err := service.GetStats(ctx, &model.FinanceStatsReq{
		NodeId: encodedTraderID, NodeType: "trader", TimeType: "custom",
		StartDate: statDate, EndDate: statDate, Platform: "douyin",
	})
	if err != nil {
		t.Fatalf("query trader stats failed: %v", err)
	}
	children, err := service.GetChildren(ctx, &model.FinanceChildrenReq{
		NodeId: encodedTraderID, NodeType: "trader", TimeType: "custom",
		StartDate: statDate, EndDate: statDate, Platform: "douyin",
	})
	if err != nil {
		t.Fatalf("query trader account children failed: %v", err)
	}
	rolledUp := model.FinanceStatsSummary{}
	for _, child := range children.Children {
		childStats := financeSummaryFromNodeStats(child)
		mergeFinanceStatsSummary(&rolledUp, &childStats)
	}
	want := parent.Summary
	want.ValidOrderCostMatch = 0
	want.CommissionOrderCount = 0
	rolledUp.ValidOrderCostMatch = 0
	rolledUp.CommissionOrderCount = 0
	if !reflect.DeepEqual(rolledUp, want) {
		t.Fatalf("trader accounts do not roll up to parent: accounts=%+v parent=%+v", rolledUp, want)
	}
	t.Logf("account rollup date=%s trader=%d accounts=%d orders=%d pay=%.2f ad=%.2f profit=%.2f",
		statDate, traderID, len(children.Children), rolledUp.OrderCount, rolledUp.PayAmount, rolledUp.AdCostAmount, rolledUp.RealtimeProfit)
}

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
