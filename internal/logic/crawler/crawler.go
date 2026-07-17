package crawler

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	netUrl "net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/grand"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

// ===== 并发处理工具类 =====

const crawlerWorkerConcurrency = 1

// WorkerPool 工作池实现（基于 semaphore channel）
type WorkerPool struct {
	sem chan struct{}
	wg  sync.WaitGroup
}

// NewWorkerPool 创建新的工作池
func NewWorkerPool(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		sem: make(chan struct{}, maxWorkers),
	}
}

// Submit 提交任务到工作池
func (wp *WorkerPool) Submit(fn func()) {
	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		wp.sem <- struct{}{}        // 获取许可证
		defer func() { <-wp.sem }() // 释放许可证
		fn()
	}()
}

// Wait 等待所有任务完成
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

// ===== 重试函数 =====

// retryWithBackoff 带指数级退避的重试函数
func (s *sCrawler) retryWithBackoff(fn func() error, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if i < maxRetries-1 {
				// 指数级退避：1秒、2秒、4秒
				wait := time.Duration(math.Pow(2, float64(i))) * time.Second
				time.Sleep(wait)
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
	ShopID           string // 账号 ID
	ShopName         string // 账号名称
	PaymentAmount    string // 短视频用户支付金额
	ClickCount       string // 短视频点击次数
	SalesVolume      string // 短视频成交件数
	DeliveryVideoNum string // 带货视频数
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
	DetailAccessStatus   string // 详情访问状态：available/removed/category_mismatch
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
	Videos      []VideoAnalysis
	Status      videoPageAccessStatus
	Note        string
	Diagnostics []VideoResponseDiagnostic
}

type videoScrapeResult struct {
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

// RunBrowserTask 核心爬虫方法 - 完整实现需求流程（含优化）
func (s *sCrawler) RunBrowserTask(ctx context.Context, req *CrawlReq) (res []Result, err error) {
	res = make([]Result, 0)
	taskId := strings.TrimSpace(req.TaskId)
	if taskId == "" {
		taskId = grand.S(16)
	} else if matched, _ := regexp.MatchString(`^[A-Za-z0-9_-]{1,64}$`, taskId); !matched {
		return res, gerror.New("任务 ID 只能包含字母、数字、下划线和连字符")
	}
	if req.LoginType == "" {
		req.LoginType = "qrcode"
	}

	// ==================== 初始化优化组件 ====================
	cookieManager := service.NewCookieManager("cookies")
	checkpointManager := service.NewCheckpointManager("checkpoints")
	exportManager := service.NewExportManager("exports")

	// 注册全局爬虫服务状态（支持外部查询和强制下线）
	statusMgr := service.GetCrawlerStatusManager()

	// 检查是否有未完成的任务（断点续爬）
	existingTask, _ := checkpointManager.LoadCheckpoint(ctx, taskId)
	if existingTask != nil && existingTask.Status == "running" {
		logInfo(ctx, taskId, fmt.Sprintf("🔄 发现未完成的任务，尝试从第 %d 页恢复...", existingTask.CurrentPage))
	}

	defer func() {
		if r := recover(); r != nil {
			panicErr := fmt.Errorf("爬虫任务 panic：%v", r)
			logError(ctx, taskId, fmt.Sprintf("💥 爬虫任务异常终止 | 错误：%+v", panicErr))
			// 保存失败状态
			checkpointManager.FailTask(ctx, taskId, panicErr.Error())
			err = panicErr
		}
	}()

	// checkCancelled 检查 context 是否已取消（用于强制下线）
	checkCancelled := func() bool {
		select {
		case <-ctx.Done():
			logWarn(ctx, taskId, "⚠️ 任务已被取消（强制下线）")
			checkpointManager.FailTask(ctx, taskId, "强制下线")
			return true
		default:
			return false
		}
	}

	// ==================== 步骤 1: 尝试 Cookie 自动登录（有头模式） ====================
	homeURL := "https://compass.jinritemai.com/shop"
	var browser *rod.Browser
	var page *rod.Page
	isLoggedIn := false

	// 检查是否有已保存的 Cookie
	hasSavedCookies := cookieManager.HasCompassCookies(ctx, taskId)

	if hasSavedCookies {
		logInfo(ctx, taskId, "🚀 步骤 1: 检测到已保存的 Cookie，以有头模式启动浏览器尝试自动登录")
		browser, err = s.startBrowser(ctx, taskId, false)
		if err != nil {
			checkpointManager.FailTask(ctx, taskId, err.Error())
			return res, gerror.Wrap(err, "浏览器启动失败")
		}

		page, err = browser.Page(proto.TargetCreateTarget{URL: homeURL})
		if err != nil {
			browser.Close()
			checkpointManager.FailTask(ctx, taskId, err.Error())
			return res, gerror.Wrap(err, "打开首页失败")
		}
		_ = page.WaitLoad()

		// 注入 Cookie 并刷新页面
		if s.injectCookies(ctx, page, cookieManager, taskId) {
			isLoggedIn = s.checkIsLoggedIn(ctx, page, taskId)
		}

		if isLoggedIn {
			logSuccess(ctx, taskId, "✅ Cookie 自动登录成功！继续使用当前有头浏览器")
			// 更新 Cookie（刷新过期时间）
			allCookies, _ := page.Cookies(nil)
			if len(allCookies) > 0 {
				cookieManager.SaveCookies(ctx, allCookies, taskId)
				logInfo(ctx, taskId, "💾 Cookie 已更新")
			}
		} else {
			logWarn(ctx, taskId, "⚠️ Cookie 已过期或无效，重新启动登录浏览器")
			page = nil
			browser.Close()
			browser = nil
		}
	} else {
		logInfo(ctx, taskId, "🚀 步骤 1: 无已保存的 Cookie，需要人工登录")
	}

	// ==================== 步骤 2: 需要人工登录时，以有头模式启动 ====================
	if !isLoggedIn {
		logInfo(ctx, taskId, "🖥️ 步骤 2: 以有头模式（显示浏览器窗口）启动，等待人工登录")
		headedBrowser, startErr := s.startBrowser(ctx, taskId, false)
		if startErr != nil {
			checkpointManager.FailTask(ctx, taskId, startErr.Error())
			return res, gerror.Wrap(startErr, "有头浏览器启动失败")
		}

		headedPage, pageErr := headedBrowser.Page(proto.TargetCreateTarget{URL: homeURL})
		if pageErr != nil {
			headedBrowser.Close()
			checkpointManager.FailTask(ctx, taskId, pageErr.Error())
			return res, gerror.Wrap(pageErr, "打开首页失败")
		}
		_ = headedPage.WaitLoad()

		// 执行登录（等待用户扫码或输入账号密码）
		logInfo(ctx, taskId, "📝 步骤 3: 请在浏览器窗口中完成登录（扫码或账号密码）")
		loginSuccess := s.performLogin(ctx, headedPage, taskId, req.LoginType, req.Username, req.Password)
		// 凭据只用于本次登录，使用后立即从请求对象中清空。
		req.Username, req.Password = "", ""
		if !loginSuccess {
			headedBrowser.Close()
			checkpointManager.FailTask(ctx, taskId, "登录失败")
			return res, gerror.New("登录失败，请检查账号密码或重新尝试")
		}
		logSuccess(ctx, taskId, "✅ 登录成功")

		// 保存 Cookie
		allCookies, _ := headedPage.Cookies(nil)
		if err := cookieManager.SaveCookies(ctx, allCookies, taskId); err != nil {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 保存 Cookie 失败：%v", err))
		} else {
			logInfo(ctx, taskId, fmt.Sprintf("💾 已保存 %d 个 Cookie 到文件", len(allCookies)))
		}

		// 全程复用完成登录的同一个浏览器和页面，避免切换环境触发风控。
		browser = headedBrowser
		page = headedPage
		isLoggedIn = true
		logSuccess(ctx, taskId, "✅ 保持当前有头浏览器，开始 UI 采集")
	}
	defer browser.Close()

	// 更新状态消息
	statusMgr.SetMessage("爬取数据中")

	// 检查是否已取消
	if checkCancelled() {
		return res, gerror.New("任务已取消")
	}

	// ==================== 步骤 4: 提前注册浏览器原生响应监听 ====================
	logInfo(ctx, taskId, "📊 步骤 4: 监听浏览器原生榜单响应并扩大 Inspector 缓存")

	// 扩大 Inspector Cache 容量到 100MB，防止响应体被驱逐导致 "evicted from inspector cache"
	cacheSize := 100 * 1024 * 1024
	networkEnable := proto.NetworkEnable{MaxTotalBufferSize: &cacheSize}
	if err := networkEnable.Call(page); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 启用网络追踪失败：%v", err))
	}

	// 初始化分页参数
	startPage := 1
	videoStartIndex := 0
	resumeVideoPhase := false
	if existingTask != nil && existingTask.CurrentStep >= 8 && existingTask.CurrentProductIndex > 0 &&
		(existingTask.Status == "paused" || existingTask.Status == "running") {
		videoStartIndex = existingTask.CurrentProductIndex
		resumeVideoPhase = true
		logInfo(ctx, taskId, fmt.Sprintf("🔄 从视频采集断点继续：商品索引=%d，已抓视频=%d 条", videoStartIndex, existingTask.VideoCount))
	} else if existingTask != nil && existingTask.Status == "running" {
		startPage = existingTask.CurrentPage
		logInfo(ctx, taskId, fmt.Sprintf("🔄 从第 %d 页继续爬取", startPage))
	}

	// 只监听当前有头浏览器已完成的原生响应，不用 Go HTTP 重放罗盘请求。
	stopObserver, pageCh := s.setupBrowserResponseObserver(ctx, page, taskId)
	defer stopObserver()

	logInfo(ctx, taskId, "✅ API 拦截器已就绪，开始导航")

	// ==================== 步骤 5: 导航到商品榜单页面 ====================
	logInfo(ctx, taskId, "📊 步骤 5: 导航到商品榜单页面")

	productRankURL := "https://compass.jinritemai.com/shop/chance/product-rank"
	if err := page.Navigate(productRankURL); err != nil {
		checkpointManager.FailTask(ctx, taskId, err.Error())
		return res, gerror.Wrap(err, "跳转到商品榜单失败")
	}
	page.WaitLoad()
	time.Sleep(3 * time.Second)

	// ==================== 步骤 6: 点击短视频榜 Tab ====================
	logInfo(ctx, taskId, "🎬 步骤 6: 点击短视频榜 Tab")
	if err := s.clickShortVideoTab(ctx, page, taskId); err != nil {
		checkpointManager.FailTask(ctx, taskId, err.Error())
		return res, gerror.Wrap(err, "点击短视频榜失败")
	}
	time.Sleep(2 * time.Second)

	if checkCancelled() {
		return res, gerror.New("任务已取消")
	}

	// ==================== 步骤 7: 分页抓取 200 条数据（支持断点续爬） ====================
	logInfo(ctx, taskId, "📈 步骤 7: 分页抓取商品榜单数据（20 页×10 条=200 条）")

	var products []ProductInfo
	if resumeVideoPhase {
		products, err = s.loadProductsFromDB(ctx, taskId)
		if err != nil {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 从数据库加载断点商品失败，将重新采集榜单：%v", err))
		}
		if len(products) > 0 {
			logInfo(ctx, taskId, fmt.Sprintf("✅ 从数据库加载 %d 条商品，跳过榜单重抓，直接恢复视频采集", len(products)))
		}
	}
	if len(products) == 0 {
		products = s.collectInterceptedData(ctx, page, taskId, checkpointManager, startPage, pageCh)
	}
	if len(products) == 0 {
		checkpointManager.FailTask(ctx, taskId, "未抓取到数据")
		logWarn(ctx, taskId, "⚠️ 未抓取到有效数据")
		return res, gerror.New("未抓取到商品数据")
	}
	logInfo(ctx, taskId, fmt.Sprintf("✅ 抓取完成，共获取 %d 条商品数据", len(products)))

	if checkCancelled() {
		return res, gerror.New("任务已取消")
	}

	// ==================== 步骤 8: 优先获取视频数据 ====================
	// 每个商品解析成功后立即入库并排队解析视频直链，避免长任务中途停止丢失已抓数据。
	logInfo(ctx, taskId, "📹 步骤 8: 通过页面 UI 获取视频，发现后立即入库并解析下载直链")
	videoScrape := s.scrapeVideoAnalysisDOMWithSinkFrom(ctx, browser, products, taskId, videoStartIndex, checkpointManager, func(videos []VideoAnalysis) {
		if err := s.persistAndQueueVideos(ctx, taskId, videos); err != nil {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 视频即时入库失败：%v", err))
		}
	})
	videoAnalysisList := videoScrape.Videos
	logInfo(ctx, taskId, fmt.Sprintf("✅ 视频分析抓取完成，共获取 %d 条视频数据", len(videoAnalysisList)))
	if videoScrape.Paused {
		_ = checkpointManager.UpdateVideoProgress(ctx, taskId, videoScrape.NextProductIndex, len(products), s.countVideoRows(ctx, taskId), "paused", videoScrape.StopReason)
		logWarn(ctx, taskId, fmt.Sprintf("⏸️ 本轮视频采集已暂停：next_index=%d/%d，原因=%s", videoScrape.NextProductIndex, len(products), videoScrape.StopReason))

		modelProducts := s.convertToModelProducts(products, nil)
		modelVideos := s.convertToModelVideos(taskId, videoAnalysisList)
		exportFiles, exportErr := exportManager.ExportAll(ctx, taskId, modelProducts, modelVideos)
		if exportErr != nil {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 暂停前导出数据失败：%v", exportErr))
		} else {
			logInfo(ctx, taskId, fmt.Sprintf("✅ 暂停前已导出数据：%d 个文件", len(exportFiles)))
		}
		_ = s.updateCrawlTaskSummary(ctx, taskId, "paused", exportFiles)
		res = append(res, Result{
			URL:           productRankURL,
			Title:         fmt.Sprintf("抖音商品短视频榜_暂停_%d 条商品_%d 条视频", len(products), s.countVideoRows(ctx, taskId)),
			Content:       fmt.Sprintf("本轮达到视频详情查看上限，已暂停；下次将从第 %d 个商品继续", videoScrape.NextProductIndex+1),
			ProductCount:  len(products),
			VideoCount:    s.countVideoRows(ctx, taskId),
			Paused:        true,
			NextIndex:     videoScrape.NextProductIndex,
			StopReason:    videoScrape.StopReason,
			Products:      products,
			VideoAnalysis: videoAnalysisList,
			ExportFiles:   exportFiles,
		})
		return res, nil
	}
	_ = checkpointManager.UpdateVideoProgress(ctx, taskId, len(products), len(products), s.countVideoRows(ctx, taskId), "running", "")

	if checkCancelled() {
		return res, gerror.New("任务已取消")
	}

	// ==================== 步骤 9: 获取商品详情数据 ====================
	productDetails := make([]ProductDetail, 0)
	if g.Cfg().MustGet(ctx, "crawler.productDetail.enabled", false).Bool() {
		limit := g.Cfg().MustGet(ctx, "crawler.productDetail.maxViews", 9).Int()
		detailProducts := limitProductsForDetail(products, limit)
		if len(detailProducts) < len(products) {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 商品详情查看存在官方额度限制，本次仅访问前 %d/%d 个商品详情", len(detailProducts), len(products)))
		}
		logInfo(ctx, taskId, "🔍 步骤 9: 获取商品详情数据（首次上架时间、最低到手价）")
		productDetails = s.scrapeProductDetails(ctx, browser, detailProducts, taskId)
		logInfo(ctx, taskId, fmt.Sprintf("✅ 详情抓取完成，共获取 %d 条详情数据", len(productDetails)))
	} else {
		logInfo(ctx, taskId, "⏭️ 步骤 9: 已跳过商品详情访问，避免消耗官方详情查看次数")
	}
	_ = checkpointManager.UpdateProgress(ctx, taskId, 9, 20, len(products))

	if checkCancelled() {
		return res, gerror.New("任务已取消")
	}

	// ==================== 步骤 10: 保存数据到数据库 ====================
	logInfo(ctx, taskId, "💾 步骤 10: 保存数据到数据库")
	if err := s.saveAllData(ctx, taskId, products, productDetails, nil); err != nil {
		checkpointManager.FailTask(ctx, taskId, err.Error())
		logError(ctx, taskId, fmt.Sprintf("❌ 数据入库失败：%v", err))
		return res, gerror.Wrap(err, "保存数据失败")
	}
	logInfo(ctx, taskId, "✅ 数据入库成功")

	// ==================== 步骤 11: 导出数据文件 ====================
	logInfo(ctx, taskId, "📥 步骤 11: 导出数据文件（CSV/Excel）")
	modelProducts := s.convertToModelProducts(products, productDetails)
	modelVideos := s.convertToModelVideos(taskId, videoAnalysisList)
	exportFiles, err := exportManager.ExportAll(ctx, taskId, modelProducts, modelVideos)
	if err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 数据导出失败：%v", err))
	} else {
		logInfo(ctx, taskId, fmt.Sprintf("✅ 数据导出成功：%d 个文件", len(exportFiles)))
		for key, file := range exportFiles {
			logInfo(ctx, taskId, fmt.Sprintf("  - %s: %s", key, file))
		}
	}

	// 标记任务完成
	checkpointManager.CompleteTask(ctx, taskId, len(products))
	_ = s.updateCrawlTaskSummary(ctx, taskId, "completed", exportFiles)

	// 构造返回结果
	res = append(res, Result{
		URL:   productRankURL,
		Title: fmt.Sprintf("抖音商品短视频榜_%d 条商品_%d 条视频", len(products), len(videoAnalysisList)),
		Content: fmt.Sprintf("成功抓取%d条商品数据，%d条详情数据，%d条视频分析数据，已导出%d个文件",
			len(products), len(productDetails), len(videoAnalysisList), len(exportFiles)),
		ProductCount:   len(products),
		VideoCount:     len(videoAnalysisList),
		Products:       products,
		ProductDetails: productDetails,
		VideoAnalysis:  videoAnalysisList,
		ExportFiles:    exportFiles,
	})

	logInfo(ctx, taskId, fmt.Sprintf("🎉 爬虫任务全部完成！商品：%d 条，详情：%d 条，视频分析：%d 条",
		len(products), len(productDetails), len(videoAnalysisList)))
	return res, nil
}

// RunProductRankTask 商品榜单抓取任务（独立任务，用于调度器）
func (s *sCrawler) RunProductRankTask(ctx context.Context) error {
	taskId := fmt.Sprintf("product_rank_%d", time.Now().UnixMilli())
	logInfo(ctx, taskId, "🚀 开始执行商品榜单抓取任务")

	// 获取状态管理器
	statusMgr := service.GetCrawlerStatusManager()
	if statusMgr.IsOnline() {
		return gerror.New("爬虫服务正在运行，请稍后再试")
	}

	// 创建任务上下文，并确保强制下线可以真正取消后续操作
	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctx = taskCtx

	// 设置上线状态
	statusMgr.SetOnline(taskId, "product_rank", cancel)
	defer statusMgr.SetOffline()

	// 启动浏览器
	browser, err := s.startBrowser(ctx, taskId, false)
	if err != nil {
		return gerror.Wrap(err, "启动浏览器失败")
	}
	defer browser.Close()

	// 打开首页
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return gerror.Wrap(err, "打开首页失败")
	}

	// Cookie 登录
	cookieManager := service.NewCookieManager("cookies")
	hasSavedCookies := cookieManager.HasCompassCookies(ctx, taskId)
	if !hasSavedCookies {
		return gerror.New("未找到已保存的 Cookie，请先执行登录操作")
	}

	// 注入 Cookie
	if !s.injectCookies(ctx, page, cookieManager, taskId) {
		return gerror.New("Cookie 注入失败")
	}
	if err := page.Navigate("https://compass.jinritemai.com/shop"); err != nil {
		return gerror.Wrap(err, "打开罗盘首页失败")
	}
	_ = page.WaitLoad()

	// 验证登录状态
	if !s.checkIsLoggedIn(ctx, page, taskId) {
		return gerror.New("Cookie 已过期，请重新登录")
	}

	// 更新状态消息
	statusMgr.SetMessage("抓取商品榜单中")

	// 监听浏览器自身的榜单响应，不重放内部接口。
	stopObserver, pageCh := s.setupBrowserResponseObserver(ctx, page, taskId)
	defer stopObserver()

	// 导航到商品榜单页面
	productRankURL := "https://compass.jinritemai.com/shop/chance/product-rank"
	if err := page.Navigate(productRankURL); err != nil {
		return gerror.Wrap(err, "跳转到商品榜单失败")
	}
	page.WaitLoad()
	time.Sleep(3 * time.Second)

	// 点击短视频榜 Tab
	if err := s.clickShortVideoTab(ctx, page, taskId); err != nil {
		return gerror.Wrap(err, "点击短视频榜失败")
	}
	time.Sleep(2 * time.Second)

	// 分页抓取数据
	checkpointManager := service.NewCheckpointManager("checkpoints")
	products := s.collectInterceptedData(ctx, page, taskId, checkpointManager, 1, pageCh)
	if len(products) == 0 {
		return gerror.New("未抓取到商品数据")
	}

	logInfo(ctx, taskId, fmt.Sprintf("✅ 商品榜单抓取完成，共 %d 条", len(products)))
	_ = checkpointManager.CompleteTask(ctx, taskId, len(products))
	return nil
}

