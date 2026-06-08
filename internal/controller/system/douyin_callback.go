package system

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"go-crawler/internal/service/platform/oceanengine"

	"github.com/oceanengine/ad_open_sdk_go/models"
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
		r.Response.WriteJsonExit(g.Map{
			"code": 400,
			"msg":  "缺少 auth_code 参数",
		})
		return
	}

	g.Log().Infof(ctx, "[QianchuanCallback] 收到OAuth回调, auth_code=%s", authCode)

	// Step0: auth_code 换 token
	tokenResp, err := oceanengine.ExchangeToken(ctx, authCode)
	if err != nil {
		g.Log().Errorf(ctx, "[QianchuanCallback] 换取token失败: %v", err)
		r.Response.WriteJsonExit(g.Map{
			"code": 500,
			"msg":  "换取access_token失败: " + err.Error(),
		})
		return
	}

	accessToken := *tokenResp.Data.AccessToken
	advertiserIDs := tokenResp.Data.AdvertiserIds

	g.Log().Infof(ctx, "[QianchuanCallback] token换取成功, 授权账户数=%d", len(advertiserIDs))

	// 保存 token 信息
	saveTokenRecord(ctx, accessToken, tokenResp)

	// Step1-3: 遍历每个广告主，拉取数据
	client := oceanengine.NewClient(accessToken)
	pullDays := g.Cfg().MustGet(ctx, "platform.qianchuan.pullDays", 30).Int()

	type accountResult struct {
		AdvertiserID   int64                         `json:"advertiser_id"`
		AdvertiserName string                        `json:"advertiser_name"`
		TotalCombined  int                           `json:"total_combined"`
		Saved          int                           `json:"saved"`
		Combined       []oceanengine.CombinedReport  `json:"combined,omitempty"`
		Error          string                        `json:"error,omitempty"`
	}

	var results []accountResult
	totalSaved := 0

	for _, advID := range advertiserIDs {
		ar := accountResult{AdvertiserID: advID}

		pullResult, err := client.PullAll(ctx, advID, pullDays)
		if err != nil {
			ar.Error = err.Error()
			g.Log().Errorf(ctx, "[QianchuanCallback] 拉取广告主%d数据失败: %v", advID, err)
			results = append(results, ar)
			continue
		}

		ar.AdvertiserName = pullResult.AdvertiserName
		ar.TotalCombined = pullResult.TotalCombined

		// 入库
		saved := 0
		for _, item := range pullResult.Combined {
			data := g.Map{
				"advertiser_id":   item.AdvertiserID,
				"advertiser_name": item.AdvertiserName,
				"ad_id":           item.AdID,
				"ad_name":         item.AdName,
				"product_id":      item.ProductID,
				"product_name":    item.ProductName,
				"aweme_id":        item.AwemeID,
				"aweme_name":      item.AwemeName,
				"aweme_show_id":   item.AwemeShowID,
				"stat_cost":       item.StatCost,
				"stat_datetime":   item.StatDatetime,
				"show_cnt":        item.ShowCnt,
				"click_cnt":       item.ClickCnt,
				"ctr":             item.Ctr,
				"convert_cnt":     item.ConvertCnt,
				"convert_cost":    item.ConvertCost,
				"updated_at":      time.Now(),
			}

			count, _ := g.DB().Model("qianchuan_ad_report").Ctx(ctx).
				Where("ad_id", item.AdID).
				Where("stat_datetime", item.StatDatetime).
				Count()

			if count > 0 {
				g.DB().Model("qianchuan_ad_report").Ctx(ctx).
					Where("ad_id", item.AdID).
					Where("stat_datetime", item.StatDatetime).
					Data(data).Update()
			} else {
				_, err := g.DB().Model("qianchuan_ad_report").Ctx(ctx).Data(data).Insert()
				if err != nil {
					g.Log().Errorf(ctx, "[QianchuanCallback] 入库失败 ad_id=%d: %v", item.AdID, err)
					continue
				}
			}
			saved++
		}
		ar.Saved = saved
		totalSaved += saved

		g.Log().Infof(ctx, "[QianchuanCallback] 广告主%d(%s): 拉取%d条, 入库%d条",
			advID, pullResult.AdvertiserName, pullResult.TotalCombined, saved)

		results = append(results, ar)
	}

	// 记录回调日志
	logCallbackReq(ctx, authCode, advertiserIDs, results)

	msg := "授权成功"
	if totalSaved > 0 {
		msg = "授权成功，已拉取数据"
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  msg,
		"data": g.Map{
			"advertiser_count": len(results),
			"total_saved":      totalSaved,
			"results":          results,
		},
	})
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

	// 保存 token 到 qianchuan_token 表，供定时拉取使用
	if tokenResp, ok := resp.(*models.Oauth2AccessTokenResponse); ok {
		idsJSON, _ := json.Marshal(tokenResp.Data.AdvertiserIds)
		g.DB().Model("qianchuan_token").Ctx(ctx).Data(g.Map{
			"access_token":   accessToken,
			"advertiser_ids": string(idsJSON),
			"expires_in":     tokenResp.Data.ExpiresIn,
		}).Insert()
	}
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
