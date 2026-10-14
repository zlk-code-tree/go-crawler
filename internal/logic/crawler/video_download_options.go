package crawler

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// VideoDownloadOptions bounds both queued and synchronous URL resolutions.
type VideoDownloadOptions struct {
	Workers         int
	QueueCapacity   int
	TaskTimeout     time.Duration
	HTTPTimeout     time.Duration
	RetryAttempts   int
	RetryDelay      time.Duration
	RefillInterval  time.Duration
	ShutdownTimeout time.Duration
}

func DefaultVideoDownloadOptions() VideoDownloadOptions {
	return VideoDownloadOptions{
		Workers: 2, QueueCapacity: 1000, TaskTimeout: 120 * time.Second,
		HTTPTimeout: 15 * time.Second, RetryAttempts: 3, RetryDelay: time.Second,
		RefillInterval: 5 * time.Second, ShutdownTimeout: 30 * time.Second,
	}
}

func (o VideoDownloadOptions) validate() error {
	if o.Workers < 1 || o.Workers > 8 {
		return fmt.Errorf("crawler.video.resolver.workers 必须为 1..8")
	}
	if o.QueueCapacity < 1 || o.QueueCapacity > 10000 {
		return fmt.Errorf("crawler.video.resolver.queueCapacity 必须为 1..10000")
	}
	if o.RetryAttempts < 1 || o.RetryAttempts > 5 {
		return fmt.Errorf("crawler.video.resolver.retryAttempts 必须为 1..5")
	}
	if o.TaskTimeout <= 0 || o.TaskTimeout > 10*time.Minute ||
		o.HTTPTimeout <= 0 || o.HTTPTimeout > time.Minute ||
		o.RetryDelay <= 0 || o.RetryDelay > time.Minute ||
		o.RefillInterval <= 0 || o.RefillInterval > time.Minute ||
		o.ShutdownTimeout <= 0 || o.ShutdownTimeout > 2*time.Minute {
		return fmt.Errorf("视频解析超时、重试间隔和补投间隔必须为正数且在允许范围内")
	}
	return nil
}

func VideoDownloadOptionsFromConfig(ctx context.Context) (VideoDownloadOptions, error) {
	o := DefaultVideoDownloadOptions()
	integer := func(key string, fallback int) (int, error) {
		value, err := g.Cfg().Get(ctx, "crawler.video.resolver."+key, fallback)
		if err != nil {
			return 0, fmt.Errorf("读取视频解析配置 %s 失败：%w", key, err)
		}
		return value.Int(), nil
	}
	for _, setting := range []struct {
		key    string
		target *int
	}{
		{"workers", &o.Workers}, {"queueCapacity", &o.QueueCapacity}, {"retryAttempts", &o.RetryAttempts},
	} {
		value, err := integer(setting.key, *setting.target)
		if err != nil {
			return o, err
		}
		*setting.target = value
	}
	for _, setting := range []struct {
		key     string
		target  *time.Duration
		maximum int
	}{
		{"timeoutSeconds", &o.TaskTimeout, 600}, {"httpTimeoutSeconds", &o.HTTPTimeout, 60},
		{"retryDelaySeconds", &o.RetryDelay, 60}, {"refillIntervalSeconds", &o.RefillInterval, 60},
		{"shutdownTimeoutSeconds", &o.ShutdownTimeout, 120},
	} {
		seconds, err := integer(setting.key, int(*setting.target/time.Second))
		if err != nil {
			return o, err
		}
		// Validate seconds before multiplication, including very large inputs.
		if seconds < 1 || seconds > setting.maximum {
			return o, fmt.Errorf("crawler.video.resolver.%s 必须为 1..%d 秒", setting.key, setting.maximum)
		}
		*setting.target = time.Duration(seconds) * time.Second
	}
	return o, o.validate()
}
