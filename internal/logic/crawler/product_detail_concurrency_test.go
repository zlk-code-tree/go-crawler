package crawler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
)

func TestProductDetailBatchOptionsBounds(t *testing.T) {
	for _, test := range []struct {
		concurrency int
		timeout     int
		valid       bool
	}{{1, 60, true}, {8, 300, true}, {0, 60, false}, {9, 60, false}, {1, 0, false}, {1, 301, false}} {
		options, err := validateProductDetailBatchOptions(test.concurrency, test.timeout)
		if (err == nil) != test.valid {
			t.Fatalf("concurrency=%d timeout=%d: options=%+v err=%v", test.concurrency, test.timeout, options, err)
		}
		if test.valid && (options.Concurrency != test.concurrency || options.TaskTimeout != time.Duration(test.timeout)*time.Second) {
			t.Fatalf("unexpected validated options: %+v", options)
		}
	}
}

func TestProductDetailAccessIssueScope(t *testing.T) {
	for _, test := range []struct {
		text   string
		status string
		fatal  bool
	}{{"今日查看次数已达上限", "daily_limit", true}, {"请重新登录", "restricted", true}, {"环境异常，请完成验证", "restricted", true}, {"对方已关闭竞对详情功能", "restricted", false}, {"经营的类目与对方不匹配", "available", false}, {"首次上架 2026-10-13", "available", false}} {
		status, _, err := productDetailAccessIssue([]string{test.text})
		var fatal *fatalCrawlerTaskError
		if status != test.status || errors.As(err, &fatal) != test.fatal {
			t.Fatalf("text=%q status=%q err=%v", test.text, status, err)
		}
		if test.status == "restricted" && err == nil {
			t.Fatalf("permission failure for %q was silently accepted", test.text)
		}
	}
	// A local permission message must not hide a global quota error in another node.
	status, _, err := productDetailAccessIssue([]string{"对方已关闭竞对详情功能", "今日查看已达上限"})
	var fatal *fatalCrawlerTaskError
	if status != "daily_limit" || !errors.As(err, &fatal) {
		t.Fatalf("global access failure was hidden: status=%s err=%v", status, err)
	}
}

type detailTargetCleanupClient struct {
	t      *testing.T
	cancel context.CancelFunc
	closed bool
}

func (c *detailTargetCleanupClient) Event() <-chan *cdp.Event { return nil }

func (c *detailTargetCleanupClient) Call(ctx context.Context, _ string, method string, _ interface{}) ([]byte, error) {
	switch method {
	case "Target.createTarget":
		return []byte(`{"targetId":"detail-page"}`), nil
	case "Target.attachToTarget":
		c.cancel()
		return nil, context.Canceled
	case "Target.closeTarget":
		if ctx.Err() != nil {
			c.t.Fatalf("page cleanup reused cancelled context: %v", ctx.Err())
		}
		if _, ok := ctx.Deadline(); !ok {
			c.t.Fatal("page cleanup has no timeout")
		}
		c.closed = true
		return []byte(`{"success":true}`), nil
	default:
		c.t.Fatalf("unexpected DevTools method: %s", method)
		return nil, errors.New("unexpected DevTools method")
	}
}

func TestProductDetailClosesTargetWhenPageAttachmentIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := &detailTargetCleanupClient{t: t, cancel: cancel}
	browser := rod.New().Context(ctx).Client(client)
	page, err := openProductDetailPage(ctx, browser)
	if page != nil || !errors.Is(err, context.Canceled) || !client.closed {
		t.Fatalf("cancelled target attachment leaked page: page=%v err=%v closed=%v", page, err, client.closed)
	}
}

func TestProductDetailCancelledParsingDoesNotTouchPage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	detail, err := (&sCrawler{}).scrapeSingleProductDetailByParsing(ctx, nil, ProductInfo{}, "cancelled")
	if detail != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled parsing continued: detail=%v err=%v", detail, err)
	}
}
