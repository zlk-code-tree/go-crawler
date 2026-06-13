package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

// CronListReq 定时任务列表
type CronListReq struct {
	g.Meta `path:"/task-schedule/cron-list" method:"get" tags:"TaskSchedule" summary:"定时任务列表"`
}

type CronListRes struct {
	List []CronTaskItem `json:"list"`
}

type CronTaskItem struct {
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

// CronSaveReq 保存定时任务
type CronSaveReq struct {
	g.Meta        `path:"/task-schedule/cron-save" method:"post" tags:"TaskSchedule" summary:"保存定时任务"`
	Id            int64  `json:"id"`
	Name          string `json:"name" v:"required#任务名称不能为空"`
	CronExpr      string `json:"cronExpr" v:"required#Cron表达式不能为空"`
	ApiUrl        string `json:"apiUrl"`
	ApiMethod     string `json:"apiMethod" d:"GET"`
	ApiParams     string `json:"apiParams"`
	RetryTimes    int    `json:"retryTimes"`
	RetryInterval int    `json:"retryInterval"`
	NextTask      string `json:"nextTask"`
}

type CronSaveRes struct{}

// CronStartReq 启动任务
type CronStartReq struct {
	g.Meta `path:"/task-schedule/cron-start" method:"post" tags:"TaskSchedule" summary:"启动定时任务"`
	Name   string `json:"name" v:"required#任务名称不能为空"`
}

type CronStartRes struct{}

// CronStopReq 停止任务
type CronStopReq struct {
	g.Meta `path:"/task-schedule/cron-stop" method:"post" tags:"TaskSchedule" summary:"停止定时任务"`
	Name   string `json:"name" v:"required#任务名称不能为空"`
}

type CronStopRes struct{}

// CronTriggerReq 手动触发
type CronTriggerReq struct {
	g.Meta `path:"/task-schedule/cron-trigger" method:"post" tags:"TaskSchedule" summary:"手动触发任务"`
	Name   string `json:"name" v:"required#任务名称不能为空"`
}

type CronTriggerRes struct {
	Message string `json:"message"`
}

// CronDeleteReq 删除任务
type CronDeleteReq struct {
	g.Meta `path:"/task-schedule/cron-delete" method:"post" tags:"TaskSchedule" summary:"删除定时任务"`
	Name   string `json:"name" v:"required#任务名称不能为空"`
}

type CronDeleteRes struct{}
