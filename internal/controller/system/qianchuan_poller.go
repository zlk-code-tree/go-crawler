package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"go-crawler/internal/service"
	"go-crawler/internal/service/platform/oceanengine"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// QianchuanPollerController 千川数据拉取控制器
type QianchuanPollerController struct{}

func NewQianchuanPollerController() *QianchuanPollerController {
	return &QianchuanPollerController{}
}

// TokenStatus 检查千川授权状态 GET /api/qianchuan/token-status
func (c *QianchuanPollerController) TokenStatus(r *ghttp.Request) {
	row, _ := g.DB().Model("qianchuan_token").Ctx(r.Context()).OrderDesc("id").Limit(1).One()
	hasToken := !row.IsEmpty()
	expired := false
	expiresAt := ""
	if hasToken {
		expiresAt = row["expires_at"].String()
		if expiresAt != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", expiresAt); err == nil && time.Now().After(t) {
				expired = true
			}
		}
	}

	appId := g.Cfg().MustGet(r.Context(), "platform.qianchuan.appId").String()
	redirectUri := g.Cfg().MustGet(r.Context(), "platform.qianchuan.redirectUri").String()
	scope := "[22000000,26501000,26502000]"
	authUrl := fmt.Sprintf("https://ad.oceanengine.com/openapi/audit/oauth.html?app_id=%s&redirect_uri=%s&scope=%s", appId, url.QueryEscape(redirectUri), url.QueryEscape(scope))

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": g.Map{
			"authorized": hasToken && !expired,
			"expired":    expired,
			"expiresAt":  expiresAt,
			"authUrl":    authUrl,
		},
		"msg": "ok",
	})
}

// AuthUrl 获取巨量千川OAuth授权链接 GET /api/qianchuan/auth-url
func (c *QianchuanPollerController) AuthUrl(r *ghttp.Request) {
	appId := g.Cfg().MustGet(r.Context(), "platform.qianchuan.appId").String()
	redirectUri := g.Cfg().MustGet(r.Context(), "platform.qianchuan.redirectUri").String()
	scope := "[22000000,26501000,26502000]"
	authUrl := fmt.Sprintf("https://ad.oceanengine.com/openapi/audit/oauth.html?app_id=%s&redirect_uri=%s&scope=%s", appId, url.QueryEscape(redirectUri), url.QueryEscape(scope))
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": g.Map{
			"authUrl":      authUrl,
			"appId":        appId,
			"redirectUri":  redirectUri,
			"callbackPath": "/api/douyin/callback",
		},
		"msg": "ok",
	})
}

// Start 启动每日定时拉取 POST /api/qianchuan-poller/start
func (c *QianchuanPollerController) Start(r *ghttp.Request) {
	poller := oceanengine.GetPoller()
	if poller.IsRunning() {
		r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "定时拉取已在运行中"})
		return
	}
	poller.Start(r.Context())
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "定时拉取已启动（每天04:37执行）"})
}

// Stop 停止每日定时拉取 POST /api/qianchuan-poller/stop
func (c *QianchuanPollerController) Stop(r *ghttp.Request) {
	poller := oceanengine.GetPoller()
	if !poller.IsRunning() {
		r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "定时拉取未在运行"})
		return
	}
	poller.Stop()
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "定时拉取已停止"})
}

// Status 查看定时任务状态 GET /api/qianchuan-poller/status
func (c *QianchuanPollerController) Status(r *ghttp.Request) {
	poller := oceanengine.GetPoller()
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": g.Map{
			"running": poller.IsRunning(),
		},
		"msg": "ok",
	})
}

// ManualTrigger 手动拉取千川数据 POST /api/qianchuan-poller/trigger
func (c *QianchuanPollerController) ManualTrigger(r *ghttp.Request) {
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()
	userId := r.GetCtxVar("userId").Int64()

	reqParams, _ := json.Marshal(g.Map{"startDate": startDate, "endDate": endDate, "days": r.Get("days").String()})
	logId, err := service.TaskLogStart(r.Context(), "千川数据手动拉取", "qianchuan_pull", "/api/qianchuan-poller/trigger", string(reqParams))
	if err != nil {
		g.Log().Errorf(r.Context(), "[QianchuanPoller] 写入任务日志失败: %v", err)
	}

	if startDate != "" && endDate != "" {
		g.Log().Infof(r.Context(), "[QianchuanPoller] 收到异步拉取请求: %s~%s, logId=%d, userId=%d", startDate, endDate, logId, userId)
		go c.runQianchuanPullByDate(context.Background(), startDate, endDate, logId, userId)
	} else {
		days := r.Get("days", 1).Int()
		if days <= 0 {
			days = 1
		}
		if days > 90 {
			r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "拉取天数不能超过90天"})
			return
		}
		g.Log().Infof(r.Context(), "[QianchuanPoller] 收到异步拉取请求: days=%d, logId=%d, userId=%d", days, logId, userId)
		go c.runQianchuanPullByDays(context.Background(), days, logId, userId)
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"logId": logId}, "msg": "任务已启动，后台拉取中"})
}

func (c *QianchuanPollerController) runQianchuanPullByDate(ctx context.Context, startDate, endDate string, logId, userId int64) {
	overallStart := time.Now()
	poller := oceanengine.GetPoller()
	result, err := poller.ManualPullByDate(ctx, startDate, endDate)
	durationMs := int(time.Since(overallStart).Milliseconds())

	if err != nil {
		if logId > 0 {
			service.TaskLogFailed(ctx, logId, err.Error(), durationMs)
			service.CreateNotification(ctx, "千川数据拉取失败", `{"error":"`+err.Error()+`"}`, "task_result", "qianchuan_pull", logId, service.NotifyLevelImportant, userId)
		}
		return
	}

	respBody, _ := json.Marshal(g.Map{
		"totalSaved": result.TotalSaved,
		"totalPlans": result.TotalPlans,
		"errors":     result.Errors,
		"durationMs": durationMs,
	})
	level := service.NotifyLevelNormal
	if len(result.Errors) > 0 {
		level = service.NotifyLevelImportant
	}
	if logId > 0 {
		service.TaskLogSuccess(ctx, logId, "", string(respBody), durationMs)
		service.CreateNotification(ctx, "千川数据拉取完成", string(respBody), "task_result", "qianchuan_pull", logId, level, userId)
	}
}

func (c *QianchuanPollerController) runQianchuanPullByDays(ctx context.Context, days int, logId, userId int64) {
	overallStart := time.Now()
	poller := oceanengine.GetPoller()
	result, err := poller.ManualPull(ctx, days)
	durationMs := int(time.Since(overallStart).Milliseconds())

	if err != nil {
		if logId > 0 {
			service.TaskLogFailed(ctx, logId, err.Error(), durationMs)
			service.CreateNotification(ctx, "千川数据拉取失败", `{"error":"`+err.Error()+`"}`, "task_result", "qianchuan_pull", logId, service.NotifyLevelImportant, userId)
		}
		return
	}

	respBody, _ := json.Marshal(g.Map{
		"totalSaved": result.TotalSaved,
		"totalPlans": result.TotalPlans,
		"errors":     result.Errors,
		"durationMs": durationMs,
	})
	level := service.NotifyLevelNormal
	if len(result.Errors) > 0 {
		level = service.NotifyLevelImportant
	}
	if logId > 0 {
		service.TaskLogSuccess(ctx, logId, "", string(respBody), durationMs)
		service.CreateNotification(ctx, "千川数据拉取完成", string(respBody), "task_result", "qianchuan_pull", logId, level, userId)
	}
}
