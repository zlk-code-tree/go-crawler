package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestScheduler(handler func(context.Context, TaskType) error) *Scheduler {
	s := &Scheduler{
		tasks:          make(map[TaskType]*ScheduleTask),
		cycleInterval:  time.Hour,
		currentCancels: make(map[TaskType]context.CancelFunc),
		taskHandler:    handler,
	}
	s.initDefaultTasks()
	return s
}

func tickTestScheduler(s *Scheduler) {
	s.mu.RLock()
	generation := s.generation
	s.mu.RUnlock()
	s.tickGeneration(generation)
}

func waitSchedulerSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for scheduler")
	}
}

func waitSchedulerIdle(t *testing.T, s *Scheduler) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for s.HasRunningTask() {
		if time.Now().After(deadline) {
			t.Fatal("scheduler did not release task ownership")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestSchedulerPipelineDoesNotReenterDuringVideo(t *testing.T) {
	videoStarted, releaseVideo := make(chan struct{}), make(chan struct{})
	var ranks, videos atomic.Int32
	s := newTestScheduler(func(ctx context.Context, taskType TaskType) error {
		if taskType == TaskTypeProductRank {
			ranks.Add(1)
			return nil
		}
		if videos.Add(1) == 1 {
			close(videoStarted)
		}
		select {
		case <-releaseVideo:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	// 再现第二轮开始前仍保留的上一轮 NextRunAt。
	s.tasks[TaskTypeProductRank].NextRunAt = time.Now().Add(-time.Hour)
	s.Start()
	t.Cleanup(func() { s.CancelRunningTasks(true); waitSchedulerIdle(t, s) })
	if err := s.TriggerTask(TaskTypeProductRank); err != nil {
		t.Fatal(err)
	}
	waitSchedulerSignal(t, videoStarted)
	for i := 0; i < 20; i++ {
		tickTestScheduler(s)
	}
	for _, kind := range []TaskType{TaskTypeProductRank, TaskTypeVideoPool} {
		if err := s.TriggerTask(kind); err == nil {
			t.Fatalf("accepted %s while video phase was running", kind)
		}
	}
	if task := s.GetTaskStatus(TaskTypeProductRank); task.Status != TaskStatusWaiting || !task.NextRunAt.IsZero() {
		t.Fatalf("rank should wait for this video phase: %+v", task)
	}
	close(releaseVideo)
	waitSchedulerIdle(t, s)
	if ranks.Load() != 1 || videos.Load() != 1 {
		t.Fatalf("unexpected executions: ranks=%d videos=%d", ranks.Load(), videos.Load())
	}
	if task := s.GetTaskStatus(TaskTypeProductRank); task.Status != TaskStatusIdle || !task.NextRunAt.After(time.Now()) {
		t.Fatalf("next cycle was not scheduled after video completion: %+v", task)
	}
}

func TestSchedulerConcurrentTriggersReserveOnce(t *testing.T) {
	release := make(chan struct{})
	s := newTestScheduler(func(ctx context.Context, _ TaskType) error {
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	t.Cleanup(func() { s.CancelRunningTasks(true); waitSchedulerIdle(t, s) })
	var accepted atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if s.TriggerTask(TaskTypeVideoPool) == nil {
				accepted.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("expected exactly one accepted trigger, got %d", accepted.Load())
	}
	if !s.HasRunningTask() {
		t.Fatal("successful trigger must reserve task before returning")
	}
	close(release)
	waitSchedulerIdle(t, s)
}

func TestSchedulerCancelRetainsOwnershipUntilHandlerExits(t *testing.T) {
	started, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var videos atomic.Int32
	s := newTestScheduler(func(ctx context.Context, taskType TaskType) error {
		if taskType == TaskTypeVideoPool {
			videos.Add(1)
			return nil
		}
		close(started)
		<-ctx.Done()
		close(canceled)
		<-release // 浏览器清理完成之前不得释放执行权。
		return nil
	})
	defer s.Stop()
	if err := s.TriggerTask(TaskTypeProductRank); err != nil {
		t.Fatal(err)
	}
	waitSchedulerSignal(t, started)
	got := s.CancelRunningTasks(false)
	waitSchedulerSignal(t, canceled)
	if len(got) != 1 || got[0] != TaskTypeProductRank {
		t.Fatalf("unexpected canceled tasks: %v", got)
	}
	if !s.HasRunningTask() || s.GetTaskStatus(TaskTypeProductRank).Status != TaskStatusRunning {
		t.Error("cancel cleared running state before the handler exited")
	}
	if err := s.TriggerTask(TaskTypeVideoPool); err == nil {
		t.Error("accepted another task during cancellation cleanup")
	}
	close(release)
	waitSchedulerIdle(t, s)
	if videos.Load() != 0 {
		t.Fatal("canceled rank started a video phase")
	}
	if task := s.GetTaskStatus(TaskTypeProductRank); task.LastError != "任务已取消" {
		t.Fatalf("cancellation reported as success: %+v", task)
	}
}

func TestSchedulerPauseRequiresVideoResume(t *testing.T) {
	var ranks, videos atomic.Int32
	s := newTestScheduler(func(_ context.Context, taskType TaskType) error {
		if taskType == TaskTypeProductRank {
			ranks.Add(1)
			return nil
		}
		if videos.Add(1) == 1 {
			return fmt.Errorf("video: %w", &TaskPausedError{Reason: "每日查看配额已用完"})
		}
		return nil
	})
	if err := s.TriggerTask(TaskTypeProductRank); err != nil {
		t.Fatal(err)
	}
	waitSchedulerIdle(t, s)
	video := s.GetTaskStatus(TaskTypeVideoPool)
	if video.Status != TaskStatusPaused || video.LastError == "" {
		t.Fatalf("pause was not preserved: %+v", video)
	}
	if rank := s.GetTaskStatus(TaskTypeProductRank); rank.Status != TaskStatusWaiting || !rank.NextRunAt.IsZero() {
		t.Fatalf("paused video should hold rank cycle: %+v", rank)
	}
	s.Start()
	defer s.Stop()
	tickTestScheduler(s)
	if err := s.TriggerTask(TaskTypeProductRank); err == nil {
		t.Fatal("rank trigger must not replace paused video")
	}
	if err := s.TriggerTask(TaskTypeVideoPool); err != nil {
		t.Fatal(err)
	}
	waitSchedulerIdle(t, s)
	if ranks.Load() != 1 || videos.Load() != 2 {
		t.Fatalf("resume did not continue only video: ranks=%d videos=%d", ranks.Load(), videos.Load())
	}
	if video = s.GetTaskStatus(TaskTypeVideoPool); video.Status != TaskStatusIdle || video.LastError != "" {
		t.Fatalf("video did not complete after resume: %+v", video)
	}
}

func TestSchedulerStopStartTerminatesOldLoops(t *testing.T) {
	s := newTestScheduler(func(context.Context, TaskType) error { return errors.New("test failure") })
	defer s.Stop()
	for i := 0; i < 10; i++ {
		s.Start()
		s.mu.RLock()
		done, generation := s.loopDone, s.generation
		s.mu.RUnlock()
		s.Stop()
		waitSchedulerSignal(t, done)
		s.Start()
		s.mu.Lock()
		s.tasks[TaskTypeProductRank].LastRunAt = time.Now().Add(-time.Hour)
		s.tasks[TaskTypeProductRank].NextRunAt = time.Now().Add(-time.Minute)
		s.mu.Unlock()
		// 已退出的旧 ticker 即使有缓冲 tick，也不能干扰新调度循环。
		s.tickGeneration(generation)
		if s.HasRunningTask() {
			t.Fatal("old scheduler generation started a task after restart")
		}
		s.Stop()
	}
}

func TestSchedulerStopPreventsNextPipelinePhase(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var videos atomic.Int32
	s := newTestScheduler(func(_ context.Context, taskType TaskType) error {
		if taskType == TaskTypeProductRank {
			close(started)
			<-release
		} else {
			videos.Add(1)
		}
		return nil
	})
	s.Start()
	if err := s.TriggerTask(TaskTypeProductRank); err != nil {
		t.Fatal(err)
	}
	waitSchedulerSignal(t, started)
	s.Stop()
	// 重启调度器不恢复旧流水线的下一阶段。
	s.Start()
	defer s.Stop()
	close(release)
	waitSchedulerIdle(t, s)
	if videos.Load() != 0 {
		t.Fatal("stopped pipeline started a video phase")
	}
}

func TestSchedulerMissingHandlerRejectsTrigger(t *testing.T) {
	s := newTestScheduler(nil)
	if err := s.TriggerTask(TaskTypeProductRank); err == nil {
		t.Fatal("trigger succeeded without a handler")
	}
	if task := s.GetTaskStatus(TaskTypeProductRank); task.RunCount != 0 || task.Status != TaskStatusIdle {
		t.Fatalf("rejected trigger changed task state: %+v", task)
	}
}

func TestSchedulerRankFailureDoesNotStartVideo(t *testing.T) {
	var videos atomic.Int32
	s := newTestScheduler(func(_ context.Context, taskType TaskType) error {
		if taskType == TaskTypeProductRank {
			return errors.New("rank save failed")
		}
		videos.Add(1)
		return nil
	})
	if err := s.TriggerTask(TaskTypeProductRank); err != nil {
		t.Fatal(err)
	}
	waitSchedulerIdle(t, s)
	if videos.Load() != 0 {
		t.Fatal("failed rank started a video phase")
	}
	if rank := s.GetTaskStatus(TaskTypeProductRank); rank.LastError != "rank save failed" || !rank.NextRunAt.After(time.Now()) {
		t.Fatalf("rank failure did not retain error and retry interval: %+v", rank)
	}
}
