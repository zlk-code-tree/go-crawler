package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckpointManagerSeparatesVideoPoolAndManualTasks(t *testing.T) {
	ctx := context.Background()
	manager := NewCheckpointManager(t.TempDir())
	tasks := []*CrawlTask{
		{TaskId: "pool", TaskType: string(TaskTypeVideoPool), Status: "paused", CurrentStep: 8, CurrentProductIndex: 2},
		{TaskId: "manual", Status: "paused", CurrentStep: 8, CurrentProductIndex: 5},
		{TaskId: "rank", TaskType: string(TaskTypeProductRank), Status: "running", CurrentStep: 8, CurrentProductIndex: 3},
		{TaskId: "completed-pool", TaskType: string(TaskTypeVideoPool), Status: "completed", CurrentStep: 8, CurrentProductIndex: 10},
	}
	for index, task := range tasks {
		if err := manager.SaveCheckpoint(ctx, task); err != nil {
			t.Fatal(err)
		}
		modified := time.Date(2026, time.October, 1, 0, index, 0, 0, time.UTC)
		if err := os.Chtimes(filepath.Join(manager.checkpointDir, "task_"+task.TaskId+".json"), modified, modified); err != nil {
			t.Fatal(err)
		}
	}

	pool, err := manager.FindLatestVideoPoolCheckpoint(ctx)
	if err != nil || pool == nil || pool.TaskId != "pool" {
		t.Fatalf("video pool resumed a different task type: task=%+v, err=%v", pool, err)
	}
	manual, err := manager.FindLatestResumableCheckpoint(ctx)
	if err != nil || manual == nil || manual.TaskId != "manual" {
		t.Fatalf("manual collection resumed a different task type: task=%+v, err=%v", manual, err)
	}
}

func TestCheckpointManagerResumesVideoPoolAtIndexZero(t *testing.T) {
	ctx := context.Background()
	manager := NewCheckpointManager(t.TempDir())
	if err := manager.SaveCheckpoint(ctx, &CrawlTask{
		TaskId: "pool-at-start", TaskType: string(TaskTypeVideoPool), Status: "paused",
		CurrentStep: 8, CurrentProductIndex: 0, ProductCount: 10,
	}); err != nil {
		t.Fatal(err)
	}
	task, err := manager.FindLatestVideoPoolCheckpoint(ctx)
	if err != nil || task == nil || task.TaskId != "pool-at-start" || task.CurrentProductIndex != 0 {
		t.Fatalf("index zero was discarded instead of resumed: task=%+v, err=%v", task, err)
	}
}

func TestCheckpointManagerResumesManualRankingPage(t *testing.T) {
	ctx := context.Background()
	manager := NewCheckpointManager(t.TempDir())
	if err := manager.SaveCheckpoint(ctx, &CrawlTask{
		TaskId: "manual-ranking", Status: "running", CurrentStep: 6, CurrentPage: 7,
	}); err != nil {
		t.Fatal(err)
	}
	task, err := manager.FindLatestResumableCheckpoint(ctx)
	if err != nil || task == nil || task.TaskId != "manual-ranking" || task.CurrentPage != 7 {
		t.Fatalf("manual ranking progress was ignored: task=%+v, err=%v", task, err)
	}
}

func TestCheckpointManagerRejectsUnsafeTaskID(t *testing.T) {
	ctx := context.Background()
	manager := NewCheckpointManager(t.TempDir())
	for _, taskID := range []string{"", "../escape", "task/child", "task\\child"} {
		if err := manager.SaveCheckpoint(ctx, &CrawlTask{TaskId: taskID}); err == nil {
			t.Errorf("SaveCheckpoint accepted unsafe task ID %q", taskID)
		}
		if _, err := manager.LoadCheckpoint(ctx, taskID); err == nil {
			t.Errorf("LoadCheckpoint accepted unsafe task ID %q", taskID)
		}
	}
}

func TestCheckpointManagerVideoProgressPreservesSourceTask(t *testing.T) {
	ctx := context.Background()
	manager := NewCheckpointManager(t.TempDir())
	const taskID, sourceID = "pool-resume", "original-ranking-snapshot"
	if err := manager.SaveCheckpoint(ctx, &CrawlTask{
		TaskId: taskID, TaskType: string(TaskTypeVideoPool), SourceTaskId: sourceID,
		Status: "running", CurrentStep: 8,
	}); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"paused", "running"} {
		if err := manager.UpdateVideoProgress(ctx, taskID, 3, 10, 7, status, "test reason"); err != nil {
			t.Fatal(err)
		}
		task, err := manager.LoadCheckpoint(ctx, taskID)
		if err != nil || task == nil {
			t.Fatalf("could not load video progress: task=%+v, err=%v", task, err)
		}
		if task.SourceTaskId != sourceID || task.TaskType != string(TaskTypeVideoPool) || task.TaskId != taskID {
			t.Fatalf("progress update replaced the original product snapshot identity: %+v", task)
		}
		if task.CurrentProductIndex != 3 || task.ProductCount != 10 || task.VideoCount != 7 || task.Status != status {
			t.Fatalf("progress fields did not survive persistence: %+v", task)
		}
	}
}

func TestCheckpointManagerCompletionDeletesOnlyCompletedTask(t *testing.T) {
	ctx := context.Background()
	manager := NewCheckpointManager(t.TempDir())
	for _, id := range []string{"finished", "unfinished"} {
		if err := manager.SaveCheckpoint(ctx, &CrawlTask{
			TaskId: id, TaskType: string(TaskTypeVideoPool), Status: "paused", CurrentStep: 8,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := manager.CompleteTask(ctx, "finished", 10); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(manager.checkpointDir, "task_finished.json")); !os.IsNotExist(err) {
		t.Fatalf("completed checkpoint still exists: %v", err)
	}
	if task, err := manager.LoadCheckpoint(ctx, "finished"); err != nil || task != nil {
		t.Fatalf("completed task can still resume: task=%+v, err=%v", task, err)
	}
	if task, err := manager.FindLatestVideoPoolCheckpoint(ctx); err != nil || task == nil || task.TaskId != "unfinished" {
		t.Fatalf("completion removed another task's checkpoint: task=%+v, err=%v", task, err)
	}
	if err := manager.CompleteTask(ctx, "finished", 10); err != nil {
		t.Fatalf("completion must tolerate an already removed checkpoint: %v", err)
	}
}
