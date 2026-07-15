package system

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"go-crawler/internal/service/platform/oceanengine"
)

// DouyinCallbackController 抖音/巨量千川 OAuth 回调控制器
type DouyinCallbackController struct{}

func NewDouyinCallbackController() *DouyinCallbackController {
	return &DouyinCallbackController{}
}

// Callback 巨量千川 OAuth 授权回调
// GET /api/douyin/callback?auth_code=xxx&state=xxx
func (c *DouyinCallbackController) Callback(r *ghttp.Request) {
	ctx := r.Context()
	authCode := r.GetQuery("auth_code").String()

	if authCode == "" {
		r.Response.Write(`<html><body><h2>授权失败</h2><p>缺少 auth_code 参数</p></body></html>`)
		return
	}

	g.Log().Infof(ctx, "[QianchuanCallback] 收到OAuth回调, auth_code=%s", authCode)

	// Step0: auth_code 换 token
	tokenResp, err := oceanengine.ExchangeToken(ctx, authCode)
	if err != nil {
		g.Log().Errorf(ctx, "[QianchuanCallback] 换取token失败: %v", err)
		r.Response.Writef(`<html><body><h2>授权失败</h2><p>换取access_token失败: %s</p></body></html>`, err.Error())
		return
	}

	accessToken := *tokenResp.Data.AccessToken
	advertiserIDs := tokenResp.Data.AdvertiserIds

	g.Log().Infof(ctx, "[QianchuanCallback] token换取成功, 原始授权账户IDs=%v, 共%d个", advertiserIDs, len(advertiserIDs))

	// 将原始授权账户ID解析为千川广告主ID（原始IDs可能是巨量引擎平台账户ID，需要解析）
	resolvedIDs, resolveErr := oceanengine.ResolveQianchuanAdvertiserIDs(ctx, accessToken, advertiserIDs)
	if resolveErr != nil {
		g.Log().Warningf(ctx, "[QianchuanCallback] 解析千川广告主ID失败，使用原始IDs: %v", resolveErr)
	} else if len(resolvedIDs) > 0 {
		advertiserIDs = resolvedIDs
		g.Log().Infof(ctx, "[QianchuanCallback] 千川广告主IDs解析成功: %v (共%d个)", resolvedIDs, len(resolvedIDs))
	} else {
		g.Log().Warningf(ctx, "[QianchuanCallback] 解析千川广告主ID返回空列表，使用原始IDs")
	}

	// 保存 token 信息
	saveTokenRecord(ctx, accessToken, tokenResp)

	// Step1-3: 遍历每个千川广告主，拉取数据
	client := oceanengine.NewClient(accessToken)
	pullDays := g.Cfg().MustGet(ctx, "platform.qianchuan.pullDays", 30).Int()

	type accountResult struct {
		AdvertiserID     int64                        `json:"advertiser_id"`
		AdvertiserName   string                       `json:"advertiser_name"`
		TotalCombined    int                          `json:"total_combined"`
		TotalPlanReports int                          `json:"total_plan_reports"`
		Saved            int                          `json:"saved"`
		PlanSaved        int                          `json:"plan_saved"`
		Combined         []oceanengine.CombinedReport `json:"combined,omitempty"`
		Error            string                       `json:"error,omitempty"`
	}

	var results []accountResult
	totalSaved := 0

	for _, advID := range advertiserIDs {
		ar := accountResult{AdvertiserID: advID}

		// 校验广告主是否活跃（外部/无效账户跳过拉取）
		if !oceanengine.IsAdvertiserActive(ctx, advID) {
			g.Log().Infof(ctx, "[QianchuanCallback] 广告主%d标记为非活跃，跳过拉取", advID)
			continue
		}

		pullResult, err := client.PullAll(ctx, advID, pullDays)
		if err != nil {
			ar.Error = err.Error()
			g.Log().Errorf(ctx, "[QianchuanCallback] 拉取广告主%d数据失败: %v", advID, err)
			results = append(results, ar)
			continue
		}

		ar.AdvertiserName = pullResult.AdvertiserName
		ar.TotalCombined = pullResult.TotalCombined
		ar.TotalPlanReports = pullResult.TotalPlanReports

		// 入库（写入账户维度日结报表）
		saved := 0
		for _, item := range pullResult.Combined {
			_, err := g.DB().Ctx(ctx).Exec(ctx,
				`INSERT INTO qianchuan_advertiser_daily_report
					(advertiser_id, stat_datetime,
					stat_cost, show_cnt, click_cnt, ctr, cpc_platform, cpm_platform,
					convert_cnt, convert_cost, convert_rate, deep_convert_cnt, deep_convert_cost,
					pay_order_amount, pay_order_count, pay_order_cost_per_order, pay_order_coupon_amount,
					create_order_amount, create_order_count, create_order_coupon_amount, create_order_roi,
					prepay_order_amount, prepay_order_count, prepay_and_pay_order_roi,
					all_order_pay_gmv_7days, all_order_pay_count_7days, all_order_pay_roi_7days,
					all_order_refund_amount_7d, all_order_refund_count_7d,
					all_order_refund_amount_14d, all_order_refund_count_14d,
					dy_follow, total_play)
				VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
				ON DUPLICATE KEY UPDATE
					stat_cost=VALUES(stat_cost), show_cnt=VALUES(show_cnt), click_cnt=VALUES(click_cnt),
					ctr=VALUES(ctr), cpc_platform=VALUES(cpc_platform), cpm_platform=VALUES(cpm_platform),
					convert_cnt=VALUES(convert_cnt), convert_cost=VALUES(convert_cost), convert_rate=VALUES(convert_rate),
					deep_convert_cnt=VALUES(deep_convert_cnt), deep_convert_cost=VALUES(deep_convert_cost),
					pay_order_amount=VALUES(pay_order_amount), pay_order_count=VALUES(pay_order_count),
					pay_order_cost_per_order=VALUES(pay_order_cost_per_order), pay_order_coupon_amount=VALUES(pay_order_coupon_amount),
					create_order_amount=VALUES(create_order_amount), create_order_count=VALUES(create_order_count),
					create_order_coupon_amount=VALUES(create_order_coupon_amount), create_order_roi=VALUES(create_order_roi),
					prepay_order_amount=VALUES(prepay_order_amount), prepay_order_count=VALUES(prepay_order_count),
					prepay_and_pay_order_roi=VALUES(prepay_and_pay_order_roi),
					all_order_pay_gmv_7days=VALUES(all_order_pay_gmv_7days), all_order_pay_count_7days=VALUES(all_order_pay_count_7days),
					all_order_pay_roi_7days=VALUES(all_order_pay_roi_7days),
					all_order_refund_amount_7d=VALUES(all_order_refund_amount_7d), all_order_refund_count_7d=VALUES(all_order_refund_count_7d),
					all_order_refund_amount_14d=VALUES(all_order_refund_amount_14d), all_order_refund_count_14d=VALUES(all_order_refund_count_14d),
					dy_follow=VALUES(dy_follow), total_play=VALUES(total_play)`,
				item.AdvertiserID, item.StatDatetime,
				item.StatCost, item.ShowCnt, item.ClickCnt, item.Ctr, item.CpcPlatform, item.CpmPlatform,
				item.ConvertCnt, item.ConvertCost, item.ConvertRate, item.DeepConvertCnt, item.DeepConvertCost,
				item.PayOrderAmount, item.PayOrderCount, item.PayOrderCostPerOrder, item.PayOrderCouponAmount,
				item.CreateOrderAmount, item.CreateOrderCount, item.CreateOrderCouponAmount, item.CreateOrderRoi,
				item.PrepayOrderAmount, item.PrepayOrderCount, item.PrepayAndPayOrderRoi,
				item.AllOrderPayGmv7days, item.AllOrderPayCount7days, item.AllOrderPayRoi7days,
				item.AllOrderRefundAmount7d, item.AllOrderRefundCount7d,
				item.AllOrderRefundAmount14d, item.AllOrderRefundCount14d,
				item.DyFollow, item.TotalPlay)
			if err != nil {
				g.Log().Errorf(ctx, "[QianchuanCallback] 入库失败 advID=%d date=%s: %v", advID, item.StatDatetime, err)
				continue
			}
			saved++
		}
		ar.Saved = saved
		totalSaved += saved

		planSaved, planFailed := oceanengine.SavePlanReports(ctx, pullResult.PlanReports)
		ar.PlanSaved = planSaved
		totalSaved += planSaved
		if planFailed > 0 {
			g.Log().Warningf(ctx, "[QianchuanCallback] 广告主%d计划明细入库失败%d条", advID, planFailed)
		}

		g.Log().Infof(ctx, "[QianchuanCallback] 广告主%d(%s): 账户报表%d条/入库%d条, 计划报表%d条/入库%d条",
			advID, pullResult.AdvertiserName, pullResult.TotalCombined, saved, pullResult.TotalPlanReports, planSaved)

		results = append(results, ar)
	}

	// 记录回调日志
	logCallbackReq(ctx, authCode, advertiserIDs, results)

	g.Log().Infof(ctx, "[QianchuanCallback] 授权完成: 账户数=%d, 入库=%d条", len(results), totalSaved)

	// 重定向到系统首页
	// HTTP 302 不支持 # 片段，用 JS 跳转
	r.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	r.Response.Write(`<!DOCTYPE html><html><head><meta charset="utf-8"><script>location.replace("/#/order/douyin");</script></head><body><p>授权成功，正在跳转...</p></body></html>`)
}

