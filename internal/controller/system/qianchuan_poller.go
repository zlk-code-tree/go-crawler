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

	// 动态生成千川授权链接
	appId := g.Cfg().MustGet(r.Context(), "platform.qianchuan.appId").String()
	rid := g.Cfg().MustGet(r.Context(), "platform.qianchuan.rid").String()
	materialAuth := g.Cfg().MustGet(r.Context(), "platform.qianchuan.materialAuth", 1).String()
	authUrl := fmt.Sprintf("https://qianchuan.jinritemai.com/openapi/qc/audit/oauth.html?app_id=%s&state=from_system&material_auth=%s&rid=%s",
		appId, materialAuth, rid)

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
	rid := g.Cfg().MustGet(r.Context(), "platform.qianchuan.rid").String()
	materialAuth := g.Cfg().MustGet(r.Context(), "platform.qianchuan.materialAuth", 1).String()
	authUrl := fmt.Sprintf("https://qianchuan.jinritemai.com/openapi/qc/audit/oauth.html?app_id=%s&state=from_system&material_auth=%s&rid=%s",
		appId, materialAuth, rid)
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": g.Map{
			"authUrl":      authUrl,
			"callbackPath": "/api/douyin/callback",
		},
		"msg": "ok",
	})
}

// SyncAdvertisers 同步广告主列表 POST /api/qianchuan-poller/sync-advertisers
// 授权后调用此接口，获取千川广告主列表并更新数据库
func (c *QianchuanPollerController) SyncAdvertisers(r *ghttp.Request) {
	ctx := r.Context()

	// 获取最新 token
	row, err := g.DB().Model("qianchuan_token").Ctx(ctx).OrderDesc("id").Limit(1).One()
	if err != nil || row.IsEmpty() {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "未找到token，请先完成OAuth授权"})
		return
	}

	token := row["access_token"].String()
	tokenId := row["id"].Int64()

	// Step1: 调用 oauth2/advertiser/get 获取授权的店铺账户列表
	client := g.Client()
	client.SetHeader("Content-Type", "application/json")
	resp, err := client.Get(ctx, "https://api.oceanengine.com/open_api/oauth2/advertiser/get/", g.Map{
		"access_token": token,
	})
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "调用advertiser/get失败: " + err.Error()})
		return
	}
	defer resp.Close()
	body := resp.ReadAllString()

	var advResp struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
		Data    struct {
			List []struct {
				AdvertiserId   int64  `json:"advertiser_id"`
				AdvertiserName string `json:"advertiser_name"`
			} `json:"list"`
		} `json:"data"`
	}
	json.Unmarshal([]byte(body), &advResp)

	if advResp.Code != 0 {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": g.Map{"raw": body}, "msg": fmt.Sprintf("advertiser/get返回错误: code=%d, msg=%s", advResp.Code, advResp.Message)})
		return
	}

	// Step2: 用授权账号ID作为shop_id，调用 qianchuan/shop/advertiser/list/ 获取千川广告主
	// 注意：必须新建client避免 Content-Type 干扰GET参数传递
	var allAdvIDs []int64
	var allAdvNames []g.Map
	var rawResults []g.Map

	// 将 oauth2/advertiser/get 返回的ID作为 shop_id 尝试
	for _, shop := range advResp.Data.List {
		shopId := shop.AdvertiserId
		g.Log().Infof(ctx, "[SyncAdvertisers] 查询店铺%d下的千川广告主", shopId)

		// 注意：千川shop/advertiser/list API要求通过 Access-Token 请求头传递token
		// 分页循环拉取，避免遗漏大店铺的千川广告主
		page := int64(1)
		pageSize := int64(100)
		shopTotal := 0
		for {
			qcClient := g.Client()
			qcClient.SetHeader("Access-Token", token)
			apiURL := fmt.Sprintf("https://api.oceanengine.com/open_api/v1.0/qianchuan/shop/advertiser/list/?shop_id=%d&page=%d&page_size=%d",
				shopId, page, pageSize)
			advListResp, err := qcClient.Get(ctx, apiURL)
			if err != nil {
				g.Log().Warningf(ctx, "[SyncAdvertisers] 查询店铺%d第%d页失败: %v", shopId, page, err)
				rawResults = append(rawResults, g.Map{"shopId": shopId, "page": page, "error": err.Error()})
				break
			}
			advListBody := advListResp.ReadAllString()
			advListResp.Close()

			var listResp struct {
				Code    int64  `json:"code"`
				Message string `json:"message"`
				Data    struct {
					List     []int64 `json:"list"`
					PageInfo struct {
						TotalNumber int64 `json:"total_number"`
						TotalPage   int64 `json:"total_page"`
					} `json:"page_info"`
				} `json:"data"`
			}
			json.Unmarshal([]byte(advListBody), &listResp)

			if listResp.Code != 0 {
				g.Log().Warningf(ctx, "[SyncAdvertisers] 店铺%d第%d页返回错误: code=%d, msg=%s", shopId, page, listResp.Code, listResp.Message)
				rawResults = append(rawResults, g.Map{"shopId": shopId, "page": page, "code": listResp.Code, "error": listResp.Message})
				break
			}

			for _, advId := range listResp.Data.List {
				allAdvIDs = append(allAdvIDs, advId)
				allAdvNames = append(allAdvNames, g.Map{"id": advId, "shopId": shopId})
			}
			shopTotal += len(listResp.Data.List)

			if listResp.Data.PageInfo.TotalPage <= 0 || page >= listResp.Data.PageInfo.TotalPage {
				break
			}
			page++
		}
		rawResults = append(rawResults, g.Map{"shopId": shopId, "count": shopTotal, "total": shopTotal})
		g.Log().Infof(ctx, "[SyncAdvertisers] 店铺%d: 找到%d个千川广告主(分页完成)", shopId, shopTotal)
	}

	// 去重
	uniqueMap := make(map[int64]bool)
	var uniqueIDs []int64
	for _, id := range allAdvIDs {
		if !uniqueMap[id] {
			uniqueMap[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	// 全失败保护：如果 Step2 全部失败（allAdvIDs 为空），不更新数据库，避免清空已有广告主列表
	if len(uniqueIDs) == 0 {
		failedShops := make([]g.Map, 0, len(rawResults))
		for _, r := range rawResults {
			if e, ok := r["error"]; ok {
				failedShops = append(failedShops, g.Map{"shopId": r["shopId"], "error": e})
			}
		}
		r.Response.WriteJsonExit(g.Map{
			"code": 500,
			"data": g.Map{
				"platformAccounts": advResp.Data.List,
				"failedShops":      failedShops,
				"raw":              rawResults,
			},
			"msg": fmt.Sprintf("所有平台账户解析千川广告主均失败（共%d个），已跳过更新。可能原因：1)应用缺少千川权限 2)平台账户无千川广告主。当前数据库保留原有%d个广告主", len(advResp.Data.List), 0),
		})
		return
	}

	// 更新数据库
	idsJSON, _ := json.Marshal(uniqueIDs)
	g.DB().Model("qianchuan_token").Ctx(ctx).Where("id", tokenId).Data(g.Map{
		"advertiser_ids": string(idsJSON),
	}).Update()

	// 统计失败的shop
	failedCount := 0
	for _, r := range rawResults {
		if _, ok := r["error"]; ok {
			failedCount++
		}
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": g.Map{
			"count":            len(uniqueIDs),
			"advertisers":      allAdvNames,
			"platformAccounts": advResp.Data.List,
			"failedShopCount":  failedCount,
			"raw":              rawResults,
		},
		"msg": fmt.Sprintf("同步完成，共%d个千川广告主（%d个平台账户中%d个解析成功）", len(uniqueIDs), len(advResp.Data.List), len(advResp.Data.List)-failedCount),
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

// DebugToken 调试接口：查看当前token信息和广告主列表 GET /api/qianchuan-poller/debug
func (c *QianchuanPollerController) DebugToken(r *ghttp.Request) {
	ctx := r.Context()

	// 1. 从数据库读取 token 信息
	row, err := g.DB().Model("qianchuan_token").Ctx(ctx).OrderDesc("id").Limit(1).One()
	if err != nil || row.IsEmpty() {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "未找到token"})
		return
	}

	token := row["access_token"].String()
	dbAdvIds := row["advertiser_ids"].String()
	expiresAt := row["expires_at"].String()
	refreshToken := row["refresh_token"].String()
	refreshExpiresAt := row["refresh_token_expires_at"].String()

	// 2. 调用 advertiser/get API 看实际返回什么
	var apiResult interface{}
	client := g.Client()
	client.SetHeader("Content-Type", "application/json")
	resp, err := client.Get(ctx, "https://api.oceanengine.com/open_api/oauth2/advertiser/get/", g.Map{
		"access_token": token,
	})
	if err != nil {
		apiResult = map[string]interface{}{"error": err.Error()}
	} else {
		defer resp.Close()
		body := resp.ReadAllString()
		var advResp map[string]interface{}
		json.Unmarshal([]byte(body), &advResp)
		apiResult = advResp
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": g.Map{
			"tokenPrefix": token[:min(8, len(token))],
			"expiresAt":   expiresAt,
			"refreshTokenPrefix": func() string {
				if len(refreshToken) > 8 {
					return refreshToken[:8]
				}
				return refreshToken
			}(),
			"refreshExpiresAt": refreshExpiresAt,
			"dbAdvertiserIds":  dbAdvIds,
			"apiAdvertiserGet": apiResult,
		},
		"msg": "ok",
	})
}

