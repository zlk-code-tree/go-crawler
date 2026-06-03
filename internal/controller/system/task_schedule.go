package system

import (
	"context"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/service"
)

type TaskScheduleController struct{}

func NewTaskScheduleController() *TaskScheduleController {
	return &TaskScheduleController{}
}

// List 任务列表 GET /api/task-schedule/list
func (c *TaskScheduleController) List(ctx context.Context, req *v1.TaskScheduleListReq) (res *v1.TaskScheduleListRes, err error) {
	list, err := service.GetTaskList(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.TaskScheduleListRes{List: list}, nil
}

// Logs 执行日志 GET /api/task-schedule/logs
func (c *TaskScheduleController) Logs(ctx context.Context, req *v1.TaskScheduleLogsReq) (res *v1.TaskScheduleLogsRes, err error) {
	logs, total, err := service.GetTaskLogs(ctx, req.TaskType, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	return &v1.TaskScheduleLogsRes{List: logs, Total: total}, nil
}

// LogDetail 日志详情 GET /api/task-schedule/log-detail
func (c *TaskScheduleController) LogDetail(ctx context.Context, req *v1.TaskScheduleLogDetailReq) (res *v1.TaskScheduleLogDetailRes, err error) {
	return service.GetTaskLogDetail(ctx, req.Id)
}

// Stats 概览统计 GET /api/task-schedule/stats
func (c *TaskScheduleController) Stats(ctx context.Context, req *v1.TaskScheduleStatsReq) (res *v1.TaskScheduleStatsRes, err error) {
	return service.GetTaskStats(ctx)
}
