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
	resp, err := client.Post(ctx, "https://ad.oceanengine.com/open_api/oauth2/access_token/", g.Map{
		"app_id":    appId,
		"secret":    secret,
		"auth_code": authCode,
	})
	if err != nil {
		return nil, fmt.Errorf("换取access_token失败: %w", err)
	}
	defer resp.Close()

	body := resp.ReadAllString()
	g.Log().Debugf(ctx, "[Qianchuan] ExchangeToken原始响应: %s", body)
	if err := json.Unmarshal([]byte(body), &respData); err != nil {
		return nil, fmt.Errorf("解析access_token响应失败: %w, body=%s", err, body)
	}

	g.Log().Infof(ctx, "[Qianchuan] ExchangeToken响应: code=%v", respData.Code)

	if respData.Code != nil && *respData.Code != 0 {
		return nil, fmt.Errorf("换取access_token失败: code=%d, msg=%s", *respData.Code, *respData.Message)
	}
	return &respData, nil
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

// AdPlan 广告计划（含商品和抖音账号信息）
type AdPlan struct {
	AdID          int64  // 计划ID
	AdName        string // 计划名称
	AdvertiserID  int64  // 账户ID
	CampaignID    int64  // 广告组ID
	ProductID     int64  // 商品ID
	ProductName   string // 商品名称
	ProductImg    string // 商品图片
	AwemeID       int64  // 抖音账号ID
	AwemeName     string // 抖音账号名称
	AwemeAvatar   string // 抖音账号头像
	AwemeShowID   string // 抖音号
	AdCreateTime  string // 计划创建时间
	AdModifyTime  string // 计划修改时间
	Status        string // 计划状态
}

// GetAllAdPlans 获取广告主下所有广告计划（含商品和抖音账号信息）
func (c *Client) GetAllAdPlans(ctx context.Context, advertiserID int64) ([]AdPlan, error) {
	var allPlans []AdPlan
	page := int32(1)
	pageSize := int32(100)

	for {
		client := ad_open_sdk_go.Init(c.cfg)

		// requestAwemeInfo=1 返回抖音账号信息
		reqAwemeInfo := int64(1)

		// 查询所有状态的计划
		status := models.ALL_QianchuanAdGetV10FilteringStatus
		filtering := models.QianchuanAdGetV10Filtering{
			Status: &status,
		}

		resp, _, err := client.QianchuanAdGetV10Api().Get(ctx).
			AdvertiserId(advertiserID).
			Filtering(filtering).
			RequestAwemeInfo(reqAwemeInfo).
			Page(page).
			PageSize(pageSize).
			AccessToken(c.accessToken).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("获取广告计划列表失败(advertiserID=%d, page=%d): %w", advertiserID, page, err)
		}
		if resp.Code != nil && *resp.Code != 0 {
			return nil, fmt.Errorf("广告计划列表返回错误: code=%d, msg=%s", *resp.Code, *resp.Message)
		}

		for _, item := range resp.Data.List {
			plan := AdPlan{AdvertiserID: advertiserID}

			if item.AdId != nil {
				plan.AdID = *item.AdId
			}
			if item.Name != nil {
				plan.AdName = *item.Name
			}
			if item.CampaignId != nil {
				plan.CampaignID = *item.CampaignId
			}
			if item.AdCreateTime != nil {
				plan.AdCreateTime = *item.AdCreateTime
			}
			if item.AdModifyTime != nil {
				plan.AdModifyTime = *item.AdModifyTime
			}
			if item.Status != nil {
				plan.Status = string(*item.Status)
			}

			// 取第一个商品信息
			if len(item.ProductInfo) > 0 {
				p := item.ProductInfo[0]
				if p.Id != nil {
					plan.ProductID = *p.Id
				}
				if p.Name != nil {
					plan.ProductName = *p.Name
				}
				if p.Img != nil {
					plan.ProductImg = *p.Img
				}
			}

			// 取第一个抖音账号信息
			if len(item.AwemeInfo) > 0 {
				a := item.AwemeInfo[0]
				if a.AwemeId != nil {
					plan.AwemeID = *a.AwemeId
				}
				if a.AwemeName != nil {
					plan.AwemeName = *a.AwemeName
				}
				if a.AwemeAvatar != nil {
					plan.AwemeAvatar = *a.AwemeAvatar
				}
				if a.AwemeShowId != nil {
					plan.AwemeShowID = *a.AwemeShowId
				}
			}

			allPlans = append(allPlans, plan)
		}

		// 分页判断
		if resp.Data.PageInfo == nil ||
			resp.Data.PageInfo.TotalNumber == nil ||
			resp.Data.PageInfo.TotalPage == nil {
			break
		}
		if page >= *resp.Data.PageInfo.TotalPage {
			break
		}
		page++
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
	ShowCnt      float64 // 展示数
	ClickCnt     float64 // 点击数
	Ctr          float64 // 点击率
	CpmPlatform  float64 // CPM
	CpcPlatform  float64 // CPC
	ConvertCnt   float64 // 转化数
	ConvertCost  float64 // 转化成本
	ConvertRate  float64 // 转化率
}

