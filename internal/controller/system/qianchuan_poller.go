package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-crawler/internal/service"
	"go-crawler/internal/service/platform/oceanengine"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// QianchuanPollerController 千川数据拉取控制器
type QianchuanPollerController struct{}

const qianchuanNeedReauthCode = 46001

func NewQianchuanPollerController() *QianchuanPollerController {
	return &QianchuanPollerController{}
}

func buildQianchuanAuthData(ctx context.Context) g.Map {
	appId := g.Cfg().MustGet(ctx, "platform.qianchuan.appId").String()
	rid := g.Cfg().MustGet(ctx, "platform.qianchuan.rid").String()
	materialAuth := g.Cfg().MustGet(ctx, "platform.qianchuan.materialAuth", 1).String()
	authUrl := fmt.Sprintf("https://qianchuan.jinritemai.com/openapi/qc/audit/oauth.html?app_id=%s&state=from_system&material_auth=%s&rid=%s",
		appId, materialAuth, rid)
	return g.Map{
		"needReauth":   true,
		"popup":        true,
		"authUrl":      authUrl,
		"authRoute":    "/api/qianchuan/auth-url?redirect=1",
		"callbackPath": "/api/douyin/callback",
	}
}

func isQianchuanReauthError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "请重新完成巨量千川OAuth授权") ||
		strings.Contains(msg, "app is not allowed to be renewed") ||
		strings.Contains(msg, "access_token和refresh_token均已过期") ||
		strings.Contains(msg, "access_token已过期且无refresh_token")
}

func writeQianchuanReauthResponse(r *ghttp.Request, msg string) {
	r.Response.WriteStatusExit(http.StatusConflict, g.Map{
		"code": qianchuanNeedReauthCode,
		"data": buildQianchuanAuthData(r.Context()),
		"msg":  msg,
	})
}

