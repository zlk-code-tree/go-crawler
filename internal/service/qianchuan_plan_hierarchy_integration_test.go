package service

import (
	"context"
	"os"
	"testing"
	"time"

	_ "go-crawler/internal/packed"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
)

func TestQianchuanPlanHierarchyDatabaseIntegration(t *testing.T) {
	if os.Getenv("QIANCHUAN_DB_INTEGRATION") != "1" {
		t.Skip("set QIANCHUAN_DB_INTEGRATION=1 to run against the configured database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result, err := NewQianchuanPlanHierarchyService().Query(ctx, QianchuanPlanHierarchyRequest{
		Level:     QianchuanHierarchyLevelStore,
		StartDate: time.Now().AddDate(0, 0, -7).Format("2006-01-02"),
		EndDate:   time.Now().Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("query store hierarchy failed: %v", err)
	}
	t.Logf("store nodes=%d summary=%+v", len(result.List), result.Summary)

	for _, store := range result.List {
		if !store.HasChildren {
			continue
		}
		advertisers, queryErr := NewQianchuanPlanHierarchyService().Query(ctx, QianchuanPlanHierarchyRequest{
			Level:     QianchuanHierarchyLevelAdvertiser,
			StartDate: result.StartDate,
			EndDate:   result.EndDate,
			StoreID:   store.StoreID,
		})
		if queryErr != nil {
			t.Fatalf("query advertisers for store %d failed: %v", store.StoreID, queryErr)
		}
		if len(advertisers.List) == 0 {
			continue
		}
		_, queryErr = NewQianchuanPlanHierarchyService().Query(ctx, QianchuanPlanHierarchyRequest{
			Level:        QianchuanHierarchyLevelPlan,
			StartDate:    result.StartDate,
			EndDate:      result.EndDate,
			AdvertiserID: advertisers.List[0].AdvertiserID,
		})
		if queryErr != nil {
			t.Fatalf("query plans for advertiser %d failed: %v", advertisers.List[0].AdvertiserID, queryErr)
		}
		break
	}
}
