package service

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestCrawlerStatusAcquireTaskIsAtomic(t *testing.T) {
	mgr := &crawlerStatusManager{}
	const contenders = 32
	start := make(chan struct{})
	winners := make(chan func(), contenders)
	var wg sync.WaitGroup
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, finish, err := mgr.AcquireTask(context.Background(), "same-task", "qrcode")
			if err == nil {
				winners <- finish
			} else if !errors.Is(err, ErrCrawlerBusy) {
				t.Errorf("unexpected acquire error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	close(winners)
	if len(winners) != 1 {
		t.Errorf("acquired %d tasks concurrently, want exactly 1", len(winners))
	}
	for finish := range winners {
		finish()
	}
	if mgr.IsOnline() {
		t.Fatal("task stayed occupied after finish")
	}
}

func TestCrawlerStatusLeaseReuseDoesNotReleaseOwner(t *testing.T) {
	mgr := &crawlerStatusManager{}
	ctx, finish, err := mgr.AcquireTask(context.Background(), "task", "qrcode")
	if err != nil {
		t.Fatal(err)
	}
	defer finish()
	_, nestedFinish, err := mgr.AcquireTask(ctx, "task", "qrcode")
	if err != nil {
		t.Fatal(err)
	}
	nestedFinish()
	if !mgr.IsOnline() || ctx.Err() != nil {
		t.Fatal("nested task released or cancelled the controller's lease")
	}
	if _, _, err := mgr.AcquireTask(context.Background(), "task", "qrcode"); !errors.Is(err, ErrCrawlerBusy) {
		t.Fatalf("same ID without the private lease bypassed exclusion: %v", err)
	}
}

func TestCrawlerStatusForceOfflineWaitsForCleanup(t *testing.T) {
	mgr := &crawlerStatusManager{}
	ctx, finish, err := mgr.AcquireTask(context.Background(), "task", "qrcode")
	if err != nil {
		t.Fatal(err)
	}
	defer finish()
	mgr.ForceOffline()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("task context was not cancelled: %v", ctx.Err())
	}
	if !mgr.IsOnline() {
		t.Fatal("task released before browser cleanup")
	}
	if _, _, err := mgr.AcquireTask(context.Background(), "next", "qrcode"); !errors.Is(err, ErrCrawlerBusy) {
		t.Fatalf("new task started before cleanup: %v", err)
	}
	mgr.SetMessage(ctx, "爬取数据中")
	if mgr.GetStatus().Message != "正在停止，等待浏览器清理" {
		t.Fatal("stopping message was overwritten")
	}
	finish()
	_, nextFinish, err := mgr.AcquireTask(context.Background(), "next", "qrcode")
	if err != nil {
		t.Fatal(err)
	}
	nextFinish()
}

func TestCrawlerStatusStaleOwnerCannotChangeResumedTask(t *testing.T) {
	mgr := &crawlerStatusManager{}
	oldCtx, oldFinish, err := mgr.AcquireTask(context.Background(), "resumed", "qrcode")
	if err != nil {
		t.Fatal(err)
	}
	oldFinish()
	newCtx, newFinish, err := mgr.AcquireTask(context.Background(), "resumed", "qrcode")
	if err != nil {
		t.Fatal(err)
	}
	defer newFinish()
	mgr.SetMessage(newCtx, "new task")
	mgr.SetMessage(oldCtx, "old task")
	mgr.SetMessage(context.Background(), "unowned task")
	oldFinish()
	if status := mgr.GetStatus(); !status.IsOnline || status.Message != "new task" {
		t.Fatalf("stale task changed current state: %+v", status)
	}
	if newCtx.Err() != nil {
		t.Fatal("stale finish cancelled the resumed task")
	}
}

func TestCrawlerStatusRejectsCancelledTask(t *testing.T) {
	mgr := &crawlerStatusManager{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := mgr.AcquireTask(ctx, "task", "qrcode"); !errors.Is(err, context.Canceled) {
		t.Fatalf("acquire cancelled context: %v", err)
	}
	if mgr.IsOnline() {
		t.Fatal("cancelled task occupied crawler")
	}
}
