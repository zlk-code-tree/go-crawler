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

func (c *TaskScheduleController) List(ctx context.Context, req *v1.TaskScheduleListReq) (res *v1.TaskScheduleListRes, err error) {
	list, err := service.GetTaskList(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.TaskScheduleListRes{List: list}, nil
}

func (c *TaskScheduleController) Logs(ctx context.Context, req *v1.TaskScheduleLogsReq) (res *v1.TaskScheduleLogsRes, err error) {
	logs, total, err := service.GetTaskLogs(ctx, req.TaskType, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	return &v1.TaskScheduleLogsRes{List: logs, Total: total}, nil
}

func (c *TaskScheduleController) LogDetail(ctx context.Context, req *v1.TaskScheduleLogDetailReq) (res *v1.TaskScheduleLogDetailRes, err error) {
	return service.GetTaskLogDetail(ctx, req.Id)
}

func (c *TaskScheduleController) Stats(ctx context.Context, req *v1.TaskScheduleStatsReq) (res *v1.TaskScheduleStatsRes, err error) {
	return service.GetTaskStats(ctx)
}

// CronList 定时任务列表
func (c *TaskScheduleController) CronList(ctx context.Context, req *v1.CronListReq) (res *v1.CronListRes, err error) {
	list := service.GetCronManager().List(ctx)
	items := make([]v1.CronTaskItem, len(list))
	for i, t := range list {
		items[i] = v1.CronTaskItem{
			Id: t.Id, Name: t.Name, DisplayName: t.DisplayName, CronExpr: t.CronExpr,
			ApiUrl: t.ApiUrl, ApiMethod: t.ApiMethod, ApiParams: t.ApiParams,
			Status: t.Status, RetryTimes: t.RetryTimes, RetryInterval: t.RetryInterval,
			NextTask: t.NextTask, Running: t.Running, LastRunTime: t.LastRunTime,
		}
	}
	return &v1.CronListRes{List: items}, nil
}

// CronSave 保存定时任务
func (c *TaskScheduleController) CronSave(ctx context.Context, req *v1.CronSaveReq) (res *v1.CronSaveRes, err error) {
	err = service.GetCronManager().SaveDbTask(ctx, &service.CronTaskInfo{
		Name: req.Name, CronExpr: req.CronExpr, ApiUrl: req.ApiUrl,
		ApiMethod: req.ApiMethod, ApiParams: req.ApiParams,
		RetryTimes: req.RetryTimes, RetryInterval: req.RetryInterval, NextTask: req.NextTask,
		Status: 1,
	})
	return &v1.CronSaveRes{}, err
}

// CronStart 启动任务
func (c *TaskScheduleController) CronStart(ctx context.Context, req *v1.CronStartReq) (res *v1.CronStartRes, err error) {
	return &v1.CronStartRes{}, nil
}

// CronStop 停止任务
func (c *TaskScheduleController) CronStop(ctx context.Context, req *v1.CronStopReq) (res *v1.CronStopRes, err error) {
	service.GetCronManager().Stop(req.Name)
	return &v1.CronStopRes{}, nil
}

// CronTrigger 手动触发
func (c *TaskScheduleController) CronTrigger(ctx context.Context, req *v1.CronTriggerReq) (res *v1.CronTriggerRes, err error) {
	err = service.GetCronManager().Trigger(ctx, req.Name)
	if err != nil {
		return &v1.CronTriggerRes{Message: err.Error()}, nil
	}
	return &v1.CronTriggerRes{Message: "任务已触发"}, nil
}

// CronDelete 删除任务
func (c *TaskScheduleController) CronDelete(ctx context.Context, req *v1.CronDeleteReq) (res *v1.CronDeleteRes, err error) {
	err = service.GetCronManager().DeleteDbTask(req.Name)
	return &v1.CronDeleteRes{}, err
}
