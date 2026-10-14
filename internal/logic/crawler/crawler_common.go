package crawler

import (
	"context"
	"fmt"
	"math"
	netUrl "net/url"
	"strings"
	"time"
)

// retryWithBackoff 带指数级退避的重试函数
func (s *sCrawler) retryWithBackoff(ctx context.Context, fn func() error, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if i < maxRetries-1 {
				// 指数级退避：1秒、2秒、4秒
				wait := time.Duration(math.Pow(2, float64(i))) * time.Second
				if err := waitForCrawler(ctx, wait); err != nil {
					return err
				}
			}
		}
	}
	return lastErr
}

// ProductInfo 商品基本信息
type ProductInfo struct {
	Rank             int    // 排名
	ProductID        string // 商品 ID（从 API 响应获取）
	ProductName      string // 商品名称
	ImageURL         string // 商品主图地址（罗盘 product_info.image_url）
	ShopID           string // 账号 ID
	ShopName         string // 账号名称
	PaymentAmount    string // 短视频用户支付金额
	ClickCount       string // 短视频点击次数
	ClickConvertRate string // 点击成交转化率
	SalesVolume      string // 短视频成交件数
	DeliveryVideoNum string // 带货视频数
	ProductQRCodeURL string // 商品二维码
	ShopLogoURL      string // 店铺 Logo
	ShopQRCodeURL    string // 店铺二维码
	PreviousRank     *int   // 上期排名（优先官方返回，缺失时入库前补算）
	RankDelta        *int   // 排名变化：正数=上升，负数=下降
	RankTrendSource  string // 排名变化来源：official/fallback/unknown
	ProductURL       string // 商品详情页 URL
	FirstList        int    // 是否首次上榜：0-否 1-是
	PriceRange       string // 价格带
}

// buildDetailURL 根据 product_id 和 shop_id 构造罗盘商品详情页 URL
func buildDetailURL(productID, shopID string) string {
	return fmt.Sprintf(
		"https://compass.jinritemai.com/shop/chance/rank-product/detail?product_id=%s&product_shop_id=%s",
		productID, shopID,
	)
}

func buildVideoAnalysisURL(product ProductInfo) string {
	rawURL := ""
	if product.ProductID != "" && product.ShopID != "" {
		rawURL = buildDetailURL(product.ProductID, product.ShopID)
	} else {
		rawURL = product.ProductURL
	}
	if rawURL == "" && product.ProductID != "" {
		rawURL = fmt.Sprintf("https://compass.jinritemai.com/shop/chance/rank-product/detail?product_id=%s", netUrl.QueryEscape(product.ProductID))
	}
	if u, err := netUrl.Parse(rawURL); err == nil {
		q := u.Query()
		q.Set("tab", "video")
		u.RawQuery = q.Encode()
		return u.String()
	}
	return rawURL
}

func isVideoAnalysisAPIURL(rawURL, productID string) bool {
	u, err := netUrl.Parse(rawURL)
	if err != nil || !strings.HasSuffix(strings.ToLower(u.Hostname()), "jinritemai.com") {
		return false
	}
	lowerPath := strings.ToLower(u.Path)
	if strings.Contains(lowerPath, "rank-product/detail") || strings.Contains(lowerPath, "product-detail") {
		return false
	}
	q := u.Query()
	requestProductID := q.Get("product_id")
	if productID != "" && requestProductID != "" && requestProductID != productID {
		return false
	}
	return q.Get("source_entrance") == "video" || strings.Contains(lowerPath, "flow_more") ||
		(strings.Contains(lowerPath, "video") && (requestProductID != "" || q.Get("target_id") != ""))
}

func isCompetitorAccessControlAPIURL(rawURL string) bool {
	u, err := netUrl.Parse(rawURL)
	if err != nil || !strings.HasSuffix(strings.ToLower(u.Hostname()), "jinritemai.com") {
		return false
	}
	lowerPath := strings.ToLower(u.Path)
	return strings.Contains(lowerPath, "/compass_api/shop/common/competition/get_view_quota") ||
		strings.Contains(lowerPath, "/compass_api/shop/common/competition/get_display_rule")
}

// ProductDetail 商品详情数据（参考 Python 爬虫提取的字段）
type ProductDetail struct {
	ProductID            string // 商品 ID
	ProductName          string // 商品名称
	FirstShelfTime       string // 首次上架时间
	MinPrice             string // 最低到手价
	PriceRange           string // 价格带
	HasShippingInsurance *bool  // 是否有运费险
	DetailAccessStatus   string // 详情访问状态：available/removed/category_mismatch/restricted/daily_limit
	DetailAccessNote     string // 详情访问备注
}

