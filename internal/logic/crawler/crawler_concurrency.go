package crawler

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

type crawlerBatchOptions struct {
	Concurrency int
	TaskTimeout time.Duration
}

type crawlerBatchResult[T any] struct {
	Value    T
	Err      error
	Executed bool
}

// fatalCrawlerTaskError stops dispatching the batch and cancels tasks already running.
// Ordinary errors and individual task deadlines remain local to a single item.
type fatalCrawlerTaskError struct{ err error }

func (e *fatalCrawlerTaskError) Error() string { return e.err.Error() }
func (e *fatalCrawlerTaskError) Unwrap() error { return e.err }

func fatalCrawlerError(err error) error {
	if err == nil {
		return nil
	}
	return &fatalCrawlerTaskError{err: err}
}

type crawlerTaskPanicError struct {
	Value any
	Stack []byte
}

func (e *crawlerTaskPanicError) Error() string {
	return fmt.Sprintf("爬虫任务异常：%v", e.Value)
}

// runCrawlerBatch acquires capacity before starting each goroutine. Each result
// has a single writer at its input index. The function always joins started
// tasks, including their cleanup, before the caller may release shared resources.
func runCrawlerBatch[T, R any](ctx context.Context, items []T, options crawlerBatchOptions, fn func(context.Context, int, T) (R, error)) ([]crawlerBatchResult[R], error) {
	if ctx == nil {
		return nil, errors.New("批次 context 不能为空")
	}
	if options.Concurrency < 1 {
		return nil, errors.New("批次并发数必须大于 0")
	}
	if options.TaskTimeout <= 0 {
		return nil, errors.New("单项任务超时必须大于 0")
	}
	if fn == nil {
		return nil, errors.New("批次任务函数不能为空")
	}

	results := make([]crawlerBatchResult[R], len(items))
	batchCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	sem := make(chan struct{}, options.Concurrency)
	var wg sync.WaitGroup

dispatch:
	for index, item := range items {
		if batchCtx.Err() != nil {
			break
		}
		select {
		case <-batchCtx.Done():
			break dispatch
		case sem <- struct{}{}:
		}
		// Both cancellation and capacity can become ready together.
		if batchCtx.Err() != nil {
			<-sem
			break
		}
		wg.Add(1)
		go func(index int, item T) {
			defer wg.Done()
			defer func() { <-sem }()
			taskCtx, taskCancel := context.WithTimeout(batchCtx, options.TaskTimeout)
			defer taskCancel()
			results[index].Executed = true
			results[index].Value, results[index].Err = executeCrawlerTask(taskCtx, index, item, fn)
			var fatal *fatalCrawlerTaskError
			if errors.As(results[index].Err, &fatal) {
				// Publish the fatal cause before releasing capacity for another item.
				cancel(results[index].Err)
			}
		}(index, item)
	}
	wg.Wait()
	err := context.Cause(batchCtx)
	if err != nil {
		for index := range results {
			if !results[index].Executed {
				results[index].Err = err
			}
		}
	}
	return results, err
}

func executeCrawlerTask[T, R any](ctx context.Context, index int, item T, fn func(context.Context, int, T) (R, error)) (value R, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fatalCrawlerError(&crawlerTaskPanicError{Value: recovered, Stack: debug.Stack()})
		}
	}()
	if err = ctx.Err(); err != nil {
		return value, err
	}
	value, err = fn(ctx, index, item)
	if err == nil {
		err = ctx.Err()
	}
	return value, err
}
