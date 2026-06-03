package v1

import (
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

// ==================== 任务调度监控 ====================

// TaskScheduleListReq 任务列表
type TaskScheduleListReq struct {
	g.Meta `path:"/task-schedule/list" method:"get" tags:"TaskSchedule" summary:"获取任务统计列表"`
}

// TaskScheduleListRes 任务列表响应
type TaskScheduleListRes struct {
	List []model.TaskStatsItem `json:"list"`
}

// TaskScheduleLogsReq 任务执行日志
type TaskScheduleLogsReq struct {
	g.Meta   `path:"/task-schedule/logs" method:"get" tags:"TaskSchedule" summary:"获取任务执行日志"`
	TaskType string `json:"taskType" in:"query" v:"required" dc:"任务类型"`
	Page     int    `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"20" dc:"每页数量"`
}

// TaskScheduleLogsRes 任务执行日志响应
type TaskScheduleLogsRes struct {
	List  []model.TaskLogEntry `json:"list"`
	Total int64                  `json:"total"`
}

// TaskScheduleLogDetailReq 日志详情
type TaskScheduleLogDetailReq struct {
	g.Meta `path:"/task-schedule/log-detail" method:"get" tags:"TaskSchedule" summary:"获取日志详情"`
	Id     int64 `json:"id" in:"query" v:"required" dc:"日志ID"`
}

// TaskScheduleLogDetailRes 日志详情响应
type TaskScheduleLogDetailRes = model.TaskLogEntry

// TaskScheduleStatsReq 概览统计
type TaskScheduleStatsReq struct {
	g.Meta `path:"/task-schedule/stats" method:"get" tags:"TaskSchedule" summary:"获取概览统计"`
}

// TaskScheduleStatsRes 概览统计响应
type TaskScheduleStatsRes = model.TaskScheduleStats