// RunVideoPoolTask 视频池抓取任务（独立任务，用于调度器）
func (s *sCrawler) RunVideoPoolTask(ctx context.Context) error {
	taskId := fmt.Sprintf("video_pool_%d", time.Now().UnixMilli())
	logInfo(ctx, taskId, "🚀 开始执行视频池抓取任务")

	// 获取状态管理器
	statusMgr := service.GetCrawlerStatusManager()
	if statusMgr.IsOnline() {
		return gerror.New("爬虫服务正在运行，请稍后再试")
	}

	// 创建任务上下文，并确保强制下线可以真正取消后续操作
	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctx = taskCtx

	// 设置上线状态
	statusMgr.SetOnline(taskId, "video_pool", cancel)
	defer statusMgr.SetOffline()

	// 查询已抓取的商品列表（从数据库获取最新 200 条）
	products, err := s.getProductsFromDB(ctx)
	if err != nil || len(products) == 0 {
		return gerror.New("未找到已抓取的商品数据，请先执行商品榜单抓取")
	}

	logInfo(ctx, taskId, fmt.Sprintf("📋 从数据库加载 %d 条商品数据", len(products)))

	// 启动浏览器
	browser, err := s.startBrowser(ctx, taskId, false)
	if err != nil {
		return gerror.Wrap(err, "启动浏览器失败")
	}
	defer browser.Close()

	// Cookie 登录
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return gerror.Wrap(err, "打开首页失败")
	}

	cookieManager := service.NewCookieManager("cookies")
	hasSavedCookies := cookieManager.HasCompassCookies(ctx, taskId)
	if !hasSavedCookies {
		return gerror.New("未找到已保存的 Cookie，请先执行登录操作")
	}

	// 注入 Cookie
	if !s.injectCookies(ctx, page, cookieManager, taskId) {
		return gerror.New("Cookie 注入失败")
	}
	if err := page.Navigate("https://compass.jinritemai.com/shop"); err != nil {
		return gerror.Wrap(err, "打开罗盘首页失败")
	}
	_ = page.WaitLoad()

	// 验证登录状态
	if !s.checkIsLoggedIn(ctx, page, taskId) {
		return gerror.New("Cookie 已过期，请重新登录")
	}

	// 更新状态消息
	statusMgr.SetMessage("抓取视频数据中")

	// 仅使用页面 UI 采集，不重放罗盘内部接口。
	videoAnalysisList := s.scrapeVideoAnalysisDOMWithSink(ctx, browser, products, taskId, func(videos []VideoAnalysis) {
		if err := s.persistAndQueueVideos(ctx, taskId, videos); err != nil {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 视频即时入库失败：%v", err))
		}
	})
	logInfo(ctx, taskId, fmt.Sprintf("✅ 视频池抓取完成，共 %d 条视频数据", len(videoAnalysisList)))

	// 视频已在发现时逐商品入库，这里只输出最终去重统计。
	if len(videoAnalysisList) > 0 {
		videoAnalysisList = deduplicateVideos(videoAnalysisList)
		logInfo(ctx, taskId, fmt.Sprintf("✅ 去重后 %d 条视频数据", len(videoAnalysisList)))
	}

	return nil
}

type VideoDiagnosticResult struct {
	TaskId      string
	Product     ProductInfo
	AnalysisURL string
	Videos      []VideoAnalysis
	Diagnostics []VideoResponseDiagnostic
}

// RunVideoDiagnostic 只诊断一个商品，用于先验证真实接口和解析结构，避免盲目跑满 Top200。
func (s *sCrawler) RunVideoDiagnostic(ctx context.Context, productId string) (*VideoDiagnosticResult, error) {
	taskId := fmt.Sprintf("video_diagnostic_%d", time.Now().UnixMilli())
	statusMgr := service.GetCrawlerStatusManager()
	if statusMgr.IsOnline() {
		return nil, gerror.New("爬虫服务正在运行，请稍后再试")
	}
	taskCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctx = taskCtx
	statusMgr.SetOnline(taskId, "video_diagnostic", cancel)
	defer statusMgr.SetOffline()
	statusMgr.SetMessage("诊断单商品视频链路")

	products, err := s.getProductsFromDB(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, "读取最新榜单快照失败")
	}
	if len(products) == 0 {
		return nil, gerror.New("最新榜单快照为空，请先抓取 Top200")
	}
	product := products[0]
	if productId != "" {
		found := false
		for _, candidate := range products {
			if candidate.ProductID == productId {
				product, found = candidate, true
				break
			}
		}
		if !found {
			return nil, gerror.Newf("最新榜单快照中不存在商品 %s", productId)
		}
	}

	browser, err := s.startBrowser(ctx, taskId, false)
	if err != nil {
		return nil, err
	}
	defer browser.Close()
	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, gerror.Wrap(err, "创建诊断页面失败")
	}
	defer page.Close()
	cookieManager := service.NewCookieManager("cookies")
	if !cookieManager.HasCompassCookies(ctx, taskId) || !s.injectCookies(ctx, page, cookieManager, taskId) {
		return nil, gerror.New("没有可用的罗盘 Cookie，请先用账号密码登录")
	}
	if err := page.Navigate("https://compass.jinritemai.com/shop"); err != nil {
		return nil, gerror.Wrap(err, "打开罗盘首页失败")
	}
	_ = page.WaitLoad()
	if !s.checkIsLoggedIn(ctx, page, taskId) {
		return nil, gerror.New("Cookie 已过期，请重新登录")
	}

	pageResult := s.scrapeSingleProductVideosWithStatus(ctx, browser, product, taskId)
	videos := deduplicateVideos(pageResult.Videos)
	if len(videos) > 0 {
		if err := s.saveVideoAnalysis(ctx, taskId, videos); err != nil {
			return nil, gerror.Wrap(err, "保存诊断视频数据失败")
		}
	}
	return &VideoDiagnosticResult{
		TaskId: taskId, Product: product, AnalysisURL: buildVideoAnalysisURL(product), Videos: videos,
		Diagnostics: pageResult.Diagnostics,
	}, nil
}

// getProductsFromDB 从数据库获取已抓取的商品列表
func (s *sCrawler) getProductsFromDB(ctx context.Context) ([]ProductInfo, error) {
	// 只读取最新一次榜单快照，避免把多个任务的重复商品混成“最新 200 条”。
	latest, err := g.DB().Model("tiktok_ranking_list").
		Fields("task_id").
		WhereNot("task_id", "").
		OrderDesc("created_at").
		One()
	if err != nil {
		return nil, err
	}
	if latest.IsEmpty() {
		return nil, nil
	}

	taskId := latest["task_id"].String()
	result, err := g.DB().Model("tiktok_ranking_list").
		Where("task_id", taskId).
		OrderAsc("rank").
		Limit(200).
		All()
	if err != nil {
		return nil, err
	}

	var products []ProductInfo
	for _, row := range result {
		product := ProductInfo{
			ProductID:   row["product_id"].String(),
			ProductName: row["product_name"].String(),
			ShopID:      row["shop_id"].String(),
			ShopName:    row["shop_name"].String(),
			ProductURL:  row["product_url"].String(),
		}
		if product.ProductID != "" {
			products = append(products, product)
		}
	}

	return products, nil
}

// deduplicateVideos 基于 video_id 去重（借鉴 Python 爬虫的 _clean_videos_against_previous）
func deduplicateVideos(videos []VideoAnalysis) []VideoAnalysis {
	seen := make(map[string]bool)
	var result []VideoAnalysis

	for _, v := range videos {
		if v.VideoId == "" {
			continue
		}
		if !seen[v.VideoId] {
			seen[v.VideoId] = true
			result = append(result, v)
		}
	}

	return result
}

// saveVideoAnalysis 保存视频分析数据到数据库
func (s *sCrawler) saveVideoAnalysis(ctx context.Context, taskId string, videos []VideoAnalysis) error {
	if len(videos) == 0 {
		return nil
	}

	// 批量插入
	for _, video := range videos {
		if video.VideoId != "" {
			count, countErr := g.DB().Model("video_analysis_data").Ctx(ctx).
				Where("task_id", taskId).
				Where("video_id", video.VideoId).
				Count()
			if countErr == nil && count > 0 {
				continue
			}
		}
		_, err := g.DB().Model("video_analysis_data").Ctx(ctx).Insert(g.Map{
			"task_id":             taskId,
			"product_id":          video.ProductID,
			"product_name":        video.ProductName,
			"video_id":            video.VideoId,
			"video_desc":          video.VideoDesc,
			"author_name":         video.AuthorName,
			"cover_url":           video.CoverURL,
			"publish_time":        video.PublishTime,
			"source_url":          video.SourceURL,
			"download_status":     "pending",
			"download_url":        "",
			"resolve_strategy":    "",
			"payment_amount":      video.PaymentAmount,
			"expose_user_count":   video.ExposeUserCount,
			"click_user_count":    video.ClickUserCount,
			"convert_user_count":  video.ConvertUserCount,
			"click_rate":          video.ClickRate,
			"click_convert_rate":  video.ClickConvertRate,
			"expose_convert_rate": video.ExposeConvertRate,
			"order_count":         video.OrderCount,
			"sales_volume":        video.SalesVolume,
			"created_at":          time.Now(),
		})
		if err != nil {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 保存视频数据失败 video_id=%s: %v", video.VideoId, err))
		}
	}

	logInfo(ctx, taskId, fmt.Sprintf("✅ 保存 %d 条视频数据到数据库", len(videos)))
	return nil
}

func (s *sCrawler) loadProductsFromDB(ctx context.Context, taskId string) ([]ProductInfo, error) {
	rows, err := g.DB().Model("tiktok_ranking_list").
		Ctx(ctx).
		Where("task_id", taskId).
		OrderAsc("rank").
		All()
	if err != nil {
		return nil, err
	}
	products := make([]ProductInfo, 0, len(rows))
	for _, row := range rows {
		products = append(products, ProductInfo{
			Rank:             row["rank"].Int(),
			ProductID:        row["product_id"].String(),
			ShopID:           row["shop_id"].String(),
			ProductURL:       row["product_url"].String(),
			ProductName:      row["product_name"].String(),
			ShopName:         row["shop_name"].String(),
			PaymentAmount:    row["payment_amount"].String(),
			ClickCount:       row["click_count"].String(),
			SalesVolume:      row["sales_volume"].String(),
			DeliveryVideoNum: row["delivery_video_num"].String(),
			PriceRange:       row["price_range"].String(),
			FirstList:        row["first_list"].Int(),
		})
	}
	return products, nil
}

func (s *sCrawler) countVideoRows(ctx context.Context, taskId string) int {
	count, err := g.DB().Model("video_analysis_data").Ctx(ctx).Where("task_id", taskId).Count()
	if err != nil {
		return 0
	}
	return count
}

func (s *sCrawler) updateCrawlTaskSummary(ctx context.Context, taskId, status string, exportFiles map[string]string) error {
	productCount, _ := g.DB().Model("tiktok_ranking_list").Ctx(ctx).Where("task_id", taskId).Count()
	videoCount, _ := g.DB().Model("video_analysis_data").Ctx(ctx).Where("task_id", taskId).Count()
	data := g.Map{
		"product_count": productCount,
		"video_count":   videoCount,
	}
	if status != "" {
		data["status"] = status
	}
	if status == "completed" || status == "paused" || status == "failed" || status == "stopped" {
		data["end_time"] = time.Now()
	}
	if len(exportFiles) > 0 {
		if encoded, err := json.Marshal(exportFiles); err == nil {
			data["export_files"] = string(encoded)
		}
	}
	_, err := g.DB().Model("crawl_tasks").Ctx(ctx).Where("task_id", taskId).Update(data)
	return err
}

// startBrowser 启动浏览器
// startBrowser 启动浏览器 - headless: true表示无头模式，false表示有头模式（显示窗口）
func (s *sCrawler) startBrowser(ctx context.Context, taskId string, headless bool) (*rod.Browser, error) {
	l := launcher.New().
		Headless(headless).
		Append("--start-maximized").
		Append("--disable-dev-shm-usage")

	url, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("启动浏览器失败：%w", err)
	}

	browser := rod.New().ControlURL(url).SlowMotion(100 * time.Millisecond)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("连接浏览器失败：%w", err)
	}

	logInfo(ctx, taskId, fmt.Sprintf("🌐 浏览器启动成功：%s", url))
	return browser, nil
}

// checkIsLoggedIn 检测是否已登录
func (s *sCrawler) checkIsLoggedIn(ctx context.Context, page *rod.Page, taskId string) bool {
	selectors := []string{
		"//*[contains(text(), '经营概况')]",
		"//*[contains(text(), '抖音电商·罗盘')]",
		".el-container",
		".main-content",
	}

	for _, selector := range selectors {
		var err error
		if strings.HasPrefix(selector, "//") {
			_, err = page.Timeout(2 * time.Second).ElementX(selector)
		} else {
			_, err = page.Timeout(2 * time.Second).Element(selector)
		}
		if err == nil {
			logInfo(ctx, taskId, "✅ 检测到已登录状态")
			return true
		}
	}

	logInfo(ctx, taskId, "⚠️ 未检测到登录状态，需要登录")
	return false
}

// performLogin 执行登录
func (s *sCrawler) performLogin(ctx context.Context, page *rod.Page, taskId string, loginType, username, password string) bool {
	deadline := time.Now().Add(8 * time.Minute)
	logInfo(ctx, taskId, fmt.Sprintf("⏳ 等待登录完成（超时时间：%d 分钟）", 8))

	if loginType == "password" && username != "" && password != "" {
		if err := s.fillPasswordLoginForm(page, username, password); err != nil {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 自动填充账号密码失败，请在浏览器中手动输入：%v", err))
		} else {
			logInfo(ctx, taskId, "🔐 已在本地浏览器填充账号密码，如出现验证码请手动完成")
		}
	}

	// 等待用户手动完成登录（支持扫码或账号密码）
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)

		// 检测是否登录成功
		if s.checkIsLoggedIn(ctx, page, taskId) {
			return true
		}

		remaining := int(time.Until(deadline).Seconds())
		if remaining%30 == 0 {
			logInfo(ctx, taskId, fmt.Sprintf("⌛ 等待登录中...剩余时间：%d秒", remaining))
		}
	}

	logError(ctx, taskId, "⏰ 登录超时")
	return false
}