// TokenStatus 检查千川授权状态 GET /api/qianchuan/token-status
func (c *QianchuanPollerController) TokenStatus(r *ghttp.Request) {
	row, _ := g.DB().Model("qianchuan_token").Ctx(r.Context()).OrderDesc("id").Limit(1).One()
	hasToken := !row.IsEmpty()
	expired := false
	expiresAt := ""
	refreshExpiresAt := ""
	if hasToken {
		expiresAt = row["expires_at"].String()
		refreshExpiresAt = row["refresh_token_expires_at"].String()
		if expiresAt != "" {
			if t, err := time.ParseInLocation("2006-01-02 15:04:05", expiresAt, time.FixedZone("CST", 8*60*60)); err == nil && time.Now().After(t) {
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

	data := g.Map{
		"hasToken":         hasToken,
		"authorized":       hasToken && !expired,
		"expired":          expired,
		"expiresAt":        expiresAt,
		"refreshExpiresAt": refreshExpiresAt,
		"authUrl":          authUrl,
	}
	if !hasToken || expired {
		for k, v := range buildQianchuanAuthData(r.Context()) {
			data[k] = v
		}
		msg := "千川授权已失效，请重新授权"
		if !hasToken {
			msg = "未找到千川授权信息，请先授权"
		}
		r.Response.WriteStatusExit(http.StatusConflict, g.Map{
			"code": qianchuanNeedReauthCode,
			"data": data,
			"msg":  msg,
		})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": data,
		"msg":  "ok",
	})
}

// AuthUrl 获取巨量千川OAuth授权链接 GET /api/qianchuan/auth-url
func (c *QianchuanPollerController) AuthUrl(r *ghttp.Request) {
	appId := g.Cfg().MustGet(r.Context(), "platform.qianchuan.appId").String()
	rid := g.Cfg().MustGet(r.Context(), "platform.qianchuan.rid").String()
	materialAuth := g.Cfg().MustGet(r.Context(), "platform.qianchuan.materialAuth", 1).String()
	authUrl := fmt.Sprintf("https://qianchuan.jinritemai.com/openapi/qc/audit/oauth.html?app_id=%s&state=from_system&material_auth=%s&rid=%s",
		appId, materialAuth, rid)
	if r.Get("redirect").Int() == 1 {
		r.Response.RedirectTo(authUrl)
		return
	}
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

	// 获取最新可用 token（必要时自动续期）
	token, _, err := oceanengine.GetLatestToken(ctx)
	if err != nil {
		if isQianchuanReauthError(err) {
			writeQianchuanReauthResponse(r, err.Error())
			return
		}
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "获取有效token失败: " + err.Error()})
		return
	}

	// 重新读取最新记录，用于回写 advertiser_ids
	row, err := g.DB().Model("qianchuan_token").Ctx(ctx).OrderDesc("id").Limit(1).One()
	if err != nil || row.IsEmpty() {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "未找到token，请先完成OAuth授权"})
		return
	}
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

	token, err := oceanengine.GetLatestAccessToken(ctx)
	if err != nil {
		if isQianchuanReauthError(err) {
			writeQianchuanReauthResponse(r, err.Error())
			return
		}
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "获取有效token失败: " + err.Error()})
		return
	}

	// 直接用 HTTP 调用千川报表API，打印原始响应
	fieldsJSON, _ := json.Marshal([]string{"stat_cost", "pay_order_count", "create_order_count"})
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

	if startDate != "" && endDate != "" {
		start, startErr := time.Parse("2006-01-02", truncateToDate(startDate))
		end, endErr := time.Parse("2006-01-02", truncateToDate(endDate))
		if startErr != nil || endErr != nil || end.Before(start) {
			msg := "日期格式错误或结束日期早于开始日期"
			r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": msg})
			return
		}
		today := time.Now().Format("2006-01-02")
		todayDate, _ := time.Parse("2006-01-02", today)
		if start.After(todayDate) || end.After(todayDate) {
			msg := fmt.Sprintf("千川数据暂不支持拉取未来日期。当前服务器日期是 %s，请检查前端传参", today)
			r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": msg})
			return
		}
		if int(end.Sub(start).Hours()/24)+1 > 30 {
			msg := "千川日维度单次拉取不能超过30个自然日，请缩小日期范围"
			r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": msg})
			return
		}
	} else {
		days := r.Get("days", 7).Int()
		if days <= 0 {
			days = 1
		}
		if days > 30 {
			msg := "千川日维度单次拉取不能超过30天"
			r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": msg})
			return
		}
	}

	if _, err := oceanengine.GetLatestAccessToken(r.Context()); err != nil {
		if isQianchuanReauthError(err) {
			writeQianchuanReauthResponse(r, err.Error())
			return
		}
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "获取有效token失败: " + err.Error()})
		return
	}

	reqParams, _ := json.Marshal(g.Map{"startDate": startDate, "endDate": endDate, "days": r.Get("days").String()})
	logId, err := service.TaskLogStart(r.Context(), "千川数据手动拉取", "qianchuan_pull", "/api/qianchuan-poller/trigger", string(reqParams))
	if err != nil {
		g.Log().Errorf(r.Context(), "[QianchuanPoller] 写入任务日志失败: %v", err)
	}

	if startDate != "" && endDate != "" {
		g.Log().Infof(r.Context(), "[QianchuanPoller] 收到异步拉取请求: %s~%s, logId=%d, userId=%d", startDate, endDate, logId, userId)
		go c.runQianchuanPullByDate(context.Background(), startDate, endDate, logId, userId)
	} else {
		days := r.Get("days", 7).Int()
		if days <= 0 {
			days = 1
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
	startDate = truncateToDate(startDate)
	endDate = truncateToDate(endDate)

	start, startErr := time.Parse("2006-01-02", startDate)
	end, endErr := time.Parse("2006-01-02", endDate)
	if startErr != nil || endErr != nil || end.Before(start) {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "日期格式错误或结束日期早于开始日期"})
		return
	}

	if _, err := oceanengine.GetLatestAccessToken(r.Context()); err != nil {
		if isQianchuanReauthError(err) {
			writeQianchuanReauthResponse(r, err.Error())
			return
		}
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "获取有效token失败: " + err.Error()})
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
	// 支持前端传入带时分秒的日期（如 "2026-07-13 00:00:00"），截取日期部分
	startDate = truncateToDate(startDate)
	endDate = truncateToDate(endDate)

	query := g.DB().Model("qianchuan_advertiser_daily_report").Ctx(ctx).
		Where("stat_datetime >= ?", startDate).
		Where("stat_datetime <= ?", endDate)

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
		Where("stat_datetime >= ?", startDate).
		Where("stat_datetime <= ?", endDate).
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

