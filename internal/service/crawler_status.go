package service

import (
	"context"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
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
	mu       sync.RWMutex
	status   CrawlerStatus
	cancelFn context.CancelFunc // 用于强制停止爬虫任务
}

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

// SetOnline 标记服务上线
func (m *crawlerStatusManager) SetOnline(taskId, loginType string, cancelFn context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = CrawlerStatus{
		IsOnline:  true,
		TaskId:    taskId,
		LoginType: loginType,
		StartTime: time.Now(),
		Message:   "运行中",
	}
	m.cancelFn = cancelFn
	g.Log().Infof(context.Background(), "[CrawlerStatus] 服务上线 taskId=%s loginType=%s", taskId, loginType)
}

// SetOffline 标记服务下线
func (m *crawlerStatusManager) SetOffline() {
	m.mu.Lock()
	defer m.mu.Unlock()
	oldTaskId := m.status.TaskId
	m.status = CrawlerStatus{}
	m.cancelFn = nil
	g.Log().Infof(context.Background(), "[CrawlerStatus] 服务下线 taskId=%s", oldTaskId)
}

// SetMessage 更新状态消息（如"登录中"、"爬取中"等）
func (m *crawlerStatusManager) SetMessage(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.Message = msg
}

// ForceOffline 强制下线（取消正在运行的任务）
func (m *crawlerStatusManager) ForceOffline() {
	m.mu.Lock()
	cancelFn := m.cancelFn
	taskId := m.status.TaskId
	m.status = CrawlerStatus{}
	m.cancelFn = nil
	m.mu.Unlock()

	if cancelFn != nil {
		cancelFn()
	}

	// 更新数据库中的任务状态
	if taskId != "" {
		_, _ = g.DB().Model("crawl_tasks").Where("task_id", taskId).
			Where("status", "running").
			Update(g.Map{"status": "stopped", "end_time": time.Now()})
	}
	g.Log().Infof(context.Background(), "[CrawlerStatus] 强制下线 taskId=%s", taskId)
}

// IsOnline 快速检查是否在线
func (m *crawlerStatusManager) IsOnline() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status.IsOnline
}
