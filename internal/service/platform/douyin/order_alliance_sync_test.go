package douyin

import (
	"reflect"
	"testing"
)

func TestUniqueStrings(t *testing.T) {
	input := []string{"1001", "", "1002", "1001", "1003", "1002"}
	want := []string{"1001", "1002", "1003"}
	if got := uniqueStrings(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueStrings() = %v, want %v", got, want)
	}
}

func TestChunkStrings(t *testing.T) {
	values := make([]string, 45)
	for i := range values {
		values[i] = string(rune('a' + i))
	}
	chunks := chunkStrings(values, allianceOrderBatchSize)
	if len(chunks) != 3 {
		t.Fatalf("chunkStrings() returned %d chunks, want 3", len(chunks))
	}
	wantSizes := []int{20, 20, 5}
	for i, wantSize := range wantSizes {
		if len(chunks[i]) != wantSize {
			t.Fatalf("chunk %d size = %d, want %d", i, len(chunks[i]), wantSize)
		}
	}
}

func TestSplitOrderAllianceWindowsMonthly(t *testing.T) {
	windows, err := splitOrderAllianceWindows(PollParams{
		StartTime:   "2026-07-01 00:00:00",
		EndTime:     "2026-07-31 23:59:59",
		StartPage:   7,
		Concurrency: 99,
	})
	if err != nil {
		t.Fatalf("splitOrderAllianceWindows() error = %v", err)
	}
	if len(windows) != 31 {
		t.Fatalf("window count = %d, want 31", len(windows))
	}
	if windows[0].StartTime != "2026-07-01 00:00:00" || windows[0].EndTime != "2026-07-01 23:59:59" {
		t.Fatalf("first window = %s ~ %s", windows[0].StartTime, windows[0].EndTime)
	}
	if windows[30].StartTime != "2026-07-31 00:00:00" || windows[30].EndTime != "2026-07-31 23:59:59" {
		t.Fatalf("last window = %s ~ %s", windows[30].StartTime, windows[30].EndTime)
	}
	if windows[0].StartPage != 7 || windows[1].StartPage != 0 {
		t.Fatalf("start pages = %d, %d; want 7, 0", windows[0].StartPage, windows[1].StartPage)
	}
	for index, window := range windows {
		if window.Concurrency != orderAllianceMaxConcurrency {
			t.Fatalf("window %d concurrency = %d, want %d", index, window.Concurrency, orderAllianceMaxConcurrency)
		}
	}
}

func TestSplitOrderAllianceWindowsPartialDay(t *testing.T) {
	windows, err := splitOrderAllianceWindows(PollParams{
		StartTime: "2026-07-21 08:30:00",
		EndTime:   "2026-07-21 18:45:00",
	})
	if err != nil {
		t.Fatalf("splitOrderAllianceWindows() error = %v", err)
	}
	if len(windows) != 1 {
		t.Fatalf("window count = %d, want 1", len(windows))
	}
	if windows[0].Concurrency != 3 {
		t.Fatalf("default concurrency = %d, want 3", windows[0].Concurrency)
	}
}

func TestSplitOrderAllianceWindowsRejectsInvalidRange(t *testing.T) {
	_, err := splitOrderAllianceWindows(PollParams{
		StartTime: "2026-07-21 23:59:59",
		EndTime:   "2026-07-21 00:00:00",
	})
	if err == nil {
		t.Fatal("splitOrderAllianceWindows() error = nil, want invalid range error")
	}
}

func TestSplitOrderAllianceWindowsRejectsOver31Days(t *testing.T) {
	_, err := splitOrderAllianceWindows(PollParams{
		StartTime: "2026-06-01 00:00:00",
		EndTime:   "2026-07-31 23:59:59",
	})
	if err == nil {
		t.Fatal("splitOrderAllianceWindows() error = nil, want range limit error")
	}
}

func TestMergeOrderAllianceSyncResultDropsOrderIDs(t *testing.T) {
	target := newOrderAllianceSyncResult(2)
	source := &OrderAllianceSyncResult{
		Orders: &PollResult{
			StoreCount:  1,
			TotalOrders: 2,
			TotalSaved:  2,
			StoreResults: []StorePollResult{{
				StoreID: 1, StoreName: "测试店铺", TotalOrders: 2, TotalSaved: 2,
				OrderIDs: []string{"1001", "1002"},
			}},
		},
		Alliance: &AlliancePullResult{MatchedOrders: 2},
	}

	mergeOrderAllianceSyncResult(target, source)

	if len(target.Orders.StoreResults) != 1 {
		t.Fatalf("store result count = %d, want 1", len(target.Orders.StoreResults))
	}
	if len(target.Orders.StoreResults[0].OrderIDs) != 0 {
		t.Fatalf("aggregated result retained order IDs: %v", target.Orders.StoreResults[0].OrderIDs)
	}
	if target.Orders.TotalSaved != 2 || target.Alliance.MatchedOrders != 2 {
		t.Fatalf("merged totals = saved %d matched %d", target.Orders.TotalSaved, target.Alliance.MatchedOrders)
	}
}