// PlanReportList 查询千川计划维度报表 GET /api/qianchuan-poller/plan-report
func (c *QianchuanPollerController) PlanReportList(r *ghttp.Request) {
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()
	advertiserId := r.Get("advertiserId", int64(0)).Int64()
	keyword := strings.TrimSpace(r.Get("keyword").String())
	productId := strings.TrimSpace(r.Get("productId").String())
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 20).Int()
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	// 支持前端传入带时分秒的日期，截取日期部分
	if startDate == "" || endDate == "" {
		today := time.Now().Format("2006-01-02")
		startDate = today
		endDate = today
	}
	start, startErr := time.Parse("2006-01-02", truncateToDate(startDate))
	end, endErr := time.Parse("2006-01-02", truncateToDate(endDate))
	if startErr != nil || endErr != nil || end.Before(start) {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "日期格式错误或结束日期早于开始日期"})
		return
	}

	result, err := service.NewQianchuanPlanHierarchyService().Query(r.Context(), service.QianchuanPlanHierarchyRequest{
		Level:        service.QianchuanHierarchyLevelPlan,
		StartDate:    startDate,
		EndDate:      endDate,
		Keyword:      keyword,
		ProductID:    productId,
		AdvertiserID: advertiserId,
		GroupByDate:  true,
	})
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "查询失败: " + err.Error()})
		return
	}

	planIDs := make(map[int64]struct{})
	productIDs := make(map[string]struct{})
	for _, item := range result.List {
		planIDs[item.AdID] = struct{}{}
		if item.ProductID != "" && item.ProductID != "0" {
			productIDs[item.ProductID] = struct{}{}
		}
	}
	total := len(result.List)
	startIndex := (page - 1) * pageSize
	if startIndex > total {
		startIndex = total
	}
	endIndex := startIndex + pageSize
	if endIndex > total {
		endIndex = total
	}
	rows := make([]g.Map, 0, endIndex-startIndex)
	for _, item := range result.List[startIndex:endIndex] {
		rows = append(rows, g.Map{
			"advertiser_id":      item.AdvertiserID,
			"advertiser_name":    item.AdvertiserName,
			"ad_id":              item.AdID,
			"ad_name":            item.Name,
			"product_id":         item.ProductID,
			"product_name":       item.ProductName,
			"aweme_id":           item.AwemeID,
			"aweme_name":         item.AwemeName,
			"aweme_show_id":      item.AwemeShowID,
			"stat_cost":          item.StatCost,
			"pay_order_count":    item.PayOrderCount,
			"create_order_count": item.CreateOrderCount,
			"stat_datetime":      item.StatDate,
			"show_cnt":           item.ShowCnt,
			"click_cnt":          item.ClickCnt,
			"ctr":                item.Ctr,
			"convert_cnt":        item.ConvertCnt,
			"convert_cost":       item.ConvertCost,
			"valid_order_count":  item.ValidOrderCount,
			"product_quantity":   item.ProductQuantity,
			"pay_amount":         item.PayAmount,
			"cost_amount":        item.CostAmount,
			"commission":         item.Commission,
			"profit":             item.Profit,
		})
	}
	summary := result.Summary
	summaryRow := g.Map{
		"total_order_count":       summary.TotalOrderCount,
		"total_valid_order_count": summary.TotalValidOrderCount,
		"product_quantity":        summary.ProductQuantity,
		"total_pay_amount":        summary.TotalPayAmount,
		"total_cost":              summary.TotalCost,
		"total_order_cost":        summary.TotalOrderCost,
		"total_refund_cost":       summary.TotalRefundCost,
		"reserve_amount":          summary.ReserveAmount,
		"shop_service_fee":        summary.ShopServiceFee,
		"total_commission":        summary.TotalCommission,
		"commission_deduction":    summary.CommissionDeduction,
		"total_profit":            summary.TotalProfit,
		"financial_scope":         summary.FinancialScope,
		"plan_count":              len(planIDs),
		"product_count":           len(productIDs),
	}

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

