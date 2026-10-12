package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func newTestProductDetailCache(t *testing.T) *ProductDetailCache {
	t.Helper()
	return &ProductDetailCache{
		records:   make(map[string]*ProductDetailRecord),
		cachePath: filepath.Join(t.TempDir(), CacheFileName),
	}
}

func TestProductDetailCacheUpsertPersistsWithoutDeadlock(t *testing.T) {
	cache := newTestProductDetailCache(t)
	done := make(chan *ProductDetailRecord, 1)
	go func() {
		done <- cache.UpsertRecord(&ProductDetailRecord{
			ProductID: " product-1 ", FirstShelfTime: "2026-10-01", MinPrice: "19.90",
		})
	}()

	select {
	case record := <-done:
		if record == nil || record.ProductID != "product-1" || record.UpdatedAt.IsZero() {
			t.Fatalf("unexpected stored record: %+v", record)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cache upsert deadlocked")
	}

	reloaded := &ProductDetailCache{cachePath: cache.cachePath}
	reloaded.loadRecords()
	if record := reloaded.GetRecord("product-1"); record == nil || record.MinPrice != "19.90" {
		t.Fatalf("persisted record was not restored: %+v", record)
	}
}

func TestProductDetailCacheRecordsAreIndependentCopies(t *testing.T) {
	cache := newTestProductDetailCache(t)
	insurance := true
	input := &ProductDetailRecord{ProductID: "product-1", MinPrice: "19.90", HasShippingInsurance: &insurance}
	stored, err := cache.UpsertRecordWithError(input)
	if err != nil {
		t.Fatal(err)
	}
	if !input.UpdatedAt.IsZero() {
		t.Fatal("upsert mutated the caller's record")
	}

	input.MinPrice = "input mutation"
	insurance = false
	stored.MinPrice = "returned mutation"
	*stored.HasShippingInsurance = false
	read := cache.GetRecord("product-1")
	read.MinPrice = "read mutation"
	*read.HasShippingInsurance = false
	batch := cache.GetRecords([]string{"product-1"})
	batch["product-1"].MinPrice = "batch mutation"
	*batch["product-1"].HasShippingInsurance = false

	actual := cache.GetRecord("product-1")
	if actual.MinPrice != "19.90" || actual.HasShippingInsurance == nil || !*actual.HasShippingInsurance {
		t.Fatalf("caller mutation leaked into the cache: %+v", actual)
	}
}

func TestProductDetailCacheConcurrentWritesKeepCompleteSnapshot(t *testing.T) {
	cache := newTestProductDetailCache(t)
	if _, err := cache.UpsertRecordWithError(&ProductDetailRecord{ProductID: "initial"}); err != nil {
		t.Fatal(err)
	}

	const writes = 32
	start := make(chan struct{})
	errors := make(chan error, writes+1)
	stopReader := make(chan struct{})
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		<-start
		for {
			select {
			case <-stopReader:
				return
			default:
			}
			data, err := os.ReadFile(cache.cachePath)
			if err != nil {
				errors <- err
				return
			}
			if !json.Valid(data) {
				errors <- fmt.Errorf("reader observed incomplete JSON: %q", data)
				return
			}
		}
	}()

	var writers sync.WaitGroup
	for i := 0; i < writes; i++ {
		writers.Add(1)
		go func(index int) {
			defer writers.Done()
			<-start
			insurance := true
			id := fmt.Sprintf("product-%d", index)
			input := &ProductDetailRecord{ProductID: id, HasShippingInsurance: &insurance}
			stored, err := cache.UpsertRecordWithError(input)
			if err != nil {
				errors <- err
				return
			}
			// These mutations must not race with another writer's serialization.
			insurance = false
			input.MinPrice = "input mutation"
			stored.MinPrice = "returned mutation"
			read := cache.GetRecord(id)
			*read.HasShippingInsurance = false
		}(i)
	}
	close(start)
	writers.Wait()
	close(stopReader)
	<-readerDone
	close(errors)
	for err := range errors {
		t.Error(err)
	}

	data, err := os.ReadFile(cache.cachePath)
	if err != nil {
		t.Fatal(err)
	}
	var records map[string]*ProductDetailRecord
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != writes+1 {
		t.Fatalf("final snapshot lost updates: got %d records, want %d", len(records), writes+1)
	}
	for id, record := range records {
		if id != "initial" && (record.MinPrice != "" || record.HasShippingInsurance == nil || !*record.HasShippingInsurance) {
			t.Errorf("snapshot contains caller mutations for %s: %+v", id, record)
		}
	}
	files, err := os.ReadDir(filepath.Dir(cache.cachePath))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name() != CacheFileName {
		t.Fatalf("unexpected temporary files after persistence: %v", files)
	}
}

func TestProductDetailCacheReportsPersistenceFailure(t *testing.T) {
	cache := newTestProductDetailCache(t)
	// A directory at the destination prevents replacement after the temp file is written.
	if err := os.Mkdir(cache.cachePath, 0755); err != nil {
		t.Fatal(err)
	}
	stored, err := cache.UpsertRecordWithError(&ProductDetailRecord{ProductID: "product-1"})
	if err == nil {
		t.Fatal("expected a persistence error")
	}
	if stored == nil || cache.GetRecord("product-1") == nil {
		t.Fatal("the in-memory record should remain available after a persistence error")
	}
	files, err := os.ReadDir(filepath.Dir(cache.cachePath))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name() != CacheFileName || !files[0].IsDir() {
		t.Fatalf("failed write changed the destination or leaked a temp file: %v", files)
	}
}

func TestProductDetailCacheHandlesEmptyRecords(t *testing.T) {
	cache := newTestProductDetailCache(t)
	if err := os.WriteFile(cache.cachePath, []byte("null"), 0644); err != nil {
		t.Fatal(err)
	}
	cache.loadRecords()
	for _, input := range []*ProductDetailRecord{nil, {ProductID: "  "}} {
		if record, err := cache.UpsertRecordWithError(input); record != nil || err == nil {
			t.Fatalf("expected invalid input to be rejected, got record=%+v, err=%v", record, err)
		}
	}
	if cache.IsRecordComplete(nil) || !cache.IsRecordStale(nil) {
		t.Fatal("nil records must be treated as incomplete and stale")
	}
	if _, err := cache.UpsertRecordWithError(&ProductDetailRecord{ProductID: "product-1"}); err != nil {
		t.Fatal(err)
	}
}