func (s *sCrawler) fillPasswordLoginForm(page *rod.Page, username, password string) error {
	// 先尝试切换到账号/密码登录面板。找不到时继续在当前面板定位输入框。
	for _, xpath := range []string{
		"//*[contains(text(), '密码登录')]",
		"//*[contains(text(), '账号登录')]",
	} {
		if elem, err := page.Timeout(1500 * time.Millisecond).ElementX(xpath); err == nil {
			_ = elem.Click(proto.InputMouseButtonLeft, 1)
			time.Sleep(500 * time.Millisecond)
			break
		}
	}

	findInput := func(selectors []string) (*rod.Element, error) {
		var lastErr error
		for _, selector := range selectors {
			elem, err := page.Timeout(1500 * time.Millisecond).Element(selector)
			if err == nil {
				return elem, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}

	usernameInput, err := findInput([]string{
		"input[placeholder*='手机号']", "input[placeholder*='账号']",
		"input[type='tel']", "input[type='text']",
	})
	if err != nil || usernameInput == nil {
		return gerror.New("未找到账号输入框")
	}
	passwordInput, err := findInput([]string{"input[type='password']", "input[placeholder*='密码']"})
	if err != nil || passwordInput == nil {
		return gerror.New("未找到密码输入框")
	}

	_ = usernameInput.SelectAllText()
	if err := usernameInput.Input(username); err != nil {
		return gerror.Wrap(err, "输入账号失败")
	}
	_ = passwordInput.SelectAllText()
	if err := passwordInput.Input(password); err != nil {
		return gerror.Wrap(err, "输入密码失败")
	}

	if button, err := page.Timeout(2 * time.Second).ElementX("//button[contains(., '登录')]"); err == nil {
		if err := button.Click(proto.InputMouseButtonLeft, 1); err != nil {
			return gerror.Wrap(err, "点击登录按钮失败")
		}
	}
	return nil
}

// clickShortVideoTab 点击短视频榜 Tab
func (s *sCrawler) clickShortVideoTab(ctx context.Context, page *rod.Page, taskId string) error {
	// 尝试多种选择器定位短视频榜 Tab
	tabSelectors := []string{
		"//*[contains(text(), '短视频榜')]",
		".ecom-tabs-tab-btn:contains('短视频榜')",
		"[data-type='short_video']",
		".tab-item:contains('短视频榜')",
	}

	for _, selector := range tabSelectors {
		var element *rod.Element
		var err error

		if strings.HasPrefix(selector, "//") {
			element, err = page.Timeout(3 * time.Second).ElementX(selector)
		} else {
			element, err = page.Timeout(3 * time.Second).Element(selector)
		}

		if err == nil {
			if err := element.Click(proto.InputMouseButtonLeft, 1); err != nil {
				logWarn(ctx, taskId, fmt.Sprintf("点击 Tab 失败：%v", err))
				continue
			}
			logInfo(ctx, taskId, fmt.Sprintf("✅ 成功点击短视频榜 Tab: %s", selector))
			time.Sleep(2 * time.Second)
			return nil
		}
	}

	// 如果直接点击失败，尝试通过 URL 导航
	shortVideoURL := "https://compass.jinritemai.com/shop/chance/product-rank?rankType=short_video"
	logInfo(ctx, taskId, fmt.Sprintf("🔄 Tab 点击失败，尝试直接导航：%s", shortVideoURL))
	if err := page.Navigate(shortVideoURL); err != nil {
		return fmt.Errorf("导航到短视频榜失败：%w", err)
	}
	page.WaitLoad()
	time.Sleep(2 * time.Second)

	return nil
}

// scrapeAllPages 分页抓取所有数据
func (s *sCrawler) scrapeAllPages(ctx context.Context, page *rod.Page, taskId string) []ProductInfo {
	var allProducts []ProductInfo
	totalPages := 20
	currentPage := 1

	for currentPage <= totalPages {
		logInfo(ctx, taskId, fmt.Sprintf("📄 正在抓取第 %d/%d 页", currentPage, totalPages))

		// 抓取当前页数据（传递页码参数）
		products := s.scrapeCurrentPage(ctx, page, taskId, currentPage)
		if len(products) > 0 {
			allProducts = append(allProducts, products...)
			logInfo(ctx, taskId, fmt.Sprintf("✅ 第 %d 页抓取完成：获取 %d 条数据", currentPage, len(products)))
		}

		// 尝试点击下一页
		if currentPage < totalPages {
			if !s.clickNextPage(ctx, page, taskId) {
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ 无法点击下一页，停止翻页（当前第 %d 页）", currentPage))
				break
			}
			time.Sleep(2 * time.Second)
		}

		currentPage++
	}

	// 去重
	allProducts = s.uniqueProducts(allProducts)
	logInfo(ctx, taskId, fmt.Sprintf("📊 分页抓取完成，共获取 %d 条商品（去重后）", len(allProducts)))
	return allProducts
}

// scrapeCurrentPage 抓取当前页数据
func (s *sCrawler) scrapeCurrentPage(ctx context.Context, page *rod.Page, taskId string, currentPage int) []ProductInfo {
	var products []ProductInfo

	// 等待表格加载
	time.Sleep(2 * time.Second)

	// 查找表格行
	rowSelectors := []string{".el-table__row", ".ant-table-row", ".table-row", "tr[data-row-key]"}
	var rows []*rod.Element

	for _, selector := range rowSelectors {
		elems, err := page.Elements(selector)
		if err == nil && len(elems) > 0 {
			rows = elems
			logInfo(ctx, taskId, fmt.Sprintf("📝 找到表格行选择器：%s, 行数：%d", selector, len(elems)))
			break
		}
	}

	if len(rows) == 0 {
		logWarn(ctx, taskId, "⚠️ 未找到表格行")
		return products
	}

	// 使用传入的页码参数
	logDebug(ctx, taskId, fmt.Sprintf("📄 当前页码：%d", currentPage))

	// 解析每行数据
	dataRowIdx := 0 // 用于跟踪实际数据行的索引（不包括表头）
	for rowIdx, row := range rows {
		// 跳过表头
		if rowIdx == 0 {
			text, _ := row.Text()
			if strings.Contains(text, "排名") || strings.Contains(text, "商品") || strings.Contains(text, "店铺") {
				logDebug(ctx, taskId, fmt.Sprintf("⏭️ 跳过表头行：%s", text))
				continue
			}
		}

		product := s.parseProductRow(ctx, row, taskId, dataRowIdx, currentPage)
		if product != nil && product.ProductName != "" {
			logDebug(ctx, taskId, fmt.Sprintf("✅ 解析商品：排名=%d, 名称=%s", product.Rank, product.ProductName))
			products = append(products, *product)
		}

		dataRowIdx++ // 每处理一行数据，增加数据行索引
	}

	logInfo(ctx, taskId, fmt.Sprintf("📊 当前页解析完成：获取 %d 条商品", len(products)))
	return products
}

// parseProductRow 解析商品行数据
func (s *sCrawler) parseProductRow(ctx context.Context, row *rod.Element, taskId string, rowIdx int, currentPage int) *ProductInfo {
	cellSelectors := []string{".el-table__cell", ".ant-table-cell", ".table-cell", "td"}
	var cells []*rod.Element

	for _, selector := range cellSelectors {
		elems, err := row.Elements(selector)
		if err == nil && len(elems) >= 6 {
			cells = elems
			break
		}
	}

	if len(cells) < 6 {
		logDebug(ctx, taskId, fmt.Sprintf("⚠️ 单元格数量不足：%d", len(cells)))
		return nil
	}

	// 获取排名（优先从页面 DOM 元素中提取真实排名）
	rankText := s.getText(cells[0])
	rank := s.parseRank(rankText)

	// 计算全局排名（确保跨页面连续，不循环）
	// 全局排名 = (页码 - 1) * 每页数量 (10) + 当前数据行号 (rowIdx + 1)
	globalRank := (currentPage-1)*10 + (rowIdx + 1)

	// 调试日志：输出第一个单元格的实际内容
	logDebug(ctx, taskId, fmt.Sprintf("🔍 数据行 %d - 单元格内容：%q, 页面排名=%d, 计算全局排名=%d", rowIdx, rankText, rank, globalRank))

	// 使用计算出的全局排名，覆盖任何页面排名
	rank = globalRank
	logDebug(ctx, taskId, fmt.Sprintf("✔️ 最终使用全局排名：%d (页码=%d, 行号=%d)", rank, currentPage, rowIdx))

	// 检测是否包含"首次上榜"标识（仅在排名列单元格内查找）
	firstList := 0

	// ===== 多维度选择器策略（借鉴 Tab 定位思路）=====

	// 策略 1: XPath 文本匹配（兼容多种 DOM 结构）- 使用相对路径避免全局匹配
	xpathSelectors := []string{
		".//text()[normalize-space()='首次上榜']",                         // 精确匹配
		".//span[contains(text(), '首次上榜')]",                           // 模糊匹配
		".//span[contains(@class, 'tag') and contains(text(), '首次')]", // 标签元素
	}

	for _, selector := range xpathSelectors {
		_, err := cells[0].Timeout(500 * time.Millisecond).ElementX(selector)
		if err == nil {
			firstList = 1
			logDebug(ctx, taskId, fmt.Sprintf("🏆 [XPath] 检测到首次上榜标识 [%s]：排名=%d", selector, rank))
			break
		}
	}

	// 策略 2: CSS 类名匹配（ElementUI 标签样式）
	if firstList == 0 {
		cssSelectors := []string{
			".el-tag",               // ElementUI 标签
			"[class*='first-list']", // first-list 类名标签
			"[class*='first-time']", // first-time 类名标签
			"[data-tag='首次上榜']",     // data 属性标记
		}

		for _, selector := range cssSelectors {
			elems, err := cells[0].Timeout(500 * time.Millisecond).Elements(selector)
			if err == nil && len(elems) > 0 {
				for _, elem := range elems {
					if text, _ := elem.Text(); strings.Contains(text, "首次") {
						firstList = 1
						logDebug(ctx, taskId, fmt.Sprintf("🏆 [CSS] 检测到首次上榜标识 [%s]：排名=%d", selector, rank))
						break
					}
				}
				if firstList == 1 {
					break
				}
			}
		}
	}

	// 获取商品名称，同时提取价格带
	productName := s.getText(cells[1])
	priceRange := ""

	// 从商品名称中提取价格带（如果包含"价格带"关键字）
	if strings.Contains(productName, "价格带") {
		parts := strings.SplitN(productName, "价格带", 2)
		if len(parts) == 2 {
			productName = strings.TrimSpace(parts[0]) // 移除价格带部分
			priceRange = strings.TrimSpace(parts[1])  // 提取价格带值
		}
	}

	// 调试日志：输出最终检测结果
	cellText := s.getText(cells[0])
	logDebug(ctx, taskId, fmt.Sprintf("📊 首次上榜检测完成 | 排名=%d | first_list=%d | 单元格文本=%q",
		rank, firstList, cellText))

	product := &ProductInfo{
		Rank:             rank,
		ProductName:      productName,
		ShopName:         s.getText(cells[2]),
		SalesVolume:      s.getText(cells[3]),
		PaymentAmount:    s.getText(cells[4]),
		ClickCount:       s.getText(cells[5]),
		DeliveryVideoNum: "0",
		FirstList:        firstList,
		PriceRange:       priceRange,
	}

	if len(cells) > 6 {
		product.DeliveryVideoNum = s.getText(cells[6])
	}

	// 尝试获取商品详情 URL（从商品名称列的链接中提取）
	if link, err := cells[1].Element("a"); err == nil {
		if href, _ := link.Attribute("href"); href != nil && *href != "" {
			detailHref := *href
			// 如果是相对路径，补全为完整 URL
			if strings.HasPrefix(detailHref, "/") {
				detailHref = "https://compass.jinritemai.com" + detailHref
			}
			product.ProductURL = detailHref
			logDebug(ctx, taskId, fmt.Sprintf("🔗 获取到商品详情 URL：%s", detailHref))
		}
	}

	logDebug(ctx, taskId, fmt.Sprintf("✅ 解析商品行 %d: 排名=%d, 名称=%s, 首次上榜=%d, URL=%s", rowIdx, product.Rank, product.ProductName, product.FirstList, product.ProductURL))
	return product
}

// clickNextPage 点击下一页
func (s *sCrawler) clickNextPage(ctx context.Context, page *rod.Page, taskId string) bool {
	// 只匹配真正的“下一页”按钮。不能使用 contains(@class, 'next')，
	// Aurora 的 jump-next / before-jump-next 也会被命中，导致一次跳过多页。
	nextSelectors := []string{
		".aurora-pagination-next",
		".ant-pagination-next",
		".el-pagination .btn-next",
		"li[title='下一页']",
		"li[aria-label='Next Page']",
	}

	// 罗盘右下角的意见反馈浮层会覆盖分页按钮。它不是数据权限遮罩，
	// 这里只让该反馈控件不再截获鼠标事件，不移除任何详情限制元素。
	if disabledCount := disableFloatingAssistantPointerEvents(page); disabledCount > 0 {
		logDebug(ctx, taskId, fmt.Sprintf("ℹ️ 已避免 %d 个悬浮助手节点遮挡分页按钮", disabledCount))
	}

	beforePage := activePaginationPage(page)
	for attempt := 1; attempt <= 12; attempt++ {
		for _, selector := range nextSelectors {
			element, err := page.Timeout(1500 * time.Millisecond).Element(selector)
			if err != nil {
				continue
			}

			if paginationElementDisabled(element) {
				if beforePage >= 20 {
					logInfo(ctx, taskId, "⚠️ 下一页按钮已禁用，已是最后一页")
					return false
				}
				continue
			}

			if err := element.Click(proto.InputMouseButtonLeft, 1); err != nil {
				logDebug(ctx, taskId, fmt.Sprintf("⚠️ 下一页暂不可点击（第 %d 次尝试）：%v", attempt, err))
				continue
			}

			// 点击成功不等于翻页成功，必须观察到活动页码恰好增加 1。
			if beforePage <= 0 {
				logInfo(ctx, taskId, "✅ 成功点击下一页（页面未暴露活动页码）")
				return true
			}
			verifyDeadline := time.Now().Add(10 * time.Second)
			for time.Now().Before(verifyDeadline) {
				afterPage := activePaginationPage(page)
				if afterPage == beforePage+1 {
					logInfo(ctx, taskId, fmt.Sprintf("✅ 翻页成功：%d → %d", beforePage, afterPage))
					return true
				}
				if afterPage > beforePage+1 {
					logWarn(ctx, taskId, fmt.Sprintf("⚠️ 页码异常跳跃：%d → %d，停止继续翻页", beforePage, afterPage))
					return false
				}
				time.Sleep(300 * time.Millisecond)
			}
			logDebug(ctx, taskId, fmt.Sprintf("⚠️ 点击后页码仍为 %d，准备重试", beforePage))
		}
		time.Sleep(time.Second)
	}

	logWarn(ctx, taskId, fmt.Sprintf("⚠️ 下一页在重试后仍不可用（当前活动页=%d）", beforePage))
	return false
}

func disableFloatingAssistantPointerEvents(page *rod.Page) int {
	result, err := page.Timeout(2 * time.Second).Eval(`() => {
		let changed = 0;
		const roots = document.querySelectorAll(
			'#DOUXIAOER_DRAG_CONTROLLER, [class*="miniFeedbackContainer"]'
		);
		roots.forEach((root) => {
			[root, ...root.querySelectorAll('*')].forEach((element) => {
				if (getComputedStyle(element).pointerEvents !== 'none') {
					element.style.setProperty('pointer-events', 'none', 'important');
					changed++;
				}
			});
		});
		return changed;
	}`)
	if err != nil || result == nil {
		return 0
	}
	return result.Value.Int()
}

func activePaginationPage(page *rod.Page) int {
	result, err := page.Timeout(2 * time.Second).Eval(`() => {
		const selectors = [
			'.aurora-pagination-item-active',
			'.ant-pagination-item-active',
			'.el-pager li.active'
		];
		for (const selector of selectors) {
			const element = document.querySelector(selector);
			if (!element) continue;
			const pageNo = Number.parseInt((element.textContent || '').trim(), 10);
			if (Number.isFinite(pageNo)) return pageNo;
		}
		return 0;
	}`)
	if err != nil || result == nil {
		return 0
	}
	return result.Value.Int()
}

func paginationElementDisabled(element *rod.Element) bool {
	if disabled, _ := element.Attribute("disabled"); disabled != nil {
		return true
	}
	if ariaDisabled, _ := element.Attribute("aria-disabled"); ariaDisabled != nil && *ariaDisabled == "true" {
		return true
	}
	if className, _ := element.Attribute("class"); className != nil {
		classLower := strings.ToLower(*className)
		return strings.Contains(classLower, "disabled")
	}
	return false
}

// dismissBlockingModal 关闭阻塞弹窗（借鉴 Python 爬虫的 _dismiss_blocking_modal）
func (s *sCrawler) dismissBlockingModal(page *rod.Page) {
	// 1. 先按 ESC 键关闭可能的弹窗
	_ = page.Keyboard.Press(input.Escape)
	time.Sleep(150 * time.Millisecond)

	// 2. 通过 JS 移除模态框 DOM 节点
	page.MustEval(`() => {
		const selectors = [
			'#ecom-login-account-modal',
			'.auxo-modal-wrap',
			'.auxo-modal-mask',
			'.ant-modal-root',
			'.ant-modal-mask',
			'.ant-modal-wrap',
			'[class*="modal"]',
			'[class*="dialog"]'
		];
		const roots = Array.from(document.querySelectorAll(selectors.join(',')));
		for (const node of roots) {
			if (node && typeof node.remove === 'function') {
				node.remove();
			}
		}
		// 恢复页面滚动
		document.body?.classList?.remove('ant-scrolling-effect');
		if (document.body) {
			document.body.style.overflow = '';
			document.body.style.pointerEvents = '';
		}
	}`)
	time.Sleep(150 * time.Millisecond)
}

// detectBlockingElements 检测阻塞元素（验证码、弹窗等）
func (s *sCrawler) detectBlockingElements(page *rod.Page) bool {
	selectors := []string{
		"#captcha", ".captcha", "[class*='captcha']",
		"#verify", ".verify", "[class*='verify']",
		"iframe[src*='captcha']",
		".auxo-modal-wrap",
	}
	for _, sel := range selectors {
		if page.MustHas(sel) {
			return true
		}
	}
	return false
}

// getCurrentPage 获取当前页码
func (s *sCrawler) getCurrentPage(page *rod.Page, taskId string) int {
	// 尝试从页面元素中获取当前页码
	pageSelectors := []string{
		".el-pagination .el-pager li.is-active",
		".el-pager li.active",
		".ant-pagination-item-active",
		".pagination-item.active",
		"li[aria-current='page']",
	}

	for _, selector := range pageSelectors {
		var element *rod.Element
		var err error

		if strings.HasPrefix(selector, "//") {
			element, err = page.Timeout(2 * time.Second).ElementX(selector)
		} else {
			element, err = page.Timeout(2 * time.Second).Element(selector)
		}

		if err == nil {
			text := s.getText(element)
			if pageNum := s.parseRank(text); pageNum > 0 {
				fmt.Printf("[DEBUG] 📄 从页面获取当前页码：%d (选择器：%s)\n", pageNum, selector)
				return pageNum
			}
		}
	}

	// 如果无法从页面获取，尝试从 URL 参数中解析
	if url := page.MustInfo().URL; url != "" {
		// 尝试解析 URL 中的 page 参数（如果有）
		// 例如：?page=2 或 ?p=2
		if strings.Contains(url, "page=") {
			parts := strings.Split(url, "page=")
			if len(parts) > 1 {
				pageNumStr := strings.Split(parts[1], "&")[0]
				if pageNum := s.parseRank(pageNumStr); pageNum > 0 {
					fmt.Printf("[DEBUG] 📄 从 URL 获取当前页码：%d\n", pageNum)
					return pageNum
				}
			}
		}
	}

	// 默认返回 1
	fmt.Printf("[WARN] ⚠️ 无法获取当前页码，使用默认值 1\n")
	return 1
}

// scrapeProductDetails 获取商品详情数据（并发版本）
func (s *sCrawler) scrapeProductDetails(ctx context.Context, browser *rod.Browser, products []ProductInfo, taskId string) []ProductDetail {
	var details []ProductDetail
	var mu sync.Mutex

	pool := NewWorkerPool(crawlerWorkerConcurrency)

	startTime := time.Now()
	logInfo(ctx, taskId, fmt.Sprintf("🚀 开始并发获取商品详情（并发数=%d，总数=%d）", crawlerWorkerConcurrency, len(products)))

	for i, product := range products {
		// 复制循环变量，避免闭包问题
		index := i
		p := product

		pool.Submit(func() {
			if ctx.Err() != nil {
				return
			}
			detail := s.scrapeSingleProductDetail(ctx, browser, p, taskId)
			if detail != nil {
				mu.Lock()
				details = append(details, *detail)
				mu.Unlock()
				logDebug(ctx, taskId, fmt.Sprintf("✅ [%d/%d] 详情获取完成：%s", index+1, len(products), p.ProductName))
			} else {
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ [%d/%d] 详情获取失败：%s", index+1, len(products), p.ProductName))
			}
		})
	}

	// 等待所有任务完成
	pool.Wait()

	elapsed := time.Since(startTime)
	logInfo(ctx, taskId, fmt.Sprintf("✅ 商品详情获取完成：%d 条，耗时 %.1f 秒（相比串行节省 %.0f 秒）",
		len(details), elapsed.Seconds(), float64(len(products))*2-elapsed.Seconds()))

	return details
}

// scrapeSingleProductDetail 获取单个商品详情（DOM 文本正则提取，与 Python 版本一致）
func (s *sCrawler) scrapeSingleProductDetail(ctx context.Context, browser *rod.Browser, product ProductInfo, taskId string) *ProductDetail {
	// 1. 检查缓存
	cache := service.GetProductDetailCache()
	cachedRecord := cache.GetRecord(product.ProductID)
	if cachedRecord != nil && !cache.IsRecordStale(cachedRecord) {
		logInfo(ctx, taskId, fmt.Sprintf("✅ 从缓存获取商品详情：%s", product.ProductName))
		return &ProductDetail{
			ProductID:      product.ProductID,
			ProductName:    product.ProductName,
			FirstShelfTime: cachedRecord.FirstShelfTime,
			MinPrice:       cachedRecord.MinPrice,
			PriceRange:     product.PriceRange,
		}
	}

	// 2. 优先使用罗盘站内详情页。榜单返回的 product_detail_h5_url 常指向
	// haohuo.jinritemai.com 公开商品页，不是罗盘分析详情页。
	detailURL := ""
	if product.ProductID != "" && product.ShopID != "" {
		detailURL = buildDetailURL(product.ProductID, product.ShopID)
	} else {
		detailURL = product.ProductURL
	}
	if detailURL == "" {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 商品 [%s] 无详情 URL，跳过详情获取", product.ProductName))
		return &ProductDetail{ProductName: product.ProductName}
	}

	// 3. 打开新标签页
	var page *rod.Page
	retryErr := s.retryWithBackoff(func() error {
		var pageErr error
		page, pageErr = browser.Page(proto.TargetCreateTarget{URL: ""})
		return pageErr
	}, 3)
	if retryErr != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 无法打开浏览器页面：%v", retryErr))
		return nil
	}
	defer page.Close()

	// 4. 导航到详情页，使用账号的真实授权和查看配额。
	logDebug(ctx, taskId, fmt.Sprintf("🔗 正在打开商品详情：%s", detailURL))
	if err := s.retryWithBackoff(func() error {
		return page.Navigate(detailURL)
	}, 3); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 打开商品详情失败：%v", err))
		return nil
	}
	page.WaitLoad()

	// 6. 纯 DOM 文本正则提取（核心方式，与 Python 版本一致）
	detail := s.scrapeSingleProductDetailByParsing(ctx, page, product, taskId)

	// 7. 保存到缓存
	if detail != nil && (detail.FirstShelfTime != "" || detail.MinPrice != "") {
		cacheRecord := &service.ProductDetailRecord{
			ProductID:          product.ProductID,
			FirstShelfTime:     detail.FirstShelfTime,
			MinPrice:           detail.MinPrice,
			DetailAccessStatus: detail.DetailAccessStatus,
			ProductName:        product.ProductName,
			ShopName:           product.ShopName,
			DetailURL:          detailURL,
		}
		cache.UpsertRecord(cacheRecord)
	}

	return detail
}

