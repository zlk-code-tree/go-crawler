package crawler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
)

type browserCloseClient struct {
	t     *testing.T
	calls int
}

func (c *browserCloseClient) Event() <-chan *cdp.Event { return nil }

func (c *browserCloseClient) Call(ctx context.Context, _ string, method string, _ interface{}) ([]byte, error) {
	if method != "Browser.close" {
		c.t.Fatalf("unexpected DevTools method %q", method)
	}
	if ctx.Err() != nil {
		c.t.Fatalf("cleanup inherited cancelled task context: %v", ctx.Err())
	}
	if _, ok := ctx.Deadline(); !ok {
		c.t.Fatal("browser cleanup must have a deadline")
	}
	c.calls++
	return []byte(`{}`), nil
}

func TestCloseBrowserAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, browserLifecycleKey{}, &browserLifecycle{})
	client := &browserCloseClient{t: t}
	browser := rod.New().Context(ctx).Client(client)
	cancel()
	closeBrowser(browser)
	closeBrowser(browser)
	if client.calls != 1 {
		t.Fatalf("browser closed %d times, want 1", client.calls)
	}
}

func TestCancelledBrowserWorkDoesNotStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	crawler := &sCrawler{}
	if browser, err := crawler.startBrowser(ctx, "cancelled", true); browser != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled startup returned browser=%v, err=%v", browser, err)
	}
	if crawler.performLogin(ctx, nil, "cancelled", "qrcode", "", "") {
		t.Fatal("cancelled login reported success")
	}
}

func TestWaitForCrawlerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- waitForCrawler(ctx, time.Hour) }()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("wait error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("task cancellation did not interrupt wait")
	}
	if err := waitForCrawler(context.Background(), 0); err != nil {
		t.Fatalf("zero delay failed: %v", err)
	}
}