// GetAdReport 获取广告计划报表
// startDate/endDate 格式: 2006-01-02
func (c *Client) GetAdReport(ctx context.Context, advertiserID int64, startDate, endDate string) ([]AdReport, error) {
	client := ad_open_sdk_go.Init(c.cfg)

	fields := []string{
		"ad_id", "advertiser_id", "stat_cost", "stat_datetime",
		"show_cnt", "click_cnt", "ctr", "cpm_platform", "cpc_platform",
		"convert_cnt", "convert_cost", "convert_rate",
	}

	timeGranularity := models.TIME_GRANULARITY_DAILY_QianchuanReportAdGetV10TimeGranularity

	filtering := models.QianchuanReportAdGetV10Filtering{
		MarketingGoal: models.ALL_QianchuanReportAdGetV10FilteringMarketingGoal,
	}

	page := int32(1)
	pageSize := int32(200)
	var allReports []AdReport

	for {
		resp, _, err := client.QianchuanReportAdGetV10Api().Get(ctx).
			AdvertiserId(advertiserID).
			StartDate(startDate).
			EndDate(endDate).
			Fields(fields).
			Filtering(filtering).
			TimeGranularity(timeGranularity).
			Page(page).
			PageSize(pageSize).
			AccessToken(c.accessToken).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("获取广告报表失败(advertiserID=%d): %w", advertiserID, err)
		}
		if resp.Code != nil && *resp.Code != 0 {
			return nil, fmt.Errorf("广告报表返回错误: code=%d, msg=%s", *resp.Code, *resp.Message)
		}

		for _, item := range resp.Data.List {
			r := AdReport{}
			if item.AdId != nil {
				r.AdID = *item.AdId
			}
			if item.AdvertiserId != nil {
				r.AdvertiserID = *item.AdvertiserId
			}
			if item.StatCost != nil {
				r.StatCost = *item.StatCost
			}
			if item.StatDatetime != nil {
				r.StatDatetime = *item.StatDatetime
			}
			if item.ShowCnt != nil {
				r.ShowCnt = *item.ShowCnt
			}
			if item.ClickCnt != nil {
				r.ClickCnt = *item.ClickCnt
			}
			if item.Ctr != nil {
				r.Ctr = *item.Ctr
			}
			if item.CpmPlatform != nil {
				r.CpmPlatform = *item.CpmPlatform
			}
			if item.CpcPlatform != nil {
				r.CpcPlatform = *item.CpcPlatform
			}
			if item.ConvertCnt != nil {
				r.ConvertCnt = *item.ConvertCnt
			}
			if item.ConvertCost != nil {
				r.ConvertCost = *item.ConvertCost
			}
			if item.ConvertRate != nil {
				r.ConvertRate = *item.ConvertRate
			}
			allReports = append(allReports, r)
		}

		if resp.Data.PageInfo == nil ||
			resp.Data.PageInfo.TotalPage == nil {
			break
		}
		if int64(page) >= *resp.Data.PageInfo.TotalPage {
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
	ProductID      int64   `json:"product_id"`
	ProductName    string  `json:"product_name"`
	AwemeID        int64   `json:"aweme_id"`
	AwemeName      string  `json:"aweme_name"`
	AwemeShowID    string  `json:"aweme_show_id"`
	StatCost       float64 `json:"stat_cost"`
	StatDatetime   string  `json:"stat_datetime"`
	ShowCnt        float64 `json:"show_cnt"`
	ClickCnt       float64 `json:"click_cnt"`
	Ctr            float64 `json:"ctr"`
	ConvertCnt     float64 `json:"convert_cnt"`
	ConvertCost    float64 `json:"convert_cost"`
}

// PullAll 拉取指定广告主的全量数据（计划列表 + 报表）
// days: 拉取最近多少天的报表数据
func (c *Client) PullAll(ctx context.Context, advertiserID int64, days int) (*PullResult, error) {
	now := time.Now()
	return c.PullAllByDate(ctx, advertiserID,
		now.AddDate(0, 0, -days).Format("2006-01-02"),
		now.Format("2006-01-02"))
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
			ShowCnt:        r.ShowCnt,
			ClickCnt:       r.ClickCnt,
			Ctr:            r.Ctr,
			ConvertCnt:     r.ConvertCnt,
			ConvertCost:    r.ConvertCost,
		}
		if plan, ok := planMap[r.AdID]; ok {
			combined.AdName = plan.AdName
			combined.ProductID = plan.ProductID
			combined.ProductName = plan.ProductName
			combined.AwemeID = plan.AwemeID
			combined.AwemeName = plan.AwemeName
			combined.AwemeShowID = plan.AwemeShowID
		}
		result.Combined = append(result.Combined, combined)
	}
	result.TotalCombined = len(result.Combined)

	return result, nil
}

// ==================== Token 管理 ====================

// GetAccessToken 返回当前使用的 access_token
func (c *Client) GetAccessToken() string {
	return c.accessToken
}