// scrapeSingleProductDetailByParsing 通过 DOM 文本正则提取商品详情（核心方式）
func (s *sCrawler) scrapeSingleProductDetailByParsing(ctx context.Context, page *rod.Page, product ProductInfo, taskId string) *ProductDetail {
	detail := &ProductDetail{
		ProductID:   product.ProductID,
		ProductName: product.ProductName,
	}

	// 随机延迟，模拟真实用户行为
	time.Sleep(time.Duration(grand.Intn(1000)+500) * time.Millisecond)

	// 收集页面文本
	texts := s.collectDetailTexts(ctx, page, taskId)

	// 1. 提取首次上架时间
	detail.FirstShelfTime = s.extractFirstShelfTimeFromTexts(ctx, texts, taskId)

	// 2. 提取最低到手价
	detail.MinPrice = s.extractMinPriceFromTexts(ctx, texts, taskId)

	// 3. 提取运费险信息
	hasInsurance := s.extractShippingInsuranceFromTexts(ctx, texts, taskId)
	detail.HasShippingInsurance = hasInsurance

	// 4. 检查商品状态
	detailAccessStatus, detailAccessNote := s.extractDetailAccessState(ctx, texts, taskId)
	detail.DetailAccessStatus = detailAccessStatus
	detail.DetailAccessNote = detailAccessNote

	if detail.FirstShelfTime == "" && detail.MinPrice == "" {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 商品 [%s] 详情页未提取到数据，可能需要更新选择器", product.ProductName))
	}

	return detail
}

// collectDetailTexts 收集页面文本（借鉴 Python 爬虫的 _collect_detail_texts）
func (s *sCrawler) collectDetailTexts(ctx context.Context, page *rod.Page, taskId string) []string {
	// 定义关键词，用于判断页面是否加载完成
	keywords := []string{"首次上架", "最低到手价", "服务保障", "运费险"}

	// 重试机制，最多尝试 4 次
	for attempt := 0; attempt < 4; attempt++ {
		var texts []string

		// 尝试从多个元素获取文本
		selectors := []string{"body", "main", "#root"}
		for _, selector := range selectors {
			if elem, err := page.Element(selector); err == nil {
				if text, err := elem.Text(); err == nil && text != "" {
					texts = append(texts, text)
				}
			}
		}

		// 如果没有获取到文本，尝试获取页面内容
		if len(texts) == 0 {
			if content, err := page.HTML(); err == nil {
				texts = append(texts, content)
			}
		}

		// 检查是否包含关键词
		for _, text := range texts {
			for _, keyword := range keywords {
				if strings.Contains(text, keyword) {
					logDebug(ctx, taskId, fmt.Sprintf("✅ 页面文本包含关键词：%s", keyword))
					return texts
				}
			}
		}

		// 等待后重试
		if attempt < 3 {
			time.Sleep(800 * time.Millisecond)
		}
	}

	// 返回最后一次收集的文本
	var texts []string
	selectors := []string{"body", "main", "#root"}
	for _, selector := range selectors {
		if elem, err := page.Element(selector); err == nil {
			if text, err := elem.Text(); err == nil && text != "" {
				texts = append(texts, text)
				break
			}
		}
	}

	if len(texts) == 0 {
		if content, err := page.HTML(); err == nil {
			texts = append(texts, content)
		}
	}

	return texts
}

// extractFirstShelfTimeFromTexts 从文本中提取首次上架时间
func (s *sCrawler) extractFirstShelfTimeFromTexts(ctx context.Context, texts []string, taskId string) string {
	// 定义多种日期格式的正则表达式
	patterns := []string{
		`首次上架\s*[:：]?\s*(\d{4}\s*年\s*\d{1,2}\s*月\s*\d{1,2}\s*日?)`,
		`首次上架\s*[:：]?\s*(\d{4}[./-]\d{1,2}[./-]\d{1,2})`,
		`上架时间\s*[:：]?\s*(\d{4}\s*年\s*\d{1,2}\s*月\s*\d{1,2}\s*日?)`,
		`上架时间\s*[:：]?\s*(\d{4}[./-]\d{1,2}[./-]\d{1,2})`,
	}

	for _, text := range texts {
		for _, pattern := range patterns {
			if matches := regexp.MustCompile(pattern).FindStringSubmatch(text); len(matches) > 1 {
				dateStr := strings.TrimSpace(matches[1])
				// 标准化日期格式
				dateStr = s.normalizeDateString(dateStr)
				logDebug(ctx, taskId, fmt.Sprintf("📅 提取到上架时间：%s", dateStr))
				return dateStr
			}
		}
	}

	return ""
}

// extractMinPriceFromTexts 从文本中提取最低到手价
func (s *sCrawler) extractMinPriceFromTexts(ctx context.Context, texts []string, taskId string) string {
	// 定义价格的正则表达式
	patterns := []string{
		`最低到手价\s*[:：]?\s*[¥￥]?\s*([\d,\s]+(?:\s*\.\s*\d+)?)`,
		`到手价\s*[:：]?\s*[¥￥]?\s*([\d,\s]+(?:\s*\.\s*\d+)?)`,
		`最低价\s*[:：]?\s*[¥￥]?\s*([\d,\s]+(?:\s*\.\s*\d+)?)`,
	}

	for _, text := range texts {
		for _, pattern := range patterns {
			if matches := regexp.MustCompile(pattern).FindStringSubmatch(text); len(matches) > 1 {
				priceStr := strings.TrimSpace(matches[1])
				// 清理价格字符串
				priceStr = strings.ReplaceAll(priceStr, " ", "")
				priceStr = strings.ReplaceAll(priceStr, ",", "")
				logDebug(ctx, taskId, fmt.Sprintf("💰 提取到最低到手价：%s", priceStr))
				return priceStr
			}
		}
	}

	return ""
}

// extractShippingInsuranceFromTexts 从文本中提取运费险信息
func (s *sCrawler) extractShippingInsuranceFromTexts(ctx context.Context, texts []string, taskId string) *bool {
	shippingInsuranceMarkers := []string{"配置运费险", "赠送运费险", "运费险"}
	noShippingInsuranceMarkers := []string{"无运费险", "未配置运费险"}
	sawServiceGuarantee := false

	for _, text := range texts {
		for _, marker := range shippingInsuranceMarkers {
			if strings.Contains(text, marker) {
				trueValue := true
				logDebug(ctx, taskId, "📦 提取到运费险：是")
				return &trueValue
			}
		}
		for _, marker := range noShippingInsuranceMarkers {
			if strings.Contains(text, marker) {
				falseValue := false
				logDebug(ctx, taskId, "📦 提取到运费险：否")
				return &falseValue
			}
		}
		if strings.Contains(text, "服务保障") {
			sawServiceGuarantee = true
		}
	}

	if sawServiceGuarantee {
		return nil
	}

	return nil
}

// extractDetailAccessState 提取商品详情访问状态
func (s *sCrawler) extractDetailAccessState(ctx context.Context, texts []string, taskId string) (string, string) {
	removedMarkers := []string{"目标商品已下架"}
	categoryMismatchMarkers := []string{
		"您经营的类目与对方不匹配",
		"经营的类目与对方不匹配",
		"类目与对方不匹配",
	}

	for _, text := range texts {
		for _, marker := range removedMarkers {
			if strings.Contains(text, marker) {
				logDebug(ctx, taskId, "⚠️ 商品已下架")
				return "removed", "目标商品已下架"
			}
		}
		for _, marker := range categoryMismatchMarkers {
			if strings.Contains(text, marker) {
				logDebug(ctx, taskId, "⚠️ 类目不匹配")
				return "category_mismatch", "经营类目不匹配"
			}
		}
	}

	return "available", ""
}

// normalizeDateString 标准化日期字符串
func (s *sCrawler) normalizeDateString(dateStr string) string {
	// 替换各种日期分隔符
	dateStr = strings.ReplaceAll(dateStr, "年", "-")
	dateStr = strings.ReplaceAll(dateStr, "月", "-")
	dateStr = strings.ReplaceAll(dateStr, "日", "")
	dateStr = strings.ReplaceAll(dateStr, ".", "-")
	dateStr = strings.ReplaceAll(dateStr, "/", "-")

	// 清理空格
	dateStr = strings.ReplaceAll(dateStr, " ", "")

	return dateStr
}

// processResponseBody 处理响应体的编码和压缩
func (s *sCrawler) processResponseBody(body []byte, headers map[string][]string) (string, error) {
	// 检查是否是gzip压缩
	if encoding, ok := headers["Content-Encoding"]; ok {
		for _, enc := range encoding {
			if strings.Contains(enc, "gzip") {
				// 解压缩gzip数据
				gzipReader, err := gzip.NewReader(bytes.NewReader(body))
				if err != nil {
					return "", err
				}
				defer gzipReader.Close()

				uncompressed, err := io.ReadAll(gzipReader)
				if err != nil {
					return "", err
				}
				body = uncompressed
				break
			}
		}
	}

	// 检查是否是deflate压缩
	if encoding, ok := headers["Content-Encoding"]; ok {
		for _, enc := range encoding {
			if strings.Contains(enc, "deflate") {
				// 解压缩deflate数据
				var buf bytes.Buffer
				deflater := flate.NewReader(bytes.NewReader(body))
				if _, err := io.Copy(&buf, deflater); err != nil {
					return "", err
				}
				deflater.Close()
				body = buf.Bytes()
				break
			}
		}
	}

	// 尝试使用不同的编码解码
	// 先尝试 UTF-8
	if utf8Content := string(body); len(utf8Content) > 0 {
		return utf8Content, nil
	}

	// 尝试 GBK
	if reader := transform.NewReader(bytes.NewReader(body), simplifiedchinese.GBK.NewDecoder()); reader != nil {
		if content, err := io.ReadAll(reader); err == nil && len(content) > 0 {
			return string(content), nil
		}
	}

	// 尝试 GB2312
	if reader := transform.NewReader(bytes.NewReader(body), simplifiedchinese.HZGB2312.NewDecoder()); reader != nil {
		if content, err := io.ReadAll(reader); err == nil && len(content) > 0 {
			return string(content), nil
		}
	}

	// 尝试 Big5
	if reader := transform.NewReader(bytes.NewReader(body), traditionalchinese.Big5.NewDecoder()); reader != nil {
		if content, err := io.ReadAll(reader); err == nil && len(content) > 0 {
			return string(content), nil
		}
	}

	// 直接尝试转换为字符串
	return string(body), nil
}

// parseProductDetailResponse 解析商品详情 API 响应 JSON
func parseProductDetailResponse(body string, product ProductInfo) *ProductDetail {
	detail := &ProductDetail{
		ProductName: product.ProductName,
	}

	// 打印响应体前1000字符，了解API响应结构
	if len(body) > 1000 {
		fmt.Printf("[DEBUG] 商品详情API响应前1000字符: %s\n", body[:1000])
	} else {
		fmt.Printf("[DEBUG] 商品详情API响应: %s\n", body)
	}

	// 尝试多种API响应结构解析
	// 结构1: {"data":{"first_shelf_time":"...", "min_price":"..."}}
	type response1 struct {
		Data struct {
			FirstShelfTime       string `json:"first_shelf_time"`
			MinPrice             string `json:"min_price"`
			HasShippingInsurance *bool  `json:"has_shipping_insurance"`
		} `json:"data"`
	}

	var resp1 response1
	if err := json.Unmarshal([]byte(body), &resp1); err == nil {
		if resp1.Data.FirstShelfTime != "" {
			detail.FirstShelfTime = resp1.Data.FirstShelfTime
		}
		if resp1.Data.MinPrice != "" {
			detail.MinPrice = resp1.Data.MinPrice
		}
		fmt.Printf("[DEBUG] 解析商品详情API响应成功：结构1\n")
		return detail
	}

	// 结构2: {"data":{"product_info":{"first_shelf_time":"...", "min_price":"..."}}}
	type response2 struct {
		Data struct {
			ProductInfo struct {
				FirstShelfTime       string `json:"first_shelf_time"`
				MinPrice             string `json:"min_price"`
				HasShippingInsurance *bool  `json:"has_shipping_insurance"`
			} `json:"product_info"`
		} `json:"data"`
	}

	var resp2 response2
	if err := json.Unmarshal([]byte(body), &resp2); err == nil {
		if resp2.Data.ProductInfo.FirstShelfTime != "" {
			detail.FirstShelfTime = resp2.Data.ProductInfo.FirstShelfTime
		}
		if resp2.Data.ProductInfo.MinPrice != "" {
			detail.MinPrice = resp2.Data.ProductInfo.MinPrice
		}
		fmt.Printf("[DEBUG] 解析商品详情API响应成功：结构2\n")
		return detail
	}

	// 结构3: {"result":{"first_shelf_time":"...", "min_price":"..."}}
	type response3 struct {
		Result struct {
			FirstShelfTime       string `json:"first_shelf_time"`
			MinPrice             string `json:"min_price"`
			HasShippingInsurance *bool  `json:"has_shipping_insurance"`
		} `json:"result"`
	}

	var resp3 response3
	if err := json.Unmarshal([]byte(body), &resp3); err == nil {
		if resp3.Result.FirstShelfTime != "" {
			detail.FirstShelfTime = resp3.Result.FirstShelfTime
		}
		if resp3.Result.MinPrice != "" {
			detail.MinPrice = resp3.Result.MinPrice
		}
		fmt.Printf("[DEBUG] 解析商品详情API响应成功：结构3\n")
		return detail
	}

	// 结构4: {"data":{"item_info":{"first_shelf_time":"...", "min_price":"..."}}}
	type response4 struct {
		Data struct {
			ItemInfo struct {
				FirstShelfTime       string `json:"first_shelf_time"`
				MinPrice             string `json:"min_price"`
				HasShippingInsurance *bool  `json:"has_shipping_insurance"`
			} `json:"item_info"`
		} `json:"data"`
	}

	var resp4 response4
	if err := json.Unmarshal([]byte(body), &resp4); err == nil {
		if resp4.Data.ItemInfo.FirstShelfTime != "" {
			detail.FirstShelfTime = resp4.Data.ItemInfo.FirstShelfTime
		}
		if resp4.Data.ItemInfo.MinPrice != "" {
			detail.MinPrice = resp4.Data.ItemInfo.MinPrice
		}
		fmt.Printf("[DEBUG] 解析商品详情API响应成功：结构4\n")
		return detail
	}

	// 结构5: 直接在根级别
	type response5 struct {
		FirstShelfTime       string `json:"first_shelf_time"`
		MinPrice             string `json:"min_price"`
		HasShippingInsurance *bool  `json:"has_shipping_insurance"`
	}

	var resp5 response5
	if err := json.Unmarshal([]byte(body), &resp5); err == nil {
		if resp5.FirstShelfTime != "" {
			detail.FirstShelfTime = resp5.FirstShelfTime
		}
		if resp5.MinPrice != "" {
			detail.MinPrice = resp5.MinPrice
		}
		fmt.Printf("[DEBUG] 解析商品详情API响应成功：结构5\n")
		return detail
	}

	// 尝试使用正则表达式提取关键信息
	detail = extractDetailByRegex(body, detail)
	if detail.FirstShelfTime != "" || detail.MinPrice != "" {
		fmt.Printf("[DEBUG] 解析商品详情API响应成功：正则表达式\n")
		return detail
	}

	// 解析失败，返回空详情
	fmt.Printf("[DEBUG] 解析商品详情API响应失败：所有结构都不匹配\n")
	return detail
}

// extractDetailByRegex 使用正则表达式从响应体中提取商品详情
func extractDetailByRegex(body string, detail *ProductDetail) *ProductDetail {
	// 提取首次上架时间
	datePatterns := []*regexp.Regexp{
		regexp.MustCompile(`"first_shelf_time"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`first_shelf_time\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`上架时间\s*:\s*"([^"]+)"`),
	}

	for _, pattern := range datePatterns {
		matches := pattern.FindStringSubmatch(body)
		if len(matches) > 1 {
			detail.FirstShelfTime = matches[1]
			break
		}
	}

	// 提取最低到手价
	pricePatterns := []*regexp.Regexp{
		regexp.MustCompile(`"min_price"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`min_price\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`最低价格\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`"price"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`price\s*:\s*"([^"]+)"`),
	}

	for _, pattern := range pricePatterns {
		matches := pattern.FindStringSubmatch(body)
		if len(matches) > 1 {
			detail.MinPrice = matches[1]
			break
		}
	}

	return detail
}

// scrapeVideoAnalysis 获取视频分析数据（并发版本）
func (s *sCrawler) scrapeVideoAnalysis(ctx context.Context, browser *rod.Browser, products []ProductInfo, taskId string) []VideoAnalysis {
	var allVideos []VideoAnalysis
	var mu sync.Mutex

	// 使用受控并发，降低浏览器压力和平台限流风险。
	pool := NewWorkerPool(crawlerWorkerConcurrency)

	startTime := time.Now()
	logInfo(ctx, taskId, fmt.Sprintf("🚀 开始并发获取视频分析（并发数=%d，总数=%d）", crawlerWorkerConcurrency, len(products)))

	for i, product := range products {
		// 复制循环变量
		index := i
		p := product

		pool.Submit(func() {
			if ctx.Err() != nil {
				return
			}
			videos := s.scrapeSingleProductVideosWithAPI(ctx, browser, p, taskId)
			if len(videos) > 0 {
				mu.Lock()
				allVideos = append(allVideos, videos...)
				mu.Unlock()
				logDebug(ctx, taskId, fmt.Sprintf("✅ [%d/%d] 视频分析完成：%s (%d 条视频)", index+1, len(products), p.ProductName, len(videos)))
			} else {
				if ctx.Err() != nil {
					return
				}
				// 如果API拦截失败，回退到页面解析方式
				videos = s.scrapeSingleProductVideos(ctx, browser, p, taskId)
				if len(videos) > 0 {
					mu.Lock()
					allVideos = append(allVideos, videos...)
					mu.Unlock()
					logDebug(ctx, taskId, fmt.Sprintf("✅ [%d/%d] 视频分析完成（页面解析）：%s (%d 条视频)", index+1, len(products), p.ProductName, len(videos)))
				} else {
					logWarn(ctx, taskId, fmt.Sprintf("⚠️ [%d/%d] 视频分析失败：%s", index+1, len(products), p.ProductName))
				}
			}
		})
	}

	// 等待所有任务完成
	pool.Wait()

	elapsed := time.Since(startTime)
	logInfo(ctx, taskId, fmt.Sprintf("✅ 视频分析获取完成：%d 条，耗时 %.1f 秒（相比串行节省 %.0f 秒）",
		len(allVideos), elapsed.Seconds(), float64(len(products))*3-elapsed.Seconds()))

	return allVideos
}

