package crawler

import (
	"context"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

type browserLifecycleKey struct{}

type browserLifecycle struct {
	launcher *launcher.Launcher
	once     sync.Once
}

// closeBrowser uses a fresh context because the task may already be cancelled.
// If the DevTools connection has stopped responding, terminate the owned process.
func closeBrowser(browser *rod.Browser) {
	if browser == nil {
		return
	}
	lifecycle, _ := browser.GetContext().Value(browserLifecycleKey{}).(*browserLifecycle)
	close := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = browser.Context(ctx).Close()
		if lifecycle != nil && lifecycle.launcher != nil {
			// A successful DevTools response does not guarantee process exit.
			// Kill the owned process before Cleanup waits for its exit channel.
			lifecycle.launcher.Kill()
			lifecycle.launcher.Cleanup()
		}
	}
	if lifecycle == nil {
		close()
		return
	}
	lifecycle.once.Do(close)
}

// waitForCrawler keeps pacing and retry delays interruptible by task cancellation.
func waitForCrawler(ctx context.Context, duration time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
