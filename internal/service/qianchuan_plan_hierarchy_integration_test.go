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

	awemes, err := NewQianchuanPlanHierarchyService().Query(ctx, QianchuanPlanHierarchyRequest{
		Level:     QianchuanHierarchyLevelAweme,
		StartDate: result.StartDate,
		EndDate:   result.EndDate,
	})
	if err != nil {
		t.Fatalf("query aweme hierarchy failed: %v", err)
	}
	if len(awemes.List) > 0 {
		aweme := awemes.List[0]
		advertisers, queryErr := NewQianchuanPlanHierarchyService().Query(ctx, QianchuanPlanHierarchyRequest{
			Level:       QianchuanHierarchyLevelAdvertiser,
			StartDate:   result.StartDate,
			EndDate:     result.EndDate,
			AwemeID:     aweme.AwemeID,
			AwemeShowID: aweme.AwemeShowID,
		})
		if queryErr != nil {
			t.Fatalf("query advertisers for aweme %s failed: %v", aweme.NodeKey, queryErr)
		}
		if aweme.HasChildren && len(advertisers.List) == 0 {
			t.Fatalf("expected advertisers for aweme %s", aweme.NodeKey)
		}
		if len(advertisers.List) > 0 {
			advertiser := advertisers.List[0]
			plans, planErr := NewQianchuanPlanHierarchyService().Query(ctx, QianchuanPlanHierarchyRequest{
				Level:        QianchuanHierarchyLevelPlan,
				StartDate:    result.StartDate,
				EndDate:      result.EndDate,
				AdvertiserID: advertiser.AdvertiserID,
				AwemeID:      advertiser.AwemeID,
				AwemeShowID:  advertiser.AwemeShowID,
			})
			if planErr != nil {
				t.Fatalf("query plans for aweme advertiser %d failed: %v", advertiser.AdvertiserID, planErr)
			}
			if advertiser.HasChildren && len(plans.List) == 0 {
				t.Fatalf("expected plans for aweme advertiser %d", advertiser.AdvertiserID)
			}
		}
	}

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

func TestQianchuanPlanDetailDatabaseIntegration(t *testing.T) {
	if os.Getenv("QIANCHUAN_DB_INTEGRATION") != "1" {
		t.Skip("set QIANCHUAN_DB_INTEGRATION=1 to run against the configured database")
	}
	statDate := os.Getenv("QIANCHUAN_INTEGRATION_DATE")
	if statDate == "" {
		t.Skip("set QIANCHUAN_INTEGRATION_DATE to verify a specific plan report date")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result, err := NewQianchuanPlanHierarchyService().Query(ctx, QianchuanPlanHierarchyRequest{
		Level:       QianchuanHierarchyLevelPlan,
		StartDate:   statDate,
		EndDate:     statDate,
		GroupByDate: true,
	})
	if err != nil {
		t.Fatalf("query plan detail failed: %v", err)
	}
	if result.Summary.FinancialScope != "douyin_order_snapshot" {
		t.Fatalf("unexpected financial scope: %+v", result.Summary)
	}
	if result.Summary.TotalValidOrderCount <= 0 || result.Summary.TotalPayAmount <= 0 {
		t.Fatalf("expected canonical order totals: %+v", result.Summary)
	}
	t.Logf("plan detail rows=%d summary=%+v", len(result.List), result.Summary)
}