// scrapeSingleProductVideosWithAPI 使用API拦截的方式获取单个商品的视频分析数据
func (s *sCrawler) scrapeSingleProductVideosWithAPI(ctx context.Context, browser *rod.Browser, product ProductInfo, taskId string) []VideoAnalysis {
	var videos []VideoAnalysis

	// 重试打开页面
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 无法打开浏览器页面：%v", err))
		return videos
	}
	defer page.Close()

	// 设置API拦截器
	videoCh := make(chan []VideoAnalysis, 1)
	router := page.HijackRequests()

	// 拦截 flow_more API 请求（精确匹配，借鉴 Python 爬虫的 FLOW_MORE_PATTERN）
	if err := router.Add("*", "", func(h *rod.Hijack) {
		reqURL := h.Request.URL().String()
		if !isVideoAnalysisAPIURL(reqURL, product.ProductID) {
			h.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}

		if err := h.LoadResponse(&http.Client{}, true); err != nil {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ LoadResponse 失败：%v", err))
			h.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}
		body := h.Response.Body()
		if len(body) == 0 {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 响应体为空，URL：%s", reqURL))
			h.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}

		// 处理响应体编码和压缩
		responseBody, err := s.processResponseBody([]byte(body), h.Response.Headers())
		if err != nil {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 处理响应体失败：%v", err))
			h.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}

		// 尝试解析视频分析数据
		videoData := parseVideoAnalysisResponse(responseBody, product)
		if len(videoData) > 0 {
			logInfo(ctx, taskId, fmt.Sprintf("🔍 拦截到 flow_more API 响应，解析到 %d 条视频数据", len(videoData)))
			select {
			case videoCh <- videoData:
			default:
				logWarn(ctx, taskId, "⚠️ videoCh 已满，丢弃数据")
			}
		}
		// LoadResponse 已经将真实响应写入 HijackResponse，交由 router 返回给页面。
	}); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 注册 flow_more API 拦截失败：%v", err))
		return videos
	}

	go router.Run()
	defer router.Stop()

	// 导航到视频分析页面
	analysisURL := buildVideoAnalysisURL(product)
	if err := s.retryWithBackoff(func() error {
		return page.Navigate(analysisURL)
	}, 3); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 打开视频分析失败：%v", err))
		return videos
	}

	page.WaitLoad()

	// 等待API响应，最多10秒
	select {
	case videos = <-videoCh:
		logInfo(ctx, taskId, fmt.Sprintf("✅ 通过API拦截获取到视频分析数据：%d 条", len(videos)))
	case <-time.After(10 * time.Second):
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 视频分析API响应超时：%s", product.ProductName))
	}

	return videos
}

// FlowMoreTemplate 存储从浏览器提取的 API 模板（借鉴 Python 爬虫的 _bootstrap_flow_more_template）
type FlowMoreTemplate struct {
	URLTemplate string
	Headers     map[string]string
}

// bootstrapFlowMoreTemplate 从浏览器提取 flow_more API 模板
// 打开第一个商品的详情页，拦截 flow_more API 响应，提取 URL 和 Headers
func (s *sCrawler) bootstrapFlowMoreTemplate(
	ctx context.Context,
	browser *rod.Browser,
	products []ProductInfo,
	taskId string,
) (*FlowMoreTemplate, error) {
	if len(products) == 0 {
		return nil, fmt.Errorf("商品列表为空")
	}

	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return nil, fmt.Errorf("无法打开浏览器页面：%v", err)
	}
	defer page.Close()

	// 设置 API 拦截器
	templateCh := make(chan *FlowMoreTemplate, 1)
	router := page.HijackRequests()

	// 拦截 flow_more API 响应
	if err := router.Add("*", "", func(h *rod.Hijack) {
		// 记录请求 URL 和 Headers
		reqURL := h.Request.URL().String()
		if !isVideoAnalysisAPIURL(reqURL, products[0].ProductID) {
			h.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}

		// 提取 Headers
		headers := make(map[string]string)
		for k, v := range h.Request.Headers() {
			headers[k] = v.Str()
		}

		template := &FlowMoreTemplate{
			URLTemplate: reqURL,
			Headers:     headers,
		}

		select {
		case templateCh <- template:
			logInfo(ctx, taskId, fmt.Sprintf("✅ 成功提取 flow_more API 模板"))
		default:
		}

		h.ContinueRequest(&proto.FetchContinueRequest{})
	}); err != nil {
		return nil, fmt.Errorf("注册 flow_more 拦截失败：%v", err)
	}

	go router.Run()
	defer router.Stop()

	// 打开第一个商品的详情页
	product := products[0]
	detailURL := buildVideoAnalysisURL(product)

	if err := s.retryWithBackoff(func() error {
		return page.Navigate(detailURL)
	}, 3); err != nil {
		return nil, fmt.Errorf("打开商品详情失败：%v", err)
	}

	page.WaitLoad()
	time.Sleep(2 * time.Second)

	// 点击视频分析 Tab 触发 flow_more 请求
	videoTabSelectors := []string{
		"//*[contains(text(), '视频分析')]",
		".//div[contains(@class, 'tab') and contains(text(), '视频')]",
	}
	for _, selector := range videoTabSelectors {
		if elem, err := page.Timeout(5 * time.Second).ElementX(selector); err == nil {
			_ = elem.Click(proto.InputMouseButtonLeft, 1)
			break
		}
	}

	// 等待 flow_more 响应
	select {
	case template := <-templateCh:
		return template, nil
	case <-time.After(15 * time.Second):
		return nil, fmt.Errorf("提取 flow_more API 模板超时")
	}
}

// replaceURLParam 替换 URL 中的查询参数（使用标准库，更可靠）
func replaceURLParam(rawURL, key, value string) string {
	// 优先使用标准库解析
	if u, err := netUrl.Parse(rawURL); err == nil {
		q := u.Query()
		q.Set(key, value)
		u.RawQuery = q.Encode()
		return u.String()
	}
	// fallback: 正则替换
	re := regexp.MustCompile(fmt.Sprintf(`(%s=)[^&]*`, key))
	return re.ReplaceAllString(rawURL, fmt.Sprintf("${1}%s", value))
}

// ===== flow_more API 响应结构（与 Python 版本一致） =====

// FlowMoreResponse flow_more API 完整响应
type FlowMoreResponse struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Msg     string       `json:"msg"`
	Data    FlowMoreData `json:"data"`
}

// FlowMoreData flow_more 响应 data 层
type FlowMoreData struct {
	VideoList []FlowVideoItem `json:"video_list"`
	Total     int             `json:"total"`
	HasMore   bool            `json:"has_more"`
}

// FlowVideoItem flow_more 视频条目（深层嵌套结构）
type FlowVideoItem struct {
	BaseInfo struct {
		VideoBaseInfo struct {
			VideoID     string `json:"video_id"`
			VideoTitle  string `json:"video_title"`
			VideoImage  string `json:"video_image"`
			PublishTime string `json:"publish_time"`
			AuthorInfo  struct {
				AuthorNickName string `json:"author_nick_name"`
			} `json:"author_info"`
		} `json:"video_base_info"`
	} `json:"base_info"`
	Metrics FlowMetrics `json:"metrics"`
}

func convertFlowVideoItems(items []FlowVideoItem, product ProductInfo) []VideoAnalysis {
	videos := make([]VideoAnalysis, 0, len(items))
	for _, item := range items {
		base := item.BaseInfo.VideoBaseInfo
		if base.VideoID == "" {
			continue
		}
		videos = append(videos, VideoAnalysis{
			ProductID: product.ProductID, ProductName: product.ProductName,
			VideoId: base.VideoID, VideoDesc: base.VideoTitle,
			AuthorName: base.AuthorInfo.AuthorNickName, CoverURL: base.VideoImage,
			PublishTime: base.PublishTime, SourceURL: fmt.Sprintf("https://www.douyin.com/video/%s", base.VideoID),
			PaymentAmount: item.Metrics.PayAmt.FormatRange(), ExposeUserCount: item.Metrics.ShowUV.FormatRange(),
			ClickUserCount: item.Metrics.DhProductClickUV.FormatRange(), ConvertUserCount: item.Metrics.PayUcnt.FormatRange(),
			ClickRate: item.Metrics.ClickRatio.FormatRange(), ClickConvertRate: item.Metrics.ClickConversion.FormatRange(),
			ExposeConvertRate: item.Metrics.ShowConversion.FormatRange(), OrderCount: item.Metrics.PayCnt.FormatRange(),
			SalesVolume: item.Metrics.PayComboCnt.FormatRange(),
		})
	}
	return videos
}

// FlowMetrics 指标集合
type FlowMetrics struct {
	PayAmt           MetricField `json:"pay_amt"`
	DhProductClickUV MetricField `json:"dh_product_click_uv"`
	PayComboCnt      MetricField `json:"pay_combo_cnt"`
	ShowUV           MetricField `json:"show_uv"`
	PayUcnt          MetricField `json:"pay_ucnt"`
	ClickRatio       MetricField `json:"click_ratio"`
	ClickConversion  MetricField `json:"click_conversion"`
	ShowConversion   MetricField `json:"show_conversion"`
	PayCnt           MetricField `json:"pay_cnt"`
}

// MetricField 指标字段（包含 value 范围和单位）
type MetricField struct {
	Value      MetricValue `json:"value"`
	ExtraValue string      `json:"extra_value"`
	Unit       string      `json:"unit"`
}

// MetricValue 指标值范围
type MetricValue struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
}

// FormatRange 格式化指标范围（处理单位转换：分→元、小数→百分比）
func (m MetricField) FormatRange() string {
	lower, upper := m.Value.Lower, m.Value.Upper
	switch m.Unit {
	case "fen": // 分 → 元
		lower /= 100
		upper /= 100
		return fmt.Sprintf("%.0f-%.0f", lower, upper)
	case "ratio": // 小数 → 百分比
		return fmt.Sprintf("%.2f%%-%.2f%%", lower*100, upper*100)
	case "wan": // 万
		return fmt.Sprintf("%.1f万-%.1f万", lower/10000, upper/10000)
	default:
		if lower == 0 && upper == 0 {
			return ""
		}
		return fmt.Sprintf("%.0f-%.0f", lower, upper)
	}
}

// FormatFloor 取范围下限值（用于 total_gmv 等排序场景）
func (m MetricField) FormatFloor() string {
	switch m.Unit {
	case "fen":
		return fmt.Sprintf("%.0f", m.Value.Lower/100)
	default:
		return fmt.Sprintf("%.0f", m.Value.Lower)
	}
}

// VideoPageResult 视频分页结果
type VideoPageResult struct {
	Videos   []VideoAnalysis
	Total    int
	PageSize int
	HasMore  bool
}

// fetchVideoPage 获取单页视频数据（使用正确的嵌套 JSON 结构 + 单位转换）
func (s *sCrawler) fetchVideoPage(
	ctx context.Context,
	template *FlowMoreTemplate,
	product ProductInfo,
	pageNo int,
) (*VideoPageResult, error) {
	// 替换 URL 中的 product_id 和 page_no
	requestURL := replaceURLParam(template.URLTemplate, "product_id", product.ProductID)
	requestURL = replaceURLParam(requestURL, "page_no", fmt.Sprintf("%d", pageNo))

	// 发送请求（带重试）
	client := &http.Client{Timeout: 30 * time.Second}
	var body []byte
	retryErr := s.retryWithBackoff(func() error {
		req, reqErr := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
		if reqErr != nil {
			return reqErr
		}
		for k, v := range template.Headers {
			lower := strings.ToLower(k)
			if strings.HasPrefix(k, ":") || lower == "host" || lower == "content-length" ||
				lower == "connection" || lower == "accept-encoding" {
				continue
			}
			req.Header.Set(k, v)
		}
		req.Header.Set("Referer", buildVideoAnalysisURL(product))

		resp, reqErr := client.Do(req)
		if reqErr != nil {
			return reqErr
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			return fmt.Errorf("视频接口返回 HTTP %d", resp.StatusCode)
		}
		var readErr error
		body, readErr = io.ReadAll(resp.Body)
		return readErr
	}, 3)
	if retryErr != nil {
		return nil, gerror.Wrap(retryErr, "请求失败（重试3次后）")
	}

	// 解析嵌套 JSON 结构
	var resp FlowMoreResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, gerror.Wrap(err, "解析响应失败")
	}
	if resp.Code != 0 {
		message := resp.Message
		if message == "" {
			message = resp.Msg
		}
		return nil, gerror.Newf("视频接口业务失败 code=%d message=%s", resp.Code, message)
	}
	if resp.Data.Total > 0 && len(resp.Data.VideoList) == 0 {
		return nil, gerror.New("视频接口返回了总数但解析列表为空，需要更新响应结构")
	}

	// 构建结果（使用正确的嵌套结构提取字段）
	result := &VideoPageResult{
		Total:    resp.Data.Total,
		PageSize: 10, // 默认每页 10 条
		HasMore:  resp.Data.HasMore,
	}

	result.Videos = convertFlowVideoItems(resp.Data.VideoList, product)

	return result, nil
}

// fetchVideosWithPagination 分页获取视频数据（借鉴 Python 爬虫的 _collect_flow_more_payloads）
func (s *sCrawler) fetchVideosWithPagination(
	ctx context.Context,
	template *FlowMoreTemplate,
	product ProductInfo,
) ([]VideoAnalysis, error) {
	var allVideos []VideoAnalysis

	// 请求第 1 页
	page1Result, err := s.fetchVideoPage(ctx, template, product, 1)
	if err != nil {
		return nil, err
	}
	allVideos = append(allVideos, page1Result.Videos...)

	// 没有更多页时直接返回
	if !page1Result.HasMore && (page1Result.Total == 0 || page1Result.PageSize == 0) {
		return allVideos, nil
	}

	// 计算总页数
	totalPages := (page1Result.Total + page1Result.PageSize - 1) / page1Result.PageSize
	if totalPages <= 1 && !page1Result.HasMore {
		return allVideos, nil
	}
	if totalPages < 2 && page1Result.HasMore {
		totalPages = 20 // 缺少 total 时用 has_more 控制，并设置安全上限
	}
	if totalPages > 20 {
		totalPages = 20
	}

	// 请求剩余页
	for page := 2; page <= totalPages; page++ {
		if err := ctx.Err(); err != nil {
			return allVideos, err
		}
		pageResult, err := s.fetchVideoPage(ctx, template, product, page)
		if err != nil {
			// 出错停止分页，返回已获取的数据
			break
		}
		allVideos = append(allVideos, pageResult.Videos...)
		if !pageResult.HasMore && page*pageResult.PageSize >= pageResult.Total {
			break
		}
		select {
		case <-ctx.Done():
			return allVideos, ctx.Err()
		case <-time.After(700 * time.Millisecond):
		}
	}

	return allVideos, nil
}

// scrapeVideoAnalysisOptimized 保留旧方法名以兼容内部调用；实际已强制切换为纯 DOM 模式。
func (s *sCrawler) scrapeVideoAnalysisOptimized(ctx context.Context, browser *rod.Browser, products []ProductInfo, taskId string) []VideoAnalysis {
	return s.scrapeVideoAnalysisDOM(ctx, browser, products, taskId)
}

// parseVideoAnalysisResponse 解析视频分析 API 响应 JSON
func parseVideoAnalysisResponse(body string, product ProductInfo) []VideoAnalysis {
	var nested FlowMoreResponse
	if err := json.Unmarshal([]byte(body), &nested); err == nil {
		if videos := convertFlowVideoItems(nested.Data.VideoList, product); len(videos) > 0 {
			return videos
		}
	}

	// 兼容早期扁平结构，便于在平台灰度切换期间继续工作。
	type videoItem struct {
		VideoId           string `json:"video_id"`
		VideoDesc         string `json:"video_desc"`
		PaymentAmount     string `json:"payment_amount"`
		ExposeUserCount   string `json:"expose_user_count"`
		ClickUserCount    string `json:"click_user_count"`
		ConvertUserCount  string `json:"convert_user_count"`
		ClickRate         string `json:"click_rate"`
		ClickConvertRate  string `json:"click_convert_rate"`
		ExposeConvertRate string `json:"expose_convert_rate"`
		OrderCount        string `json:"order_count"`
		SalesVolume       string `json:"sales_volume"`
		AuthorName        string `json:"author_name"`
		CoverURL          string `json:"video_image"`
		PublishTime       string `json:"publish_time"`
	}

	type response struct {
		Data struct {
			VideoList []videoItem `json:"video_list"`
		} `json:"data"`
	}

	var resp response
	_ = json.Unmarshal([]byte(body), &resp)

	videos := make([]VideoAnalysis, 0, len(resp.Data.VideoList))
	for _, item := range resp.Data.VideoList {
		if item.VideoId == "" {
			continue
		}
		video := VideoAnalysis{
			ProductID:         product.ProductID,
			ProductName:       product.ProductName,
			VideoId:           item.VideoId,
			VideoDesc:         item.VideoDesc,
			AuthorName:        item.AuthorName,
			CoverURL:          item.CoverURL,
			PublishTime:       item.PublishTime,
			SourceURL:         fmt.Sprintf("https://www.douyin.com/video/%s", item.VideoId),
			PaymentAmount:     item.PaymentAmount,
			ExposeUserCount:   item.ExposeUserCount,
			ClickUserCount:    item.ClickUserCount,
			ConvertUserCount:  item.ConvertUserCount,
			ClickRate:         item.ClickRate,
			ClickConvertRate:  item.ClickConvertRate,
			ExposeConvertRate: item.ExposeConvertRate,
			OrderCount:        item.OrderCount,
			SalesVolume:       item.SalesVolume,
		}
		videos = append(videos, video)
	}
	if len(videos) > 0 {
		return videos
	}

	if videos := parseGenericFlowMoreRows(body, product); len(videos) > 0 {
		return videos
	}

	// 平台改版导致字段层级变化时，从该商品专属的浏览器响应文本中
	// 兜底提取真实 video_id / aweme_id，只接受纯数字 ID。
	for _, identity := range extractAllVideoIdentitiesFromText(body) {
		videos = append(videos, VideoAnalysis{
			ProductID:   product.ProductID,
			ProductName: product.ProductName,
			VideoId:     identity.VideoId,
			VideoDesc:   product.ProductName,
			SourceURL:   identity.SourceURL,
		})
	}

	return videos
}

func parseGenericFlowMoreRows(body string, product ProductInfo) []VideoAnalysis {
	decoder := json.NewDecoder(bytes.NewReader([]byte(body)))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return nil
	}

	rows := candidateVideoRows(root)
	videos := make([]VideoAnalysis, 0, len(rows))
	seen := make(map[string]struct{})
	for _, row := range rows {
		videoID := normalizeVideoIDCandidate(firstRecursiveString(row,
			"video_id", "aweme_id", "aweme_id_str", "item_id", "item_id_str",
		))
		if videoID == "" {
			for _, identity := range extractAllVideoIdentitiesFromText(marshalCompactString(row)) {
				videoID = identity.VideoId
				break
			}
		}
		if videoID == "" {
			continue
		}
		if _, ok := seen[videoID]; ok {
			continue
		}
		seen[videoID] = struct{}{}

		videoDesc := firstRecursiveString(row,
			"video_title", "video_desc", "item_title", "title", "desc", "content",
		)
		if strings.TrimSpace(videoDesc) == "" {
			videoDesc = product.ProductName
		}

		videos = append(videos, VideoAnalysis{
			ProductID:         product.ProductID,
			ProductName:       product.ProductName,
			VideoId:           videoID,
			VideoDesc:         videoDesc,
			AuthorName:        firstRecursiveString(row, "author_nick_name", "author_name", "nick_name", "nickname"),
			CoverURL:          firstRecursiveString(row, "video_image", "cover_url", "cover", "image_url", "avatar_url"),
			PublishTime:       firstRecursiveString(row, "publish_time", "create_time", "release_time"),
			SourceURL:         "https://www.douyin.com/video/" + videoID,
			PaymentAmount:     firstMetricString(row, "pay_amt", "payment_amount", "gmv"),
			ExposeUserCount:   firstMetricString(row, "show_uv", "expose_user_count", "exposure_user_count"),
			ClickUserCount:    firstMetricString(row, "dh_product_click_uv", "click_user_count", "product_click_user_count"),
			ConvertUserCount:  firstMetricString(row, "pay_ucnt", "convert_user_count"),
			ClickRate:         firstMetricString(row, "click_ratio", "click_rate"),
			ClickConvertRate:  firstMetricString(row, "click_conversion", "click_convert_rate"),
			ExposeConvertRate: firstMetricString(row, "show_conversion", "expose_convert_rate"),
			OrderCount:        firstMetricString(row, "pay_cnt", "order_count"),
			SalesVolume:       firstMetricString(row, "pay_combo_cnt", "sales_volume"),
		})
	}
	return videos
}

