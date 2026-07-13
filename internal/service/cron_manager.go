package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcron"
)

// CronTaskInfo 定时任务信息
type CronTaskInfo struct {
	Id            int64  `json:"id"`
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	CronExpr      string `json:"cronExpr"`
	ApiUrl        string `json:"apiUrl"`
	ApiMethod     string `json:"apiMethod"`
	ApiParams     string `json:"apiParams"`
	Status        int    `json:"status"`
	RetryTimes    int    `json:"retryTimes"`
	RetryInterval int    `json:"retryInterval"`
	NextTask      string `json:"nextTask"`
	Running       bool   `json:"running"`
	LastRunTime   string `json:"lastRunTime"`
}

type builtinTask struct {
	cronExpr      string
	handler       func(context.Context) error
	notifySuccess bool
}

// CronManager 定时任务管理器
type CronManager struct {
	mu      sync.Mutex
	builtin map[string]builtinTask
	running map[string]bool
	lastRun map[string]string
}

var (
	cronManagerOnce     sync.Once
	cronManagerInstance *CronManager
)

// GetCronManager 获取单例
func GetCronManager() *CronManager {
	cronManagerOnce.Do(func() {
		cronManagerInstance = &CronManager{
			builtin: make(map[string]builtinTask),
			running: make(map[string]bool),
			lastRun: make(map[string]string),
		}
	})
	return cronManagerInstance
}

// builtinDisplayNames 内置任务中文名
var builtinDisplayNames = map[string]string{
	"douyin_order_daily_pull":      "抖店订单及达人每日对账",
	"douyin_order_today_pull":      "抖店订单及达人当日同步",
	"wechat_shop_order_daily_pull": "微信小店订单每日拉取",
	"finance_daily_aggregate":      "达人佣金日聚合",
	"order_daily_aggregate":        "抖音订单日聚合",
	"qianchuan_daily_pull":         "千川数据每日拉取",
	"qianchuan_token_refresh":      "千川Token自动续期",
	"profit_morning_push":          "盈亏早报推送",
	"profit_evening_push":          "盈亏晚报推送",
}

// RegisterBuiltin 注册内置任务
// handler 返回 error 时由 CronManager 统一记录失败日志与通知，handler 内部不应再重复记录
func (m *CronManager) RegisterBuiltin(name, cronExpr string, handler func(context.Context) error) {
	m.registerBuiltin(name, cronExpr, handler, true)
}

// RegisterBuiltinSilent 注册成功时不生成系统通知的高频维护任务，失败仍正常告警。
func (m *CronManager) RegisterBuiltinSilent(name, cronExpr string, handler func(context.Context) error) {
	m.registerBuiltin(name, cronExpr, handler, false)
}

func (m *CronManager) registerBuiltin(name, cronExpr string, handler func(context.Context) error, notifySuccess bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.builtin[name] = builtinTask{cronExpr: cronExpr, handler: handler, notifySuccess: notifySuccess}
}

// StartAll 启动所有任务
func (m *CronManager) StartAll(ctx context.Context) {
	registered := 0
	failed := 0
	for name, task := range m.builtin {
		if err := m.startWrapped(ctx, name, task.cronExpr, task.handler, task.notifySuccess); err != nil {
			failed++
		} else {
			registered++
		}
	}
	var dbTasks []CronTaskInfo
	if err := g.DB().Model("cron_task").Ctx(ctx).Where("status", 1).Scan(&dbTasks); err != nil {
		g.Log().Errorf(ctx, "[CronManager] 加载数据库定时任务失败: %v", err)
		failed++
	} else {
		for _, t := range dbTasks {
			if err := m.startDbTask(ctx, t); err != nil {
				failed++
			} else {
				registered++
			}
		}
	}
	g.Log().Infof(ctx, "[CronManager] 定时任务注册完成: 成功%d 失败%d", registered, failed)
}

// startWrapped 启动带包装的cron任务
func (m *CronManager) startWrapped(ctx context.Context, name, pattern string, handler func(context.Context) error, notifySuccess bool) error {
	_, err := gcron.AddSingleton(ctx, pattern, func(ctx context.Context) {
		m.runWithLock(ctx, name, handler, notifySuccess)
	}, name)
	if err != nil {
		g.Log().Errorf(ctx, "[CronManager] 注册任务失败 name=%s pattern=%s err=%v", name, pattern, err)
		return err
	}
	g.Log().Infof(ctx, "[CronManager] 注册任务成功 name=%s pattern=%s", name, pattern)
	return nil
}

