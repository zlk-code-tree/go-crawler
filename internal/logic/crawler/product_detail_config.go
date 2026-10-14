package crawler

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

func productDetailBatchOptions(ctx context.Context) (crawlerBatchOptions, error) {
	concurrency, err := g.Cfg().Get(ctx, "crawler.productDetail.concurrency", 1)
	if err != nil {
		return crawlerBatchOptions{}, fmt.Errorf("读取商品详情并发配置失败：%w", err)
	}
	timeout, err := g.Cfg().Get(ctx, "crawler.productDetail.timeoutSeconds", 60)
	if err != nil {
		return crawlerBatchOptions{}, fmt.Errorf("读取商品详情超时配置失败：%w", err)
	}
	return validateProductDetailBatchOptions(concurrency.Int(), timeout.Int())
}

func validateProductDetailBatchOptions(concurrency, timeoutSeconds int) (crawlerBatchOptions, error) {
	if concurrency < 1 || concurrency > 8 {
		return crawlerBatchOptions{}, fmt.Errorf("crawler.productDetail.concurrency 必须在 1 到 8 之间，当前为 %d", concurrency)
	}
	if timeoutSeconds < 1 || timeoutSeconds > 300 {
		return crawlerBatchOptions{}, fmt.Errorf("crawler.productDetail.timeoutSeconds 必须在 1 到 300 之间，当前为 %d", timeoutSeconds)
	}
	return crawlerBatchOptions{Concurrency: concurrency, TaskTimeout: time.Duration(timeoutSeconds) * time.Second}, nil
}
