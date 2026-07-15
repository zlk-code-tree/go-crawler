package service

import (
	"context"
	"math"
	"os"
	"strconv"
	"testing"
	"time"

	_ "go-crawler/internal/packed"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
)

func TestStoreCostDatabaseIntegration(t *testing.T) {
	if os.Getenv("FINANCE_STORE_COST_DB_INTEGRATION") != "1" {
		t.Skip("set FINANCE_STORE_COST_DB_INTEGRATION=1 to run against the configured database")
	}
	statDate := os.Getenv("FINANCE_STORE_COST_TEST_DATE")
	if statDate == "" {
		t.Fatal("FINANCE_STORE_COST_TEST_DATE is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	financeService := NewFinanceCommissionService()
	stores, err := financeService.GetStoreCostStats(ctx, statDate, statDate, 100)
	if err != nil {
		t.Fatalf("query store costs failed: %v", err)
	}

	accountCount := int64(0)
	unmatchedCount := int64(0)
	totalCost := 0.0
	for _, store := range stores {
		accountCount += store.AdvertiserCount
		totalCost += store.StatCost
		if !store.MatchedStore {
			unmatchedCount += store.AdvertiserCount
		}
		details, detailTotal, detailErr := financeService.GetStoreCostDetail(ctx, statDate, statDate, store.StoreID, 1, 500)
		if detailErr != nil {
			t.Fatalf("query store %d details failed: %v", store.StoreID, detailErr)
		}
		if detailTotal != store.AdvertiserCount {
			t.Fatalf("store %d account count mismatch: summary=%d detail=%d", store.StoreID, store.AdvertiserCount, detailTotal)
		}
		detailCost := 0.0
		for _, detail := range details {
			detailCost += detail.StatCost
		}
		if math.Abs(detailCost-store.StatCost) > 0.05 {
			t.Fatalf("store %d cost mismatch: summary=%.2f detail=%.2f", store.StoreID, store.StatCost, detailCost)
		}
	}

	allDetails, allErr := financeService.GetAllStoreCostDetail(ctx, statDate, statDate)
	if allErr != nil {
		t.Fatalf("query all store details failed: %v", allErr)
	}
	if int64(len(allDetails)) != accountCount {
		t.Fatalf("all store detail account count mismatch: summary=%d detail=%d", accountCount, len(allDetails))
	}
	allDetailCost := 0.0
	for _, detail := range allDetails {
		allDetailCost += detail.StatCost
	}
	if math.Abs(allDetailCost-totalCost) > 0.05 {
		t.Fatalf("all store detail cost mismatch: summary=%.2f detail=%.2f", totalCost, allDetailCost)
	}
	exportFile, exportErr := financeService.ExportAllStoreCostDetail(ctx, statDate, statDate)
	if exportErr != nil {
		t.Fatalf("export all store details failed: %v", exportErr)
	}
	if _, statErr := os.Stat(exportFile); statErr != nil {
		t.Fatalf("export all store details file missing: %v", statErr)
	}
	_ = os.Remove(exportFile)

	assertStoreCostCount(t, "FINANCE_STORE_COST_EXPECT_STORES", int64(len(stores)))
	assertStoreCostCount(t, "FINANCE_STORE_COST_EXPECT_ACCOUNTS", accountCount)
	assertStoreCostCount(t, "FINANCE_STORE_COST_EXPECT_UNMATCHED", unmatchedCount)
	t.Logf("date=%s stores=%d accounts=%d unmatched=%d totalCost=%.2f", statDate, len(stores), accountCount, unmatchedCount, totalCost)
}

func assertStoreCostCount(t *testing.T, envName string, actual int64) {
	t.Helper()
	expectedText := os.Getenv(envName)
	if expectedText == "" {
		return
	}
	expected, err := strconv.ParseInt(expectedText, 10, 64)
	if err != nil {
		t.Fatalf("invalid %s: %v", envName, err)
	}
	if actual != expected {
		t.Fatalf("%s mismatch: expected=%d actual=%d", envName, expected, actual)
	}
}