// TestReportAPI 测试报表API直接调用 GET /api/qianchuan-poller/test-report
func (c *QianchuanPollerController) TestReportAPI(r *ghttp.Request) {
	ctx := r.Context()
	advID := r.Get("advertiserId", int64(1705855590344712)).Int64()
	startDate := r.Get("startDate", "2026-06-20").String()
	endDate := r.Get("endDate", "2026-06-25").String()

	// 获取token
	row, _ := g.DB().Model("qianchuan_token").Ctx(ctx).OrderDesc("id").Limit(1).One()
	if row.IsEmpty() {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "无token"})
		return
	}
	token := row["access_token"].String()

	// 直接用 HTTP 调用千川报表API，打印原始响应
	fieldsJSON, _ := json.Marshal([]string{"stat_cost"})
	filteringJSON, _ := json.Marshal(map[string]string{"marketing_goal": "ALL"})
	apiURL := fmt.Sprintf("https://api.oceanengine.com/open_api/v1.0/qianchuan/report/ad/get/?advertiser_id=%d&start_date=%s&end_date=%s&time_granularity=TIME_GRANULARITY_DAILY&page=1&page_size=10&fields=%s&filtering=%s",
		advID, startDate, endDate, url.QueryEscape(string(fieldsJSON)), url.QueryEscape(string(filteringJSON)))

	client := g.Client()
	client.SetHeader("Access-Token", token)
	resp, err := client.Get(ctx, apiURL)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": err.Error()})
		return
	}
	body := resp.ReadAllString()
	resp.Close()

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": g.Map{
			"url":     apiURL,
			"advid":   advID,
			"dates":   startDate + "~" + endDate,
			"rawBody": body,
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
		start, startErr := time.Parse("2006-01-02", startDate)
		end, endErr := time.Parse("2006-01-02", endDate)
		if startErr != nil || endErr != nil || end.Before(start) {
			msg := "日期格式错误或结束日期早于开始日期"
			if logId > 0 {
				service.TaskLogFailed(r.Context(), logId, msg, 0)
			}
			r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": msg})
			return
		}
		today := time.Now().Format("2006-01-02")
		todayDate, _ := time.Parse("2006-01-02", today)
		if start.After(todayDate) || end.After(todayDate) {
			msg := fmt.Sprintf("千川数据暂不支持拉取未来日期。当前服务器日期是 %s，请检查前端传参", today)
			if logId > 0 {
				service.TaskLogFailed(r.Context(), logId, msg, 0)
			}
			r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": msg})
			return
		}
		if int(end.Sub(start).Hours()/24)+1 > 30 {
			msg := "千川日维度单次拉取不能超过30个自然日，请缩小日期范围"
			if logId > 0 {
				service.TaskLogFailed(r.Context(), logId, msg, 0)
			}
			r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": msg})
			return
		}
		g.Log().Infof(r.Context(), "[QianchuanPoller] 收到异步拉取请求: %s~%s, logId=%d, userId=%d", startDate, endDate, logId, userId)
		go c.runQianchuanPullByDate(context.Background(), startDate, endDate, logId, userId)
	} else {
		days := r.Get("days", 7).Int()
		if days <= 0 {
			days = 1
		}
		if days > 30 {
			msg := "千川日维度单次拉取不能超过30天"
			if logId > 0 {
				service.TaskLogFailed(r.Context(), logId, msg, 0)
			}
			r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": msg})
			return
		}
		g.Log().Infof(r.Context(), "[QianchuanPoller] 收到异步拉取请求: days=%d, logId=%d, userId=%d", days, logId, userId)
		go c.runQianchuanPullByDays(context.Background(), days, logId, userId)
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"logId": logId}, "msg": "任务已启动，后台拉取中"})
}

