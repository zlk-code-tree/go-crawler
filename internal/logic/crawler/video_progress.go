package crawler

import (
	"context"
	"errors"
	"fmt"
)

// videoCollection separates progress decisions from browser and database I/O.
// A product is committed only after its videos have been persisted.
type videoCollection struct {
	maxViews   int
	maxEmpty   int
	fetch      func(ProductInfo) videoPageResult
	persist    func([]VideoAnalysis) error
	checkpoint func(nextIndex int, status, reason string) error
	wait       func(context.Context) error
}

func (c videoCollection) run(ctx context.Context, products []ProductInfo, startIndex int) videoScrapeResult {
	result := videoScrapeResult{NextProductIndex: startIndex}
	if startIndex < 0 || startIndex > len(products) {
		result.Err = fmt.Errorf("视频断点超出商品快照范围：%d/%d", startIndex, len(products))
		return result
	}
	save := func(status, reason string) error {
		if c.checkpoint == nil {
			return nil
		}
		return c.checkpoint(result.NextProductIndex, status, reason)
	}
	stop := func(reason string, err error) videoScrapeResult {
		result.Paused, result.StopReason, result.Err = true, reason, err
		if checkpointErr := save("paused", reason); checkpointErr != nil {
			result.Err = errors.Join(result.Err, fmt.Errorf("保存视频断点失败：%w", checkpointErr))
		}
		return result
	}
	if err := save("running", ""); err != nil {
		result.Err = fmt.Errorf("初始化视频断点失败：%w", err)
		return result
	}
	views, emptyStreak := 0, 0
	for index := startIndex; index < len(products); index++ {
		if err := ctx.Err(); err != nil {
			return stop("任务被取消", err)
		}
		if c.maxViews > 0 && views >= c.maxViews {
			return stop(fmt.Sprintf("达到本轮视频详情查看保护上限：%d 次", c.maxViews), nil)
		}
		page := c.fetch(products[index])
		views++
		if err := ctx.Err(); err != nil {
			return stop("任务被取消", err)
		}
		if page.Err != nil {
			return stop("视频页面采集失败", page.Err)
		}
		if page.Status == videoPageAccessDailyLimit {
			return stop("官方每日详情查看次数已达上限："+page.Note, nil)
		}
		if page.Status == videoPageAccessRestricted && len(page.Videos) == 0 {
			return stop("视频详情访问受限，保留当前商品位置："+page.Note, nil)
		}
		videos := deduplicateVideos(page.Videos)
		if len(videos) > 0 {
			if c.persist != nil {
				if err := c.persist(videos); err != nil {
					return stop("视频入库失败，保留当前商品位置", err)
				}
			}
			result.Videos = deduplicateVideos(append(result.Videos, videos...))
			emptyStreak = 0
		} else {
			emptyStreak++
		}
		result.NextProductIndex = index + 1
		if err := save("running", ""); err != nil {
			// 已入库数据可幂等重放，不把未可靠保存的进度报告为完成。
			result.NextProductIndex = index
			result.Err = fmt.Errorf("保存视频断点失败：%w", err)
			return result
		}
		if c.maxEmpty > 0 && emptyStreak >= c.maxEmpty {
			return stop(fmt.Sprintf("连续 %d 个详情页没有可识别视频，暂停访问", emptyStreak), nil)
		}
		if index+1 < len(products) && c.wait != nil {
			if err := c.wait(ctx); err != nil {
				return stop("任务被取消", err)
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return stop("任务被取消", err)
	}
	result.Completed = true
	return result
}
