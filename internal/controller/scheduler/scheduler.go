package scheduler

import (
	"time"

	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// Controller 调度器控制器
type Controller struct{}

// New 创建调度器控制器
func New() *Controller {
	return &Controller{}
}

// Start 启动调度器
func (c *Controller) Start(r *ghttp.Request) {
	if !service.SchedulerManualTriggerEnabled(r.Context()) {
		r.Response.WriteJsonExit(g.Map{
			"code":    1,
			"message": "采集调度器已关闭，无法手动启动",
		})
		return
	}

	scheduler := service.GetScheduler()
	if scheduler.IsRunning() {
		r.Response.WriteJsonExit(g.Map{
			"code":    0,
			"message": "调度器已在运行中",
		})
		return
	}

	scheduler.Start()
	r.Response.WriteJsonExit(g.Map{
		"code":    0,
		"message": "调度器已启动",
	})
}

// Stop 停止调度器
func (c *Controller) Stop(r *ghttp.Request) {
	scheduler := service.GetScheduler()
	if !scheduler.IsRunning() {
		r.Response.WriteJsonExit(g.Map{
			"code":    0,
			"message": "调度器未在运行",
		})
		return
	}

	scheduler.Stop()
	r.Response.WriteJsonExit(g.Map{
		"code":    0,
		"message": "调度器已停止",
	})
}

// Status 获取调度器状态
func (c *Controller) Status(r *ghttp.Request) {
	scheduler := service.GetScheduler()

	status := g.Map{
		"appEnv":               service.AppEnv(r.Context()),
		"running":              scheduler.IsRunning(),
		"schedulerEnabled":     service.SchedulerEnabled(r.Context()),
		"manualTriggerEnabled": service.SchedulerManualTriggerEnabled(r.Context()),
		"cycleInterval":        scheduler.GetCycleInterval().String(),
		"pipeline":             "product_rank → video_pool → wait → loop",
		"tasks":                scheduler.GetAllTaskStatus(),
	}

	r.Response.WriteJsonExit(g.Map{
		"code":    0,
		"message": "success",
		"data":    status,
	})
}

// TriggerTask 手动触发任务
func (c *Controller) TriggerTask(r *ghttp.Request) {
	taskType := r.Get("taskType").String()
	if taskType == "" {
		r.Response.WriteJsonExit(g.Map{
			"code":    1,
			"message": "任务类型不能为空",
		})
		return
	}

	if !service.SchedulerManualTriggerEnabled(r.Context()) {
		r.Response.WriteJsonExit(g.Map{
			"code":    1,
			"message": "采集调度器已关闭，无法手动触发",
		})
		return
	}

	scheduler := service.GetScheduler()
	if err := scheduler.TriggerTask(service.TaskType(taskType)); err != nil {
		r.Response.WriteJsonExit(g.Map{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code":    0,
		"message": "任务已触发",
	})
}

// SetInterval 设置任务执行间隔
// taskType 支持: "product_rank" 或 "cycle"（设置轮次间隔）
func (c *Controller) SetInterval(r *ghttp.Request) {
	taskType := r.Get("taskType").String()
	interval := r.Get("interval").Int() // 分钟数

	if taskType == "" {
		r.Response.WriteJsonExit(g.Map{
			"code":    1,
			"message": "任务类型不能为空",
		})
		return
	}

	if interval <= 0 {
		r.Response.WriteJsonExit(g.Map{
			"code":    1,
			"message": "间隔时间必须大于 0",
		})
		return
	}

	scheduler := service.GetScheduler()

	// 设置轮次间隔
	if taskType == "cycle" {
		scheduler.SetCycleInterval(time.Duration(interval) * time.Minute)
		r.Response.WriteJsonExit(g.Map{
			"code":    0,
			"message": "轮次间隔设置成功",
		})
		return
	}

	scheduler.SetTaskInterval(service.TaskType(taskType), time.Duration(interval)*time.Minute)

	r.Response.WriteJsonExit(g.Map{
		"code":    0,
		"message": "间隔设置成功",
	})
}
