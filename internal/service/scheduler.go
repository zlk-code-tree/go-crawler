package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

// TaskType 任务类型
type TaskType string

const (
	TaskTypeProductRank TaskType = "product_rank" // 商品榜单抓取
	TaskTypeVideoPool   TaskType = "video_pool"   // 视频池抓取（商品详情+视频分析）
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusIdle    TaskStatus = "idle"    // 空闲
	TaskStatusRunning TaskStatus = "running" // 运行中
	TaskStatusWaiting TaskStatus = "waiting" // 等待执行（流水线中排队）
	TaskStatusPaused  TaskStatus = "paused"  // 等待手动恢复（配额或其他业务限制）
)

// TaskPausedError 表示任务已保留断点，需手动恢复，不能视为本轮完成。
type TaskPausedError struct {
	Reason string
}

func (e *TaskPausedError) Error() string {
	if e.Reason == "" {
		return "任务已暂停，等待手动恢复"
	}
	return e.Reason
}

// ScheduleTask 调度任务
type ScheduleTask struct {
	Type      TaskType
	Status    TaskStatus
	Interval  time.Duration
	LastRunAt time.Time
	NextRunAt time.Time
	RunCount  int
	LastError string
}

// Scheduler 任务调度器
// 流水线模式：商品榜单抓取 → 商品详情/视频抓取 → 等待间隔 → 循环
type Scheduler struct {
	mu             sync.RWMutex
	running        bool
	ticker         *time.Ticker
	stopCh         chan struct{}
	loopDone       chan struct{}
	generation     uint64 // Stop 后，旧任务不能再启动下一阶段
	tasks          map[TaskType]*ScheduleTask
	taskHandler    func(ctx context.Context, taskType TaskType) error
	cycleInterval  time.Duration // 一轮完整抓取后的等待间隔
	currentCancels map[TaskType]context.CancelFunc
}

var (
	schedulerInstance *Scheduler
	schedulerOnce     sync.Once
)

// GetScheduler 获取全局调度器实例
func GetScheduler() *Scheduler {
	schedulerOnce.Do(func() {
		schedulerInstance = &Scheduler{
			tasks:          make(map[TaskType]*ScheduleTask),
			cycleInterval:  3 * time.Hour, // 一轮抓取完成后等待 3 小时，可由配置覆盖
			currentCancels: make(map[TaskType]context.CancelFunc),
		}
		schedulerInstance.initDefaultTasks()
	})
	return schedulerInstance
}

// initDefaultTasks 初始化默认任务
func (s *Scheduler) initDefaultTasks() {
	s.tasks[TaskTypeProductRank] = &ScheduleTask{
		Type:     TaskTypeProductRank,
		Status:   TaskStatusIdle,
		Interval: 3 * time.Hour, // 一轮完整抓取的间隔，可由配置覆盖
	}

	s.tasks[TaskTypeVideoPool] = &ScheduleTask{
		Type:     TaskTypeVideoPool,
		Status:   TaskStatusIdle,
		Interval: 0, // 不独立调度，由商品抓取完成后自动触发
	}
}

// SetTaskHandler 设置任务执行处理器
func (s *Scheduler) SetTaskHandler(handler func(ctx context.Context, taskType TaskType) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.taskHandler = handler
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}

	s.running = true
	ticker := time.NewTicker(3 * time.Second)
	stopCh, loopDone := make(chan struct{}), make(chan struct{})
	s.ticker, s.stopCh, s.loopDone = ticker, stopCh, loopDone
	generation := s.generation
	go func() {
		defer close(loopDone)
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				s.tickGeneration(generation)
			}
		}
	}()
	g.Log().Info(context.Background(), "[Scheduler] 调度器已启动（流水线模式）")
}

// Stop 停止后续调度；已开始的任务继续执行，但不会再启动下一阶段。
func (s *Scheduler) Stop() {
	s.mu.Lock()
	s.stopLocked()
	s.mu.Unlock()
	g.Log().Info(context.Background(), "[Scheduler] 调度器已停止")
}

