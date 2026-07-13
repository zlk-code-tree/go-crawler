package system

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-crawler/internal/service"
	douyinSvc "go-crawler/internal/service/platform/douyin"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type DouyinPollerController struct{}

func NewDouyinPollerController() *DouyinPollerController {
	return &DouyinPollerController{}
}

// Start 启动抖店订单轮询 POST /api/douyin-poller/start
func (c *DouyinPollerController) Start(r *ghttp.Request) {
	poller := douyinSvc.GetPoller()
	if poller.IsRunning() {
		r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "轮询器已在运行"})
		return
	}
	if err := poller.Start(r.Context()); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "启动失败: " + err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "轮询器已启动"})
}

// Stop 停止轮询 POST /api/douyin-poller/stop
func (c *DouyinPollerController) Stop(r *ghttp.Request) {
	poller := douyinSvc.GetPoller()
	if !poller.IsRunning() {
		r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "轮询器未在运行"})
		return
	}
	poller.Stop()
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "轮询器已停止"})
}

// Status 获取轮询器状态 GET /api/douyin-poller/status
func (c *DouyinPollerController) Status(r *ghttp.Request) {
	poller := douyinSvc.GetPoller()
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": poller.Stats(),
		"msg":  "ok",
	})
}

// TriggerOrder 手动触发订单、精选联盟和达人匹配 POST /api/douyin-poller/trigger-order
func (c *DouyinPollerController) TriggerOrder(r *ghttp.Request) {
	opt := douyinSvc.PollParams{
		StartTime:     r.Get("startTime").String(),
		EndTime:       r.Get("endTime").String(),
		StartPage:     r.Get("startPage", 0).Int(),
		Concurrency:   r.Get("concurrency", 5).Int(),
		UseUpdateTime: r.Get("useUpdateTime", false).Bool(),
	}

	reqParams, _ := json.Marshal(g.Map{
		"startTime":     opt.StartTime,
		"endTime":       opt.EndTime,
		"startPage":     opt.StartPage,
		"concurrency":   opt.Concurrency,
		"useUpdateTime": opt.UseUpdateTime,
	})
	logID, err := service.TaskLogStart(r.Context(), "抖店订单及达人匹配", "douyin_order_pull", "/api/douyin-poller/trigger-order", string(reqParams))
	if err != nil {
		g.Log().Errorf(r.Context(), "[DouyinPoller] 写入任务日志失败: %v", err)
	}

	g.Log().Infof(r.Context(), "[DouyinPoller] 收到订单及达人匹配请求: startTime=%s endTime=%s logId=%d",
		opt.StartTime, opt.EndTime, logID)
	go c.runOrderAllianceSync(context.Background(), opt, logID, r.GetCtxVar("userId").Int64())
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"logId": logID}, "msg": "任务已启动，订单拉取完成后将自动匹配达人"})
}

func (c *DouyinPollerController) runOrderAllianceSync(ctx context.Context, opt douyinSvc.PollParams, logID, userID int64) {
	startedAt := time.Now()
	result, err := douyinSvc.NewOrderAllianceSyncService().Sync(ctx, opt)
	durationMs := int(time.Since(startedAt).Milliseconds())
	respBody, _ := json.Marshal(result)
	if err != nil {
		if logID > 0 {
			service.TaskLogFailed(ctx, logID, err.Error(), durationMs)
			service.CreateNotification(ctx, "抖店订单及达人匹配部分失败",
				fmt.Sprintf(`{"error":%q,"result":%s}`, err.Error(), respBody),
				"task_result", "douyin_order_pull", logID, service.NotifyLevelImportant, userID)
		}
		return
	}
	if logID > 0 {
		service.TaskLogSuccess(ctx, logID, "", string(respBody), durationMs)
		service.CreateNotification(ctx, "抖店订单及达人匹配完成", string(respBody),
			"task_result", "douyin_order_pull", logID, service.NotifyLevelNormal, userID)
	}
}