// runWithLock 防重复执行 + 日志记录 + 通知 + 任务链
// handler 返回 error 时记录失败，否则记录成功；handler 内部不应再重复写任务日志/通知
func (m *CronManager) runWithLock(ctx context.Context, name string, handler func(context.Context) error, notifySuccess bool) {
	m.mu.Lock()
	if m.running[name] {
		m.mu.Unlock()
		g.Log().Warningf(ctx, "[CronManager] 任务 %s 正在执行中，跳过本次调度", name)
		return
	}
	m.running[name] = true
	m.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			g.Log().Errorf(ctx, "[CronManager] 任务 %s panic: %v", name, r)
		}
		m.mu.Lock()
		m.running[name] = false
		m.lastRun[name] = time.Now().Format("2006-01-02 15:04:05")
		m.mu.Unlock()
	}()

	logId, _ := TaskLogStart(ctx, name, name, "", "")
	t0 := time.Now()
	err := handler(ctx)
	dur := int(time.Since(t0).Milliseconds())

	if err != nil {
		TaskLogFailed(ctx, logId, err.Error(), dur)
		CreateNotification(ctx, name+" 执行失败", fmt.Sprintf(`{"错误":"%s","耗时":%d}`, err.Error(), dur), "task_result", name, logId, NotifyLevelImportant, 0)
		g.Log().Errorf(ctx, "[CronManager] 任务 %s 失败: %v", name, err)
		return
	}

	TaskLogSuccess(ctx, logId, "", fmt.Sprintf(`{"durationMs":%d}`, dur), dur)
	if notifySuccess {
		CreateNotification(ctx, name+" 执行完成", fmt.Sprintf(`{"耗时":%d,"任务":"%s"}`, dur, name), "task_result", name, logId, NotifyLevelNormal, 0)
	}

	// 任务链
	if t := m.getDbTask(ctx, name); t != nil && t.NextTask != "" {
		g.Log().Infof(ctx, "[CronManager] 任务链触发: %s -> %s", name, t.NextTask)
		m.triggerByName(ctx, t.NextTask)
	}
}

func (m *CronManager) startDbTask(ctx context.Context, t CronTaskInfo) error {
	if t.Status != 1 {
		return nil
	}
	// executeHttpTask 内部已自行处理重试与日志记录，这里返回 nil 不让 CronManager 重复记录
	handler := func(ctx context.Context) error { m.executeHttpTask(ctx, t); return nil }
	return m.startWrapped(ctx, t.Name, t.CronExpr, handler, true)
}

// Stop 停止任务
func (m *CronManager) Stop(name string) {
	gcron.Remove(name)
	g.DB().Model("cron_task").Where("name", name).Data(g.Map{"status": 0}).Update()
}

// ResumeTask 恢复内置任务的定时调度（用于被 Stop 停止后重新启动）
// 仅对 builtin 任务有效，返回是否找到并恢复
func (m *CronManager) ResumeTask(ctx context.Context, name string) bool {
	task, ok := m.builtin[name]
	if !ok {
		return false
	}
	gcron.Remove(name) // 先移除避免重复注册
	return m.startWrapped(ctx, name, task.cronExpr, task.handler, task.notifySuccess) == nil
}

// Trigger 手动触发
func (m *CronManager) Trigger(ctx context.Context, name string) error {
	return m.triggerByName(ctx, name)
}

func (m *CronManager) triggerByName(ctx context.Context, name string) error {
	m.mu.Lock()
	if m.running[name] {
		m.mu.Unlock()
		return fmt.Errorf("任务 %s 正在执行中，请稍后重试", name)
	}
	m.mu.Unlock()

	go func() {
		defer func() {
			m.mu.Lock()
			m.running[name] = false
			m.lastRun[name] = time.Now().Format("2006-01-02 15:04:05")
			m.mu.Unlock()
		}()

		m.mu.Lock()
		m.running[name] = true
		m.mu.Unlock()

		if task, ok := m.builtin[name]; ok {
			m.runWithLock(ctx, name, task.handler, task.notifySuccess)
			return
		}
		if t := m.getDbTask(ctx, name); t != nil {
			m.executeHttpTask(ctx, *t)
		}
	}()
	return nil
}