// VideoAnalysis 视频分析数据
type VideoAnalysis struct {
	ProductID         string // 商品 ID
	ProductName       string // 商品名称
	VideoId           string // 视频 ID
	VideoDesc         string // 短视频信息
	AuthorName        string // 作者名称
	CoverURL          string // 封面地址
	PublishTime       string // 发布时间
	SourceURL         string // 抖音视频页地址
	PaymentAmount     string // 用户支付金额
	ExposeUserCount   string // 商品曝光人数
	ClickUserCount    string // 商品点击人数
	ConvertUserCount  string // 商品成交人数
	ClickRate         string // 商品点击率 (人数)
	ClickConvertRate  string // 商品点击 - 成交率 (人数)
	ExposeConvertRate string // 商品曝光 - 成交率 (人数)
	OrderCount        string // 商品成交订单数
	SalesVolume       string // 商品成交件数
}

type videoPageAccessStatus string

const (
	videoPageAccessAvailable  videoPageAccessStatus = "available"
	videoPageAccessRestricted videoPageAccessStatus = "restricted"
	videoPageAccessDailyLimit videoPageAccessStatus = "daily_limit"
	videoPageAccessNoData     videoPageAccessStatus = "no_data"
)

type videoPageResult struct {
	Err         error
	Videos      []VideoAnalysis
	Status      videoPageAccessStatus
	Note        string
	Diagnostics []VideoResponseDiagnostic
}

type videoScrapeResult struct {
	Err              error
	Videos           []VideoAnalysis
	Completed        bool
	Paused           bool
	NextProductIndex int
	StopReason       string
}

type VideoResponseDiagnostic struct {
	ProductID        string         `json:"product_id"`
	ResponsePath     string         `json:"response_path"`
	QueryKeys        []string       `json:"query_keys"`
	BodySize         int            `json:"body_size"`
	JSONValid        bool           `json:"json_valid"`
	BusinessCode     string         `json:"business_code"`
	Message          string         `json:"message"`
	TopLevelKeys     []string       `json:"top_level_keys"`
	DataKeys         []string       `json:"data_keys"`
	ArrayLengths     map[string]int `json:"array_lengths"`
	HasVideoID       bool           `json:"has_video_id"`
	HasAwemeID       bool           `json:"has_aweme_id"`
	HasVideoList     bool           `json:"has_video_list"`
	HasDownloadURL   bool           `json:"has_download_url"`
	RestrictionNote  string         `json:"restriction_note"`
	ParsedVideoCount int            `json:"parsed_video_count"`
	RemainingCount   string         `json:"remaining_count"`
	TargetViewable   string         `json:"target_viewable"`
	DisplayType      string         `json:"display_type"`
}

type sCrawler struct{}

func init() {
	// 初始化日志（使用 logger.go 中的 SimpleLogger）
	logger := GetLogger()
	logger.Info(context.Background(), "🚀 爬虫服务初始化")
	fmt.Printf("✅ 日志系统初始化完成 | 目录：logs/\n")
	fmt.Printf("📝 日志级别：ALL | 控制台输出：开启 | 自动轮转：开启\n")
}

func formatLog(ctx context.Context, content string, level string) string {
	return fmt.Sprintf(
		"[%s] [%s] %s",
		time.Now().Format("2006-01-02 15:04:05.000"),
		strings.ToUpper(level),
		content,
	)
}

func New() *sCrawler {
	return &sCrawler{}
}

// CrawlReq 爬虫请求
type CrawlReq struct {
	TaskId    string `json:"task_id"`
	TargetUrl string `json:"target_url"`
	LoginType string `json:"login_type"` // "qrcode" or "password"
	Username  string `json:"username"`
	Password  string `json:"password"`
}

// Result 爬虫结果
type Result struct {
	URL            string            `json:"url"`
	Title          string            `json:"title"`
	Content        string            `json:"content"`
	ProductCount   int               `json:"product_count"`
	VideoCount     int               `json:"video_count"`
	Paused         bool              `json:"paused,omitempty"`
	NextIndex      int               `json:"next_index,omitempty"`
	StopReason     string            `json:"stop_reason,omitempty"`
	Products       []ProductInfo     `json:"products,omitempty"`
	ProductDetails []ProductDetail   `json:"product_details,omitempty"`
	VideoAnalysis  []VideoAnalysis   `json:"video_analysis,omitempty"`
	ExportFiles    map[string]string `json:"export_files,omitempty"`
}