func candidateVideoRows(root any) []any {
	rows := make([]any, 0)
	if rootMap, ok := root.(map[string]any); ok {
		if data, exists := rootMap["data"]; exists {
			switch typed := data.(type) {
			case []any:
				return typed
			case map[string]any:
				for _, key := range []string{"video_list", "list", "items", "rows", "result"} {
					if list, ok := typed[key].([]any); ok {
						return list
					}
				}
			}
		}
	}
	collectLikelyVideoRows(root, &rows, 0)
	return rows
}

func collectLikelyVideoRows(value any, rows *[]any, depth int) {
	if depth > 4 || len(*rows) >= 200 {
		return
	}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if containsRecursiveKey(item, "video_id", "aweme_id", "aweme_id_str", "item_id", "item_id_str") {
				*rows = append(*rows, item)
			} else {
				collectLikelyVideoRows(item, rows, depth+1)
			}
		}
	case map[string]any:
		for _, item := range typed {
			collectLikelyVideoRows(item, rows, depth+1)
		}
	}
}

func containsRecursiveKey(value any, keys ...string) bool {
	return firstRecursiveString(value, keys...) != ""
}

func firstRecursiveString(value any, keys ...string) string {
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[strings.ToLower(key)] = struct{}{}
	}
	return firstRecursiveStringBySet(value, keySet, 0)
}

func firstRecursiveStringBySet(value any, keySet map[string]struct{}, depth int) string {
	if depth > 8 {
		return ""
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if _, ok := keySet[strings.ToLower(key)]; ok {
				if candidate := stringFromJSONValue(child); strings.TrimSpace(candidate) != "" {
					return candidate
				}
			}
		}
		keys := sortedMapKeys(typed)
		for _, key := range keys {
			if candidate := firstRecursiveStringBySet(typed[key], keySet, depth+1); strings.TrimSpace(candidate) != "" {
				return candidate
			}
		}
	case []any:
		for _, child := range typed {
			if candidate := firstRecursiveStringBySet(child, keySet, depth+1); strings.TrimSpace(candidate) != "" {
				return candidate
			}
		}
	}
	return ""
}

func stringFromJSONValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		if math.Trunc(typed) == typed {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	case nil:
		return ""
	default:
		return ""
	}
}

func normalizeVideoIDCandidate(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if numericVideoID.MatchString(candidate) {
		return candidate
	}
	return ""
}

func marshalCompactString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func firstMetricString(value any, keys ...string) string {
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[strings.ToLower(key)] = struct{}{}
	}
	metric := firstMetricValueBySet(value, keySet, 0)
	return formatGenericMetric(metric)
}

func firstMetricValueBySet(value any, keySet map[string]struct{}, depth int) any {
	if depth > 8 {
		return nil
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if _, ok := keySet[strings.ToLower(key)]; ok {
				return child
			}
		}
		keys := sortedMapKeys(typed)
		for _, key := range keys {
			if candidate := firstMetricValueBySet(typed[key], keySet, depth+1); candidate != nil {
				return candidate
			}
		}
	case []any:
		for _, child := range typed {
			if candidate := firstMetricValueBySet(child, keySet, depth+1); candidate != nil {
				return candidate
			}
		}
	}
	return nil
}

func formatGenericMetric(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case map[string]any:
		if extra := stringFromJSONValue(typed["extra_value"]); extra != "" {
			return extra
		}
		unit := stringFromJSONValue(typed["unit"])
		if valueMap, ok := typed["value"].(map[string]any); ok {
			if lower, lowerOK := numberFromJSONValue(valueMap["lower"]); lowerOK {
				if upper, upperOK := numberFromJSONValue(valueMap["upper"]); upperOK {
					return (MetricField{
						Value: MetricValue{Lower: lower, Upper: upper},
						Unit:  unit,
					}).FormatRange()
				}
			}
		}
		if valueNode, exists := typed["value"]; exists {
			return formatGenericMetric(valueNode)
		}
		if lower, lowerOK := numberFromJSONValue(typed["lower"]); lowerOK {
			if upper, upperOK := numberFromJSONValue(typed["upper"]); upperOK {
				return (MetricField{
					Value: MetricValue{Lower: lower, Upper: upper},
					Unit:  unit,
				}).FormatRange()
			}
		}
	}
	return ""
}

func numberFromJSONValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		number, err := typed.Float64()
		return number, err == nil
	case float64:
		return typed, true
	case string:
		number, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return number, err == nil
	default:
		return 0, false
	}
}

// scrapeVideoAnalysisDOM 只通过用户可见页面采集，不重放罗盘内部接口。
func (s *sCrawler) scrapeVideoAnalysisDOM(ctx context.Context, browser *rod.Browser, products []ProductInfo, taskId string) []VideoAnalysis {
	return s.scrapeVideoAnalysisDOMWithSink(ctx, browser, products, taskId, nil)
}

func (s *sCrawler) scrapeVideoAnalysisDOMWithSink(
	ctx context.Context,
	browser *rod.Browser,
	products []ProductInfo,
	taskId string,
	onDiscovered func([]VideoAnalysis),
) []VideoAnalysis {
	return s.scrapeVideoAnalysisDOMWithSinkFrom(ctx, browser, products, taskId, 0, nil, onDiscovered).Videos
}

func (s *sCrawler) scrapeVideoAnalysisDOMWithSinkFrom(
	ctx context.Context,
	browser *rod.Browser,
	products []ProductInfo,
	taskId string,
	startIndex int,
	checkpointManager *service.CheckpointManager,
	onDiscovered func([]VideoAnalysis),
) videoScrapeResult {
	allVideos := make([]VideoAnalysis, 0)
	maxDetailViews := g.Cfg().MustGet(ctx, "crawler.video.maxDetailViews", 10).Int()
	maxEmptyStreak := g.Cfg().MustGet(ctx, "crawler.video.maxEmptyStreak", 4).Int()
	emptyStreak := 0
	if startIndex < 0 {
		startIndex = 0
	}
	if startIndex > len(products) {
		startIndex = len(products)
	}
	viewsThisRun := 0
	for index := startIndex; index < len(products); index++ {
		product := products[index]
		if ctx.Err() != nil {
			break
		}
		if maxDetailViews > 0 && viewsThisRun >= maxDetailViews {
			reason := fmt.Sprintf(
				"⏸️ 已达到本轮视频详情查看保护上限：%d 次；停止继续打开详情页，避免消耗官方查看次数",
				maxDetailViews,
			)
			logWarn(ctx, taskId, reason)
			return videoScrapeResult{
				Videos:           deduplicateVideos(allVideos),
				Paused:           true,
				NextProductIndex: index,
				StopReason:       reason,
			}
		}
		logInfo(ctx, taskId, fmt.Sprintf("🎬 [%d/%d] 通过 UI 打开视频分析：%s", index+1, len(products), product.ProductName))
		viewsThisRun++
		pageResult := s.scrapeSingleProductVideosWithStatus(ctx, browser, product, taskId)
		if pageResult.Status == videoPageAccessDailyLimit {
			reason := fmt.Sprintf(
				"⏸️ 官方每日详情查看次数已达上限，停止本日视频访问；已处理=%d，总数=%d，剩余=%d，原因=%s",
				index, len(products), len(products)-index, pageResult.Note,
			)
			logWarn(ctx, taskId, reason)
			return videoScrapeResult{
				Videos:           deduplicateVideos(allVideos),
				Paused:           true,
				NextProductIndex: index,
				StopReason:       reason,
			}
		}
		videos := pageResult.Videos
		if len(videos) > 0 {
			videos = deduplicateVideos(videos)
			allVideos = append(allVideos, videos...)
			if onDiscovered != nil {
				onDiscovered(videos)
			}
			logInfo(ctx, taskId, fmt.Sprintf("✅ [%d/%d] 获取 %d 条真实视频", index+1, len(products), len(videos)))
			emptyStreak = 0
		} else {
			emptyStreak++
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ [%d/%d] 页面未发现可识别的真实 video_id：%s", index+1, len(products), product.ProductName))
			if maxEmptyStreak > 0 && emptyStreak >= maxEmptyStreak {
				reason := fmt.Sprintf(
					"⏸️ 连续 %d 个视频详情页未拿到可识别视频数据；停止本轮视频访问，避免继续空跑和消耗查看次数",
					emptyStreak,
				)
				logWarn(ctx, taskId, reason)
				return videoScrapeResult{
					Videos:           deduplicateVideos(allVideos),
					Paused:           true,
					NextProductIndex: index + 1,
					StopReason:       reason,
				}
			}
		}
		if checkpointManager != nil {
			_ = checkpointManager.UpdateVideoProgress(ctx, taskId, index+1, len(products), s.countVideoRows(ctx, taskId), "running", "")
		}
		select {
		case <-ctx.Done():
			return videoScrapeResult{
				Videos:           deduplicateVideos(allVideos),
				Paused:           true,
				NextProductIndex: index + 1,
				StopReason:       "任务被取消",
			}
		case <-time.After(1200 * time.Millisecond):
		}
	}
	return videoScrapeResult{
		Videos:           deduplicateVideos(allVideos),
		Completed:        true,
		NextProductIndex: len(products),
	}
}

func limitProductsForDetail(products []ProductInfo, limit int) []ProductInfo {
	if limit <= 0 || limit >= len(products) {
		return products
	}
	return products[:limit]
}

func (s *sCrawler) persistAndQueueVideos(ctx context.Context, taskId string, videos []VideoAnalysis) error {
	videos = deduplicateVideos(videos)
	if len(videos) == 0 {
		return nil
	}
	if err := s.saveVideoAnalysis(ctx, taskId, videos); err != nil {
		return err
	}
	if !g.Cfg().MustGet(ctx, "crawler.video.autoDownload", true).Bool() {
		return nil
	}
	videoIDs := make([]string, 0, len(videos))
	for _, video := range videos {
		videoIDs = append(videoIDs, video.VideoId)
	}
	result := GetVideoDownloadManager().EnqueueBatch(ctx, videoIDs)
	logInfo(ctx, taskId, fmt.Sprintf("🎬 视频已入库并排队解析下载直链：新排队=%d，已解析=%d，跳过=%d，失败=%d",
		result.Queued, result.Completed, result.Skipped, result.Failed))
	return nil
}

// scrapeSingleProductVideos 获取单个商品的视频分析数据（改进版）
func (s *sCrawler) scrapeSingleProductVideos(ctx context.Context, browser *rod.Browser, product ProductInfo, taskId string) []VideoAnalysis {
	return s.scrapeSingleProductVideosWithStatus(ctx, browser, product, taskId).Videos
}

func (s *sCrawler) scrapeSingleProductVideosWithStatus(ctx context.Context, browser *rod.Browser, product ProductInfo, taskId string) videoPageResult {
	result := videoPageResult{Status: videoPageAccessAvailable}

	// 重试打开页面
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 无法打开浏览器页面：%v", err))
		return result
	}
	defer page.Close()

	// 详情页的视频标识通常只存在于浏览器原生 flow_more 响应中，
	// DOM 未必包含 video_id。这里只读取页面已经收到的响应，不重放接口。
	stopObserver, videoResponseCh := s.setupBrowserVideoResponseObserver(ctx, page, taskId, product)
	defer stopObserver()

	// 导航到视频分析页面（带重试）
	analysisURL := buildVideoAnalysisURL(product)
	if err := s.retryWithBackoff(func() error {
		return page.Navigate(analysisURL)
	}, 3); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 打开视频分析失败：%v", err))
		return result
	}

	page.WaitLoad()
	time.Sleep(2 * time.Second)

	// URL 参数未触发 Tab 时，通过可见文本点击“视频分析”。
	if tab, err := page.Timeout(3 * time.Second).ElementX("//*[contains(text(), '视频分析')]"); err == nil {
		_ = tab.Click(proto.InputMouseButtonLeft, 1)
		time.Sleep(2 * time.Second)
	}
	if !s.waitForRiskControl(ctx, page, taskId) {
		return result
	}

	// 优先使用正常页面触发后，Chrome 已经接收到的原生响应。
	// 有些页面会同时渲染“竞对详情受限”遮罩和已加载的部分数据，
	// 因此先消费浏览器已经收到的响应，避免把可见/已返回的数据误判为空。
	select {
	case response := <-videoResponseCh:
		result.Diagnostics = append(result.Diagnostics, response.Diagnostics...)
		if response.Status == videoPageAccessDailyLimit {
			return response
		}
		if len(response.Videos) > 0 {
			logInfo(ctx, taskId, fmt.Sprintf("✅ 从浏览器原生视频响应解析到 %d 条视频", len(response.Videos)))
			return response
		}
		if response.Status == videoPageAccessNoData {
			result.Status = response.Status
			result.Note = response.Note
		}
	case <-ctx.Done():
		return result
	case <-time.After(5 * time.Second):
		// 没有匹配响应时继续使用 DOM 兜底。
	}

	if blocked, note := s.detectCompetitorDetailBlock(ctx, page, taskId); blocked {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 商品 [%s] 视频详情受限：%s", product.ProductName, note))
		result.Note = note
		if isDailyVideoViewLimit(note) {
			result.Status = videoPageAccessDailyLimit
		} else {
			result.Status = videoPageAccessRestricted
		}
		return result
	}

	// 获取核心数据
	coreData := s.scrapeVideoCoreData(ctx, page, taskId)

	// 获取短视频列表
	videoList := s.scrapeVideoList(ctx, page, taskId, product.ProductName)
	for _, v := range videoList {
		v.PaymentAmount = coreData.PaymentAmount
		v.ProductID = product.ProductID
		v.ProductName = product.ProductName
		result.Videos = append(result.Videos, v)
	}

	return result
}

// scrapeVideoCoreData 获取视频核心数据
func (s *sCrawler) scrapeVideoCoreData(ctx context.Context, page *rod.Page, taskId string) VideoAnalysis {
	video := VideoAnalysis{}

	// 查找核心数据指标
	metrics := map[string]*string{
		"用户支付金额": &video.PaymentAmount,
		"曝光人数":   &video.ExposeUserCount,
		"点击人数":   &video.ClickUserCount,
		"成交人数":   &video.ConvertUserCount,
		"点击率":    &video.ClickRate,
		"成交订单数":  &video.OrderCount,
		"成交件数":   &video.SalesVolume,
	}

	for label, value := range metrics {
		xpath := fmt.Sprintf("//*[contains(text(), '%s')]/following-sibling::*[1]", label)
		if elem, err := page.Timeout(2 * time.Second).ElementX(xpath); err == nil {
			if text, err := elem.Text(); err == nil {
				*value = strings.TrimSpace(text)
			}
		}
	}

	return video
}

// scrapeVideoList 获取短视频列表数据
func (s *sCrawler) scrapeVideoList(ctx context.Context, page *rod.Page, taskId string, productName string) []VideoAnalysis {
	videos := make([]VideoAnalysis, 0)
	seen := make(map[string]bool)

	videoRows, err := page.Elements("[data-video-id], [data-row-key], .video-item, .video-row, .ant-table-tbody .ant-table-row")
	if err != nil {
		return videos
	}

	for _, row := range videoRows {
		videoID, sourceURL := extractVideoIdentityFromElement(row)
		if videoID == "" || seen[videoID] {
			continue
		}
		seen[videoID] = true
		desc, _ := row.Text()
		desc = strings.TrimSpace(desc)
		if desc == "" {
			desc = productName
		}
		coverURL := ""
		if image, imageErr := row.Element("img"); imageErr == nil {
			if src, srcErr := image.Attribute("src"); srcErr == nil && src != nil {
				coverURL = *src
			}
		}
		videos = append(videos, VideoAnalysis{
			VideoId: videoID, VideoDesc: desc, CoverURL: coverURL, SourceURL: sourceURL,
		})
	}

	// 表格结构改版时，从页面上真实抖音链接补充 video_id。
	if links, linkErr := page.Elements("a[href*='/video/'], a[href*='share/video']"); linkErr == nil {
		for _, link := range links {
			videoID, sourceURL := extractVideoIdentityFromElement(link)
			if videoID == "" || seen[videoID] {
				continue
			}
			seen[videoID] = true
			desc, _ := link.Text()
			videos = append(videos, VideoAnalysis{VideoId: videoID, VideoDesc: strings.TrimSpace(desc), SourceURL: sourceURL})
		}
	}

	// React 表格可能不把 video_id 放在可见节点属性中，但会保留在
	// 页面已渲染的 JSON/链接里。只扫描当前浏览器 DOM，不发起新请求。
	if pageHTML, htmlErr := page.HTML(); htmlErr == nil {
		for _, identity := range extractAllVideoIdentitiesFromText(pageHTML) {
			if seen[identity.VideoId] {
				continue
			}
			seen[identity.VideoId] = true
			videos = append(videos, VideoAnalysis{
				VideoId: identity.VideoId, VideoDesc: productName, SourceURL: identity.SourceURL,
			})
		}
	}
	return videos
}

// setupBrowserVideoResponseObserver 读取正常详情页已经收到的视频响应。
// 它不拦截、不修改，也不会使用独立 HTTP 客户端重放罗盘接口。
func (s *sCrawler) setupBrowserVideoResponseObserver(
	ctx context.Context,
	page *rod.Page,
	taskId string,
	product ProductInfo,
) (func(), chan videoPageResult) {
	resultCh := make(chan videoPageResult, 4)
	observerCtx, cancel := context.WithCancel(ctx)
	observedPage := page.Context(observerCtx)
	cacheSize := 32 * 1024 * 1024
	if err := (proto.NetworkEnable{MaxTotalBufferSize: &cacheSize}).Call(observedPage); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 启用视频响应监听失败：%v", err))
	}

	pending := make(map[proto.NetworkRequestID]string)
	var pendingMu sync.Mutex
	wait := observedPage.EachEvent(
		func(event *proto.NetworkResponseReceived) {
			if event.Response == nil {
				return
			}
			if !isVideoAnalysisAPIURL(event.Response.URL, product.ProductID) &&
				!isCompetitorAccessControlAPIURL(event.Response.URL) {
				return
			}
			pendingMu.Lock()
			pending[event.RequestID] = event.Response.URL
			pendingMu.Unlock()
		},
		func(event *proto.NetworkLoadingFinished) {
			pendingMu.Lock()
			responseURL, ok := pending[event.RequestID]
			delete(pending, event.RequestID)
			pendingMu.Unlock()
			if !ok {
				return
			}

			bodyResult, err := (proto.NetworkGetResponseBody{RequestID: event.RequestID}).Call(observedPage)
			if err != nil {
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ 读取浏览器视频响应失败：%v", err))
				return
			}
			body := bodyResult.Body
			if bodyResult.Base64Encoded {
				decoded, decodeErr := base64.StdEncoding.DecodeString(bodyResult.Body)
				if decodeErr != nil {
					logWarn(ctx, taskId, fmt.Sprintf("⚠️ 解码浏览器视频响应失败：%v", decodeErr))
					return
				}
				body = string(decoded)
			}

			if isCompetitorAccessControlAPIURL(responseURL) {
				diagnostic := buildVideoResponseDiagnostic(body, responseURL, product.ProductID, 0)
				logAccessControlDiagnostic(ctx, taskId, diagnostic)
				return
			}

			if note := dailyVideoViewLimitMarker(body); note != "" {
				diagnostic := buildVideoResponseDiagnostic(body, responseURL, product.ProductID, 0)
				logVideoResponseDiagnostic(ctx, taskId, diagnostic)
				select {
				case resultCh <- videoPageResult{
					Status:      videoPageAccessDailyLimit,
					Note:        note,
					Diagnostics: []VideoResponseDiagnostic{diagnostic},
				}:
				default:
				}
				return
			}

			videos := parseVideoAnalysisResponse(body, product)
			diagnostic := buildVideoResponseDiagnostic(body, responseURL, product.ProductID, len(videos))
			logInfo(ctx, taskId, fmt.Sprintf(
				"🔍 读取到浏览器原生视频响应，body=%d，视频=%d，URL=%s",
				len(body), len(videos), responseURL,
			))
			logVideoResponseDiagnostic(ctx, taskId, diagnostic)
			if len(videos) == 0 {
				select {
				case resultCh <- videoPageResult{
					Status:      videoPageAccessNoData,
					Note:        "浏览器原生视频响应不包含可识别 video_id",
					Diagnostics: []VideoResponseDiagnostic{diagnostic},
				}:
				default:
				}
				return
			}
			select {
			case resultCh <- videoPageResult{
				Videos: videos, Status: videoPageAccessAvailable,
				Diagnostics: []VideoResponseDiagnostic{diagnostic},
			}:
			default:
				logWarn(ctx, taskId, "⚠️ 视频响应队列已满，丢弃重复响应")
			}
		},
	)
	go wait()
	return cancel, resultCh
}