func (s *Scheduler) stopLocked() {
	s.running = false
	s.generation++
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}
}

// IsRunning 检查调度器是否运行中
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// tickGeneration 检查到期榜单；旧调度循环不能在 Stop/Start 后触发任务。
func (s *Scheduler) tickGeneration(generation uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running || generation != s.generation || !s.isTaskDueLocked(TaskTypeProductRank, time.Now()) {
		return
	}
	if run, err := s.prepareTaskLocked(TaskTypeProductRank); err == nil {
		go s.runTask(run)
	}
}

func (s *Scheduler) isTaskDueLocked(taskType TaskType, now time.Time) bool {
	task := s.tasks[taskType]
	// 首次不自动运行；流水线中任意阶段执行时均不能再开一轮榜单。
	return task != nil && len(s.currentCancels) == 0 &&
		!task.LastRunAt.IsZero() && !task.NextRunAt.IsZero() &&
		task.Status == TaskStatusIdle && !now.Before(task.NextRunAt)
}

// scheduledRun 在返回手动触发成功之前就占用执行权，避免重复请求同时成功。
type scheduledRun struct {
	taskType   TaskType
	ctx        context.Context
	cancel     context.CancelFunc
	handler    func(context.Context, TaskType) error
	generation uint64
	runCount   int
}

func (s *Scheduler) prepareTaskLocked(taskType TaskType) (*scheduledRun, error) {
	task := s.tasks[taskType]
	if task == nil {
		return nil, fmt.Errorf("任务类型不存在: %s", taskType)
	}
	if len(s.currentCancels) > 0 {
		return nil, fmt.Errorf("采集流水线正在运行中，无法触发: %s", taskType)
	}
	if task.Status == TaskStatusWaiting {
		return nil, fmt.Errorf("任务正在等待流水线完成，请先恢复视频采集: %s", taskType)
	}
	if s.taskHandler == nil {
		return nil, errors.New("任务处理器未注册")
	}
	ctx, cancel := context.WithCancel(context.Background())
	task.Status = TaskStatusRunning
	task.LastRunAt = time.Now()
	task.NextRunAt = time.Time{}
	task.LastError = ""
	task.RunCount++
	if s.currentCancels == nil {
		s.currentCancels = make(map[TaskType]context.CancelFunc)
	}
	s.currentCancels[taskType] = cancel
	return &scheduledRun{
		taskType:   taskType,
		ctx:        ctx,
		cancel:     cancel,
		handler:    s.taskHandler,
		generation: s.generation,
		runCount:   task.RunCount,
	}, nil
}

func (s *Scheduler) runTask(run *scheduledRun) {
	for run != nil {
		g.Log().Infof(context.Background(), "[Scheduler] 开始执行任务: %s (第 %d 次)", run.taskType, run.runCount)
		var err error
		if err = run.ctx.Err(); err == nil {
			err = run.handler(run.ctx, run.taskType)
		}
		s.mu.Lock()
		// 和 CancelRunningTasks 共用锁，取消与启动下一阶段之间没有空档。
		if run.ctx.Err() != nil {
			err = run.ctx.Err()
		}
		delete(s.currentCancels, run.taskType)
		run.cancel()
		task := s.tasks[run.taskType]
		task.Status = TaskStatusIdle
		var paused *TaskPausedError
		switch {
		case errors.Is(err, context.Canceled):
			task.LastError = "任务已取消"
			g.Log().Warningf(context.Background(), "[Scheduler] 任务已取消: %s", run.taskType)
		case errors.As(err, &paused):
			task.Status = TaskStatusPaused
			task.LastError = err.Error()
			g.Log().Warningf(context.Background(), "[Scheduler] 任务已暂停: %s, 原因: %v", run.taskType, err)
		case err != nil:
			task.LastError = err.Error()
			g.Log().Errorf(context.Background(), "[Scheduler] 任务执行失败: %s, 错误: %v", run.taskType, err)
		default:
			task.LastError = ""
			g.Log().Infof(context.Background(), "[Scheduler] 任务执行完成: %s", run.taskType)
		}

		if paused != nil {
			// 视频暂停时保留本轮榜单，等待视频续抓，不抢先启动下一轮。
			if run.taskType == TaskTypeVideoPool {
				if product := s.tasks[TaskTypeProductRank]; product != nil {
					product.Status = TaskStatusWaiting
					product.NextRunAt = time.Time{}
				}
			}
			s.mu.Unlock()
			return
		}

		if run.taskType == TaskTypeProductRank && err == nil && run.generation == s.generation {
			// 在同一临界区交接执行权，tick 与手动触发均无法插入新任务。
			next, nextErr := s.prepareTaskLocked(TaskTypeVideoPool)
			if nextErr == nil {
				task.Status = TaskStatusWaiting
				s.mu.Unlock()
				run = next
				continue
			}
			task.LastError = nextErr.Error()
		}

		if run.taskType == TaskTypeVideoPool {
			if product := s.tasks[TaskTypeProductRank]; product != nil {
				product.Status = TaskStatusIdle
				product.NextRunAt = time.Now().Add(s.cycleWaitLocked())
			}
		} else {
			task.NextRunAt = time.Now().Add(s.cycleWaitLocked())
		}
		s.mu.Unlock()
		return
	}
}