// PlanHierarchy 查询店铺、抖音号、千川账户和投放计划汇总。
func (c *QianchuanPollerController) PlanHierarchy(r *ghttp.Request) {
	req := service.QianchuanPlanHierarchyRequest{
		Level:        r.Get("level", service.QianchuanHierarchyLevelStore).String(),
		StartDate:    r.Get("startDate").String(),
		EndDate:      r.Get("endDate").String(),
		Keyword:      r.Get("keyword").String(),
		ProductID:    r.Get("productId").String(),
		StoreID:      r.Get("storeId", int64(0)).Int64(),
		AccountID:    r.Get("accountId", int64(0)).Int64(),
		AdvertiserID: r.Get("advertiserId", int64(0)).Int64(),
		AwemeID:      r.Get("awemeId", int64(0)).Int64(),
		AwemeShowID:  r.Get("awemeShowId").String(),
		TalentKey:    r.Get("talentKey").String(),
	}
	result, err := service.NewQianchuanPlanHierarchyService().Query(r.Context(), req)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "查询层级汇总失败: " + err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": result})
}

// PlanAwemeExport 导出全部或搜索结果中的抖音号维度指标。
func (c *QianchuanPollerController) PlanAwemeExport(r *ghttp.Request) {
	level := r.Get("level").String()
	if level != service.QianchuanHierarchyLevelAweme {
		r.Response.WriteStatusExit(http.StatusBadRequest, g.Map{"code": 400, "msg": "只支持抖音号维度数据导出", "data": nil})
		return
	}
	keyword := strings.TrimSpace(r.Get("keyword").String())

	filename, err := service.NewQianchuanPlanHierarchyService().ExportAwemes(r.Context(), service.QianchuanPlanHierarchyRequest{
		Level:     level,
		StartDate: r.Get("startDate").String(),
		EndDate:   r.Get("endDate").String(),
		Keyword:   keyword,
	})
	if err != nil {
		r.Response.WriteStatusExit(http.StatusInternalServerError, g.Map{"code": 500, "msg": "导出失败: " + err.Error(), "data": nil})
		return
	}

	r.Response.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(filename)))
	r.Response.ServeFile(filename)
	defer os.Remove(filename)
}