// ManualTriggerOverall 手动拉取乘方数据 POST /api/qianchuan-poller/trigger-overall
func (c *QianchuanPollerController) ManualTriggerOverall(r *ghttp.Request) {
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()
	userId := r.GetCtxVar("userId").Int64()

	if startDate == "" || endDate == "" {
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		startDate = yesterday
		endDate = yesterday
	}

	start, startErr := time.Parse("2006-01-02", startDate)
	end, endErr := time.Parse("2006-01-02", endDate)
	if startErr != nil || endErr != nil || end.Before(start) {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "日期格式错误或结束日期早于开始日期"})
		return
	}

	reqParams, _ := json.Marshal(g.Map{"startDate": startDate, "endDate": endDate})
	logId, err := service.TaskLogStart(r.Context(), "千川乘方数据手动拉取", "qianchuan_overall_pull", "/api/qianchuan-poller/trigger-overall", string(reqParams))
	if err != nil {
		g.Log().Errorf(r.Context(), "[QianchuanPoller] 写入乘方任务日志失败: %v", err)
	}

	g.Log().Infof(r.Context(), "[QianchuanPoller] 收到乘方异步拉取请求: %s~%s, logId=%d, userId=%d", startDate, endDate, logId, userId)
	go c.runQianchuanOverallPullByDate(context.Background(), startDate, endDate, logId, userId)

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"logId": logId}, "msg": "乘方任务已启动，后台拉取中"})
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
		"totalSaved":        result.TotalSaved,
		"totalOverallSaved": result.TotalOverallSaved,
		"totalPlans":        result.TotalPlans,
		"totalPlanReports":  result.TotalPlanReports,
		"totalPlanSaved":    result.TotalPlanSaved,
		"totalFailed":       result.TotalFailed,
		"errors":            result.Errors,
		"durationMs":        durationMs,
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
		"totalSaved":        result.TotalSaved,
		"totalOverallSaved": result.TotalOverallSaved,
		"totalPlans":        result.TotalPlans,
		"totalPlanReports":  result.TotalPlanReports,
		"totalPlanSaved":    result.TotalPlanSaved,
		"totalFailed":       result.TotalFailed,
		"errors":            result.Errors,
		"durationMs":        durationMs,
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