// QueryShopAllianceOrder 透传调用 /buyin/queryShopAllianceOrder GET /api/douyin/query-shop-alliance-order
func (c *DouyinPollerController) QueryShopAllianceOrder(r *ghttp.Request) {
	svc := douyinSvc.NewService()
	token, _, err := svc.GetTokenForStore(r.Context(), r.Get("storeId", 1).Int())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "获取token失败: " + err.Error()})
		return
	}

	resp, err := svc.QueryShopAllianceOrder(r.Context(), token, douyinSvc.AllianceOrderQuery{
		Size:      r.Get("size", 20).Int64(),
		Cursor:    r.Get("cursor").String(),
		StartTime: r.Get("startTime").String(),
		EndTime:   r.Get("endTime").String(),
		OrderIds:  r.Get("orderIds").String(),
		TimeType:  r.Get("timeType").String(),
	})
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "调用失败: " + err.Error()})
		return
	}
	r.Response.Header().Set("Content-Type", "application/json")
	r.Response.WriteExit(resp)
}

// TriggerAlliance 手工补拉指定时间范围的精选联盟数据并匹配达人。
func (c *DouyinPollerController) TriggerAlliance(r *ghttp.Request) {
	startTime := r.Get("startTime").String()
	endTime := r.Get("endTime").String()
	if startTime == "" || endTime == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "startTime和endTime不能为空"})
		return
	}

	concurrency := r.Get("concurrency", g.Cfg().MustGet(r.Context(), "task.concurrency", 10).Int()).Int()
	if concurrency <= 0 {
		concurrency = 10
	}
	storeCount, err := g.DB().Model("store").Ctx(r.Context()).
		Where("platform", "douyin").Where("status", 1).Where("author", 1).Count()
	if err != nil || storeCount == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "没有启用且已授权的抖音店铺"})
		return
	}

	reqParams, _ := json.Marshal(g.Map{
		"startTime": startTime, "endTime": endTime, "storeCount": storeCount, "concurrency": concurrency,
	})
	logID, err := service.TaskLogStart(r.Context(), "精选联盟手动拉取", "alliance_manual_pull", "/api/douyin-poller/trigger-alliance", string(reqParams))
	if err != nil {
		g.Log().Errorf(r.Context(), "[AlliancePoller] 写入任务日志失败: %v", err)
	}

	go c.runAlliancePullByTime(context.Background(), startTime, endTime, concurrency, logID, r.GetCtxVar("userId").Int64())
	r.Response.WriteJsonExit(g.Map{
		"code": 0, "data": g.Map{"logId": logID, "storeCount": storeCount}, "msg": "任务已启动，联盟拉取完成后将自动匹配达人",
	})
}

func (c *DouyinPollerController) runAlliancePullByTime(ctx context.Context, startTime, endTime string, concurrency int, logID, userID int64) {
	startedAt := time.Now()
	result, err := douyinSvc.NewOrderAllianceSyncService().PullAllianceByTime(ctx, startTime, endTime, concurrency)
	durationMs := int(time.Since(startedAt).Milliseconds())
	respBody, _ := json.Marshal(result)
	if err != nil {
		if logID > 0 {
			service.TaskLogFailed(ctx, logID, err.Error(), durationMs)
			service.CreateNotification(ctx, "精选联盟拉取部分失败",
				fmt.Sprintf(`{"error":%q,"result":%s}`, err.Error(), respBody),
				"task_result", "alliance_manual_pull", logID, service.NotifyLevelImportant, userID)
		}
		return
	}
	if logID > 0 {
		service.TaskLogSuccess(ctx, logID, "", string(respBody), durationMs)
		service.CreateNotification(ctx, "精选联盟及达人匹配完成", string(respBody),
			"task_result", "alliance_manual_pull", logID, service.NotifyLevelNormal, userID)
	}
}
