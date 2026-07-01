package oceanengine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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
	ProductID    int64  // 商品ID
	ProductName  string // 商品名称
	AwemeID      string // 抖音账号ID
	AwemeName    string // 抖音账号名称
	AwemeShowID  string // 抖音号
}

// AuthorizedAweme 千川广告主授权抖音号
type AuthorizedAweme struct {
	AwemeID     int64
	AwemeName   string
	AwemeShowID string
	AwemeStatus string
	BindTypes   []string
}

// GetAllAdPlans 获取广告主下所有广告计划（千川 v1.0/qianchuan/ad/get/）
// 千川 API 要求 filtering.marketing_goal 必填，需要分别拉取直播和短视频两类计划
func (c *Client) GetAllAdPlans(ctx context.Context, advertiserID int64) ([]AdPlan, error) {
	var allPlans []AdPlan
	sdkClient := ad_open_sdk_go.Init(c.cfg)

	goals := []models.QianchuanAdGetV10FilteringMarketingGoal{
		models.LIVE_PROM_GOODS_QianchuanAdGetV10FilteringMarketingGoal,
		models.VIDEO_PROM_GOODS_QianchuanAdGetV10FilteringMarketingGoal,
	}

	for _, goal := range goals {
		page := int32(1)
		pageSize := int32(100)
		for {
			g.Log().Infof(ctx, "[Qianchuan] 调用千川广告列表API advID=%d page=%d goal=%s", advertiserID, page, goal)
			var resp *models.QianchuanAdGetV10Response
			err := retryWithBackoff(ctx, fmt.Sprintf("计划列表API(advID=%d,goal=%s)", advertiserID, goal), func() error {
				var e error
				resp, _, e = sdkClient.QianchuanAdGetV10Api().Get(ctx).
					AdvertiserId(advertiserID).
					Filtering(models.QianchuanAdGetV10Filtering{
						MarketingGoal: goal,
					}).
					Page(page).
					PageSize(pageSize).
					AccessToken(c.accessToken).
					Execute()
				return e
			})
			if err != nil {
				return nil, fmt.Errorf("获取广告计划列表失败(advertiserID=%d, goal=%s): %w", advertiserID, goal, err)
			}
			if resp.Code != nil && *resp.Code != 0 {
				return nil, fmt.Errorf("广告列表返回错误: code=%d, msg=%s", *resp.Code, *resp.Message)
			}

			if resp.Data != nil && resp.Data.List != nil {
				for _, item := range resp.Data.List {
					plan := AdPlan{AdvertiserID: advertiserID}
					if item.AdId != nil {
						plan.AdID = *item.AdId
					}
					if item.Name != nil {
						plan.AdName = *item.Name
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
					// 提取商品信息
					if len(item.ProductInfo) > 0 {
						if item.ProductInfo[0].Id != nil {
							plan.ProductID = *item.ProductInfo[0].Id
						}
						if item.ProductInfo[0].Name != nil {
							plan.ProductName = *item.ProductInfo[0].Name
						}
					}
					// 提取抖音号信息
					if len(item.AwemeInfo) > 0 {
						if item.AwemeInfo[0].AwemeId != nil {
							plan.AwemeID = strconv.FormatInt(*item.AwemeInfo[0].AwemeId, 10)
						}
						if item.AwemeInfo[0].AwemeName != nil {
							plan.AwemeName = *item.AwemeInfo[0].AwemeName
						}
						if item.AwemeInfo[0].AwemeShowId != nil {
							plan.AwemeShowID = *item.AwemeInfo[0].AwemeShowId
						}
					}
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

	// 5. 获取该广告主授权抖音号列表，填充计划中缺失的aweme信息
	tryFillAwemeInfo(ctx, c, &allPlans, advertiserID)

	return allPlans, nil
}

// GetAuthorizedAwemes 获取广告主下全部授权抖音号（自动翻页）
func (c *Client) GetAuthorizedAwemes(ctx context.Context, advertiserID int64) ([]AuthorizedAweme, error) {
	sdkClient := ad_open_sdk_go.Init(c.cfg)
	page := int32(1)
	pageSize := int32(100)
	var result []AuthorizedAweme

	for {
		resp, _, err := sdkClient.QianchuanAwemeAuthorizedGetV10Api().Get(ctx).
			AdvertiserId(advertiserID).
			Filtering(models.QianchuanAwemeAuthorizedGetV10Filtering{
				MarketingGoal: models.ALL_QianchuanAwemeAuthorizedGetV10FilteringMarketingGoal.Ptr(),
			}).
			Page(page).
			PageSize(pageSize).
			AccessToken(c.accessToken).
			Execute()
		if err != nil {
			return nil, fmt.Errorf("获取授权抖音号列表失败(advID=%d,page=%d): %w", advertiserID, page, err)
		}
		if resp.Code != nil && *resp.Code != 0 {
			msg := ""
			if resp.Message != nil {
				msg = *resp.Message
			}
			return nil, fmt.Errorf("授权抖音号列表返回错误(code=%d,msg=%s)", *resp.Code, msg)
		}
		if resp.Data == nil || len(resp.Data.AwemeIdList) == 0 {
			break
		}

		for _, item := range resp.Data.AwemeIdList {
			if item == nil {
				continue
			}
			aweme := AuthorizedAweme{}
			if item.AwemeId != nil {
				aweme.AwemeID = *item.AwemeId
			}
			if item.AwemeName != nil {
				aweme.AwemeName = *item.AwemeName
			}
			if item.AwemeShowId != nil {
				aweme.AwemeShowID = *item.AwemeShowId
			}
			if item.AwemeStatus != nil {
				aweme.AwemeStatus = string(*item.AwemeStatus)
			}
			for _, bindType := range item.BindType {
				if bindType != nil {
					aweme.BindTypes = append(aweme.BindTypes, string(*bindType))
				}
			}
			result = append(result, aweme)
		}

		if resp.Data.PageInfo == nil || resp.Data.PageInfo.TotalPage == nil || int64(page) >= *resp.Data.PageInfo.TotalPage {
			break
		}
		page++
	}

	return result, nil
}

// tryFillAwemeInfo 通过 qianchuan_aweme_authorized_get 接口获取授权抖音号列表，
// 填充计划列表中缺失的 aweme_info（计划页面的aweme_info可能有空）
func tryFillAwemeInfo(ctx context.Context, c *Client, allPlans *[]AdPlan, advertiserID int64) {
	awemes, err := c.GetAuthorizedAwemes(ctx, advertiserID)
	if err != nil {
		g.Log().Warningf(ctx, "[Qianchuan] 获取授权抖音号列表失败(advID=%d): %v", advertiserID, err)
		return
	}
	if len(awemes) == 0 {
		return
	}

	// 仅当只有一个授权抖音号时，用它作为缺失aweme的默认值
	// 多个授权抖音号时无法自动匹配，跳过填充避免数据错配
	if len(awemes) > 1 {
		emptyCount := 0
		for i := range *allPlans {
			if (*allPlans)[i].AwemeID == "" {
				emptyCount++
			}
		}
		if emptyCount > 0 {
			g.Log().Warningf(ctx, "[Qianchuan] advID=%d 有 %d 个授权抖音号，但 %d 个计划缺少aweme数据，无法自动匹配，已跳过",
				advertiserID, len(awemes), emptyCount)
		}
		return
	}

	firstAweme := awemes[0]
	awemeID := strconv.FormatInt(firstAweme.AwemeID, 10)
	awemeName := firstAweme.AwemeName
	awemeShowID := firstAweme.AwemeShowID

	count := 0
	for i := range *allPlans {
		if (*allPlans)[i].AwemeID == "" {
			(*allPlans)[i].AwemeID = awemeID
			(*allPlans)[i].AwemeName = awemeName
			(*allPlans)[i].AwemeShowID = awemeShowID
			count++
		}
	}
	if count > 0 {
		g.Log().Infof(ctx, "[Qianchuan] 从授权抖音号列表填充%d个计划的aweme数据(advID=%d, aweme=%s/%s/%s)",
			count, advertiserID, awemeID, awemeName, awemeShowID)
	}
}

// ==================== Step3: 获取广告主日结报表（账户维度） ====================

// AdReport 广告主日结报表（账户维度，来自 report/advertiser/get API）
type AdReport struct {
	AdvertiserID            int64   // 账户ID
	StatDatetime            string  // 统计日期
	StatCost                float64 // 消耗（元）
	ShowCnt                 int64   // 展示数
	ClickCnt                int64   // 点击数
	Ctr                     float64 // 点击率
	CpcPlatform             float64 // CPC
	CpmPlatform             float64 // CPM
	ConvertCnt              int64   // 转化数
	ConvertCost             float64 // 转化成本
	ConvertRate             float64 // 转化率
	DeepConvertCnt          int64   // 深度转化数
	DeepConvertCost         float64 // 深度转化成本
	PayOrderAmount          float64 // 成交金额
	PayOrderCount           int64   // 成交订单数
	PayOrderCostPerOrder    float64 // 成交订单成本
	PayOrderCouponAmount    float64 // 成交优惠券金额
	CreateOrderAmount       float64 // 下单金额
	CreateOrderCount        int64   // 下单数
	CreateOrderCouponAmount float64 // 下单优惠券金额
	CreateOrderRoi          float64 // 下单ROI
	PrepayOrderAmount       float64 // 预售金额
	PrepayOrderCount        int64   // 预售订单数
	PrepayAndPayOrderRoi    float64 // 预售+成交ROI
	AllOrderPayGmv7days     float64 // 7日支付GMV
	AllOrderPayCount7days   int64   // 7日支付数
	AllOrderPayRoi7days     float64 // 7日ROI
	AllOrderRefundAmount7d  float64 // 7日退款金额
	AllOrderRefundCount7d   int64   // 7日退款数
	AllOrderRefundAmount14d float64 // 14日退款金额
	AllOrderRefundCount14d  int64   // 14日退款数
	DyFollow                int64   // 新增关注
	TotalPlay               int64   // 播放数
}

// AdPlanReport 广告计划日结报表（计划维度，来自 report/ad/get API）
type AdPlanReport struct {
	AdvertiserID int64
	AdID         int64
	StatDatetime string
	StatCost     float64
	ShowCnt      int64
	ClickCnt     int64
	Ctr          float64
	ConvertCnt   int64
	ConvertCost  float64
}

// GetAdReport 获取广告主日结报表（千川 v1.0/qianchuan/report/advertiser/get/）
// 账户维度：每个广告主每天返回一条汇总数据
func (c *Client) GetAdReport(ctx context.Context, advertiserID int64, startDate, endDate string) ([]AdReport, error) {
	page := int32(1)
	pageSize := int32(200)
	var allReports []AdReport

	// 账户维度报表请求的字段
	reportFields := []string{
		"stat_cost", "show_cnt", "click_cnt", "ctr", "cpc_platform", "cpm_platform",
		"convert_cnt", "convert_cost", "convert_rate", "deep_convert_cnt", "deep_convert_cost",
		"pay_order_amount", "pay_order_count", "pay_order_cost_per_order", "pay_order_coupon_amount",
		"create_order_amount", "create_order_count", "create_order_coupon_amount", "create_order_roi",
		"prepay_order_amount", "prepay_order_count", "prepay_and_pay_order_roi",
		"all_order_pay_gmv_7days", "all_order_pay_count_7days", "all_order_pay_roi_7days",
		"all_order_refund_amount_7d", "all_order_refund_count_7d",
		"all_order_refund_amount_14d", "all_order_refund_count_14d",
		"dy_follow", "total_play",
	}

	sdkClient := ad_open_sdk_go.Init(c.cfg)

	for {
		g.Log().Infof(ctx, "[Qianchuan] 调用千川账户报表API advID=%d page=%d, date=%s~%s", advertiserID, page, startDate, endDate)

		var resp *models.QianchuanReportAdvertiserGetV10Response
		var rawResp *http.Response
		err := retryWithBackoff(ctx, fmt.Sprintf("账户报表API(advID=%d)", advertiserID), func() error {
			var e error
			resp, rawResp, e = sdkClient.QianchuanReportAdvertiserGetV10Api().Get(ctx).
				AdvertiserId(advertiserID).
				StartDate(startDate).
				EndDate(endDate).
				Fields(reportFields).
				Filtering(models.QianchuanReportAdvertiserGetV10Filtering{
					MarketingGoal: models.ALL_QianchuanReportAdvertiserGetV10FilteringMarketingGoal,
				}).
				TimeGranularity(models.TIME_GRANULARITY_DAILY_QianchuanReportAdvertiserGetV10TimeGranularity).
				Page(page).
				PageSize(pageSize).
				AccessToken(c.accessToken).
				Execute()
			return e
		})
		if err != nil {
			return nil, fmt.Errorf("获取账户报表失败(advertiserID=%d): %w", advertiserID, err)
		}

		codeVal := int64(0)
		msgVal := ""
		if resp.Code != nil {
			codeVal = *resp.Code
		}
		if resp.Message != nil {
			msgVal = *resp.Message
		}
		listLen := 0
		if resp.Data != nil && resp.Data.List != nil {
			listLen = len(resp.Data.List)
		}
		httpStatus := 0
		if rawResp != nil {
			httpStatus = rawResp.StatusCode
		}

		g.Log().Infof(ctx, "[Qianchuan] 账户报表API: advID=%d, HTTP=%d, code=%d, msg=%s, list=%d",
			advertiserID, httpStatus, codeVal, msgVal, listLen)

		if codeVal != 0 {
			return nil, fmt.Errorf("账户报表返回错误: code=%d, msg=%s", codeVal, msgVal)
		}

		if resp.Data != nil && resp.Data.List != nil {
			for _, item := range resp.Data.List {
				r := AdReport{AdvertiserID: advertiserID}
				if item.StatDatetime != nil {
					r.StatDatetime = normalizeStatDate(*item.StatDatetime)
				}
				if item.StatCost != nil {
					r.StatCost = *item.StatCost
				}
				if item.ShowCnt != nil {
					r.ShowCnt = int64(*item.ShowCnt)
				}
				if item.ClickCnt != nil {
					r.ClickCnt = int64(*item.ClickCnt)
				}
				if item.Ctr != nil {
					r.Ctr = *item.Ctr
				}
				if item.CpcPlatform != nil {
					r.CpcPlatform = *item.CpcPlatform
				}
				if item.CpmPlatform != nil {
					r.CpmPlatform = *item.CpmPlatform
				}
				if item.ConvertCnt != nil {
					r.ConvertCnt = int64(*item.ConvertCnt)
				}
				if item.ConvertCost != nil {
					r.ConvertCost = *item.ConvertCost
				}
				if item.ConvertRate != nil {
					r.ConvertRate = *item.ConvertRate
				}
				if item.DeepConvertCnt != nil {
					r.DeepConvertCnt = int64(*item.DeepConvertCnt)
				}
				if item.DeepConvertCost != nil {
					r.DeepConvertCost = *item.DeepConvertCost
				}
				if item.PayOrderAmount != nil {
					r.PayOrderAmount = *item.PayOrderAmount
				}
				if item.PayOrderCount != nil {
					r.PayOrderCount = int64(*item.PayOrderCount)
				}
				if item.PayOrderCostPerOrder != nil {
					r.PayOrderCostPerOrder = *item.PayOrderCostPerOrder
				}
				if item.PayOrderCouponAmount != nil {
					r.PayOrderCouponAmount = *item.PayOrderCouponAmount
				}
				if item.CreateOrderAmount != nil {
					r.CreateOrderAmount = *item.CreateOrderAmount
				}
				if item.CreateOrderCount != nil {
					r.CreateOrderCount = int64(*item.CreateOrderCount)
				}
				if item.CreateOrderCouponAmount != nil {
					r.CreateOrderCouponAmount = *item.CreateOrderCouponAmount
				}
				if item.CreateOrderRoi != nil {
					r.CreateOrderRoi = *item.CreateOrderRoi
				}
				if item.PrepayOrderAmount != nil {
					r.PrepayOrderAmount = *item.PrepayOrderAmount
				}
				if item.PrepayOrderCount != nil {
					r.PrepayOrderCount = int64(*item.PrepayOrderCount)
				}
				if item.PrepayAndPayOrderRoi != nil {
					r.PrepayAndPayOrderRoi = *item.PrepayAndPayOrderRoi
				}
				if item.AllOrderPayGmv7days != nil {
					r.AllOrderPayGmv7days = *item.AllOrderPayGmv7days
				}
				if item.AllOrderPayCount7days != nil {
					r.AllOrderPayCount7days = int64(*item.AllOrderPayCount7days)
				}
				if item.AllOrderPayRoi7days != nil {
					r.AllOrderPayRoi7days = *item.AllOrderPayRoi7days
				}
				if item.AllOrderRefundAmount7d != nil {
					r.AllOrderRefundAmount7d = *item.AllOrderRefundAmount7d
				}
				if item.AllOrderRefundCount7d != nil {
					r.AllOrderRefundCount7d = int64(*item.AllOrderRefundCount7d)
				}
				if item.AllOrderRefundAmount14d != nil {
					r.AllOrderRefundAmount14d = *item.AllOrderRefundAmount14d
				}
				if item.AllOrderRefundCount14d != nil {
					r.AllOrderRefundCount14d = int64(*item.AllOrderRefundCount14d)
				}
				if item.DyFollow != nil {
					r.DyFollow = int64(*item.DyFollow)
				}
				if item.TotalPlay != nil {
					r.TotalPlay = int64(*item.TotalPlay)
				}
				allReports = append(allReports, r)
			}
			g.Log().Infof(ctx, "[Qianchuan] 千川账户报表(advID=%d): 本页%d条", advertiserID, len(resp.Data.List))
		}

		if resp.Data == nil || resp.Data.PageInfo == nil || resp.Data.PageInfo.TotalPage == nil || page >= int32(*resp.Data.PageInfo.TotalPage) {
			break
		}
		page++
	}

	g.Log().Infof(ctx, "[Qianchuan] 千川账户报表汇总(advID=%d): 共%d条(%s~%s)", advertiserID, len(allReports), startDate, endDate)
	return allReports, nil
}

// GetUniPromotionReport 获取全域推广账户维度消耗。
// 该接口返回区间聚合数据，不返回日列表；为了财务日统计准确，这里按天调用并组装为日报。
func (c *Client) GetUniPromotionReport(ctx context.Context, advertiserID int64, startDate, endDate string) ([]AdReport, error) {
	start, err := time.Parse("2006-01-02", normalizeStatDate(startDate))
	if err != nil {
		return nil, fmt.Errorf("开始日期格式错误: %w", err)
	}
	end, err := time.Parse("2006-01-02", normalizeStatDate(endDate))
	if err != nil {
		return nil, fmt.Errorf("结束日期格式错误: %w", err)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("结束日期早于开始日期")
	}

	reportFields := []string{
		"stat_cost", "total_pay_order_gmv_for_roi2", "total_pay_order_count_for_roi2",
	}
	sdkClient := ad_open_sdk_go.Init(c.cfg)
	reports := make([]AdReport, 0, int(end.Sub(start).Hours()/24)+1)

	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		var resp *models.QianchuanReportUniPromotionGetV10Response
		var rawResp *http.Response
		err := retryWithBackoff(ctx, fmt.Sprintf("全域推广报表API(advID=%d,date=%s)", advertiserID, date), func() error {
			var e error
			resp, rawResp, e = sdkClient.QianchuanReportUniPromotionGetV10Api().Get(ctx).
				AdvertiserId(advertiserID).
				StartDate(date + " 00:00:00").
				EndDate(date + " 23:59:59").
				Fields(reportFields).
				MarketingGoal(models.ALL_QianchuanReportUniPromotionGetV10MarketingGoal).
				OrderPlatform(models.ALL_QianchuanReportUniPromotionGetV10OrderPlatform).
				AccessToken(c.accessToken).
				Execute()
			return e
		})
		if err != nil {
			return nil, fmt.Errorf("获取全域推广报表失败(advertiserID=%d,date=%s): %w", advertiserID, date, err)
		}

		codeVal := int64(0)
		msgVal := ""
		if resp.Code != nil {
			codeVal = *resp.Code
		}
		if resp.Message != nil {
			msgVal = *resp.Message
		}
		httpStatus := 0
		if rawResp != nil {
			httpStatus = rawResp.StatusCode
		}
		g.Log().Infof(ctx, "[Qianchuan] 全域推广报表API: advID=%d, date=%s, HTTP=%d, code=%d, msg=%s",
			advertiserID, date, httpStatus, codeVal, msgVal)

		if codeVal != 0 {
			return nil, fmt.Errorf("全域推广报表返回错误: code=%d, msg=%s", codeVal, msgVal)
		}
		if resp.Data == nil {
			continue
		}

		r := AdReport{AdvertiserID: advertiserID, StatDatetime: date}
		if resp.Data.StatCost != nil {
			r.StatCost = *resp.Data.StatCost
		}
		if resp.Data.TotalPayOrderGmvForRoi2 != nil {
			r.PayOrderAmount = *resp.Data.TotalPayOrderGmvForRoi2
		}
		if resp.Data.TotalPayOrderCountForRoi2 != nil {
			r.PayOrderCount = int64(*resp.Data.TotalPayOrderCountForRoi2)
		}
		if r.StatCost != 0 || r.PayOrderAmount != 0 || r.PayOrderCount != 0 {
			reports = append(reports, r)
		}
	}

	g.Log().Infof(ctx, "[Qianchuan] 全域推广报表汇总(advID=%d): 共%d条(%s~%s)", advertiserID, len(reports), startDate, endDate)
	return reports, nil
}

// AuthorReport 达人维度日报表
type AuthorReport struct {
	AdvertiserID  int64
	AwemeID       int64
	AwemeName     string
	StatDatetime  string
	StatCost      float64
	PayOrderGmv   float64
	PayOrderCount int64
}

// GetAuthorDimensionReport 获取全域推广分维度数据(抖音号维度)
// 调用 /open_api/v1.0/qianchuan/report/uni_promotion/dimension_data/author/get/
// 该接口需要指定 aweme_id，本方法用于拉取单个达人的消耗数据。
func (c *Client) GetAuthorDimensionReport(ctx context.Context, advertiserID, awemeID int64, startDate, endDate string) (*AuthorReport, error) {
	reports, err := c.GetAuthorDimensionReports(ctx, advertiserID, awemeID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	if len(reports) == 0 {
		return nil, nil
	}

	total := &AuthorReport{
		AdvertiserID: advertiserID,
		AwemeID:      awemeID,
		AwemeName:    reports[0].AwemeName,
		StatDatetime: reports[0].StatDatetime,
	}
	for _, item := range reports {
		total.StatCost += item.StatCost
		total.PayOrderGmv += item.PayOrderGmv
		total.PayOrderCount += item.PayOrderCount
		if total.AwemeName == "" && item.AwemeName != "" {
			total.AwemeName = item.AwemeName
		}
	}
	if total.StatCost == 0 && total.PayOrderGmv == 0 {
		return nil, nil
	}
	return total, nil
}

// GetAuthorDimensionReports 获取全域推广分维度数据(抖音号维度)，按天返回明细
func (c *Client) GetAuthorDimensionReports(ctx context.Context, advertiserID, awemeID int64, startDate, endDate string) ([]AuthorReport, error) {
	sdkClient := ad_open_sdk_go.Init(c.cfg)
	metrics := []*models.QianchuanReportUniPromotionDimensionDataAuthorGetV10Metrics{
		models.STAT_COST_QianchuanReportUniPromotionDimensionDataAuthorGetV10Metrics.Ptr(),
		models.TOTAL_PAY_ORDER_GMV_FOR_ROI2_QianchuanReportUniPromotionDimensionDataAuthorGetV10Metrics.Ptr(),
		models.TOTAL_PAY_ORDER_COUNT_FOR_ROI2_QianchuanReportUniPromotionDimensionDataAuthorGetV10Metrics.Ptr(),
	}

	var resp *models.QianchuanReportUniPromotionDimensionDataAuthorGetV10Response
	var rawResp *http.Response
	err := retryWithBackoff(ctx, fmt.Sprintf("达人维度报表API(advID=%d,awemeID=%d)", advertiserID, awemeID), func() error {
		var e error
		resp, rawResp, e = sdkClient.QianchuanReportUniPromotionDimensionDataAuthorGetV10Api().Get(ctx).
			AccessToken(c.accessToken).
			AdvertiserId(advertiserID).
			AwemeId(awemeID).
			MarketingGoal(models.ALL_QianchuanReportUniPromotionDimensionDataAuthorGetV10MarketingGoal).
			Metrics(metrics).
			StartTime(startDate + " 00:00:00").
			EndTime(endDate + " 23:59:59").
			Dimension(models.TIME_GRANULARITY_DAILY_QianchuanReportUniPromotionDimensionDataAuthorGetV10Dimension).
			Page(1).
			PageSize(100).
			Execute()
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("获取达人维度报表失败(advID=%d,awemeID=%d): %w", advertiserID, awemeID, err)
	}

	codeVal := int64(0)
	msgVal := ""
	if resp.Code != nil {
		codeVal = *resp.Code
	}
	if resp.Message != nil {
		msgVal = *resp.Message
	}
	httpStatus := 0
	if rawResp != nil {
		httpStatus = rawResp.StatusCode
	}
	g.Log().Infof(ctx, "[Qianchuan] 达人维度报表API: advID=%d, awemeID=%d, HTTP=%d, code=%d, msg=%s",
		advertiserID, awemeID, httpStatus, codeVal, msgVal)

	if codeVal != 0 {
		return nil, fmt.Errorf("达人维度报表返回错误: code=%d, msg=%s", codeVal, msgVal)
	}
	if resp.Data == nil || resp.Data.List == nil || len(resp.Data.List) == 0 {
		return nil, nil
	}

	result := make([]AuthorReport, 0, len(resp.Data.List))
	for _, item := range resp.Data.List {
		report := AuthorReport{
			AdvertiserID: advertiserID,
			AwemeID:      awemeID,
		}
		if item.StatDatetime != nil {
			report.StatDatetime = *item.StatDatetime
		}
		if item.StatCost != nil {
			report.StatCost = *item.StatCost
		}
		if item.TotalPayOrderGmvForRoi2 != nil {
			report.PayOrderGmv = *item.TotalPayOrderGmvForRoi2
		}
		if item.TotalPayOrderCountForRoi2 != nil {
			report.PayOrderCount = int64(*item.TotalPayOrderCountForRoi2)
		}
		if report.StatCost == 0 && report.PayOrderGmv == 0 && report.PayOrderCount == 0 {
			continue
		}
		result = append(result, report)
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// GetAdPlanReport 获取广告计划日结报表（千川 v1.0/qianchuan/report/ad/get/）
// 计划维度：返回 ad_id + stat_cost 等基础消耗指标，商品/抖音号由 GetAllAdPlans 补齐。
func (c *Client) GetAdPlanReport(ctx context.Context, advertiserID int64, startDate, endDate string) ([]AdPlanReport, error) {
	page := int32(1)
	pageSize := int32(200)
	var allReports []AdPlanReport

	reportFields := []string{
		"stat_cost", "show_cnt", "click_cnt", "ctr", "convert_cnt", "convert_cost",
	}

	sdkClient := ad_open_sdk_go.Init(c.cfg)
	for {
		g.Log().Infof(ctx, "[Qianchuan] 调用千川计划报表API advID=%d page=%d, date=%s~%s", advertiserID, page, startDate, endDate)

		var resp *models.QianchuanReportAdGetV10Response
		var rawResp *http.Response
		err := retryWithBackoff(ctx, fmt.Sprintf("计划报表API(advID=%d)", advertiserID), func() error {
			var e error
			resp, rawResp, e = sdkClient.QianchuanReportAdGetV10Api().Get(ctx).
				AdvertiserId(advertiserID).
				StartDate(startDate).
				EndDate(endDate).
				Fields(reportFields).
				Filtering(models.QianchuanReportAdGetV10Filtering{
					MarketingGoal: models.ALL_QianchuanReportAdGetV10FilteringMarketingGoal,
				}).
				TimeGranularity(models.TIME_GRANULARITY_DAILY_QianchuanReportAdGetV10TimeGranularity).
				Page(page).
				PageSize(pageSize).
				AccessToken(c.accessToken).
				Execute()
			return e
		})
		if err != nil {
			return nil, fmt.Errorf("获取计划报表失败(advertiserID=%d): %w", advertiserID, err)
		}

		codeVal := int64(0)
		msgVal := ""
		if resp.Code != nil {
			codeVal = *resp.Code
		}
		if resp.Message != nil {
			msgVal = *resp.Message
		}
		listLen := 0
		if resp.Data != nil && resp.Data.List != nil {
			listLen = len(resp.Data.List)
		}
		httpStatus := 0
		if rawResp != nil {
			httpStatus = rawResp.StatusCode
		}

		g.Log().Infof(ctx, "[Qianchuan] 计划报表API: advID=%d, HTTP=%d, code=%d, msg=%s, list=%d",
			advertiserID, httpStatus, codeVal, msgVal, listLen)

		if codeVal != 0 {
			return nil, fmt.Errorf("计划报表返回错误: code=%d, msg=%s", codeVal, msgVal)
		}

		if resp.Data != nil && resp.Data.List != nil {
			for _, item := range resp.Data.List {
				r := AdPlanReport{AdvertiserID: advertiserID}
				if item.AdvertiserId != nil {
					r.AdvertiserID = *item.AdvertiserId
				}
				if item.AdId != nil {
					r.AdID = *item.AdId
				}
				if item.StatDatetime != nil {
					r.StatDatetime = normalizeStatDate(*item.StatDatetime)
				}
				if item.StatCost != nil {
					r.StatCost = *item.StatCost
				}
				if item.ShowCnt != nil {
					r.ShowCnt = int64(*item.ShowCnt)
				}
				if item.ClickCnt != nil {
					r.ClickCnt = int64(*item.ClickCnt)
				}
				if item.Ctr != nil {
					r.Ctr = *item.Ctr
				}
				if item.ConvertCnt != nil {
					r.ConvertCnt = int64(*item.ConvertCnt)
				}
				if item.ConvertCost != nil {
					r.ConvertCost = *item.ConvertCost
				}
				allReports = append(allReports, r)
			}
		}

		if resp.Data == nil || resp.Data.PageInfo == nil || resp.Data.PageInfo.TotalPage == nil || page >= int32(*resp.Data.PageInfo.TotalPage) {
			break
		}
		page++
	}

	g.Log().Infof(ctx, "[Qianchuan] 千川计划报表汇总(advID=%d): 共%d条(%s~%s)", advertiserID, len(allReports), startDate, endDate)
	return allReports, nil
}

// ==================== Step4: 组合数据结构 ====================

// PullResult 一次拉取的结果
type PullResult struct {
	AdvertiserID     int64               `json:"advertiser_id"`
	AdvertiserName   string              `json:"advertiser_name"`
	Company          string              `json:"company"`
	Plans            []AdPlan            `json:"-"`
	Reports          []AdReport          `json:"-"`
	PlanReports      []AdPlanDailyReport `json:"-"`
	Combined         []CombinedReport    `json:"combined"`
	TotalPlans       int                 `json:"total_plans"`
	TotalReports     int                 `json:"total_reports"`
	TotalPlanReports int                 `json:"total_plan_reports"`
	TotalCombined    int                 `json:"total_combined"`
}

// CombinedReport 账户维度日结报表数据
type CombinedReport struct {
	AdvertiserID            int64   `json:"advertiser_id"`
	StatDatetime            string  `json:"stat_datetime"`
	StatCost                float64 `json:"stat_cost"`
	ShowCnt                 int64   `json:"show_cnt"`
	ClickCnt                int64   `json:"click_cnt"`
	Ctr                     float64 `json:"ctr"`
	CpcPlatform             float64 `json:"cpc_platform"`
	CpmPlatform             float64 `json:"cpm_platform"`
	ConvertCnt              int64   `json:"convert_cnt"`
	ConvertCost             float64 `json:"convert_cost"`
	ConvertRate             float64 `json:"convert_rate"`
	DeepConvertCnt          int64   `json:"deep_convert_cnt"`
	DeepConvertCost         float64 `json:"deep_convert_cost"`
	PayOrderAmount          float64 `json:"pay_order_amount"`
	PayOrderCount           int64   `json:"pay_order_count"`
	PayOrderCostPerOrder    float64 `json:"pay_order_cost_per_order"`
	PayOrderCouponAmount    float64 `json:"pay_order_coupon_amount"`
	CreateOrderAmount       float64 `json:"create_order_amount"`
	CreateOrderCount        int64   `json:"create_order_count"`
	CreateOrderCouponAmount float64 `json:"create_order_coupon_amount"`
	CreateOrderRoi          float64 `json:"create_order_roi"`
	PrepayOrderAmount       float64 `json:"prepay_order_amount"`
	PrepayOrderCount        int64   `json:"prepay_order_count"`
	PrepayAndPayOrderRoi    float64 `json:"prepay_and_pay_order_roi"`
	AllOrderPayGmv7days     float64 `json:"all_order_pay_gmv_7days"`
	AllOrderPayCount7days   int64   `json:"all_order_pay_count_7days"`
	AllOrderPayRoi7days     float64 `json:"all_order_pay_roi_7days"`
	AllOrderRefundAmount7d  float64 `json:"all_order_refund_amount_7d"`
	AllOrderRefundCount7d   int64   `json:"all_order_refund_count_7d"`
	AllOrderRefundAmount14d float64 `json:"all_order_refund_amount_14d"`
	AllOrderRefundCount14d  int64   `json:"all_order_refund_count_14d"`
	DyFollow                int64   `json:"dy_follow"`
	TotalPlay               int64   `json:"total_play"`
}

// AdPlanDailyReport 计划维度日结报表，含计划基础信息和消耗指标。
type AdPlanDailyReport struct {
	AdvertiserID int64   `json:"advertiser_id"`
	AdID         int64   `json:"ad_id"`
	AdName       string  `json:"ad_name"`
	ProductID    int64   `json:"product_id"`
	ProductName  string  `json:"product_name"`
	AwemeID      string  `json:"aweme_id"`
	AwemeName    string  `json:"aweme_name"`
	AwemeShowID  string  `json:"aweme_show_id"`
	StatDatetime string  `json:"stat_datetime"`
	StatCost     float64 `json:"stat_cost"`
	ShowCnt      int64   `json:"show_cnt"`
	ClickCnt     int64   `json:"click_cnt"`
	Ctr          float64 `json:"ctr"`
	ConvertCnt   int64   `json:"convert_cnt"`
	ConvertCost  float64 `json:"convert_cost"`
}

// PullAll 拉取指定广告主的全量数据（计划列表 + 报表）
// days: 拉取最近多少天的报表数据
func (c *Client) PullAll(ctx context.Context, advertiserID int64, days int) (*PullResult, error) {
	if days <= 0 {
		days = 1
	}
	if days > 30 {
		days = 30
	}
	now := time.Now()
	startDate := now.AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	endDate := now.Format("2006-01-02")
	g.Log().Infof(ctx, "[Qianchuan] PullAll: advertiserID=%d, dateRange=%s~%s, token前8位=%s", advertiserID, startDate, endDate, c.accessToken[:min(8, len(c.accessToken))])
	return c.PullAllByDate(ctx, advertiserID, startDate, endDate)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func normalizeStatDate(statDatetime string) string {
	if len(statDatetime) >= 10 {
		return statDatetime[:10]
	}
	return statDatetime
}

// PullAllByDate 拉取指定广告主在指定日期范围内的日结报表（账户维度）
// 账户维度报表已包含所有汇总数据，不需要与计划合并
func (c *Client) PullAllByDate(ctx context.Context, advertiserID int64, startDate, endDate string) (*PullResult, error) {
	result := &PullResult{AdvertiserID: advertiserID}

	// 获取账户维度日结报表：标准推广 + 全域推广。
	reports, err := c.GetAdReport(ctx, advertiserID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("获取账户报表失败: %w", err)
	}
	uniReports, uniErr := c.GetUniPromotionReport(ctx, advertiserID, startDate, endDate)
	if uniErr != nil {
		g.Log().Warningf(ctx, "[Qianchuan] 获取全域推广报表失败(advID=%d): %v，将仅使用标准推广消耗", advertiserID, uniErr)
	} else {
		reports = mergeAdReports(reports, uniReports)
	}
	result.Reports = reports
	result.TotalReports = len(reports)

	// 直接将报表数据转为 CombinedReport（账户维度无需与计划合并）
	for _, r := range reports {
		result.Combined = append(result.Combined, CombinedReport{
			AdvertiserID:            advertiserID,
			StatDatetime:            r.StatDatetime,
			StatCost:                r.StatCost,
			ShowCnt:                 r.ShowCnt,
			ClickCnt:                r.ClickCnt,
			Ctr:                     r.Ctr,
			CpcPlatform:             r.CpcPlatform,
			CpmPlatform:             r.CpmPlatform,
			ConvertCnt:              r.ConvertCnt,
			ConvertCost:             r.ConvertCost,
			ConvertRate:             r.ConvertRate,
			DeepConvertCnt:          r.DeepConvertCnt,
			DeepConvertCost:         r.DeepConvertCost,
			PayOrderAmount:          r.PayOrderAmount,
			PayOrderCount:           r.PayOrderCount,
			PayOrderCostPerOrder:    r.PayOrderCostPerOrder,
			PayOrderCouponAmount:    r.PayOrderCouponAmount,
			CreateOrderAmount:       r.CreateOrderAmount,
			CreateOrderCount:        r.CreateOrderCount,
			CreateOrderCouponAmount: r.CreateOrderCouponAmount,
			CreateOrderRoi:          r.CreateOrderRoi,
			PrepayOrderAmount:       r.PrepayOrderAmount,
			PrepayOrderCount:        r.PrepayOrderCount,
			PrepayAndPayOrderRoi:    r.PrepayAndPayOrderRoi,
			AllOrderPayGmv7days:     r.AllOrderPayGmv7days,
			AllOrderPayCount7days:   r.AllOrderPayCount7days,
			AllOrderPayRoi7days:     r.AllOrderPayRoi7days,
			AllOrderRefundAmount7d:  r.AllOrderRefundAmount7d,
			AllOrderRefundCount7d:   r.AllOrderRefundCount7d,
			AllOrderRefundAmount14d: r.AllOrderRefundAmount14d,
			AllOrderRefundCount14d:  r.AllOrderRefundCount14d,
			DyFollow:                r.DyFollow,
			TotalPlay:               r.TotalPlay,
		})
	}
	result.TotalCombined = len(result.Combined)

	planReports, planReportErr := c.GetAdPlanReport(ctx, advertiserID, startDate, endDate)
	if planReportErr != nil {
		g.Log().Warningf(ctx, "[Qianchuan] 获取计划报表失败(advID=%d): %v", advertiserID, planReportErr)
	} else {
		var totalPlans int
		result.PlanReports, totalPlans = c.combinePlanReports(ctx, advertiserID, planReports)
		result.TotalPlanReports = len(result.PlanReports)
		result.TotalPlans = totalPlans
	}

	g.Log().Infof(ctx, "[Qianchuan] PullAll结果: reports=%d, planReports=%d, combined=%d(%s~%s)", result.TotalReports, result.TotalPlanReports, result.TotalCombined, startDate, endDate)
	return result, nil
}

func mergeAdReports(standardReports, uniReports []AdReport) []AdReport {
	reportMap := make(map[string]AdReport, len(standardReports)+len(uniReports))
	order := make([]string, 0, len(standardReports)+len(uniReports))

	for _, r := range standardReports {
		r.StatDatetime = normalizeStatDate(r.StatDatetime)
		key := fmt.Sprintf("%d:%s", r.AdvertiserID, r.StatDatetime)
		if _, ok := reportMap[key]; !ok {
			order = append(order, key)
		}
		reportMap[key] = r
	}

	for _, r := range uniReports {
		r.StatDatetime = normalizeStatDate(r.StatDatetime)
		key := fmt.Sprintf("%d:%s", r.AdvertiserID, r.StatDatetime)
		existing, ok := reportMap[key]
		if !ok {
			order = append(order, key)
			reportMap[key] = r
			continue
		}
		existing.StatCost += r.StatCost
		existing.PayOrderAmount += r.PayOrderAmount
		existing.PayOrderCount += r.PayOrderCount
		reportMap[key] = existing
	}

	merged := make([]AdReport, 0, len(reportMap))
	for _, key := range order {
		merged = append(merged, reportMap[key])
	}
	return merged
}

func (c *Client) combinePlanReports(ctx context.Context, advertiserID int64, reports []AdPlanReport) ([]AdPlanDailyReport, int) {
	plans, err := c.GetAllAdPlans(ctx, advertiserID)
	if err != nil {
		g.Log().Warningf(ctx, "[Qianchuan] 获取计划基础信息失败(advID=%d): %v，计划明细将只保留ad_id和消耗", advertiserID, err)
	}

	planMap := make(map[int64]AdPlan, len(plans))
	for _, p := range plans {
		planMap[p.AdID] = p
	}

	result := make([]AdPlanDailyReport, 0, len(reports))
	for _, r := range reports {
		item := AdPlanDailyReport{
			AdvertiserID: r.AdvertiserID,
			AdID:         r.AdID,
			StatDatetime: r.StatDatetime,
			StatCost:     r.StatCost,
			ShowCnt:      r.ShowCnt,
			ClickCnt:     r.ClickCnt,
			Ctr:          r.Ctr,
			ConvertCnt:   r.ConvertCnt,
			ConvertCost:  r.ConvertCost,
		}
		if p, ok := planMap[r.AdID]; ok {
			item.AdName = p.AdName
			item.ProductID = p.ProductID
			item.ProductName = p.ProductName
			item.AwemeID = p.AwemeID
			item.AwemeName = p.AwemeName
			item.AwemeShowID = p.AwemeShowID
		}
		result = append(result, item)
	}
	return result, len(plans)
}

// ==================== Token 管理 ====================

// GetAccessToken 返回当前使用的 access_token
func (c *Client) GetAccessToken() string {
	return c.accessToken
}

// retryWithBackoff 对 fn 执行指数退避重试，最多3次
// 只重试返回error的情况，不重试业务错误(code!=0)
func retryWithBackoff(ctx context.Context, name string, fn func() error) error {
	const maxRetries = 3
	backoffs := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			g.Log().Warningf(ctx, "[Qianchuan] %s 第%d次重试(backoff=%v)", name, i, backoffs[i-1])
			time.Sleep(backoffs[i-1])
		}
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
	}
	return fmt.Errorf("%s 重试%d次后仍失败: %w", name, maxRetries, lastErr)
}

// GetShopIDs 获取巨量引擎平台广告主ID列表（即 shop_id）
func GetShopIDs(ctx context.Context, accessToken string) ([]int64, error) {
	return getAdvertiserIDs(ctx, accessToken)
}

// ResolveQianchuanAdvertiserIDs 将巨量引擎广告主ID（shop_id）解析为千川广告主ID
// 通过 qianchuan/shop/advertiser/list/ 接口查询每个 shop 下的千川广告主
func ResolveQianchuanAdvertiserIDs(ctx context.Context, accessToken string, shopIDs []int64) ([]int64, error) {
	if len(shopIDs) == 0 {
		return nil, nil
	}

	client := g.Client()
	var allAdvIDs []int64
	uniqueMap := make(map[int64]bool)

	for _, shopId := range shopIDs {
		g.Log().Infof(ctx, "[Qianchuan] 解析店铺%d下的千川广告主", shopId)

		page := int64(1)
		pageSize := int64(100)
		for {
			// 注意：千川shop/advertiser/list API要求通过 Access-Token 请求头传递token
			// （不是URL query参数，也不是Content-Type: application/json + g.Map）
			apiURL := fmt.Sprintf("https://api.oceanengine.com/open_api/v1.0/qianchuan/shop/advertiser/list/?shop_id=%d&page=%d&page_size=%d",
				shopId, page, pageSize)
			resp, err := client.SetHeader("Access-Token", accessToken).Get(ctx, apiURL)
			if err != nil {
				g.Log().Warningf(ctx, "[Qianchuan] 查询店铺%d失败: %v", shopId, err)
				break
			}
			body := resp.ReadAllString()
			resp.Close()

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
			if err := json.Unmarshal([]byte(body), &listResp); err != nil {
				g.Log().Warningf(ctx, "[Qianchuan] 解析店铺%d响应失败: %v, body=%s", shopId, err, body[:min(len(body), 200)])
				break
			}
			if listResp.Code != 0 {
				g.Log().Warningf(ctx, "[Qianchuan] 店铺%d返回错误: code=%d, msg=%s", shopId, listResp.Code, listResp.Message)
				break
			}

			for _, advId := range listResp.Data.List {
				if !uniqueMap[advId] {
					uniqueMap[advId] = true
					allAdvIDs = append(allAdvIDs, advId)
				}
			}
			g.Log().Infof(ctx, "[Qianchuan] 店铺%d 第%d页: 找到%d个千川广告主", shopId, page, len(listResp.Data.List))

			if listResp.Data.PageInfo.TotalPage <= 0 || page >= listResp.Data.PageInfo.TotalPage {
				break
			}
			page++
		}
	}

	g.Log().Infof(ctx, "[Qianchuan] 广告主解析完成: 共%d个千川广告主", len(allAdvIDs))
	return allAdvIDs, nil
}

// RefreshAccessToken 使用 refresh_token 续期 access_token
// 调用巨量千川 /open_api/oauth2/renew_token/ 接口
func RefreshAccessToken(ctx context.Context, refreshToken string) (*models.Oauth2RenewTokenResponse, error) {
	cfg := g.Cfg()
	appId := cfg.MustGet(ctx, "platform.qianchuan.appId").Int64()
	secret := cfg.MustGet(ctx, "platform.qianchuan.appSecret").String()

	g.Log().Infof(ctx, "[Qianchuan] RefreshAccessToken: appId=%d, refreshToken前%d位=%s", appId, min(8, len(refreshToken)), refreshToken[:min(8, len(refreshToken))])

	sdkClient := ad_open_sdk_go.Init(config.NewConfiguration())
	resp, _, err := sdkClient.Oauth2RenewTokenApi().Post(ctx).
		Oauth2RenewTokenRequest(models.Oauth2RenewTokenRequest{
			AppId:        appId,
			RefreshToken: refreshToken,
			Secret:       secret,
		}).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("续期token失败: %w", err)
	}
	if resp.Code != nil && *resp.Code != 0 {
		return nil, fmt.Errorf("续期token失败: code=%d, msg=%s", *resp.Code, *resp.Message)
	}

	g.Log().Infof(ctx, "[Qianchuan] RefreshAccessToken成功")
	return resp, nil
}