func (m *CronManager) getDbTask(ctx context.Context, name string) *CronTaskInfo {
	var t CronTaskInfo
	err := g.DB().Model("cron_task").Ctx(ctx).Where("name", name).Scan(&t)
	if err != nil || t.Id == 0 {
		return nil
	}
	return &t
}

func (m *CronManager) executeHttpTask(ctx context.Context, t CronTaskInfo) {
	logId, _ := TaskLogStart(ctx, t.Name, t.Name, t.ApiUrl, t.ApiParams)
	t0 := time.Now()
	var lastErr error

	for attempt := 0; attempt <= t.RetryTimes; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(t.RetryInterval) * time.Second)
			g.Log().Infof(ctx, "[CronManager] 任务 %s 第%d次重试", t.Name, attempt)
		}
		client := g.Client()
		client.SetTimeout(5 * time.Minute)
		var err error

		if t.ApiMethod == "POST" {
			var params map[string]interface{}
			json.Unmarshal([]byte(t.ApiParams), &params)
			_, err = client.Post(ctx, t.ApiUrl, params)
		} else {
			_, err = client.Get(ctx, t.ApiUrl)
		}

		if err == nil {
			dur := int(time.Since(t0).Milliseconds())
			TaskLogSuccess(ctx, logId, "", fmt.Sprintf(`{"耗时":%d}`, dur), dur)
			CreateNotification(ctx, t.Name+" 执行完成", fmt.Sprintf(`{"耗时":%d}`, dur), "task_result", t.Name, logId, NotifyLevelNormal, 0)
			if t.NextTask != "" {
				m.triggerByName(ctx, t.NextTask)
			}
			return
		}
		lastErr = err
	}

	dur := int(time.Since(t0).Milliseconds())
	TaskLogFailed(ctx, logId, lastErr.Error(), dur)
	CreateNotification(ctx, t.Name+" 执行失败", fmt.Sprintf(`{"错误":"%s"}`, lastErr.Error()), "task_result", t.Name, logId, NotifyLevelImportant, 0)
}

// List 获取所有任务状态
func (m *CronManager) List(ctx context.Context) []CronTaskInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []CronTaskInfo
	for name, task := range m.builtin {
		dn := builtinDisplayNames[name]
		if dn == "" {
			dn = name
		}
		list = append(list, CronTaskInfo{
			Name:        name,
			DisplayName: dn,
			CronExpr:    task.cronExpr,
			Status:      1,
			Running:     m.running[name],
			LastRunTime: m.lastRun[name],
		})
	}
	var dbTasks []CronTaskInfo
	g.DB().Model("cron_task").Ctx(ctx).OrderAsc("id").Scan(&dbTasks)
	for i := range dbTasks {
		dbTasks[i].Running = m.running[dbTasks[i].Name]
		dbTasks[i].LastRunTime = m.lastRun[dbTasks[i].Name]
	}
	list = append(list, dbTasks...)
	return list
}

// SaveDbTask 保存数据库任务
func (m *CronManager) SaveDbTask(ctx context.Context, t *CronTaskInfo) error {
	count, _ := g.DB().Model("cron_task").Ctx(ctx).Where("name", t.Name).Count()
	if count > 0 {
		_, err := g.DB().Model("cron_task").Ctx(ctx).Where("name", t.Name).Data(t).Update()
		return err
	}
	_, err := g.DB().Model("cron_task").Ctx(ctx).Data(t).Insert()
	if err == nil {
		// 启动或停止 cron
		if t.Status == 1 {
			if _, ok := m.builtin[t.Name]; !ok {
				m.startDbTask(ctx, *t)
			}
		} else {
			gcron.Remove(t.Name)
		}
	}
	return err
}

// DeleteDbTask 删除数据库任务
func (m *CronManager) DeleteDbTask(name string) error {
	m.Stop(name)
	_, err := g.DB().Model("cron_task").Where("name", name).Delete()
	return err
}