func (c *QianchuanPollerController) runQianchuanOverallPullByDate(ctx context.Context, startDate, endDate string, logId, userId int64) {
	overallStart := time.Now()
	poller := oceanengine.GetPoller()
	result, err := poller.ManualPullOverallByDate(ctx, startDate, endDate)
	durationMs := int(time.Since(overallStart).Milliseconds())

	if err != nil {
		if logId > 0 {
			service.TaskLogFailed(ctx, logId, err.Error(), durationMs)
			service.CreateNotification(ctx, "千川乘方数据拉取失败", `{"error":"`+err.Error()+`"}`, "task_result", "qianchuan_overall_pull", logId, service.NotifyLevelImportant, userId)
		}
		return
	}

	respBody, _ := json.Marshal(g.Map{
		"totalSaved":        result.TotalSaved,
		"totalOverallSaved": result.TotalOverallSaved,
		"totalFailed":       result.TotalFailed,
		"errors":            result.Errors,
		"durationMs":        durationMs,
	})
	level := service.NotifyLevelNormal
	if len(result.Errors) > 0 {
		level = service.NotifyLevelImportant
	}
	if logId > 0 {
		service.TaskLogSuccess(ctx, logId, "", string(respBody), durationMs)
		service.CreateNotification(ctx, "千川乘方数据拉取完成", string(respBody), "task_result", "qianchuan_overall_pull", logId, level, userId)
	}
}

