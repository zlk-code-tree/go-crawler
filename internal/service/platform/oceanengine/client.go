package oceanengine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	ad_open_sdk_go "github.com/oceanengine/ad_open_sdk_go"
	"github.com/oceanengine/ad_open_sdk_go/config"
	"github.com/oceanengine/ad_open_sdk_go/models"

	"github.com/gogf/gf/v2/frame/g"
)

// Client 巨量千川（Ocean Engine）SDK 封装
type Client struct {
	cfg         *config.Configuration
	accessToken string
}

// NewClient 创建巨量千川客户端
func NewClient(accessToken string) *Client {
	cfg := config.NewConfiguration()
	return &Client{cfg: cfg, accessToken: accessToken}
}

// ==================== Step0: 用 auth_code 换取 access_token ====================

// ExchangeToken 用 auth_code 换取 access_token
func ExchangeToken(ctx context.Context, authCode string) (*models.Oauth2AccessTokenResponse, error) {
	cfg := g.Cfg()
	appId := cfg.MustGet(ctx, "platform.qianchuan.appId").Int64()
	secret := cfg.MustGet(ctx, "platform.qianchuan.appSecret").String()

	g.Log().Infof(ctx, "[Qianchuan] ExchangeToken: appId=%d, secret=%s, authCode=%s", appId, secret[:8]+"...", authCode[:8]+"...")

	// 使用原始 HTTP 请求，避免 SDK 的 omitempty 等问题
	var respData models.Oauth2AccessTokenResponse
	client := g.Client()
	client.SetHeader("Content-Type", "application/json")
	resp, err := client.Post(ctx, "https://api.oceanengine.com/open_api/oauth2/access_token/", g.Map{
		"app_id":    appId,
		"secret":    secret,
		"auth_code": authCode,
	})
	if err != nil {
		return nil, fmt.Errorf("换取access_token失败: %w", err)
	}
	defer resp.Close()

	body := resp.ReadAllString()
	g.Log().Infof(ctx, "[Qianchuan] ExchangeToken原始响应: %s", body)
	if err := json.Unmarshal([]byte(body), &respData); err != nil {
		return nil, fmt.Errorf("解析access_token响应失败: %w, body=%s", err, body)
	}

	if respData.Code != nil && *respData.Code != 0 {
		return nil, fmt.Errorf("换取access_token失败: code=%d, msg=%s", *respData.Code, *respData.Message)
	}

	// 如果 advertiser_ids 为空，调用 advertiser/get 接口获取
	if respData.Data != nil && len(respData.Data.AdvertiserIds) == 0 {
		g.Log().Info(ctx, "[Qianchuan] token响应无advertiser_ids，调用advertiser/get获取")
		advIds, err := getAdvertiserIDs(ctx, *respData.Data.AccessToken)
		if err != nil {
			g.Log().Warningf(ctx, "[Qianchuan] 获取广告主列表失败: %v", err)
		} else {
			respData.Data.AdvertiserIds = advIds
			g.Log().Infof(ctx, "[Qianchuan] 从advertiser/get获取到 %d 个广告主", len(advIds))
		}
	}

	return &respData, nil
}

