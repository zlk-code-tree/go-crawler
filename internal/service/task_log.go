package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

// ==================== FailAlert 连续失败告警 ====================

const (
	consecutiveFailThreshold = 5 // 连续失败阈值
)

type FailAlert struct {
	mu               sync.Mutex
	failCounters     map[string]int       // taskType → 连续失败次数
	lastAlertTime    map[string]time.Time // taskType → 上次告警时间（防止重复告警）
	feishuWebhookUrl string
}

var failAlertInstance *FailAlert
var failAlertOnce sync.Once

func getFailAlert() *FailAlert {
	failAlertOnce.Do(func() {
		failAlertInstance = &FailAlert{
			failCounters:  make(map[string]int),
			lastAlertTime: make(map[string]time.Time),
			feishuWebhookUrl: g.Cfg().MustGet(context.Background(),
				"task.feishuWebhook", "").String(),
		}
	})
	return failAlertInstance
}

// RecordFail 记录一次失败，返回是否需要告警
func (a *FailAlert) RecordFail(ctx context.Context, taskType, taskName, errorMsg string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.failCounters[taskType]++
	count := a.failCounters[taskType]

	shouldAlert := count >= consecutiveFailThreshold
	if shouldAlert {
		// 30分钟内不重复告警
		if last, ok := a.lastAlertTime[taskType]; ok && time.Since(last) < 30*time.Minute {
			return false
		}
		a.lastAlertTime[taskType] = time.Now()

		if a.feishuWebhookUrl != "" {
			go a.sendFeishuAlert(taskType, taskName, count, errorMsg)
		}
	}
	return shouldAlert
}

// RecordSuccess 成功后重置计数器
func (a *FailAlert) RecordSuccess(taskType string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.failCounters, taskType)
}

// GetConsecutiveFails 获取某个任务当前的连续失败次数
func (a *FailAlert) GetConsecutiveFails(taskType string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.failCounters[taskType]
}

// sendFeishuAlert 发送飞书机器人告警
func (a *FailAlert) sendFeishuAlert(taskType, taskName string, failCount int, errorMsg string) {
	ctx := context.Background()
	msg := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title":    map[string]string{"content": "⚠️ 任务调度告警", "tag": "plain_text"},
				"template": "red",
			},
			"elements": []map[string]interface{}{
				{"tag": "div", "text": map[string]string{"content": fmt.Sprintf("任务：%s（%s）\n连续失败：%d 次\n最近错误：%s\n时间：%s", taskName, taskType, failCount, errorMsg, time.Now().Format("2006-01-02 15:04:05")), "tag": "lark_md"}},
			},
		},
	}
	body, _ := json.Marshal(msg)
	g.Log().Infof(ctx, "[FailAlert] 发送飞书告警: task=%s, fails=%d", taskType, failCount)
	resp, err := http.Post(a.feishuWebhookUrl, "application/json", bytes.NewReader(body))
	if err != nil {
		g.Log().Errorf(ctx, "[FailAlert] 飞书告警发送失败: %v", err)
		return
	}
	resp.Body.Close()
	g.Log().Infof(ctx, "[FailAlert] 飞书告警已发送, status=%d", resp.StatusCode)
}

// ==================== 日志操作方法 ====================