// ==================== 辅助函数 ====================

func saveTokenRecord(ctx context.Context, accessToken string, resp interface{}) {
	// 保存到 third_party_api_log
	b, _ := json.Marshal(resp)
	g.DB().Model("third_party_api_log").Ctx(ctx).Data(g.Map{
		"platform":       "qianchuan",
		"api_name":       "oauthCallback",
		"api_path":       "/api/douyin/callback",
		"request_params": accessToken,
		"response_body":  string(b),
		"status":         1,
	}).Insert()

	// 直接从 resp 序列化的 JSON 中解析，避免 SDK 模型 omitempty 丢失 refresh_token 等字段
	var parsed struct {
		Code    *int64  `json:"code"`
		Message *string `json:"message"`
		Data    struct {
			AccessToken           *string `json:"access_token"`
			AdvertiserIds         []int64 `json:"advertiser_ids"`
			ExpiresIn             *int64  `json:"expires_in"`
			RefreshToken          *string `json:"refresh_token"`
			RefreshTokenExpiresIn *int64  `json:"refresh_token_expires_in"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		g.Log().Errorf(ctx, "[QianchuanCallback] 解析token响应失败: %v", err)
		return
	}
	if parsed.Code != nil && *parsed.Code != 0 {
		msg := ""
		if parsed.Message != nil {
			msg = *parsed.Message
		}
		g.Log().Errorf(ctx, "[QianchuanCallback] token响应错误: code=%d, msg=%s", *parsed.Code, msg)
		return
	}

	accountIDs := parsed.Data.AdvertiserIds
	expiresAt := ""
	if parsed.Data.ExpiresIn != nil && *parsed.Data.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(*parsed.Data.ExpiresIn) * time.Second).Format("2006-01-02 15:04:05")
	}
	refreshToken := ""
	refreshTokenExpiresAt := ""
	if parsed.Data.RefreshToken != nil && *parsed.Data.RefreshToken != "" {
		refreshToken = *parsed.Data.RefreshToken
		g.Log().Infof(ctx, "[QianchuanCallback] 获取到refresh_token, 长度=%d", len(refreshToken))
	} else {
		g.Log().Warning(ctx, "[QianchuanCallback] token响应中无refresh_token，token过期后需重新授权")
	}
	if parsed.Data.RefreshTokenExpiresIn != nil && *parsed.Data.RefreshTokenExpiresIn > 0 {
		refreshTokenExpiresAt = time.Now().Add(time.Duration(*parsed.Data.RefreshTokenExpiresIn) * time.Second).Format("2006-01-02 15:04:05")
	}

	// 将授权返回的平台账户ID解析为千川广告主ID
	// token交换返回的AdvertiserIds可能是巨量引擎平台账户ID（店铺级别），
	// 需要调用 qianchuan/shop/advertiser/list/ 接口解析为真正的千川广告主ID
	qianchuanIDs, err := oceanengine.ResolveQianchuanAdvertiserIDs(ctx, accessToken, accountIDs)
	if err != nil {
		g.Log().Warningf(ctx, "[QianchuanCallback] 解析千川广告主ID失败，使用原始IDs: %v", err)
		qianchuanIDs = accountIDs
	}
	if len(qianchuanIDs) == 0 {
		g.Log().Warningf(ctx, "[QianchuanCallback] 解析千川广告主ID返回空列表，使用原始IDs")
		qianchuanIDs = accountIDs
	}
	g.Log().Infof(ctx, "[QianchuanCallback] 千川广告主IDs: 原始=%v -> 解析后=%v", accountIDs, qianchuanIDs)

	idsJSON, _ := json.Marshal(qianchuanIDs)
	g.DB().Model("qianchuan_token").Ctx(ctx).Data(g.Map{
		"access_token":             accessToken,
		"refresh_token":            refreshToken,
		"advertiser_ids":           string(idsJSON),
		"expires_in":               parsed.Data.ExpiresIn,
		"expires_at":               expiresAt,
		"refresh_token_expires_at": refreshTokenExpiresAt,
	}).Insert()
}

func logCallbackReq(ctx context.Context, authCode string, advertiserIDs []int64, results interface{}) {
	resultJSON, _ := json.Marshal(results)
	respBody := string(resultJSON)
	if len(respBody) > 50000 {
		respBody = respBody[:50000] + "...[truncated]"
	}

	reqBody, _ := json.Marshal(g.Map{
		"auth_code":      authCode,
		"advertiser_ids": advertiserIDs,
	})

	g.DB().Model("third_party_api_log").Ctx(ctx).Data(g.Map{
		"platform":       "qianchuan",
		"api_name":       "qianchuanPullAll",
		"api_path":       "/api/douyin/callback",
		"request_params": string(reqBody),
		"response_body":  respBody,
		"status":         1,
	}).Insert()
}