func buildVideoResponseDiagnostic(body, rawURL, productID string, parsedVideoCount int) VideoResponseDiagnostic {
	diagnostic := VideoResponseDiagnostic{
		ProductID:        productID,
		BodySize:         len(body),
		ParsedVideoCount: parsedVideoCount,
		ArrayLengths:     map[string]int{},
		HasVideoID:       strings.Contains(body, "video_id"),
		HasAwemeID:       strings.Contains(body, "aweme_id"),
		HasVideoList:     strings.Contains(body, "video_list"),
		HasDownloadURL:   strings.Contains(body, "download_url") || strings.Contains(body, "play_addr"),
		RestrictionNote:  firstRestrictionMarker(body),
	}
	if u, err := netUrl.Parse(rawURL); err == nil {
		diagnostic.ResponsePath = u.Path
		diagnostic.QueryKeys = sortedQueryKeys(u)
	} else {
		diagnostic.ResponsePath = "<invalid-url>"
	}

	var parsed any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return diagnostic
	}
	diagnostic.JSONValid = true
	root, ok := parsed.(map[string]any)
	if !ok {
		return diagnostic
	}
	diagnostic.TopLevelKeys = sortedMapKeys(root)
	diagnostic.BusinessCode = firstStringValue(root, "code", "st", "status", "err_no", "errno")
	diagnostic.Message = limitDiagnosticMessage(firstStringValue(root, "message", "msg", "status_msg", "err_msg"))
	if data, ok := root["data"].(map[string]any); ok {
		diagnostic.DataKeys = sortedMapKeys(data)
		collectArrayLengths("data", data, diagnostic.ArrayLengths, 2)
		diagnostic.RemainingCount = firstStringValue(data, "remaining_cnt", "remaining_count", "remain_count")
		diagnostic.TargetViewable = firstStringValue(data, "target_id_viewable", "target_viewable", "viewable")
		diagnostic.DisplayType = firstStringValue(data, "display_type")
	} else if dataList, ok := root["data"].([]any); ok {
		diagnostic.ArrayLengths["data"] = len(dataList)
	}
	collectArrayLengths("", root, diagnostic.ArrayLengths, 1)
	return diagnostic
}

func logAccessControlDiagnostic(ctx context.Context, taskId string, diagnostic VideoResponseDiagnostic) {
	logInfo(ctx, taskId, fmt.Sprintf(
		"🧭 竞对访问控制诊断：product_id=%s path=%s query_keys=%s body=%d json=%t code=%s remaining_cnt=%s target_viewable=%s display_type=%s restrict=%s top_keys=%s data_keys=%s",
		diagnostic.ProductID,
		diagnostic.ResponsePath,
		strings.Join(diagnostic.QueryKeys, ","),
		diagnostic.BodySize,
		diagnostic.JSONValid,
		diagnostic.BusinessCode,
		diagnostic.RemainingCount,
		diagnostic.TargetViewable,
		diagnostic.DisplayType,
		diagnostic.RestrictionNote,
		strings.Join(diagnostic.TopLevelKeys, ","),
		strings.Join(diagnostic.DataKeys, ","),
	))
}

func logVideoResponseDiagnostic(ctx context.Context, taskId string, diagnostic VideoResponseDiagnostic) {
	logInfo(ctx, taskId, fmt.Sprintf(
		"🧭 视频响应结构诊断：product_id=%s path=%s query_keys=%s body=%d json=%t code=%s parsed=%d has_video_id=%t has_aweme_id=%t has_video_list=%t has_download_url=%t remaining_cnt=%s target_viewable=%s display_type=%s restrict=%s top_keys=%s data_keys=%s arrays=%s",
		diagnostic.ProductID,
		diagnostic.ResponsePath,
		strings.Join(diagnostic.QueryKeys, ","),
		diagnostic.BodySize,
		diagnostic.JSONValid,
		diagnostic.BusinessCode,
		diagnostic.ParsedVideoCount,
		diagnostic.HasVideoID,
		diagnostic.HasAwemeID,
		diagnostic.HasVideoList,
		diagnostic.HasDownloadURL,
		diagnostic.RemainingCount,
		diagnostic.TargetViewable,
		diagnostic.DisplayType,
		diagnostic.RestrictionNote,
		strings.Join(diagnostic.TopLevelKeys, ","),
		strings.Join(diagnostic.DataKeys, ","),
		formatArrayLengths(diagnostic.ArrayLengths),
	))
}

func sortedQueryKeys(u *netUrl.URL) []string {
	keys := make([]string, 0, len(u.Query()))
	for key := range u.Query() {
		if key == "msToken" || key == "a_bogus" || key == "verifyFp" || key == "fp" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > 40 {
		return keys[:40]
	}
	return keys
}

func firstStringValue(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			switch typed := value.(type) {
			case string:
				return typed
			case float64, bool:
				return fmt.Sprint(typed)
			case nil:
				return ""
			default:
				return fmt.Sprintf("%v", typed)
			}
		}
	}
	return ""
}

func limitDiagnosticMessage(message string) string {
	message = strings.TrimSpace(message)
	if len([]rune(message)) <= 80 {
		return message
	}
	runes := []rune(message)
	return string(runes[:80])
}

func collectArrayLengths(prefix string, value any, result map[string]int, depth int) {
	if depth < 0 || len(result) >= 20 {
		return
	}
	switch typed := value.(type) {
	case []any:
		if prefix != "" {
			result[prefix] = len(typed)
		}
		if len(typed) > 0 {
			collectArrayLengths(prefix+"[]", typed[0], result, depth-1)
		}
	case map[string]any:
		keys := sortedMapKeys(typed)
		for _, key := range keys {
			nextPrefix := key
			if prefix != "" {
				nextPrefix = prefix + "." + key
			}
			collectArrayLengths(nextPrefix, typed[key], result, depth-1)
		}
	}
}

func formatArrayLengths(values map[string]int) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", key, values[key]))
	}
	return strings.Join(parts, ",")
}

func firstRestrictionMarker(text string) string {
	if marker := dailyVideoViewLimitMarker(text); marker != "" {
		return marker
	}
	for _, marker := range []string{
		"对方已关闭竞对详情功能",
		"仍可查看部分数据",
		"详情查看访问限制",
		"无权查看",
		"暂无权限",
		"无法查看详情",
		"没有权限查看",
	} {
		if strings.Contains(text, marker) {
			return marker
		}
	}
	return ""
}

func (s *sCrawler) detectCompetitorDetailBlock(ctx context.Context, page *rod.Page, taskId string) (bool, string) {
	texts := make([]string, 0, 2)
	if elem, err := page.Timeout(500 * time.Millisecond).Element("body"); err == nil {
		if text, textErr := elem.Text(); textErr == nil {
			texts = append(texts, text)
		}
	}
	if html, err := page.HTML(); err == nil {
		texts = append(texts, html)
	}
	blockMarkers := []string{
		"查看次数已达上限",
		"今日查看次数已达上限",
		"今日详情查看次数已达上限",
		"今日查看已达上限",
		"已达今日查看上限",
		"每天仅可查看10次",
		"每日仅可查看10次",
		"明日再来查看",
		"对方已关闭竞对详情功能",
		"仍可查看部分数据",
		"详情查看访问限制",
		"无权查看",
		"暂无权限",
		"无法查看详情",
		"没有权限查看",
	}
	for _, text := range texts {
		for _, marker := range blockMarkers {
			if strings.Contains(text, marker) {
				logDebug(ctx, taskId, fmt.Sprintf("⚠️ 检测到详情权限限制：%s", marker))
				return true, marker
			}
		}
	}
	return false, ""
}

func dailyVideoViewLimitMarker(text string) string {
	markers := []string{
		"查看次数已达上限",
		"今日查看次数已达上限",
		"今日详情查看次数已达上限",
		"今日查看已达上限",
		"已达今日查看上限",
		"每天仅可查看10次",
		"每日仅可查看10次",
		"明日再来查看",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return marker
		}
	}
	return ""
}

func isDailyVideoViewLimit(note string) bool {
	return dailyVideoViewLimitMarker(note) != ""
}

var videoIDPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?:douyin\.com/(?:share/)?video/)(\d{6,32})`),
	regexp.MustCompile(`(?:video_id|aweme_id|item_id)(?:_str)?(?:\\*["'])?\s*[:=]\s*(?:\\*["'])?(\d{6,32})`),
}

type videoIdentity struct {
	VideoId   string
	SourceURL string
}

func extractAllVideoIdentitiesFromText(candidate string) []videoIdentity {
	result := make([]videoIdentity, 0)
	seen := make(map[string]struct{})
	for _, pattern := range videoIDPatterns {
		for _, match := range pattern.FindAllStringSubmatch(candidate, -1) {
			if len(match) < 2 {
				continue
			}
			if _, ok := seen[match[1]]; ok {
				continue
			}
			seen[match[1]] = struct{}{}
			result = append(result, videoIdentity{
				VideoId: match[1], SourceURL: "https://www.douyin.com/video/" + match[1],
			})
		}
	}
	return result
}

func extractVideoIdentityFromElement(element *rod.Element) (string, string) {
	if element == nil {
		return "", ""
	}
	for _, attribute := range []string{"data-video-id", "data-aweme-id", "data-row-key"} {
		if value, err := element.Attribute(attribute); err == nil && value != nil {
			if matched, _ := regexp.MatchString(`^\d{10,24}$`, *value); matched {
				return *value, "https://www.douyin.com/video/" + *value
			}
		}
	}
	candidates := make([]string, 0, 4)
	if href, err := element.Attribute("href"); err == nil && href != nil {
		candidates = append(candidates, *href)
	}
	if links, err := element.Elements("a[href]"); err == nil {
		for _, link := range links {
			if href, hrefErr := link.Attribute("href"); hrefErr == nil && href != nil {
				candidates = append(candidates, *href)
			}
		}
	}
	if html, err := element.HTML(); err == nil {
		candidates = append(candidates, html)
	}
	for _, candidate := range candidates {
		if videoID, sourceURL := extractVideoIdentityFromText(candidate); videoID != "" {
			return videoID, sourceURL
		}
	}
	return "", ""
}

func extractVideoIdentityFromText(candidate string) (string, string) {
	for _, pattern := range videoIDPatterns {
		if match := pattern.FindStringSubmatch(candidate); len(match) > 1 {
			sourceURL := "https://www.douyin.com/video/" + match[1]
			if strings.HasPrefix(candidate, "http") && !strings.Contains(candidate, "<") {
				sourceURL = candidate
			}
			return match[1], sourceURL
		}
	}
	return "", ""
}

func (s *sCrawler) waitForRiskControl(ctx context.Context, page *rod.Page, taskId string) bool {
	riskTexts := []string{"环境异常", "操作频繁", "请完成验证", "访问异常"}
	hasRisk := func() bool {
		for _, text := range riskTexts {
			if _, err := page.Timeout(700 * time.Millisecond).ElementX(fmt.Sprintf("//*[contains(text(), '%s')]", text)); err == nil {
				return true
			}
		}
		return false
	}
	if !hasRisk() {
		return true
	}
	service.GetCrawlerStatusManager().SetMessage("检测到环境验证，等待手动处理")
	logWarn(ctx, taskId, "⚠️ 检测到环境异常或验证页，已暂停自动操作，请在浏览器中手动处理")
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(3 * time.Second):
		}
		if !hasRisk() {
			service.GetCrawlerStatusManager().SetMessage("环境验证已完成，继续采集")
			return true
		}
	}
	logError(ctx, taskId, "❌ 等待手动处理环境验证超时")
	return false
}

// saveAllData 保存所有数据到数据库
func (s *sCrawler) saveAllData(ctx context.Context, taskId string, products []ProductInfo, details []ProductDetail, videos []VideoAnalysis) error {
	// 保存商品榜单数据
	var rankingList []*model.TikTokRankingListInput

	// 创建详情数据映射，使用ProductID作为键
	detailMap := make(map[string]*ProductDetail)
	for i := range details {
		if details[i].ProductID != "" {
			detail := details[i] // 创建副本
			detailMap[details[i].ProductID] = &detail
		}
	}

	for _, p := range products {
		record := &model.TikTokRankingListInput{
			TaskId:           taskId,
			ProductId:        p.ProductID,
			ShopId:           p.ShopID,
			ProductUrl:       p.ProductURL,
			Rank:             p.Rank,
			ProductName:      p.ProductName,
			ShopName:         p.ShopName,
			PaymentAmount:    p.PaymentAmount,
			ClickCount:       p.ClickCount,
			SalesVolume:      p.SalesVolume,
			DeliveryVideoNum: p.DeliveryVideoNum,
			FirstList:        p.FirstList,
			CreatedAt:        time.Now(),
		}

		// 合并详情数据，使用ProductID匹配
		if detail, ok := detailMap[p.ProductID]; ok {
			record.FirstShelfTime = detail.FirstShelfTime
			record.MinPrice = detail.MinPrice
			record.DetailAccessStatus = detail.DetailAccessStatus
			record.DetailAccessNote = detail.DetailAccessNote

			// 处理运费险字段
			if detail.HasShippingInsurance != nil {
				insuranceVal := 0
				if *detail.HasShippingInsurance {
					insuranceVal = 1
				}
				record.HasShippingInsurance = &insuranceVal
			}

			logInfo(ctx, taskId, fmt.Sprintf("✅ 匹配商品详情数据：%s (ProductID: %s)", p.ProductName, p.ProductID))
		} else {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 未找到商品详情数据：%s (ProductID: %s)", p.ProductName, p.ProductID))
		}

		rankingList = append(rankingList, record)
	}

	// 保存商品数据（包含详情信息）
	if len(rankingList) > 0 {
		// 这里不是批量插入，而是更新已保存的商品数据，添加详情信息
		for _, record := range rankingList {
			// 尝试根据ProductId更新商品数据，添加详情信息
			if err := service.TikTokRankingList().UpdateByProductId(ctx, record); err != nil {
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ 更新商品详情数据失败：%v", err))
			} else {
				logInfo(ctx, taskId, fmt.Sprintf("✅ 更新商品详情数据：%s", record.ProductName))
			}
		}
	}

	// 保存视频分析数据
	var videoData []*model.VideoAnalysisData
	for _, v := range videos {
		// 跳过空数据
		if v.VideoId == "" && v.VideoDesc == "" {
			continue
		}
		record := &model.VideoAnalysisData{
			TaskId:            taskId,
			ProductId:         v.ProductID,
			ProductName:       v.ProductName,
			VideoId:           v.VideoId,
			VideoDesc:         v.VideoDesc,
			AuthorName:        v.AuthorName,
			CoverURL:          v.CoverURL,
			PublishTime:       v.PublishTime,
			SourceURL:         v.SourceURL,
			DownloadStatus:    "pending",
			DownloadURL:       "",
			ResolveStrategy:   "",
			PaymentAmount:     v.PaymentAmount,
			ExposeUserCount:   v.ExposeUserCount,
			ClickUserCount:    v.ClickUserCount,
			ConvertUserCount:  v.ConvertUserCount,
			ClickRate:         v.ClickRate,
			ClickConvertRate:  v.ClickConvertRate,
			ExposeConvertRate: v.ExposeConvertRate,
			OrderCount:        v.OrderCount,
			SalesVolume:       v.SalesVolume,
			CreatedAt:         time.Now(),
		}
		videoData = append(videoData, record)
	}

	if len(videoData) > 0 {
		// 批量保存视频分析数据
		batchSize := 50
		for i := 0; i < len(videoData); i += batchSize {
			end := i + batchSize
			if end > len(videoData) {
				end = len(videoData)
			}
			batch := videoData[i:end]
			if err := service.VideoAnalysis().CreateBatch(ctx, batch); err != nil {
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ 保存视频分析数据失败：%v", err))
			} else {
				logInfo(ctx, taskId, fmt.Sprintf("✅ 保存视频分析数据：%d 条（批次 %d/%d）", len(batch), i/batchSize+1, (len(videoData)+batchSize-1)/batchSize))
			}
			// 随机延迟，避免数据库压力过大
			time.Sleep(time.Duration(grand.Intn(500)+200) * time.Millisecond)
		}
	} else {
		logInfo(ctx, taskId, "ℹ️ 没有视频分析数据需要保存")
	}

	return nil
}

// ==================== 工具函数 ====================

// convertToModelProducts 转换 ProductInfo 为 model.TikTokRankingListInput（按 product_id 关联详情）
func (s *sCrawler) convertToModelProducts(products []ProductInfo, details []ProductDetail) []model.TikTokRankingListInput {
	// 构建详情数据映射：product_id -> detail
	detailMap := make(map[string]*ProductDetail)
	for i := range details {
		if details[i].ProductID != "" {
			d := details[i]
			detailMap[details[i].ProductID] = &d
		}
	}

	var result []model.TikTokRankingListInput
	for _, p := range products {
		record := model.TikTokRankingListInput{
			ProductId:        p.ProductID,
			ShopId:           p.ShopID,
			ProductUrl:       p.ProductURL,
			Rank:             p.Rank,
			ProductName:      p.ProductName,
			ShopName:         p.ShopName,
			PaymentAmount:    p.PaymentAmount,
			ClickCount:       p.ClickCount,
			SalesVolume:      p.SalesVolume,
			DeliveryVideoNum: p.DeliveryVideoNum,
			FirstList:        p.FirstList,
			CreatedAt:        time.Now(),
		}
		// 按 product_id 匹配详情数据
		if detail, ok := detailMap[p.ProductID]; ok {
			record.FirstShelfTime = detail.FirstShelfTime
			record.MinPrice = detail.MinPrice
			record.DetailAccessStatus = detail.DetailAccessStatus
			record.DetailAccessNote = detail.DetailAccessNote
			if detail.HasShippingInsurance != nil {
				val := 0
				if *detail.HasShippingInsurance {
					val = 1
				}
				record.HasShippingInsurance = &val
			}
		}
		result = append(result, record)
	}
	return result
}

func (s *sCrawler) convertToModelVideos(taskId string, videos []VideoAnalysis) []model.VideoAnalysisData {
	result := make([]model.VideoAnalysisData, 0, len(videos))
	for _, v := range videos {
		if v.VideoId == "" && v.VideoDesc == "" {
			continue
		}
		result = append(result, model.VideoAnalysisData{
			TaskId: taskId, ProductId: v.ProductID, ProductName: v.ProductName,
			VideoId: v.VideoId, VideoDesc: v.VideoDesc, AuthorName: v.AuthorName,
			CoverURL: v.CoverURL, PublishTime: v.PublishTime, SourceURL: v.SourceURL,
			DownloadStatus: "pending", PaymentAmount: v.PaymentAmount,
			ExposeUserCount: v.ExposeUserCount, ClickUserCount: v.ClickUserCount,
			ConvertUserCount: v.ConvertUserCount, ClickRate: v.ClickRate,
			ClickConvertRate: v.ClickConvertRate, ExposeConvertRate: v.ExposeConvertRate,
			OrderCount: v.OrderCount, SalesVolume: v.SalesVolume, CreatedAt: time.Now(),
		})
	}
	return result
}

type browserRankingPage struct {
	PageNo   int
	Products []ProductInfo
}