// DailyReportList 查询千川日结报表 GET /api/qianchuan-poller/daily-report
func (c *QianchuanPollerController) DailyReportList(r *ghttp.Request) {
	ctx := r.Context()
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()
	advertiserId := r.Get("advertiserId", int64(0)).Int64()
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 20).Int()

	if startDate == "" || endDate == "" {
		today := time.Now().Format("2006-01-02")
		startDate = today
		endDate = today
	}

	query := g.DB().Model("qianchuan_advertiser_daily_report").Ctx(ctx).
		Where("LEFT(stat_datetime, 10) >= ?", startDate).
		Where("LEFT(stat_datetime, 10) <= ?", endDate)

	if advertiserId > 0 {
		query = query.Where("advertiser_id", advertiserId)
	}

	totalRow, _ := query.Clone().Fields("COUNT(*) as total").One()
	total := totalRow["total"].Int64()

	offset := (page - 1) * pageSize
	rows, err := query.Clone().
		Order("stat_datetime DESC, advertiser_id ASC").
		Limit(pageSize).
		Offset(offset).
		All()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "查询失败: " + err.Error()})
		return
	}

	summaryRow, _ := g.DB().Model("qianchuan_advertiser_daily_report").Ctx(ctx).
		Where("LEFT(stat_datetime, 10) >= ?", startDate).
		Where("LEFT(stat_datetime, 10) <= ?", endDate).
		Fields(`
			COALESCE(SUM(stat_cost),0) as total_cost,
			COALESCE(SUM(pay_order_amount),0) as total_pay_amount,
			COALESCE(SUM(pay_order_count),0) as total_pay_count,
			COALESCE(SUM(create_order_amount),0) as total_create_amount,
			COALESCE(SUM(create_order_count),0) as total_create_count,
			COALESCE(SUM(all_order_refund_amount_7d),0) as total_refund_7d,
			COALESCE(SUM(all_order_refund_count_7d),0) as total_refund_count_7d,
			COUNT(DISTINCT advertiser_id) as adv_count
		`).One()

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": g.Map{
			"list":      rows,
			"total":     total,
			"page":      page,
			"pageSize":  pageSize,
			"summary":   summaryRow,
			"startDate": startDate,
			"endDate":   endDate,
		},
		"msg": "ok",
	})
}

