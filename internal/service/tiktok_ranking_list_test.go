package service

import (
	"reflect"
	"testing"

	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/container/gvar"
	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

func TestRankingReplayKeepsTaskSnapshotsIndependent(t *testing.T) {
	input := []*model.TikTokRankingListInput{
		{TaskId: " task-a ", ProductId: " product-1 ", Rank: 1, MinPrice: "19.90"},
		{TaskId: "task-a", ProductId: "product-1", Rank: 2},
		{TaskId: "task-b", ProductId: "product-1", Rank: 10, MinPrice: "29.90"},
	}
	batches, err := prepareRankingBatches(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(batches) != 2 || len(batches["task-a"]) != 1 || len(batches["task-b"]) != 1 {
		t.Fatalf("deduplication mixed task snapshots: %+v", batches)
	}
	if batches["task-a"][0].MinPrice != "19.90" || batches["task-b"][0].MinPrice != "29.90" {
		t.Fatal("deduplication replaced the first captured product or another task's values")
	}
	if input[0].TaskId != " task-a " || input[0].ProductId != " product-1 " {
		t.Fatal("normalization mutated the caller's record")
	}
}

func TestRankingReplaySkipsCommittedRowsAndInsertsMissingProducts(t *testing.T) {
	existing := gdb.Result{{
		"product_id": gvar.New("product-1"), "rank": gvar.New(1), "product_name": gvar.New("saved product"),
		"min_price": gvar.New("19.90"), "first_shelf_time": gvar.New("2026-10-01"),
	}, {
		"product_id": gvar.New(""), "rank": gvar.New(2), "product_name": gvar.New("DOM product"),
	}}
	replay := []*model.TikTokRankingListInput{
		{TaskId: "task-a", ProductId: "product-1", Rank: 1},
		{TaskId: "task-a", Rank: 2, ProductName: "DOM product"},
		{TaskId: "task-a", ProductId: "product-3", Rank: 3},
		{TaskId: "task-a", ProductId: "product-3", Rank: 3},
	}
	pending := missingRankingRecords(replay, existing)
	if len(pending) != 1 || pending[0].ProductId != "product-3" {
		t.Fatalf("replay repeated committed products or lost an uncommitted one: %+v", pending)
	}
	if existing[0]["min_price"].String() != "19.90" || existing[0]["first_shelf_time"].String() != "2026-10-01" {
		t.Fatal("replay overwrote saved detail values")
	}
	existing = append(existing, gdb.Record{
		"product_id": gvar.New("product-3"), "rank": gvar.New(3), "product_name": gvar.New("third product"),
	})
	if pending := missingRankingRecords(replay, existing); len(pending) != 0 {
		t.Fatalf("fully committed replay still produced writes: %+v", pending)
	}
}

func TestRankingBatchRejectsRecordsWithoutReplayIdentity(t *testing.T) {
	for _, input := range []*model.TikTokRankingListInput{
		nil,
		{ProductId: "product-1"},
		{TaskId: "task-a"},
		{TaskId: "task-a", ProductName: "missing rank"},
		{TaskId: "task-a", Rank: 1},
	} {
		if batches, err := prepareRankingBatches([]*model.TikTokRankingListInput{input}); err == nil || batches != nil {
			t.Fatalf("record without a stable task/product identity was accepted: %+v", input)
		}
	}
	if batches, err := prepareRankingBatches([]*model.TikTokRankingListInput{
		{TaskId: "task-a", Rank: 1, ProductName: "DOM product"},
	}); err != nil || len(batches["task-a"]) != 1 {
		t.Fatalf("valid DOM fallback identity rejected: batches=%+v, err=%v", batches, err)
	}
}

func TestRankingDetailUpdateRequiresTaskAndProduct(t *testing.T) {
	for _, input := range []*model.TikTokRankingListInput{
		nil, {ProductId: "product-1", MinPrice: "19.90"}, {TaskId: "task-a", MinPrice: "19.90"},
	} {
		if _, _, err := rankingDetailUpdate(input); err == nil {
			t.Fatalf("unsafe update without a task/product scope was accepted: %+v", input)
		}
	}
	where, fields, err := rankingDetailUpdate(&model.TikTokRankingListInput{
		TaskId: " task-a ", ProductId: " product-1 ", MinPrice: "19.90",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(where, g.Map{"task_id": "task-a", "product_id": "product-1"}) {
		t.Fatalf("detail update could affect another task: %+v", where)
	}
	if !reflect.DeepEqual(fields, g.Map{"min_price": "19.90"}) {
		t.Fatalf("partial detail update would clear unrelated fields: %+v", fields)
	}
}

func TestRankingDetailUpdatePreservesUnknownValuesAndWritesFalseInsurance(t *testing.T) {
	input := &model.TikTokRankingListInput{
		TaskId: "task-a", ProductId: "product-1", FirstShelfTime: " ", MinPrice: "\t",
	}
	_, fields, err := rankingDetailUpdate(input)
	if err != nil || len(fields) != 0 {
		t.Fatalf("unknown details would clear stored values: fields=%+v, err=%v", fields, err)
	}
	insurance := 0
	input.HasShippingInsurance = &insurance
	input.DetailAccessStatus = "removed"
	input.DetailAccessNote = "product removed"
	_, fields, err = rankingDetailUpdate(input)
	if err != nil {
		t.Fatal(err)
	}
	want := g.Map{"has_shipping_insurance": 0, "detail_access_status": "removed", "detail_access_note": "product removed"}
	if !reflect.DeepEqual(fields, want) {
		t.Fatalf("known false insurance or access status was lost: got %+v, want %+v", fields, want)
	}
}