// OverallReportList 查询千川乘方日报表 GET /api/qianchuan-poller/overall-report
func (c *QianchuanPollerController) OverallReportList(r *ghttp.Request) {
	ctx := r.Context()
	startDate := r.Get("startDate").String()
	endDate := r.Get("endDate").String()
	advertiserId := r.Get("advertiserId", int64(0)).Int64()
	companyId := r.Get("company_id", r.Get("companyId", 0).Int64()).Int64()
	departmentId := r.Get("department_id", r.Get("departmentId", 0).Int64()).Int64()
	traderId := r.Get("trader_id", r.Get("traderId", 0).Int64()).Int64()
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 20).Int()

	if startDate == "" || endDate == "" {
		today := time.Now().Format("2006-01-02")
		startDate = today
		endDate = today
	}
	startDate = truncateToDate(startDate)
	endDate = truncateToDate(endDate)
	start, startErr := time.Parse("2006-01-02", startDate)
	end, endErr := time.Parse("2006-01-02", endDate)
	if startErr != nil || endErr != nil || end.Before(start) {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "日期格式错误或结束日期早于开始日期"})
		return
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	whereSQL := "WHERE LEFT(qopr.stat_datetime, 10) >= ? AND LEFT(qopr.stat_datetime, 10) <= ?"
	args := []interface{}{startDate, endDate}
	if advertiserId > 0 {
		whereSQL += " AND qopr.advertiser_id = ?"
		args = append(args, advertiserId)
	}
	companyScopeIDs, scopeErr := resolveOverallReportCompanyScope(ctx, companyId, departmentId, traderId)
	if scopeErr != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "解析乘方组织范围失败: " + scopeErr.Error()})
		return
	}
	if len(companyScopeIDs) > 0 {
		whereSQL += fmt.Sprintf(" AND qopr.company_id IN (%s)", qianchuanQueryPlaceholders(len(companyScopeIDs)))
		for _, id := range companyScopeIDs {
			args = append(args, id)
		}
	}

	groupSQL := `
		LEFT(qopr.stat_datetime, 10),
		qopr.advertiser_id,
		qopr.account_id,
		qopr.company_id,
		qopr.department_id,
		qopr.trader_id
	`
	countSQL := fmt.Sprintf(`
		SELECT COUNT(1) AS total
		FROM (
			SELECT 1
			FROM qianchuan_overall_promotion_daily_report qopr
			%s
			GROUP BY %s
		) grouped
	`, whereSQL, groupSQL)
	countRow, err := g.DB().Ctx(ctx).GetOne(ctx, countSQL, args...)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询乘方日报总数失败: " + err.Error()})
		return
	}
	total := countRow["total"].Int()

	listSQL := fmt.Sprintf(`
		SELECT
			grouped.id,
			grouped.account_id,
			grouped.company_id,
			grouped.department_id,
			grouped.trader_id,
			grouped.advertiser_id,
			grouped.stat_datetime,
			grouped.adlab_scene,
			grouped.data_period,
			grouped.marketing_goal,
			grouped.order_platform,
			grouped.shop_estimated_comission_cost,
			grouped.stat_cost_for_overall_roi2,
			grouped.stat_cost_for_roi2,
			grouped.total_cost_per_pay_order_for_roi2,
			grouped.total_cost_per_pay_order_settle_for_overall_roi2_1h,
			grouped.total_cost_per_pay_order_settle_for_roi2_1h,
			grouped.total_ecom_platform_subsidy_amount_for_roi2,
			grouped.total_order_real_settle_amount_for_roi2_1h,
			grouped.total_order_settle_amount_for_roi2_1h,
			grouped.total_order_settle_amount_rate_for_roi2_1h,
			grouped.total_order_settle_count_for_roi2_1h,
			grouped.total_order_settle_count_rate_for_roi2_1h,
			grouped.total_pay_order_count_for_roi2,
			grouped.total_pay_order_coupon_amount_for_roi2,
			grouped.total_pay_order_gmv_for_roi2,
			grouped.total_pay_order_gmv_include_coupon_for_roi2,
			grouped.total_prepay_and_pay_order_roi2,
			grouped.total_prepay_and_pay_settle_overall_roi2_1h,
			grouped.total_prepay_and_pay_settle_roi2_1h,
			grouped.total_prepay_order_count_for_roi2,
			grouped.total_prepay_order_gmv_for_roi2,
			grouped.total_refund_order_count_for_roi2_1h,
			grouped.total_refund_order_gmv_for_roi2_1h_all,
			grouped.total_refund_order_gmv_for_roi2_1h_rate,
			grouped.total_unfinished_estimate_order_gmv_for_roi2,
			grouped.no_refund_ecom_coupon_amount_for_roi2,
			grouped.no_refund_ecom_platform_subsidy_amount_for_roi2,
			COALESCE(NULLIF(a.account_name, ''), '') AS account_name,
			COALESCE(NULLIF(sc.name, ''), '') AS company_name,
			COALESCE(NULLIF(sd.name, ''), '') AS department_name,
			COALESCE(NULLIF(t.name, ''), '') AS trader_name
		FROM (
			SELECT
				MIN(qopr.id) AS id,
				qopr.account_id,
				qopr.company_id,
				qopr.department_id,
				qopr.trader_id,
				qopr.advertiser_id,
				LEFT(qopr.stat_datetime, 10) AS stat_datetime,
				MAX(qopr.adlab_scene) AS adlab_scene,
				MAX(qopr.data_period) AS data_period,
				MAX(qopr.marketing_goal) AS marketing_goal,
				MAX(qopr.order_platform) AS order_platform,
				COALESCE(SUM(qopr.shop_estimated_comission_cost), 0) AS shop_estimated_comission_cost,
				COALESCE(SUM(qopr.stat_cost_for_overall_roi2), 0) AS stat_cost_for_overall_roi2,
				COALESCE(SUM(qopr.stat_cost_for_roi2), 0) AS stat_cost_for_roi2,
				CASE WHEN SUM(qopr.total_pay_order_count_for_roi2) > 0
					THEN SUM(qopr.stat_cost_for_roi2) / SUM(qopr.total_pay_order_count_for_roi2)
					ELSE 0 END AS total_cost_per_pay_order_for_roi2,
				CASE WHEN SUM(qopr.total_order_settle_count_for_roi2_1h) > 0
					THEN SUM(qopr.stat_cost_for_overall_roi2) / SUM(qopr.total_order_settle_count_for_roi2_1h)
					ELSE 0 END AS total_cost_per_pay_order_settle_for_overall_roi2_1h,
				CASE WHEN SUM(qopr.total_order_settle_count_for_roi2_1h) > 0
					THEN SUM(qopr.stat_cost_for_roi2) / SUM(qopr.total_order_settle_count_for_roi2_1h)
					ELSE 0 END AS total_cost_per_pay_order_settle_for_roi2_1h,
				COALESCE(SUM(qopr.total_ecom_platform_subsidy_amount_for_roi2), 0) AS total_ecom_platform_subsidy_amount_for_roi2,
				COALESCE(SUM(qopr.total_order_real_settle_amount_for_roi2_1h), 0) AS total_order_real_settle_amount_for_roi2_1h,
				COALESCE(SUM(qopr.total_order_settle_amount_for_roi2_1h), 0) AS total_order_settle_amount_for_roi2_1h,
				0 AS total_order_settle_amount_rate_for_roi2_1h,
				COALESCE(SUM(qopr.total_order_settle_count_for_roi2_1h), 0) AS total_order_settle_count_for_roi2_1h,
				0 AS total_order_settle_count_rate_for_roi2_1h,
				COALESCE(SUM(qopr.total_pay_order_count_for_roi2), 0) AS total_pay_order_count_for_roi2,
				COALESCE(SUM(qopr.total_pay_order_coupon_amount_for_roi2), 0) AS total_pay_order_coupon_amount_for_roi2,
				COALESCE(SUM(qopr.total_pay_order_gmv_for_roi2), 0) AS total_pay_order_gmv_for_roi2,
				COALESCE(SUM(qopr.total_pay_order_gmv_include_coupon_for_roi2), 0) AS total_pay_order_gmv_include_coupon_for_roi2,
				CASE WHEN SUM(qopr.stat_cost_for_roi2) > 0
					THEN SUM(qopr.total_pay_order_gmv_for_roi2) / SUM(qopr.stat_cost_for_roi2)
					ELSE 0 END AS total_prepay_and_pay_order_roi2,
				CASE WHEN SUM(qopr.stat_cost_for_overall_roi2) > 0
					THEN SUM(qopr.total_order_settle_amount_for_roi2_1h) / SUM(qopr.stat_cost_for_overall_roi2)
					ELSE 0 END AS total_prepay_and_pay_settle_overall_roi2_1h,
				CASE WHEN SUM(qopr.stat_cost_for_roi2) > 0
					THEN SUM(qopr.total_order_settle_amount_for_roi2_1h) / SUM(qopr.stat_cost_for_roi2)
					ELSE 0 END AS total_prepay_and_pay_settle_roi2_1h,
				COALESCE(SUM(qopr.total_prepay_order_count_for_roi2), 0) AS total_prepay_order_count_for_roi2,
				COALESCE(SUM(qopr.total_prepay_order_gmv_for_roi2), 0) AS total_prepay_order_gmv_for_roi2,
				COALESCE(SUM(qopr.total_refund_order_count_for_roi2_1h), 0) AS total_refund_order_count_for_roi2_1h,
				COALESCE(SUM(qopr.total_refund_order_gmv_for_roi2_1h_all), 0) AS total_refund_order_gmv_for_roi2_1h_all,
				0 AS total_refund_order_gmv_for_roi2_1h_rate,
				COALESCE(SUM(qopr.total_unfinished_estimate_order_gmv_for_roi2), 0) AS total_unfinished_estimate_order_gmv_for_roi2,
				COALESCE(SUM(qopr.no_refund_ecom_coupon_amount_for_roi2), 0) AS no_refund_ecom_coupon_amount_for_roi2,
				COALESCE(SUM(qopr.no_refund_ecom_platform_subsidy_amount_for_roi2), 0) AS no_refund_ecom_platform_subsidy_amount_for_roi2
			FROM qianchuan_overall_promotion_daily_report qopr
			%s
			GROUP BY %s
		) grouped
		LEFT JOIN account a ON grouped.account_id = a.id
		LEFT JOIN sys_company sc ON grouped.company_id = sc.id
		LEFT JOIN sys_department sd ON grouped.department_id = sd.id
		LEFT JOIN trader t ON grouped.trader_id = t.id
		ORDER BY grouped.stat_datetime DESC, grouped.stat_cost_for_overall_roi2 DESC
		LIMIT ? OFFSET ?
	`, whereSQL, groupSQL)
	listArgs := append([]interface{}{}, args...)
	listArgs = append(listArgs, pageSize, (page-1)*pageSize)
	list, err := g.DB().Ctx(ctx).GetAll(ctx, listSQL, listArgs...)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询乘方日报失败: " + err.Error()})
		return
	}

	summarySQL := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(stat_cost_for_overall_roi2),0) AS overallCost,
			COALESCE(SUM(total_order_settle_amount_for_roi2_1h),0) AS settleAmount,
			COALESCE(SUM(total_order_settle_count_for_roi2_1h),0) AS settleOrderCount
		FROM qianchuan_overall_promotion_daily_report qopr
		%s
	`, whereSQL)
	summaryRow, _ := g.DB().Ctx(ctx).GetOne(ctx, summarySQL, args...)

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

func resolveOverallReportCompanyScope(ctx context.Context, companyId, departmentId, traderId int64) ([]int64, error) {
	if traderId > 0 {
		row, err := g.DB().Model("trader").Ctx(ctx).
			Where("id", traderId).
			Where("status", 1).
			Fields("company_id").
			One()
		if err != nil || row.IsEmpty() {
			return nil, err
		}
		companyId = row["company_id"].Int64()
	} else if departmentId > 0 {
		row, err := g.DB().Model("sys_department").Ctx(ctx).
			Where("id", departmentId).
			Where("status", 1).
			Fields("company_id").
			One()
		if err != nil || row.IsEmpty() {
			return nil, err
		}
		companyId = row["company_id"].Int64()
	}
	if companyId <= 0 {
		return nil, nil
	}
	rootCompanyID, err := resolveOverallReportRootCompany(ctx, companyId)
	if err != nil {
		return nil, err
	}

	rows, err := g.DB().GetAll(ctx, `
		WITH RECURSIVE company_tree AS (
			SELECT id FROM sys_company WHERE id = ? AND status = 1
			UNION ALL
			SELECT child.id
			FROM sys_company child
			INNER JOIN company_tree parent ON child.parent_id = parent.id
			WHERE child.status = 1
		)
		SELECT id FROM company_tree
	`, rootCompanyID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if id := row["id"].Int64(); id > 0 {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func resolveOverallReportRootCompany(ctx context.Context, companyId int64) (int64, error) {
	currentID := companyId
	for currentID > 0 {
		row, err := g.DB().Model("sys_company").Ctx(ctx).
			Where("id", currentID).
			Where("status", 1).
			Fields("id, parent_id").
			One()
		if err != nil || row.IsEmpty() {
			return currentID, err
		}
		parentID := row["parent_id"].Int64()
		if parentID <= 0 {
			return row["id"].Int64(), nil
		}
		currentID = parentID
	}
	return companyId, nil
}

func qianchuanQueryPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
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

	token, err := oceanengine.GetLatestAccessToken(ctx)
	if err != nil {
		if isQianchuanReauthError(err) {
			writeQianchuanReauthResponse(r, err.Error())
			return
		}
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "获取有效token失败: " + err.Error()})
		return
	}

	client := oceanengine.NewClient(token)
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

// truncateToDate 截取日期部分，兼容前端传入带时分秒的日期格式（如 "2026-07-13 00:00:00"）
func truncateToDate(s string) string {
	if len(s) > 10 {
		return s[:10]
	}
	return s
}