// getAdvertiserIDs 调用 /oauth2/advertiser/get/ 获取已授权广告主列表
func getAdvertiserIDs(ctx context.Context, accessToken string) ([]int64, error) {
	client := g.Client()
	resp, err := client.Get(ctx, "https://api.oceanengine.com/open_api/oauth2/advertiser/get/", g.Map{
		"access_token": accessToken,
	})
	if err != nil {
		return nil, fmt.Errorf("请求advertiser/get失败: %w", err)
	}
	defer resp.Close()

	body := resp.ReadAllString()
	g.Log().Infof(ctx, "[Qianchuan] advertiser/get原始响应: %s", body)

	var advResp struct {
		Code    *int64 `json:"code"`
		Message string `json:"message"`
		Data    struct {
			List []struct {
				AdvertiserId   *int64  `json:"advertiser_id"`
				AdvertiserName *string `json:"advertiser_name"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &advResp); err != nil {
		return nil, fmt.Errorf("解析advertiser/get响应失败: %w", err)
	}
	if advResp.Code != nil && *advResp.Code != 0 {
		return nil, fmt.Errorf("advertiser/get返回错误: code=%d, msg=%s", *advResp.Code, advResp.Message)
	}

	var ids []int64
	for _, item := range advResp.Data.List {
		if item.AdvertiserId != nil {
			ids = append(ids, *item.AdvertiserId)
		}
	}
	return ids, nil
}

// ==================== Step1: 获取广告主信息（账户名称） ====================

// AdvertiserInfo 广告主信息
type AdvertiserInfo struct {
	AdvertiserID   int64
	AdvertiserName string
	Company        string
}

// GetAdvertiserInfo 获取广告主信息
func (c *Client) GetAdvertiserInfo(ctx context.Context, advertiserID int64) (*AdvertiserInfo, error) {
	client := ad_open_sdk_go.Init(c.cfg)

	resp, _, err := client.AdvertiserInfoV2Api().Get(ctx).
		AdvertiserIds([]int64{advertiserID}).
		Fields([]string{"id", "name", "company"}).
		AccessToken(c.accessToken).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("获取广告主信息失败(advertiserID=%d): %w", advertiserID, err)
	}
	if resp.Code != nil && *resp.Code != 0 {
		return nil, fmt.Errorf("广告主信息返回错误: code=%d, msg=%s", *resp.Code, *resp.Message)
	}

	info := &AdvertiserInfo{
		AdvertiserID: advertiserID,
	}
	if len(resp.Data) > 0 {
		d := resp.Data[0]
		if d.Id != nil {
			info.AdvertiserID = *d.Id
		}
		if d.Name != nil {
			info.AdvertiserName = *d.Name
		}
		if d.Company != nil {
			info.Company = *d.Company
		}
	}
	return info, nil
}

// ==================== Step2: 获取广告计划列表（商品信息+抖音账号） ====================

// AdPlan 广告计划
type AdPlan struct {
	AdID         int64  // 计划ID
	AdName       string // 计划名称
	AdvertiserID int64  // 账户ID
	AdCreateTime string // 计划创建时间
	AdModifyTime string // 计划修改时间
	Status       string // 计划状态
}

// GetAllAdPlans 获取广告主下所有广告计划（千川 v1.0/qianchuan/ad/get/）
// 千川 API 要求 filtering.marketing_goal 必填，需要分别拉取直播和短视频两类计划
func (c *Client) GetAllAdPlans(ctx context.Context, advertiserID int64) ([]AdPlan, error) {
	var allPlans []AdPlan
	client := ad_open_sdk_go.Init(c.cfg)

	goals := []models.QianchuanAdGetV10FilteringMarketingGoal{
		models.LIVE_PROM_GOODS_QianchuanAdGetV10FilteringMarketingGoal,
		models.VIDEO_PROM_GOODS_QianchuanAdGetV10FilteringMarketingGoal,
	}

	for _, goal := range goals {
		page := int32(1)
		pageSize := int32(100)
		for {
			g.Log().Infof(ctx, "[Qianchuan] 调用千川广告列表API advID=%d page=%d goal=%s", advertiserID, page, goal)
			resp, _, err := client.QianchuanAdGetV10Api().Get(ctx).
				AdvertiserId(advertiserID).
				Filtering(models.QianchuanAdGetV10Filtering{
					MarketingGoal: goal,
				}).
				Page(page).
				PageSize(pageSize).
				AccessToken(c.accessToken).
				Execute()
			if err != nil {
				return nil, fmt.Errorf("获取广告计划列表失败(advertiserID=%d, goal=%s): %w", advertiserID, goal, err)
			}
			if resp.Code != nil && *resp.Code != 0 {
				return nil, fmt.Errorf("广告列表返回错误: code=%d, msg=%s", *resp.Code, *resp.Message)
			}

			if resp.Data != nil && resp.Data.List != nil {
				for _, item := range resp.Data.List {
					plan := AdPlan{AdvertiserID: advertiserID}
					if item.AdId != nil { plan.AdID = *item.AdId }
					if item.Name != nil { plan.AdName = *item.Name }
					if item.AdCreateTime != nil { plan.AdCreateTime = *item.AdCreateTime }
					if item.AdModifyTime != nil { plan.AdModifyTime = *item.AdModifyTime }
					if item.Status != nil { plan.Status = string(*item.Status) }
					allPlans = append(allPlans, plan)
				}
				g.Log().Infof(ctx, "[Qianchuan] 千川广告列表(advID=%d, goal=%s): 本页%d条", advertiserID, goal, len(resp.Data.List))
			}

			if resp.Data == nil || resp.Data.PageInfo == nil || resp.Data.PageInfo.TotalPage == nil || page >= *resp.Data.PageInfo.TotalPage {
				break
			}
			page++
		}
	}

	return allPlans, nil
}

// ==================== Step3: 获取广告计划报表（消耗数据） ====================

// AdReport 广告计划报表
type AdReport struct {
	AdID         int64   // 计划ID
	AdvertiserID int64   // 账户ID
	StatCost     float64 // 消耗（元）
	StatDatetime string  // 统计日期
}

// GetAdReport 获取广告报表（千川 v1.0/qianchuan/report/ad/get/）
func (c *Client) GetAdReport(ctx context.Context, advertiserID int64, startDate, endDate string) ([]AdReport, error) {
	page := int32(1)
	pageSize := int32(200)
	var allReports []AdReport

	client := ad_open_sdk_go.Init(c.cfg)

	for {
		g.Log().Infof(ctx, "[Qianchuan] 调用千川报表API advID=%d page=%d, date=%s~%s", advertiserID, page, startDate, endDate)
		resp, _, err := client.QianchuanReportAdGetV10Api().Get(ctx).
			AdvertiserId(advertiserID).
			StartDate(startDate).
			EndDate(endDate).
			Fields([]string{"ad_id", "advertiser_id", "stat_cost", "stat_datetime"}).
			TimeGranularity("STAT_TIME_GRANULARITY_DAILY").
			Page(page).
			PageSize(pageSize).
			AccessToken(c.accessToken).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("获取广告报表失败(advertiserID=%d): %w", advertiserID, err)
		}
		if resp.Code != nil && *resp.Code != 0 {
			return nil, fmt.Errorf("报表返回错误: code=%d, msg=%s", *resp.Code, *resp.Message)
		}

		if resp.Data != nil && resp.Data.List != nil {
			for _, item := range resp.Data.List {
				r := AdReport{}
				if item.AdId != nil { r.AdID = *item.AdId }
				if item.AdvertiserId != nil { r.AdvertiserID = *item.AdvertiserId }
				if item.StatCost != nil { r.StatCost = *item.StatCost }
				if item.StatDatetime != nil { r.StatDatetime = *item.StatDatetime }
				allReports = append(allReports, r)
			}
			g.Log().Infof(ctx, "[Qianchuan] 千川报表(advID=%d): 本页%d条", advertiserID, len(resp.Data.List))
		}

		if resp.Data == nil || resp.Data.PageInfo == nil || resp.Data.PageInfo.TotalPage == nil || int64(page) >= *resp.Data.PageInfo.TotalPage {
			break
		}
		page++
	}

	return allReports, nil
}

// ==================== Step4: 组合数据结构 ====================

// PullResult 一次拉取的结果
type PullResult struct {
	AdvertiserID   int64     `json:"advertiser_id"`
	AdvertiserName string    `json:"advertiser_name"`
	Company        string    `json:"company"`
	Plans          []AdPlan  `json:"-"`
	Reports        []AdReport `json:"-"`
	Combined       []CombinedReport `json:"combined"`
	TotalPlans     int       `json:"total_plans"`
	TotalReports   int       `json:"total_reports"`
	TotalCombined  int       `json:"total_combined"`
}

// CombinedReport 组合后的报表数据（计划信息 + 报表消耗）
type CombinedReport struct {
	AdvertiserID   int64   `json:"advertiser_id"`
	AdvertiserName string  `json:"advertiser_name"`
	AdID           int64   `json:"ad_id"`
	AdName         string  `json:"ad_name"`
	StatCost       float64 `json:"stat_cost"`
	StatDatetime   string  `json:"stat_datetime"`
}

// PullAll 拉取指定广告主的全量数据（计划列表 + 报表）
// days: 拉取最近多少天的报表数据
func (c *Client) PullAll(ctx context.Context, advertiserID int64, days int) (*PullResult, error) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -days).Format("2006-01-02")
	endDate := now.Format("2006-01-02")
	g.Log().Infof(ctx, "[Qianchuan] PullAll: advertiserID=%d, dateRange=%s~%s, token前8位=%s", advertiserID, startDate, endDate, c.accessToken[:min(8, len(c.accessToken))])
	return c.PullAllByDate(ctx, advertiserID, startDate, endDate)
}

func min(a, b int) int {
	if a < b { return a }
	return b
}

// PullAllByDate 拉取指定广告主在指定日期范围内的全量数据
// startDate/endDate 格式: 2006-01-02
func (c *Client) PullAllByDate(ctx context.Context, advertiserID int64, startDate, endDate string) (*PullResult, error) {
	result := &PullResult{AdvertiserID: advertiserID}

	// 1. 获取广告主名称
	advInfo, err := c.GetAdvertiserInfo(ctx, advertiserID)
	if err != nil {
		g.Log().Warningf(ctx, "[Qianchuan] 获取广告主信息失败(advertiserID=%d): %v", advertiserID, err)
	} else {
		result.AdvertiserName = advInfo.AdvertiserName
		result.Company = advInfo.Company
	}

	// 2. 获取所有广告计划（含商品和抖音账号信息）
	plans, err := c.GetAllAdPlans(ctx, advertiserID)
	if err != nil {
		return nil, fmt.Errorf("获取广告计划失败: %w", err)
	}
	result.Plans = plans
	result.TotalPlans = len(plans)
	g.Log().Infof(ctx, "[Qianchuan] 获取到 %d 条广告计划", len(plans))

	// 3. 获取广告报表
	reports, err := c.GetAdReport(ctx, advertiserID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("获取广告报表失败: %w", err)
	}
	result.Reports = reports
	result.TotalReports = len(reports)
	g.Log().Infof(ctx, "[Qianchuan] 获取到 %d 条报表数据(%s~%s)", len(reports), startDate, endDate)

	// 4. 以计划ID为key，关联计划和报表数据
	planMap := make(map[int64]*AdPlan)
	for i := range plans {
		planMap[plans[i].AdID] = &plans[i]
	}

	for _, r := range reports {
		combined := CombinedReport{
			AdvertiserID:   advertiserID,
			AdvertiserName: result.AdvertiserName,
			AdID:           r.AdID,
			StatCost:       r.StatCost,
			StatDatetime:   r.StatDatetime,
		}
		if plan, ok := planMap[r.AdID]; ok {
			combined.AdName = plan.AdName
		}
		result.Combined = append(result.Combined, combined)
	}
	result.TotalCombined = len(result.Combined)

	g.Log().Infof(ctx, "[Qianchuan] PullAll结果: plans=%d, reports=%d, combined=%d", result.TotalPlans, result.TotalReports, result.TotalCombined)
	return result, nil
}

// ==================== Token 管理 ====================

// GetAccessToken 返回当前使用的 access_token
func (c *Client) GetAccessToken() string {
	return c.accessToken
}
