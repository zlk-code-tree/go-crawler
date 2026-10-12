package crawler

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type videoProgressCheckpoint struct {
	next   int
	status string
	reason string
}

func videoProgressProducts() []ProductInfo {
	return []ProductInfo{{ProductID: "first"}, {ProductID: "second"}, {ProductID: "third"}, {ProductID: "fourth"}}
}

func successfulVideoPage(product ProductInfo) videoPageResult {
	return videoPageResult{
		Status: videoPageAccessAvailable,
		Videos: []VideoAnalysis{{ProductID: product.ProductID, VideoId: "video-" + product.ProductID}},
	}
}

func TestVideoCollectionPersistenceFailureKeepsCurrentProduct(t *testing.T) {
	writeErr := errors.New("database write failed")
	var fetched []string
	var checkpoints []videoProgressCheckpoint
	collection := videoCollection{
		fetch: func(product ProductInfo) videoPageResult {
			fetched = append(fetched, product.ProductID)
			return successfulVideoPage(product)
		},
		persist: func([]VideoAnalysis) error { return writeErr },
		checkpoint: func(next int, status, reason string) error {
			checkpoints = append(checkpoints, videoProgressCheckpoint{next, status, reason})
			return nil
		},
	}
	result := collection.run(context.Background(), videoProgressProducts(), 1)
	if !errors.Is(result.Err, writeErr) || result.Completed || !result.Paused || result.NextProductIndex != 1 {
		t.Fatalf("failed write advanced the product or reported completion: %+v", result)
	}
	if len(result.Videos) != 0 || !reflect.DeepEqual(fetched, []string{"second"}) {
		t.Fatalf("unpersisted data was accepted or later products fetched: result=%+v, fetched=%v", result, fetched)
	}
	last := checkpoints[len(checkpoints)-1]
	if last.next != 1 || last.status != "paused" || last.reason == "" {
		t.Fatalf("failed product was not checkpointed for retry: %+v", last)
	}
}

func TestVideoCollectionCheckpointFailureStopsFetching(t *testing.T) {
	for _, failureCall := range []int{1, 2} {
		name := "initial_checkpoint"
		if failureCall == 2 {
			name = "after_persisting_product"
		}
		t.Run(name, func(t *testing.T) {
			checkpointErr := errors.New("checkpoint write failed")
			fetchCount, persistCount, checkpointCount := 0, 0, 0
			collection := videoCollection{
				fetch: func(product ProductInfo) videoPageResult {
					fetchCount++
					return successfulVideoPage(product)
				},
				persist: func([]VideoAnalysis) error { persistCount++; return nil },
				checkpoint: func(int, string, string) error {
					checkpointCount++
					if checkpointCount == failureCall {
						return checkpointErr
					}
					return nil
				},
			}
			result := collection.run(context.Background(), videoProgressProducts(), 0)
			if !errors.Is(result.Err, checkpointErr) || result.Completed || result.NextProductIndex != 0 {
				t.Fatalf("checkpoint failure lost the retry position: %+v", result)
			}
			wantCount := failureCall - 1
			if fetchCount != wantCount || persistCount != wantCount {
				t.Fatalf("continued after checkpoint failure: fetch=%d, persist=%d; want %d each", fetchCount, persistCount, wantCount)
			}
		})
	}
}

func TestVideoCollectionCancellationKeepsCurrentProduct(t *testing.T) {
	for _, cancelBeforeFetch := range []bool{true, false} {
		name := "during_fetch"
		if cancelBeforeFetch {
			name = "before_fetch"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if cancelBeforeFetch {
				cancel()
			}
			fetchCount, persistCount := 0, 0
			collection := videoCollection{
				fetch: func(product ProductInfo) videoPageResult {
					fetchCount++
					cancel()
					return successfulVideoPage(product)
				},
				persist: func([]VideoAnalysis) error { persistCount++; return nil },
			}
			result := collection.run(ctx, videoProgressProducts(), 1)
			if !errors.Is(result.Err, context.Canceled) || result.Completed || !result.Paused || result.NextProductIndex != 1 {
				t.Fatalf("cancellation advanced the product or reported completion: %+v", result)
			}
			wantFetchCount := 1
			if cancelBeforeFetch {
				wantFetchCount = 0
			}
			if fetchCount != wantFetchCount || persistCount != 0 || len(result.Videos) != 0 {
				t.Fatalf("unexpected work after cancellation: fetched=%d, persisted=%d, result=%+v", fetchCount, persistCount, result)
			}
		})
	}
}

