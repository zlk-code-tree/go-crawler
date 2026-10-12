package service

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CrawlerStatus 爬虫服务全局状态
type CrawlerStatus struct {
	IsOnline  bool      `json:"isOnline"`
	TaskId    string    `json:"taskId"`
	LoginType string    `json:"loginType"`
	StartTime time.Time `json:"startTime"`
	Message   string    `json:"message"`
}

// crawlerStatusManager 全局爬虫服务状态管理器（单例）
type crawlerStatusManager struct {
	mu     sync.RWMutex
	status CrawlerStatus
	active *crawlerTaskLease
}

// The private context key proves ownership; a matching task ID alone does not.
type crawlerTaskLeaseKey struct{}

type crawlerTaskLease struct {
	manager  *crawlerStatusManager
	taskID   string
	cancel   context.CancelFunc
	stopping bool
}

var ErrCrawlerBusy = errors.New("爬虫服务正在运行或停止中，请稍后再试")

var (
	statusInstance *crawlerStatusManager
	statusOnce     sync.Once
)

// GetCrawlerStatusManager 获取全局状态管理器实例
func GetCrawlerStatusManager() *crawlerStatusManager {
	statusOnce.Do(func() {
		statusInstance = &crawlerStatusManager{}
	})
	return statusInstance
}

// GetStatus 获取当前服务状态
func (m *crawlerStatusManager) GetStatus() CrawlerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// AcquireTask atomically reserves the crawler until finish is called after cleanup.
// A controller may pass the returned context to the task runner, which can reuse
// the same lease. Only the original owner receives a release function.
func (m *crawlerStatusManager) AcquireTask(ctx context.Context, taskId, loginType string) (context.Context, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if lease, ok := ctx.Value(crawlerTaskLeaseKey{}).(*crawlerTaskLease); ok {
		if lease == m.active && lease.manager == m && lease.taskID == taskId && !lease.stopping {
			return ctx, func() {}, nil
		}
		return nil, nil, ErrCrawlerBusy
	}
	if m.active != nil {
		return nil, nil, ErrCrawlerBusy
	}
	taskCtx, cancel := context.WithCancel(ctx)
	lease := &crawlerTaskLease{manager: m, taskID: taskId, cancel: cancel}
	m.active = lease
	m.status = CrawlerStatus{
		IsOnline:  true,
		TaskId:    taskId,
		LoginType: loginType,
		StartTime: time.Now(),
		Message:   "运行中",
	}
	var once sync.Once
	finish := func() {
		once.Do(func() {
			cancel()
			m.mu.Lock()
			defer m.mu.Unlock()
			if m.active == lease {
				m.active = nil
				m.status = CrawlerStatus{}
			}
		})
	}
	return context.WithValue(taskCtx, crawlerTaskLeaseKey{}, lease), finish, nil
}

// SetMessage 更新状态消息（如"登录中"、"爬取中"等）
func (m *crawlerStatusManager) SetMessage(ctx context.Context, msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	lease, _ := ctx.Value(crawlerTaskLeaseKey{}).(*crawlerTaskLease)
	if lease != nil && lease == m.active && !lease.stopping && ctx.Err() == nil {
		m.status.Message = msg
	}
}

// ForceOffline requests cancellation. The task stays occupied until its owner
// finishes browser cleanup, so another task cannot overlap the stopping task.
func (m *crawlerStatusManager) ForceOffline() {
	m.mu.Lock()
	lease := m.active
	if lease != nil {
		lease.stopping = true
		m.status.Message = "正在停止，等待浏览器清理"
	}
	m.mu.Unlock()
	if lease != nil {
		lease.cancel()
	}
}

// IsOnline 快速检查是否在线
func (m *crawlerStatusManager) IsOnline() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status.IsOnline
}