func (s *Scheduler) cycleWaitLocked() time.Duration {
	if product := s.tasks[TaskTypeProductRank]; product != nil && product.Interval > 0 {
		return product.Interval
	}
	return s.cycleInterval
}

// TriggerTask 手动触发任务；整个流水线同一时间只允许一个采集阶段运行。
func (s *Scheduler) TriggerTask(taskType TaskType) error {
	s.mu.Lock()
	run, err := s.prepareTaskLocked(taskType)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	go s.runTask(run)
	return nil
}

// CancelRunningTasks 取消正在执行的采集任务；stopScheduler=true 时同时停止后续调度循环。
func (s *Scheduler) CancelRunningTasks(stopScheduler bool) []TaskType {
	s.mu.Lock()
	defer s.mu.Unlock()
	if stopScheduler {
		s.stopLocked()
	}
	canceled := make([]TaskType, 0, len(s.currentCancels))
	for taskType, cancel := range s.currentCancels {
		cancel()
		canceled = append(canceled, taskType)
	}
	// 执行权和 running 状态只在 handler 真正退出后释放。
	return canceled
}

// HasRunningTask 是否存在正在执行的采集任务。
func (s *Scheduler) HasRunningTask() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.currentCancels) > 0
}

// SetCycleInterval 设置一轮完整抓取后的等待间隔
func (s *Scheduler) SetCycleInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cycleInterval = interval
	g.Log().Infof(context.Background(), "[Scheduler] 设置轮次间隔: %v", interval)
}

// GetCycleInterval 获取轮次间隔
func (s *Scheduler) GetCycleInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cycleInterval
}

// GetTaskStatus 获取任务状态
func (s *Scheduler) GetTaskStatus(taskType TaskType) *ScheduleTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskType]
	if !exists {
		return nil
	}

	// 返回副本
	taskCopy := *task
	return &taskCopy
}

// GetAllTaskStatus 获取所有任务状态
func (s *Scheduler) GetAllTaskStatus() map[TaskType]*ScheduleTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[TaskType]*ScheduleTask)
	for k, v := range s.tasks {
		taskCopy := *v
		result[k] = &taskCopy
	}
	return result
}

// SetTaskInterval 设置任务执行间隔
func (s *Scheduler) SetTaskInterval(taskType TaskType, interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskType]
	if exists {
		task.Interval = interval
		// 如果任务还未执行过，更新下次执行时间
		if task.LastRunAt.IsZero() {
			task.NextRunAt = time.Now().Add(interval)
		}
		g.Log().Infof(context.Background(), "[Scheduler] 设置任务间隔: %s = %v", taskType, interval)
	}
}
