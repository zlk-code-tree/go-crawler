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
)

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
	mu              sync.RWMutex
	running         bool
	ticker          *time.Ticker
	tasks           map[TaskType]*ScheduleTask
	taskHandler     func(ctx context.Context, taskType TaskType) error
	cycleInterval   time.Duration // 一轮完整抓取后的等待间隔
	triggerCh       chan TaskType // 流水线触发通道
	currentCancels  map[TaskType]context.CancelFunc
	pipelineStarted bool
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
			triggerCh:      make(chan TaskType, 2),
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
		g.Log().Warning(context.Background(), "[Scheduler] 调度器已在运行")
		return
	}

	s.running = true
	s.ticker = time.NewTicker(3 * time.Second)

	// 主循环：定时检查是否该开始新一轮抓取
	go func() {
		g.Log().Info(context.Background(), "[Scheduler] 调度器已启动（流水线模式）")
		for range s.ticker.C {
			s.tick()
		}
	}()

	if !s.pipelineStarted {
		s.pipelineStarted = true
		// 流水线循环：商品榜单完成 → 立即触发商品详情/视频
		go func() {
			for taskType := range s.triggerCh {
				g.Log().Infof(context.Background(), "[Scheduler] 流水线触发任务: %s", taskType)
				s.executeTask(taskType)
			}
		}()
	}
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
	g.Log().Info(context.Background(), "[Scheduler] 调度器已停止")
}

// IsRunning 检查调度器是否运行中
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// tick 检查并执行到期任务（只负责触发商品榜单，视频由流水线触发）
func (s *Scheduler) tick() {
	s.mu.RLock()
	if s.taskHandler == nil {
		s.mu.RUnlock()
		return
	}
	s.mu.RUnlock()

	now := time.Now()

	// 只检查商品榜单抓取任务是否到期
	if s.isTaskDue(TaskTypeProductRank, now) {
		s.executeTask(TaskTypeProductRank)
	}
}

// isTaskDue 检查任务是否到期
func (s *Scheduler) isTaskDue(taskType TaskType, now time.Time) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskType]
	if !exists {
		return false
	}

	// 如果是首次运行（LastRunAt 为零值），且调度器已运行超过 10 秒
	if task.LastRunAt.IsZero() {
		return false // 首次不自动运行，等待手动触发或配置
	}

	return task.Status == TaskStatusIdle && now.After(task.NextRunAt)
}

// executeTask 执行任务
func (s *Scheduler) executeTask(taskType TaskType) {
	s.mu.Lock()
	task, exists := s.tasks[taskType]
	if !exists || task.Status == TaskStatusRunning {
		s.mu.Unlock()
		return
	}

	// 标记为运行中
	runCtx, cancel := context.WithCancel(context.Background())
	task.Status = TaskStatusRunning
	task.LastRunAt = time.Now()
	task.RunCount++
	s.currentCancels[taskType] = cancel
	handler := s.taskHandler
	if handler == nil {
		task.Status = TaskStatusIdle
		task.LastError = "任务处理器未注册"
		delete(s.currentCancels, taskType)
		cancel()
		s.mu.Unlock()
		g.Log().Errorf(context.Background(), "[Scheduler] 无法执行任务 %s: %s", taskType, task.LastError)
		return
	}
	s.mu.Unlock()

	g.Log().Infof(context.Background(), "[Scheduler] 开始执行任务: %s (第 %d 次)", taskType, task.RunCount)

	// 执行任务
	err := handler(runCtx, taskType)
	if err == nil && runCtx.Err() != nil {
		err = runCtx.Err()
	}

	// 更新任务状态
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.currentCancels, taskType)
	cancel()

	if errors.Is(err, context.Canceled) {
		task.LastError = "任务已取消"
		g.Log().Warningf(context.Background(), "[Scheduler] 任务已取消: %s", taskType)
	} else if err != nil {
		task.LastError = err.Error()
		g.Log().Errorf(context.Background(), "[Scheduler] 任务执行失败: %s, 错误: %v", taskType, err)
	} else {
		task.LastError = ""
		g.Log().Infof(context.Background(), "[Scheduler] 任务执行完成: %s", taskType)
	}

	task.Status = TaskStatusIdle

	// 流水线：商品榜单完成后立即触发视频池抓取
	if taskType == TaskTypeProductRank && err == nil {
		g.Log().Info(context.Background(), "[Scheduler] 商品榜单抓取完成，立即触发商品详情/视频抓取")
		if s.running {
			go func() { s.triggerCh <- TaskTypeVideoPool }()
		} else {
			go s.executeTask(TaskTypeVideoPool)
		}
		return
	}

	// 视频池完成后，设置下一轮的等待时间
	if taskType == TaskTypeVideoPool {
		cycleWait := s.cycleInterval
		productTask := s.tasks[TaskTypeProductRank]
		if productTask != nil && productTask.Interval > 0 {
			cycleWait = productTask.Interval
		}
		productTask.NextRunAt = time.Now().Add(cycleWait)
		g.Log().Infof(context.Background(), "[Scheduler] 一轮抓取完成，%v 后开始下一轮", cycleWait)
		return
	}

	task.NextRunAt = time.Now().Add(task.Interval)
}

// TriggerTask 手动触发任务
func (s *Scheduler) TriggerTask(taskType TaskType) error {
	s.mu.Lock()
	task, exists := s.tasks[taskType]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("任务类型不存在: %s", taskType)
	}

	if task.Status == TaskStatusRunning {
		s.mu.Unlock()
		return fmt.Errorf("任务正在运行中: %s", taskType)
	}

	s.mu.Unlock()

	g.Log().Infof(context.Background(), "[Scheduler] 手动触发任务: %s", taskType)
	go s.executeTask(taskType)
	return nil
}

// CancelRunningTasks 取消正在执行的采集任务；stopScheduler=true 时同时停止后续调度循环。
func (s *Scheduler) CancelRunningTasks(stopScheduler bool) []TaskType {
	s.mu.Lock()
	cancels := make(map[TaskType]context.CancelFunc, len(s.currentCancels))
	for taskType, cancel := range s.currentCancels {
		cancels[taskType] = cancel
	}

	drained := 0
	if stopScheduler {
		s.running = false
		if s.ticker != nil {
			s.ticker.Stop()
			s.ticker = nil
		}
		for {
			select {
			case <-s.triggerCh:
				drained++
			default:
				goto drainedDone
			}
		}
	}

drainedDone:
	s.mu.Unlock()

	canceled := make([]TaskType, 0, len(cancels))
	for taskType, cancel := range cancels {
		cancel()
		canceled = append(canceled, taskType)
	}

	if stopScheduler {
		g.Log().Infof(context.Background(), "[Scheduler] 已停止调度循环，并清理待触发任务 %d 个", drained)
	}
	if len(canceled) > 0 {
		g.Log().Infof(context.Background(), "[Scheduler] 已下发任务取消信号: %v", canceled)
	} else {
		g.Log().Info(context.Background(), "[Scheduler] 当前没有运行中的采集任务")
	}

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