func TestVideoCollectionViewLimitResumesRemainingProducts(t *testing.T) {
	products := videoProgressProducts()
	var fetched, persisted []string
	collection := videoCollection{
		maxViews: 2,
		fetch: func(product ProductInfo) videoPageResult {
			fetched = append(fetched, product.ProductID)
			return successfulVideoPage(product)
		},
		persist: func(videos []VideoAnalysis) error {
			for _, video := range videos {
				persisted = append(persisted, video.ProductID)
			}
			return nil
		},
	}
	first := collection.run(context.Background(), products, 0)
	if first.Err != nil || first.Completed || !first.Paused || first.NextProductIndex != 2 {
		t.Fatalf("unexpected result at the view limit: %+v", first)
	}
	second := collection.run(context.Background(), products, first.NextProductIndex)
	if second.Err != nil || !second.Completed || second.Paused || second.NextProductIndex != len(products) {
		t.Fatalf("resume did not complete the remaining products: %+v", second)
	}
	want := []string{"first", "second", "third", "fourth"}
	if !reflect.DeepEqual(fetched, want) || !reflect.DeepEqual(persisted, want) {
		t.Fatalf("resume skipped or repeated products: fetched=%v, persisted=%v", fetched, persisted)
	}
}

func TestVideoCollectionDailyLimitRetriesSameProduct(t *testing.T) {
	limited := true
	var fetched []string
	collection := videoCollection{
		fetch: func(product ProductInfo) videoPageResult {
			fetched = append(fetched, product.ProductID)
			if limited {
				return videoPageResult{Status: videoPageAccessDailyLimit, Note: "quota reached"}
			}
			return successfulVideoPage(product)
		},
	}
	first := collection.run(context.Background(), videoProgressProducts(), 2)
	if first.Err != nil || first.Completed || !first.Paused || first.NextProductIndex != 2 {
		t.Fatalf("daily limit advanced past the uncollected product: %+v", first)
	}
	limited = false
	second := collection.run(context.Background(), videoProgressProducts(), first.NextProductIndex)
	if second.Err != nil || !second.Completed || second.NextProductIndex != 4 {
		t.Fatalf("retry after the daily limit did not complete: %+v", second)
	}
	if !reflect.DeepEqual(fetched, []string{"third", "third", "fourth"}) {
		t.Fatalf("daily-limit product was not retried: %v", fetched)
	}
}

func TestVideoCollectionRestrictedPageKeepsCurrentProduct(t *testing.T) {
	var fetched []string
	var last videoProgressCheckpoint
	collection := videoCollection{
		fetch: func(product ProductInfo) videoPageResult {
			fetched = append(fetched, product.ProductID)
			return videoPageResult{Status: videoPageAccessRestricted, Note: "verification required"}
		},
		checkpoint: func(next int, status, reason string) error {
			last = videoProgressCheckpoint{next, status, reason}
			return nil
		},
	}
	result := collection.run(context.Background(), videoProgressProducts(), 1)
	if result.Err != nil || result.Completed || !result.Paused || result.NextProductIndex != 1 {
		t.Fatalf("restricted page advanced past the uncollected product: %+v", result)
	}
	if !reflect.DeepEqual(fetched, []string{"second"}) || last.next != 1 || last.status != "paused" || last.reason == "" {
		t.Fatalf("restricted product was not checkpointed for retry: fetched=%v, checkpoint=%+v", fetched, last)
	}
}

func TestVideoCollectionEmptyThresholdPausesAtNextProduct(t *testing.T) {
	var fetched []string
	var last videoProgressCheckpoint
	collection := videoCollection{
		maxEmpty: 2,
		fetch: func(product ProductInfo) videoPageResult {
			fetched = append(fetched, product.ProductID)
			return videoPageResult{Status: videoPageAccessNoData}
		},
		persist: func([]VideoAnalysis) error {
			t.Fatal("empty results must not be persisted")
			return nil
		},
		checkpoint: func(next int, status, reason string) error {
			last = videoProgressCheckpoint{next, status, reason}
			return nil
		},
	}
	result := collection.run(context.Background(), videoProgressProducts(), 1)
	if result.Err != nil || result.Completed || !result.Paused || result.NextProductIndex != 3 {
		t.Fatalf("empty threshold produced an incorrect resume position: %+v", result)
	}
	if !reflect.DeepEqual(fetched, []string{"second", "third"}) || last.next != 3 || last.status != "paused" || last.reason == "" {
		t.Fatalf("incorrect empty-result checkpoint: fetched=%v, checkpoint=%+v", fetched, last)
	}
}

func TestVideoCollectionFetchFailureKeepsCurrentProduct(t *testing.T) {
	fetchErr := errors.New("page navigation failed")
	fetchCount := 0
	collection := videoCollection{
		fetch: func(ProductInfo) videoPageResult {
			fetchCount++
			return videoPageResult{Err: fetchErr}
		},
	}
	result := collection.run(context.Background(), videoProgressProducts(), 1)
	if !errors.Is(result.Err, fetchErr) || result.Completed || result.NextProductIndex != 1 || fetchCount != 1 {
		t.Fatalf("fetch failure advanced past the uncollected product: %+v, fetched=%d", result, fetchCount)
	}
}