// OverallReportList 查询千川乘方日报表 GET /api/qianchuan-poller/overall-report
func (c *QianchuanPollerController) OverallReportList(r *ghttp.Request) {
	ctx := r.Context()
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()
	advertiserId := r.Get("advertiserId", int64(0)).Int64()
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 20).Int()

	if startDate == "" || endDate == "" {
		today := time.Now().Format("2006-01-02")
		startDate = today
		endDate = today
	}

	query := g.DB().Model("qianchuan_overall_promotion_daily_report qopr").Ctx(ctx).
		Where("LEFT(qopr.stat_datetime, 10) >= ?", startDate).
		Where("LEFT(qopr.stat_datetime, 10) <= ?", endDate)

	if advertiserId > 0 {
		query = query.Where("qopr.advertiser_id", advertiserId)
	}

	total, err := query.Count()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询乘方日报总数失败: " + err.Error()})
		return
	}

	list, err := query.
		Fields(`
			qopr.*,
			COALESCE(NULLIF(a.account_name, ''), '') AS account_name,
			COALESCE(NULLIF(sc.name, ''), '') AS company_name,
			COALESCE(NULLIF(sd.name, ''), '') AS department_name,
			COALESCE(NULLIF(t.name, ''), '') AS trader_name
		`).
		LeftJoin("account a", "qopr.account_id = a.id").
		LeftJoin("sys_company sc", "qopr.company_id = sc.id").
		LeftJoin("sys_department sd", "qopr.department_id = sd.id").
		LeftJoin("trader t", "qopr.trader_id = t.id").
		OrderDesc("stat_datetime").
		OrderDesc("stat_cost_for_overall_roi2").
		Page(page, pageSize).
		All()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询乘方日报失败: " + err.Error()})
		return
	}

	summaryQuery := g.DB().Model("qianchuan_overall_promotion_daily_report").Ctx(ctx).
		Fields(`
			COALESCE(SUM(stat_cost_for_overall_roi2),0) AS overallCost,
			COALESCE(SUM(total_order_settle_amount_for_roi2_1h),0) AS settleAmount,
			COALESCE(SUM(total_order_settle_count_for_roi2_1h),0) AS settleOrderCount
		`).
		Where("LEFT(stat_datetime, 10) >= ?", startDate).
		Where("LEFT(stat_datetime, 10) <= ?", endDate)
	if advertiserId > 0 {
		summaryQuery = summaryQuery.Where("advertiser_id", advertiserId)
	}
	summaryRow, _ := summaryQuery.One()

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": g.Map{
			"list": list,
			"pagination": g.Map{
				"page":     page,
				"pageSize": pageSize,
				"total":    total,
			},
			"summary": g.Map{
				"overallCost":      summaryRow["overallCost"],
				"settleAmount":     summaryRow["settleAmount"],
				"settleOrderCount": summaryRow["settleOrderCount"],
			},
		},
		"msg": "ok",
	})
}

// OverallReportDetailList 查询千川乘方明细（达人维度） GET /api/qianchuan-poller/overall-report-detail
func (c *QianchuanPollerController) OverallReportDetailList(r *ghttp.Request) {
	ctx := r.Context()
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()
	advertiserId := r.Get("advertiserId").Int64()
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 20).Int()
	anchorShowId := r.Get("anchorShowId").String()

	if advertiserId <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "advertiserId不能为空"})
		return
	}
	if startDate == "" || endDate == "" {
		today := time.Now().Format("2006-01-02")
		startDate = today
		endDate = today
	}

	row, err := g.DB().Model("qianchuan_token").Ctx(ctx).OrderDesc("id").Limit(1).One()
	if err != nil || row.IsEmpty() {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "未找到有效token，请先完成千川授权"})
		return
	}

	client := oceanengine.NewClient(row["access_token"].String())
	result, err := client.GetOverallProductAwemeReport(ctx, advertiserId, startDate, endDate, page, pageSize, anchorShowId)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询乘方抖音号明细失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": g.Map{
			"list": result.List,
			"pagination": g.Map{
				"page":     result.Page,
				"pageSize": result.PageSize,
				"total":    result.Total,
			},
		},
		"msg": "ok",
	})
}

// DailyReportAdvertisers 获取有日结数据的广告主列表 GET /api/qianchuan-poller/daily-report-advertisers
func (c *QianchuanPollerController) DailyReportAdvertisers(r *ghttp.Request) {
	ctx := r.Context()
	rows, err := g.DB().Model("qianchuan_advertiser_daily_report").Ctx(ctx).
		Fields("advertiser_id, COUNT(*) as days, MIN(stat_datetime) as first_date, MAX(stat_datetime) as last_date").
		Group("advertiser_id").
		Order("advertiser_id ASC").
		All()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "查询失败"})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": rows, "msg": "ok"})
}
