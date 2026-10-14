package crawler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCrawlerBatchBoundsConcurrencyAndPreservesOrder(t *testing.T) {
	const total, concurrency = 18, 3
	items := make([]int, total)
	releases := make([]chan struct{}, total)
	for index := range items {
		items[index] = index
		releases[index] = make(chan struct{})
	}
	started := make(chan int, total)
	var running, peak atomic.Int32
	done := make(chan []crawlerBatchResult[int], 1)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go func() {
		results, err := runCrawlerBatch(ctx, items, crawlerBatchOptions{Concurrency: concurrency, TaskTimeout: time.Second}, func(ctx context.Context, index, item int) (int, error) {
			active := running.Add(1)
			defer running.Add(-1)
			for previous := peak.Load(); active > previous; previous = peak.Load() {
				if peak.CompareAndSwap(previous, active) {
					break
				}
			}
			started <- index
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-releases[index]:
				return item * 10, nil
			}
		})
		if err != nil {
			t.Errorf("batch failed: %v", err)
		}
		done <- results
	}()
	initial := make([]int, concurrency)
	for index := range initial {
		select {
		case initial[index] = <-started:
		case <-ctx.Done():
			t.Fatal("initial workers did not start")
		}
	}
	select {
	case extra := <-started:
		t.Fatalf("task %d started without available capacity", extra)
	case <-time.After(20 * time.Millisecond):
	}
	// Complete the first three tasks in reverse startup order.
	for index := len(initial) - 1; index >= 0; index-- {
		close(releases[initial[index]])
	}
	for index := concurrency; index < total; index++ {
		select {
		case next := <-started:
			close(releases[next])
		case <-ctx.Done():
			t.Fatal("batch did not dispatch remaining items")
		}
	}
	select {
	case results := <-done:
		for index, result := range results {
			if !result.Executed || result.Err != nil || result.Value != index*10 {
				t.Fatalf("result %d does not retain input order: %+v", index, result)
			}
		}
	case <-ctx.Done():
		t.Fatal("batch did not join workers")
	}
	if peak.Load() != concurrency || running.Load() != 0 {
		t.Fatalf("peak=%d, remaining=%d", peak.Load(), running.Load())
	}
}

func TestCrawlerBatchCancellationJoinsCleanupAndStopsDispatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started, cleaning, cleanup := make(chan struct{}), make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	var calls atomic.Int32
	go func() {
		results, err := runCrawlerBatch(ctx, []int{0, 1, 2}, crawlerBatchOptions{Concurrency: 1, TaskTimeout: time.Hour}, func(ctx context.Context, _, item int) (int, error) {
			calls.Add(1)
			close(started)
			<-ctx.Done()
			close(cleaning)
			<-cleanup
			return item, ctx.Err()
		})
		if !results[0].Executed || results[1].Executed || results[2].Executed {
			t.Errorf("unexpected dispatched results: %+v", results)
		}
		done <- err
	}()
	waitCrawlerTestSignal(t, started)
	cancel()
	waitCrawlerTestSignal(t, cleaning)
	select {
	case err := <-done:
		t.Fatalf("returned before worker cleanup completed: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(cleanup)
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || calls.Load() != 1 {
			t.Fatalf("err=%v, calls=%d", err, calls.Load())
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled batch did not finish")
	}
}

func TestCrawlerBatchOrdinaryErrorAndTimeoutContinue(t *testing.T) {
	wantErr := errors.New("item failure")
	results, err := runCrawlerBatch(context.Background(), []int{0, 1, 2}, crawlerBatchOptions{Concurrency: 1, TaskTimeout: 10 * time.Millisecond}, func(ctx context.Context, _, item int) (int, error) {
		switch item {
		case 0:
			return 0, wantErr
		case 1:
			<-ctx.Done()
			return 0, ctx.Err()
		default:
			return 42, nil
		}
	})
	if err != nil || !errors.Is(results[0].Err, wantErr) || !errors.Is(results[1].Err, context.DeadlineExceeded) || results[2].Err != nil || results[2].Value != 42 {
		t.Fatalf("ordinary errors stopped or corrupted batch: results=%+v, err=%v", results, err)
	}
}

func TestCrawlerBatchFatalErrorCancelsOtherWorkers(t *testing.T) {
	wantErr := errors.New("quota exhausted")
	secondStarted := make(chan struct{})
	var calls, cleaned atomic.Int32
	results, err := runCrawlerBatch(context.Background(), []int{0, 1, 2, 3}, crawlerBatchOptions{Concurrency: 2, TaskTimeout: time.Second}, func(ctx context.Context, _, item int) (int, error) {
		calls.Add(1)
		if item == 0 {
			select {
			case <-secondStarted:
				return 0, fatalCrawlerError(wantErr)
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}
		close(secondStarted)
		<-ctx.Done()
		cleaned.Add(1)
		return 0, ctx.Err()
	})
	if !errors.Is(err, wantErr) || calls.Load() != 2 || cleaned.Load() != 1 || results[2].Executed || results[3].Executed {
		t.Fatalf("fatal batch handling: err=%v, calls=%d, cleaned=%d, results=%+v", err, calls.Load(), cleaned.Load(), results)
	}
}

func TestCrawlerBatchPanicCapturesStackAndStopsBatch(t *testing.T) {
	results, err := runCrawlerBatch(context.Background(), []int{0, 1}, crawlerBatchOptions{Concurrency: 1, TaskTimeout: time.Second}, func(context.Context, int, int) (int, error) {
		panic("unexpected detail parser state")
	})
	var panicErr *crawlerTaskPanicError
	if !errors.As(err, &panicErr) || !strings.Contains(string(panicErr.Stack), "TestCrawlerBatchPanicCapturesStackAndStopsBatch") || results[1].Executed {
		t.Fatalf("panic was not preserved as fatal error: %v, results=%+v", err, results)
	}
}

func TestCrawlerBatchValidatesParametersAndPrecancelledContext(t *testing.T) {
	fn := func(context.Context, int, int) (int, error) { t.Fatal("invalid batch executed a task"); return 0, nil }
	for index, options := range []crawlerBatchOptions{{Concurrency: 0, TaskTimeout: time.Second}, {Concurrency: -1, TaskTimeout: time.Second}, {Concurrency: 1}, {Concurrency: 1, TaskTimeout: -time.Second}} {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			if _, err := runCrawlerBatch(context.Background(), []int{1}, options, fn); err == nil {
				t.Fatal("invalid options accepted")
			}
		})
	}
	options := crawlerBatchOptions{Concurrency: 1, TaskTimeout: time.Second}
	if _, err := runCrawlerBatch[int, int](context.Background(), []int{1}, options, nil); err == nil {
		t.Fatal("nil task accepted")
	}
	if _, err := runCrawlerBatch(nil, []int{1}, options, fn); err == nil {
		t.Fatal("nil context accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results, err := runCrawlerBatch(ctx, []int{1}, options, fn)
	if !errors.Is(err, context.Canceled) || results[0].Executed || !errors.Is(results[0].Err, context.Canceled) {
		t.Fatalf("pre-cancelled batch handling: results=%+v, err=%v", results, err)
	}
}

func waitCrawlerTestSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for worker signal")
	}
}