// TaskLogStart 记录任务开始，返回 logId
func TaskLogStart(ctx context.Context, taskName, taskType, apiUrl, requestParams string) (int64, error) {
	result, err := g.DB().Ctx(ctx).Model("task_execution_log").Data(g.Map{
		"task_name":      taskName,
		"task_type":      taskType,
		"status":         0, // 运行中
		"start_time":     time.Now().Format("2006-01-02 15:04:05"),
		"api_url":        apiUrl,
		"request_params": requestParams,
	}).Insert()
	if err != nil {
		return 0, err
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// TaskLogProgress 更新后台任务进度，status 保持为运行中。
func TaskLogProgress(ctx context.Context, logID int64, progress interface{}, durationMs int) {
	if logID <= 0 {
		return
	}
	responseBody, err := json.Marshal(progress)
	if err != nil {
		return
	}
	if len(responseBody) > 50000 {
		responseBody = append(responseBody[:50000], []byte("...[truncated]")...)
	}
	_, _ = g.DB().Ctx(ctx).Model("task_execution_log").Where("id", logID).Data(g.Map{
		"duration_ms":   durationMs,
		"response_body": string(responseBody),
	}).Update()
}

// TaskLogSuccess 标记任务成功
func TaskLogSuccess(ctx context.Context, logId int64, responseCode, responseBody string, durationMs int) {
	// 截断过长的响应体（保留前50000字符）
	if len(responseBody) > 50000 {
		responseBody = responseBody[:50000] + "...[truncated]"
	}
	g.DB().Ctx(ctx).Model("task_execution_log").Where("id", logId).Data(g.Map{
		"status":        1,
		"end_time":      time.Now().Format("2006-01-02 15:04:05"),
		"duration_ms":   durationMs,
		"response_code": responseCode,
		"response_body": responseBody,
	}).Update()

	// 重置该任务类型的失败计数
	getFailAlert().RecordSuccess(g.Cfg().MustGet(ctx, "task.log_task_type", "douyin_poller").String())
}

// TaskLogFailed 标记任务失败
func TaskLogFailed(ctx context.Context, logId int64, errorMsg string, durationMs int) {
	g.DB().Ctx(ctx).Model("task_execution_log").Where("id", logId).Data(g.Map{
		"status":      2,
		"end_time":    time.Now().Format("2006-01-02 15:04:05"),
		"duration_ms": durationMs,
		"error_msg":   errorMsg,
	}).Update()

	// 触发失败告警
	getFailAlert().RecordFail(ctx, "douyin_poller", "抖店订单轮询", errorMsg)
}

// TaskLogSkip 记录跳过（如未授权店铺）
func TaskLogSkip(ctx context.Context, taskName, taskType, apiUrl, reason string) {
	g.DB().Ctx(ctx).Model("task_execution_log").Data(g.Map{
		"task_name":  taskName,
		"task_type":  taskType,
		"status":     3, // 跳过
		"start_time": time.Now().Format("2006-01-02 15:04:05"),
		"end_time":   time.Now().Format("2006-01-02 15:04:05"),
		"api_url":    apiUrl,
		"error_msg":  reason,
	}).Insert()
}

// GetTaskList 获取任务聚合列表（按 task_type 分组统计）
func GetTaskList(ctx context.Context) ([]model.TaskStatsItem, error) {
	rows, err := g.DB().GetAll(ctx, `
		SELECT
			task_type,
			MAX(task_name) as task_name,
			COUNT(*) as total_count,
			SUM(CASE WHEN status=1 THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN status=2 THEN 1 ELSE 0 END) as fail_count,
			SUM(CASE WHEN status=3 THEN 1 ELSE 0 END) as skip_count,
			AVG(CASE WHEN duration_ms>0 THEN duration_ms END) as avg_duration_ms
		FROM task_execution_log
		GROUP BY task_type
		ORDER BY MAX(created_at) DESC
	`)
	if err != nil {
		return nil, err
	}

	var list []model.TaskStatsItem
	for _, row := range rows {
		total := row["total_count"].Int()
		success := row["success_count"].Int()
		rate := float64(0)
		if total > 0 {
			rate = float64(success) / float64(total) * 100
		}
		list = append(list, model.TaskStatsItem{
			TaskType:      row["task_type"].String(),
			TaskName:      row["task_name"].String(),
			TotalCount:    total,
			SuccessCount:  success,
			FailCount:     row["fail_count"].Int(),
			SkipCount:     row["skip_count"].Int(),
			SuccessRate:   rate,
			AvgDurationMs: row["avg_duration_ms"].Int(),
		})
	}

	// 补充实时状态（运行中/空闲）+连续失败次数
	alert := getFailAlert()
	for i := range list {
		list[i].LastStatus = getTaskLastStatus(ctx, list[i].TaskType)
		if list[i].LastStatus == 0 {
			list[i].LastStatusDesc = "运行中"
		} else if list[i].LastStatus == 1 {
			list[i].LastStatusDesc = "空闲"
		} else if list[i].LastStatus == 2 {
			list[i].LastStatusDesc = "异常"
		}
		// 最后运行时间
		last, _ := g.DB().Ctx(ctx).Model("task_execution_log").
			Where("task_type", list[i].TaskType).
			Order("id DESC").Fields("start_time").One()
		if !last.IsEmpty() {
			list[i].LastRunTime = last["start_time"].String()
		}
		// 连续失败次数（用于前端红色高亮判断）
		if fails := alert.GetConsecutiveFails(list[i].TaskType); fails > 0 {
			if list[i].LastStatusDesc == "空闲" {
				list[i].LastStatusDesc = "异常"
			}
		}
	}

	return list, nil
}

func getTaskLastStatus(ctx context.Context, taskType string) int {
	row, err := g.DB().Ctx(ctx).Model("task_execution_log").
		Where("task_type", taskType).
		Order("id DESC").Fields("status").One()
	if err != nil || row.IsEmpty() {
		return 1 // 默认空闲
	}
	return row["status"].Int()
}

// GetTaskLogs 获取某个任务的执行日志（分页）
func GetTaskLogs(ctx context.Context, taskType string, page, pageSize int) ([]model.TaskLogEntry, int64, error) {
	m := g.DB().Ctx(ctx).Model("task_execution_log").Where("task_type", taskType)

	total, err := m.Count()
	if err != nil {
		return nil, 0, err
	}

	var logs []model.TaskLogEntry
	err = m.Order("id DESC").Page(page, pageSize).Scan(&logs)
	if err != nil {
		return nil, 0, err
	}
	if logs == nil {
		logs = []model.TaskLogEntry{}
	}
	return logs, int64(total), nil
}

// GetTaskLogDetail 获取单条日志详情
func GetTaskLogDetail(ctx context.Context, logId int64) (*model.TaskLogEntry, error) {
	var log *model.TaskLogEntry
	err := g.DB().Ctx(ctx).Model("task_execution_log").Where("id", logId).Scan(&log)
	if err != nil {
		return nil, err
	}
	return log, nil
}

// GetTaskStats 概览统计
func GetTaskStats(ctx context.Context) (*model.TaskScheduleStats, error) {
	today := time.Now().Format("2006-01-02")

	// 任务类型数
	types, _ := g.DB().Ctx(ctx).Model("task_execution_log").
		Fields("COUNT(DISTINCT task_type) as cnt").One()
	totalTasks := 0
	if !types.IsEmpty() {
		totalTasks = types["cnt"].Int()
	}

	// 运行中
	running, _ := g.DB().Ctx(ctx).Model("task_execution_log").
		Where("status", 0).Count()

	// 今日成功
	todaySuccess, _ := g.DB().Ctx(ctx).Model("task_execution_log").
		Where("status", 1).
		WhereGTE("start_time", today+" 00:00:00").
		WhereLTE("start_time", today+" 23:59:59").
		Count()

	// 今日失败
	todayFail, _ := g.DB().Ctx(ctx).Model("task_execution_log").
		Where("status", 2).
		WhereGTE("start_time", today+" 00:00:00").
		WhereLTE("start_time", today+" 23:59:59").
		Count()

	return &model.TaskScheduleStats{
		TotalTasks:   totalTasks,
		RunningCount: running,
		TodaySuccess: todaySuccess,
		TodayFail:    todayFail,
	}, nil
}

// CleanOldLogs 清理7天前的详细日志（保留统计信息，清除 response_body）
func CleanOldLogs(ctx context.Context) error {
	cutoff := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	_, err := g.DB().Ctx(ctx).Model("task_execution_log").
		WhereLTE("created_at", cutoff+" 23:59:59").
		WhereNot("response_body", "").
		Data(g.Map{"response_body": ""}).
		Update()
	if err != nil {
		return err
	}
	g.Log().Infof(ctx, "[TaskLog] 已清理 %s 之前的历史日志详情", cutoff)
	return nil
}