// setupBrowserResponseObserver 只读取 Chrome 已经收到的榜单响应。
// 它不暂停、篡改或用 Go HTTP 客户端重放罗盘请求。
func (s *sCrawler) setupBrowserResponseObserver(ctx context.Context, page *rod.Page, taskId string) (func(), chan browserRankingPage) {
	pageCh := make(chan browserRankingPage, 4)
	observerCtx, cancel := context.WithCancel(ctx)
	observedPage := page.Context(observerCtx)
	cacheSize := 100 * 1024 * 1024
	if err := (proto.NetworkEnable{MaxTotalBufferSize: &cacheSize}).Call(observedPage); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 启用浏览器网络响应监听失败：%v", err))
	}
	pending := make(map[proto.NetworkRequestID]string)
	var pendingMu sync.Mutex
	wait := observedPage.EachEvent(
		func(event *proto.NetworkResponseReceived) {
			if event.Response == nil || !strings.Contains(event.Response.URL, "video_bring_good") {
				return
			}
			pendingMu.Lock()
			pending[event.RequestID] = event.Response.URL
			pendingMu.Unlock()
		},
		func(event *proto.NetworkLoadingFinished) {
			pendingMu.Lock()
			responseURL, ok := pending[event.RequestID]
			delete(pending, event.RequestID)
			pendingMu.Unlock()
			if !ok {
				return
			}
			result, err := (proto.NetworkGetResponseBody{RequestID: event.RequestID}).Call(observedPage)
			if err != nil {
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ 读取浏览器响应体失败：%v", err))
				return
			}
			body := result.Body
			if result.Base64Encoded {
				decoded, decodeErr := base64.StdEncoding.DecodeString(result.Body)
				if decodeErr != nil {
					logWarn(ctx, taskId, fmt.Sprintf("⚠️ 解码浏览器响应体失败：%v", decodeErr))
					return
				}
				body = string(decoded)
			}
			products := parseProductRankResponse(body)
			pageNo := rankingPageNo(responseURL)
			logInfo(ctx, taskId, fmt.Sprintf("🔍 读取到浏览器榜单响应，实际页=%d，body=%d，商品=%d，URL=%s", pageNo, len(body), len(products), responseURL))
			if len(products) == 0 {
				return
			}
			select {
			case pageCh <- browserRankingPage{PageNo: pageNo, Products: products}:
			default:
				logWarn(ctx, taskId, "⚠️ 榜单响应队列已满，丢弃过期响应")
			}
		},
	)
	go wait()
	return cancel, pageCh
}

func rankingPageNo(responseURL string) int {
	parsedURL, err := netUrl.Parse(responseURL)
	if err != nil {
		return 0
	}
	pageNo, _ := strconv.Atoi(parsedURL.Query().Get("page_no"))
	return pageNo
}

func (s *sCrawler) triggerRankingPageRefresh(ctx context.Context, page *rod.Page, taskId string, targetPage int) bool {
	if disabledCount := disableFloatingAssistantPointerEvents(page); disabledCount > 0 {
		logDebug(ctx, taskId, fmt.Sprintf("ℹ️ 刷新分页前已避免 %d 个悬浮助手节点遮挡分页按钮", disabledCount))
	}
	result, err := page.Timeout(5*time.Second).Eval(`(targetPage) => {
		const pageNumber = Number(targetPage);
		const selectors = [
			'.aurora-pagination-item',
			'.ant-pagination-item',
			'.el-pager li',
			'li[title]'
		];
		const items = Array.from(document.querySelectorAll(selectors.join(',')))
			.filter((item) => Number.parseInt((item.textContent || item.getAttribute('title') || '').trim(), 10) > 0);
		const findItem = (pageNo) => items.find((item) => {
			const textNo = Number.parseInt((item.textContent || '').trim(), 10);
			const titleNo = Number.parseInt((item.getAttribute('title') || '').trim(), 10);
			return textNo === pageNo || titleNo === pageNo;
		});
		const active = document.querySelector(
			'.aurora-pagination-item-active, .ant-pagination-item-active, .el-pager li.active'
		);
		const activeNo = active ? Number.parseInt((active.textContent || '').trim(), 10) : 0;
		const target = findItem(pageNumber);
		if (!target) return {clicked: false, reason: 'target_not_found', activeNo};
		if (activeNo === pageNumber && pageNumber > 1) {
			const previous = findItem(pageNumber - 1);
			if (previous) {
				previous.dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true, view: window}));
				window.setTimeout(() => {
					target.dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true, view: window}));
				}, 700);
				return {clicked: true, reason: 'toggle_previous_then_target', activeNo};
			}
		}
		target.dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true, view: window}));
		return {clicked: true, reason: 'click_target', activeNo};
	}`, targetPage)
	if err != nil || result == nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 触发第 %d 页刷新失败：%v", targetPage, err))
		return false
	}
	logInfo(ctx, taskId, fmt.Sprintf("🔁 已触发第 %d 页榜单刷新：%s", targetPage, result.Value.String()))
	time.Sleep(1500 * time.Millisecond)
	return true
}

// collectInterceptedData 从 pageCh 收集拦截到的数据并翻页
func (s *sCrawler) collectInterceptedData(ctx context.Context, page *rod.Page, taskId string, checkpointManager *service.CheckpointManager, startPage int, pageCh chan browserRankingPage) []ProductInfo {
	var allProducts []ProductInfo
	totalPages := 20
	currentPage := startPage

	for currentPage <= totalPages {
		if ctx.Err() != nil {
			return s.uniqueProducts(allProducts)
		}
		logInfo(ctx, taskId, fmt.Sprintf("📄 正在处理第 %d/%d 页", currentPage, totalPages))

		// 等待当前页数据。超时后先尝试触发当前页重新请求；
		// 仍然没有目标页有效响应时才停止，避免 UI 已翻页但接口请求偶发丢失导致只抓第一页。
		var pageProducts []ProductInfo
		responseMatched := false
		recoverAttempts := 0
		for !responseMatched {
			responseTimer := time.NewTimer(25 * time.Second)
			select {
			case response := <-pageCh:
				if !responseTimer.Stop() {
					select {
					case <-responseTimer.C:
					default:
					}
				}
				if response.PageNo > 0 && response.PageNo != currentPage {
					logWarn(ctx, taskId, fmt.Sprintf("⚠️ 收到非预期榜单页：期望=%d，实际=%d，忽略该响应", currentPage, response.PageNo))
					continue
				}
				pageProducts = response.Products
				responseMatched = true
			case <-ctx.Done():
				if !responseTimer.Stop() {
					select {
					case <-responseTimer.C:
					default:
					}
				}
				return s.uniqueProducts(allProducts)
			case <-responseTimer.C:
				recoverAttempts++
				if recoverAttempts <= 2 && s.triggerRankingPageRefresh(ctx, page, taskId, currentPage) {
					logWarn(ctx, taskId, fmt.Sprintf("⚠️ 第 %d 页等待浏览器响应超时，已触发第 %d 次刷新重试", currentPage, recoverAttempts))
					continue
				}
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ 第 %d 页等待浏览器响应超时，停止翻页", currentPage))
				allProducts = s.uniqueProducts(allProducts)
				logInfo(ctx, taskId, fmt.Sprintf("📊 浏览器榜单响应采集完成，共获取 %d 条商品（去重后）", len(allProducts)))
				return allProducts
			}
		}

		if len(pageProducts) > 0 {
			allProducts = append(allProducts, pageProducts...)
			logInfo(ctx, taskId, fmt.Sprintf("✅ 第 %d 页解析到 %d 条，立即保存", currentPage, len(pageProducts)))
			s.savePageData(ctx, taskId, pageProducts, currentPage)
		}

		if checkpointManager != nil {
			_ = checkpointManager.UpdateProgress(ctx, taskId, 6, currentPage, len(allProducts))
		}

		if currentPage < totalPages {
			if !s.clickNextPage(ctx, page, taskId) {
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ 无法点击下一页，停止翻页（当前第 %d 页）", currentPage))
				break
			}
		}

		currentPage++
	}

	allProducts = s.uniqueProducts(allProducts)
	logInfo(ctx, taskId, fmt.Sprintf("📊 浏览器榜单响应采集完成，共获取 %d 条商品（去重后）", len(allProducts)))
	return allProducts
}

// interceptProductRankAPI 已废弃，逻辑拆分到浏览器响应监听 + collectInterceptedData。
// 保留此方法签名避免外部编译报错
func (s *sCrawler) interceptProductRankAPI(ctx context.Context, page *rod.Page, taskId string, checkpointManager *service.CheckpointManager, startPage int) []ProductInfo {
	return s.collectInterceptedData(ctx, page, taskId, checkpointManager, startPage, make(chan browserRankingPage, 1))
}

// parseProductRankResponse 解析商品榜单 API 响应 JSON
func parseProductRankResponse(body string) []ProductInfo {
	// 打印响应体前1000字符，了解API响应结构
	if len(body) > 1000 {
		fmt.Printf("[DEBUG] API响应前1000字符: %s\n", body[:1000])
	} else {
		fmt.Printf("[DEBUG] API响应: %s\n", body)
	}

	// 使用简单的 JSON 解析，避免引入额外依赖
	// 结构：data.data_result[i].product_info.*
	type shopInfo struct {
		ShopID   string `json:"shop_id"`
		ShopName string `json:"shop_name"`
	}
	type productInfo struct {
		Rank             int        `json:"rank"`
		ID               string     `json:"id"`
		Name             string     `json:"name"`
		PriceBin         string     `json:"price_bin"`
		ProductDetailURL string     `json:"product_detail_h5_url"`
		NewlyOnRanking   bool       `json:"newly_on_ranking"`
		ShopList         []shopInfo `json:"shop_list"`
	}
	type valueItem struct {
		Value interface{} `json:"value"`
		Unit  string      `json:"unit"`
	}
	type valueRange struct {
		ValueRange []valueItem `json:"value_range"`
	}
	type dataItem struct {
		ProductInfo productInfo `json:"product_info"`
		PayAmt      valueRange  `json:"pay_amt"`
		NewPayAmt   valueRange  `json:"new_pay_amt"`
		PayComboCnt valueRange  `json:"pay_combo_cnt"`
		ClickCnt    valueRange  `json:"product_click_cnt"`
		VideoCnt    valueRange  `json:"video_cnt"`
	}
	type pageResult struct {
		Total    int `json:"total"`
		PageNo   int `json:"page_no"`
		PageSize int `json:"page_size"`
	}
	type dataWrapper struct {
		DataResult []dataItem `json:"data_result"`
		PageResult pageResult `json:"page_result"`
	}
	type apiResponse struct {
		Data dataWrapper `json:"data"`
	}

	var resp apiResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		fmt.Printf("[ERROR] JSON解析失败: %v\n", err)
		return nil
	}

	// 打印解析结果
	fmt.Printf("[DEBUG] 解析到DataResult长度: %d\n", len(resp.Data.DataResult))
	if len(resp.Data.DataResult) > 0 {
		fmt.Printf("[DEBUG] 第一个DataItem: %+v\n", resp.Data.DataResult[0])
	}

	formatRange := func(vr valueRange) string {
		items := vr.ValueRange
		if len(items) == 0 {
			return ""
		}
		first := items[0]
		last := items[len(items)-1]

		// 转换Value为字符串
		firstValue := ""
		switch v := first.Value.(type) {
		case string:
			firstValue = v
		case float64:
			firstValue = fmt.Sprintf("%v", v)
		case int:
			firstValue = fmt.Sprintf("%d", v)
		case int64:
			firstValue = fmt.Sprintf("%d", v)
		}

		lastValue := ""
		switch v := last.Value.(type) {
		case string:
			lastValue = v
		case float64:
			lastValue = fmt.Sprintf("%v", v)
		case int:
			lastValue = fmt.Sprintf("%d", v)
		case int64:
			lastValue = fmt.Sprintf("%d", v)
		}

		if firstValue == "" {
			return ""
		}
		if firstValue == lastValue {
			return firstValue
		}
		return firstValue + "-" + lastValue
	}

	var products []ProductInfo
	for i, item := range resp.Data.DataResult {
		info := item.ProductInfo
		if info.ID == "" || info.Name == "" {
			fmt.Printf("[DEBUG] 跳过无效商品数据: ID=%s, Name=%s\n", info.ID, info.Name)
			continue
		}
		shopName := ""
		shopID := ""
		if len(info.ShopList) > 0 {
			shopName = info.ShopList[0].ShopName
			shopID = info.ShopList[0].ShopID
		}
		firstList := 0
		if info.NewlyOnRanking {
			firstList = 1
		}
		// 优先使用 pay_amt，回退到 new_pay_amt
		payAmt := item.PayAmt
		if len(payAmt.ValueRange) == 0 {
			payAmt = item.NewPayAmt
		}
		product := ProductInfo{
			Rank:             info.Rank,
			ProductID:        info.ID,
			ProductName:      info.Name,
			ShopID:           shopID,
			ShopName:         shopName,
			PaymentAmount:    formatRange(payAmt),
			ClickCount:       formatRange(item.ClickCnt),
			SalesVolume:      formatRange(item.PayComboCnt),
			DeliveryVideoNum: formatRange(item.VideoCnt),
			ProductURL:       info.ProductDetailURL,
			FirstList:        firstList,
			PriceRange:       info.PriceBin,
		}
		products = append(products, product)
		fmt.Printf("[DEBUG] 解析到商品 %d: %s\n", i+1, product.ProductName)
	}
	fmt.Printf("[DEBUG] 最终解析到 %d 条商品\n", len(products))
	return products
}

// scrapeAllPagesWithCheckpoint 支持断点续爬的分页抓取
func (s *sCrawler) scrapeAllPagesWithCheckpoint(ctx context.Context, page *rod.Page, taskId string, checkpointManager *service.CheckpointManager, startPage int) []ProductInfo {
	var allProducts []ProductInfo
	totalPages := 20
	currentPage := startPage

	for currentPage <= totalPages {
		logInfo(ctx, taskId, fmt.Sprintf("📄 正在抓取第 %d/%d 页", currentPage, totalPages))

		// 抓取当前页数据（传递页码参数）
		products := s.scrapeCurrentPage(ctx, page, taskId, currentPage)
		if len(products) > 0 {
			allProducts = append(allProducts, products...)
			logInfo(ctx, taskId, fmt.Sprintf("✅ 第 %d 页抓取完成：获取 %d 条数据", currentPage, len(products)))
			// 【新增修改点】每抓取完一页，立即将数据保存到数据库，修复写入不成功的问题
			s.savePageData(ctx, taskId, products, currentPage)
		}

		// 保存进度
		checkpointManager.UpdateProgress(ctx, taskId, 6, currentPage, len(allProducts))

		// 尝试点击下一页
		if currentPage < totalPages {
			if !s.clickNextPage(ctx, page, taskId) {
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ 无法点击下一页，停止翻页（当前第 %d 页）", currentPage))
				break
			}
			time.Sleep(2 * time.Second)
		}

		currentPage++
	}

	// 去重
	allProducts = s.uniqueProducts(allProducts)
	logInfo(ctx, taskId, fmt.Sprintf("📊 分页抓取完成，共获取 %d 条商品（去重后）", len(allProducts)))
	return allProducts
}

// savePageData 保存单页数据到数据库
func (s *sCrawler) savePageData(ctx context.Context, taskId string, products []ProductInfo, pageNum int) {
	if len(products) == 0 {
		return
	}

	// 统计价格带信息
	priceRangeCount := 0
	for _, p := range products {
		if p.PriceRange != "" {
			priceRangeCount++
		}
	}

	// 从缓存获取详情数据
	cache := service.GetProductDetailCache()

	// 转换为数据库模型
	var rankingList []*model.TikTokRankingListInput
	for _, p := range products {
		record := &model.TikTokRankingListInput{
			TaskId:           taskId,
			ProductId:        p.ProductID,
			ShopId:           p.ShopID,
			ProductUrl:       p.ProductURL,
			Rank:             p.Rank,
			ProductName:      p.ProductName,
			ShopName:         p.ShopName,
			PaymentAmount:    p.PaymentAmount,
			ClickCount:       p.ClickCount,
			SalesVolume:      p.SalesVolume,
			DeliveryVideoNum: p.DeliveryVideoNum,
			FirstList:        p.FirstList,
			PriceRange:       p.PriceRange,
			CreatedAt:        time.Now(),
		}

		// 从缓存获取详情数据
		if cachedRecord := cache.GetRecord(p.ProductID); cachedRecord != nil && !cache.IsRecordStale(cachedRecord) {
			record.FirstShelfTime = cachedRecord.FirstShelfTime
			record.MinPrice = cachedRecord.MinPrice
			logInfo(ctx, taskId, fmt.Sprintf("✅ 从缓存获取商品详情数据：%s (ProductID: %s)", p.ProductName, p.ProductID))
		} else {
			record.FirstShelfTime = ""
			record.MinPrice = ""
		}

		rankingList = append(rankingList, record)
	}

	// 批量插入
	if err := service.TikTokRankingList().CreateBatch(ctx, rankingList); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 保存第 %d 页数据失败：%v", pageNum, err))
	} else {
		logInfo(ctx, taskId, fmt.Sprintf("💾 第 %d 页数据已保存：%d 条 (其中 %d 条包含价格带)", pageNum, len(rankingList), priceRangeCount))
	}
}

func (s *sCrawler) getText(el *rod.Element) string {
	if el == nil {
		return ""
	}
	text, _ := el.Text()
	return strings.TrimSpace(text)
}

// parseRank 解析排名（处理 "#" 符号、纯文本等）
func (s *sCrawler) parseRank(str string) int {
	// 移除常见符号和空格
	str = strings.ReplaceAll(str, "#", "")
	str = strings.ReplaceAll(str, " ", "")
	str = strings.TrimSpace(str)

	// 尝试转换为整数
	val, err := strconv.Atoi(str)
	if err == nil {
		return val
	}

	// 如果转换失败，返回 0，表示无法从页面获取真实排名
	// 调用方应该跳过这条数据或使用其他逻辑处理
	return 0
}

func (s *sCrawler) toInt(str string) int {
	return s.parseRank(str)
}

func (s *sCrawler) uniqueProducts(products []ProductInfo) []ProductInfo {
	// 使用商品名称去重（因为排名是页内排名，每页都是 1-10）
	productMap := make(map[string]bool)
	var unique []ProductInfo

	for _, item := range products {
		// 使用商品名称作为去重 key
		if !productMap[item.ProductName] {
			productMap[item.ProductName] = true
			unique = append(unique, item)
		}
	}

	// 按排名排序
	sort.Slice(unique, func(i, j int) bool {
		return unique[i].Rank < unique[j].Rank
	})

	return unique
}

// injectCookies 将保存的 Cookie 注入浏览器页面
func (s *sCrawler) injectCookies(ctx context.Context, page *rod.Page, cookieManager *service.CookieManager, taskId string) bool {
	cookies, hasCookies, err := cookieManager.LoadCookies(ctx, taskId)
	if err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 加载 Cookie 失败：%v", err))
		return false
	}
	if !hasCookies || len(cookies) == 0 {
		logInfo(ctx, taskId, "ℹ️ 没有可用的 Cookie")
		return false
	}

	// 转换为 CookieParam 格式用于注入
	var cookieParams []*proto.NetworkCookieParam
	for _, c := range cookies {
		param := &proto.NetworkCookieParam{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			HTTPOnly: c.HTTPOnly,
			Secure:   c.Secure,
		}
		// 只设置有过期时间的 Cookie
		if c.Expires > 0 {
			param.Expires = c.Expires
		}
		cookieParams = append(cookieParams, param)
	}

	// 注入 Cookie 到浏览器
	err = proto.NetworkSetCookies{Cookies: cookieParams}.Call(page)
	if err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 注入 Cookie 失败：%v", err))
		return false
	}

	logInfo(ctx, taskId, fmt.Sprintf("🍪 成功注入 %d 个 Cookie", len(cookieParams)))

	// 刷新页面让 Cookie 生效
	if err := page.Reload(); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 刷新页面失败：%v", err))
		return false
	}
	page.WaitLoad()
	time.Sleep(3 * time.Second)

	return true
}
