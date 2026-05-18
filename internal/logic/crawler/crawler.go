package crawler

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
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
	ShopID           string // 店铺 ID
	ShopName         string // 店铺名称
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
	Products       []ProductInfo     `json:"products,omitempty"`
	ProductDetails []ProductDetail   `json:"product_details,omitempty"`
	VideoAnalysis  []VideoAnalysis   `json:"video_analysis,omitempty"`
	ExportFiles    map[string]string `json:"export_files,omitempty"`
}

// RunBrowserTask 核心爬虫方法 - 完整实现需求流程（含优化）
func (s *sCrawler) RunBrowserTask(ctx context.Context, req *CrawlReq) (res []Result, err error) {
	res = make([]Result, 0)
	taskId := grand.S(16)

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

	// ==================== 步骤 1: 尝试 Cookie 自动登录（无头模式） ====================
	homeURL := "https://compass.jinritemai.com/shop"
	var browser *rod.Browser
	var page *rod.Page
	isLoggedIn := false

	// 检查是否有已保存的 Cookie
	hasSavedCookies := cookieManager.HasCompassCookies(ctx, taskId)

	if hasSavedCookies {
		logInfo(ctx, taskId, "🚀 步骤 1: 检测到已保存的 Cookie，以无头模式启动浏览器尝试自动登录")
		browser, err = s.startBrowser(ctx, taskId, true)
		if err != nil {
			checkpointManager.FailTask(ctx, taskId, err.Error())
			return res, gerror.Wrap(err, "无头浏览器启动失败")
		}

		page, err = browser.Page(proto.TargetCreateTarget{URL: homeURL})
		if err != nil {
			browser.Close()
			checkpointManager.FailTask(ctx, taskId, err.Error())
			return res, gerror.Wrap(err, "打开首页失败")
		}
		page = page.Timeout(15 * time.Minute)
		page.WaitLoad()

		// 注入 Cookie 并刷新页面
		if s.injectCookies(ctx, page, cookieManager, taskId) {
			isLoggedIn = s.checkIsLoggedIn(ctx, page, taskId)
		}

		if isLoggedIn {
			logSuccess(ctx, taskId, "✅ Cookie 自动登录成功！继续无头模式爬取")
			// 更新 Cookie（刷新过期时间）
			allCookies, _ := page.Cookies(nil)
			if len(allCookies) > 0 {
				cookieManager.SaveCookies(ctx, allCookies, taskId)
				logInfo(ctx, taskId, "💾 Cookie 已更新")
			}
		} else {
			logWarn(ctx, taskId, "⚠️ Cookie 已过期或无效，关闭无头浏览器，准备有头模式登录")
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
		headedPage = headedPage.Timeout(15 * time.Minute)
		headedPage.WaitLoad()

		// 执行登录（等待用户扫码或输入账号密码）
		logInfo(ctx, taskId, "📝 步骤 3: 请在浏览器窗口中完成登录（扫码或账号密码）")
		loginSuccess := s.performLogin(ctx, headedPage, taskId, req.LoginType, req.Username, req.Password)
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

		// 关闭有头浏览器
		headedBrowser.Close()
		logInfo(ctx, taskId, "🖥️ 有头浏览器已关闭")

		// 以无头模式重新启动浏览器
		logInfo(ctx, taskId, "🔄 切换到无头模式继续爬取...")
		browser, err = s.startBrowser(ctx, taskId, true)
		if err != nil {
			checkpointManager.FailTask(ctx, taskId, err.Error())
			return res, gerror.Wrap(err, "无头浏览器启动失败")
		}

		page, err = browser.Page(proto.TargetCreateTarget{URL: homeURL})
		if err != nil {
			browser.Close()
			checkpointManager.FailTask(ctx, taskId, err.Error())
			return res, gerror.Wrap(err, "无头模式打开首页失败")
		}
		page = page.Timeout(15 * time.Minute)
		page.WaitLoad()

		// 注入刚保存的 Cookie
		if !s.injectCookies(ctx, page, cookieManager, taskId) {
			browser.Close()
			checkpointManager.FailTask(ctx, taskId, "无头模式注入Cookie失败")
			return res, gerror.New("切换无头模式后注入 Cookie 失败")
		}

		// 验证无头模式下登录状态
		if !s.checkIsLoggedIn(ctx, page, taskId) {
			browser.Close()
			checkpointManager.FailTask(ctx, taskId, "无头模式登录验证失败")
			return res, gerror.New("切换无头模式后登录状态验证失败")
		}
		logSuccess(ctx, taskId, "✅ 无头模式登录验证成功，开始后台爬取")
	}
	defer browser.Close()

	// 更新状态消息
	statusMgr.SetMessage("爬取数据中")

	// 检查是否已取消
	if checkCancelled() {
		return res, gerror.New("任务已取消")
	}

	// ==================== 步骤 4: 提前注册 API 拦截（必须在导航和点击 Tab 之前） ====================
	logInfo(ctx, taskId, "📊 步骤 4: 注册 API 拦截器并扩大 Inspector 缓存")

	// 扩大 Inspector Cache 容量到 100MB，防止响应体被驱逐导致 "evicted from inspector cache"
	cacheSize := 100 * 1024 * 1024
	networkEnable := proto.NetworkEnable{MaxTotalBufferSize: &cacheSize}
	if err := networkEnable.Call(page); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 启用网络追踪失败：%v", err))
	}

	// 初始化分页参数
	startPage := 1
	if existingTask != nil && existingTask.Status == "running" {
		startPage = existingTask.CurrentPage
		logInfo(ctx, taskId, fmt.Sprintf("🔄 从第 %d 页继续爬取", startPage))
	}

	// 提前注册 Hijack，确保后续所有 API 请求（包括第 1 页）都能被拦截
	router, pageCh := s.setupAPIInterceptor(ctx, page, taskId)
	go router.Run()
	defer router.Stop() //nolint:errcheck

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

	products := s.collectInterceptedData(ctx, page, taskId, checkpointManager, startPage, pageCh)
	if len(products) == 0 {
		checkpointManager.FailTask(ctx, taskId, "未抓取到数据")
		logWarn(ctx, taskId, "⚠️ 未抓取到有效数据")
		return res, gerror.New("未抓取到商品数据")
	}
	logInfo(ctx, taskId, fmt.Sprintf("✅ 抓取完成，共获取 %d 条商品数据", len(products)))

	if checkCancelled() {
		return res, gerror.New("任务已取消")
	}

	// ==================== 步骤 8: 获取商品详情数据 ====================
	logInfo(ctx, taskId, "🔍 步骤 8: 获取商品详情数据（首次上架时间、最低到手价）")
	productDetails := s.scrapeProductDetails(ctx, browser, products, taskId)
	logInfo(ctx, taskId, fmt.Sprintf("✅ 详情抓取完成，共获取 %d 条详情数据", len(productDetails)))

	if checkCancelled() {
		return res, gerror.New("任务已取消")
	}

	// ==================== 步骤 9: 获取视频分析数据 ====================
	logInfo(ctx, taskId, "📹 步骤 9: 获取视频分析数据（优化版：直接 HTTP 请求）")
	videoAnalysisList := s.scrapeVideoAnalysisOptimized(ctx, browser, products, taskId)
	logInfo(ctx, taskId, fmt.Sprintf("✅ 视频分析抓取完成，共获取 %d 条视频数据", len(videoAnalysisList)))

	if checkCancelled() {
		return res, gerror.New("任务已取消")
	}

	// ==================== 步骤 10: 保存数据到数据库 ====================
	logInfo(ctx, taskId, "💾 步骤 10: 保存数据到数据库")
	if err := s.saveAllData(ctx, taskId, products, productDetails, videoAnalysisList); err != nil {
		checkpointManager.FailTask(ctx, taskId, err.Error())
		logError(ctx, taskId, fmt.Sprintf("❌ 数据入库失败：%v", err))
		return res, gerror.Wrap(err, "保存数据失败")
	}
	logInfo(ctx, taskId, "✅ 数据入库成功")

	// ==================== 步骤 11: 导出数据文件 ====================
	logInfo(ctx, taskId, "📥 步骤 11: 导出数据文件（CSV/Excel）")
	modelProducts := s.convertToModelProducts(products, productDetails)
	exportFiles, err := exportManager.ExportAll(ctx, taskId, modelProducts, []model.VideoAnalysisData{})
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

	// 创建任务上下文
	_, cancel := context.WithCancel(ctx)
	defer cancel()

	// 设置上线状态
	statusMgr.SetOnline(taskId, "product_rank", cancel)
	defer statusMgr.SetOffline()

	// 启动浏览器
	browser, err := s.startBrowser(ctx, taskId, true)
	if err != nil {
		return gerror.Wrap(err, "启动浏览器失败")
	}
	defer browser.Close()

	// 打开首页
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return gerror.Wrap(err, "打开首页失败")
	}
	page = page.Timeout(15 * time.Minute)

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

	// 验证登录状态
	if !s.checkIsLoggedIn(ctx, page, taskId) {
		return gerror.New("Cookie 已过期，请重新登录")
	}

	// 更新状态消息
	statusMgr.SetMessage("抓取商品榜单中")

	// 注册 API 拦截器
	router, pageCh := s.setupAPIInterceptor(ctx, page, taskId)
	go router.Run()
	defer router.Stop()

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
	products := s.collectInterceptedData(ctx, page, taskId, nil, 1, pageCh)
	if len(products) == 0 {
		return gerror.New("未抓取到商品数据")
	}

	logInfo(ctx, taskId, fmt.Sprintf("✅ 商品榜单抓取完成，共 %d 条", len(products)))
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

	// 创建任务上下文
	_, cancel := context.WithCancel(ctx)
	defer cancel()

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
	browser, err := s.startBrowser(ctx, taskId, true)
	if err != nil {
		return gerror.Wrap(err, "启动浏览器失败")
	}
	defer browser.Close()

	// Cookie 登录
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return gerror.Wrap(err, "打开首页失败")
	}
	page = page.Timeout(15 * time.Minute)

	cookieManager := service.NewCookieManager("cookies")
	hasSavedCookies := cookieManager.HasCompassCookies(ctx, taskId)
	if !hasSavedCookies {
		return gerror.New("未找到已保存的 Cookie，请先执行登录操作")
	}

	// 注入 Cookie
	if !s.injectCookies(ctx, page, cookieManager, taskId) {
		return gerror.New("Cookie 注入失败")
	}

	// 验证登录状态
	if !s.checkIsLoggedIn(ctx, page, taskId) {
		return gerror.New("Cookie 已过期，请重新登录")
	}

	// 更新状态消息
	statusMgr.SetMessage("抓取视频数据中")

	// 使用优化版视频抓取（直接 HTTP 请求 + 分页）
	videoAnalysisList := s.scrapeVideoAnalysisOptimized(ctx, browser, products, taskId)
	logInfo(ctx, taskId, fmt.Sprintf("✅ 视频池抓取完成，共 %d 条视频数据", len(videoAnalysisList)))

	// 保存视频数据到数据库
	if len(videoAnalysisList) > 0 {
		// 去重
		videoAnalysisList = deduplicateVideos(videoAnalysisList)
		logInfo(ctx, taskId, fmt.Sprintf("✅ 去重后 %d 条视频数据", len(videoAnalysisList)))

		// 保存到数据库
		if err := s.saveVideoAnalysis(ctx, taskId, videoAnalysisList); err != nil {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 保存视频数据失败：%v", err))
		}
	}

	return nil
}

// getProductsFromDB 从数据库获取已抓取的商品列表
func (s *sCrawler) getProductsFromDB(ctx context.Context) ([]ProductInfo, error) {
	// 查询最新的商品数据
	result, err := g.DB().Model("tiktok_ranking_list").
		OrderDesc("created_at").
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
		_, err := g.DB().Model("video_analysis_data").Insert(g.Map{
			"task_id":             taskId,
			"product_id":          video.ProductID,
			"product_name":        video.ProductName,
			"video_id":            video.VideoId,
			"video_desc":          video.VideoDesc,
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

// startBrowser 启动浏览器
// startBrowser 启动浏览器 - headless: true表示无头模式，false表示有头模式（显示窗口）
func (s *sCrawler) startBrowser(ctx context.Context, taskId string, headless bool) (*rod.Browser, error) {
	l := launcher.New().
		Headless(headless).
		NoSandbox(true).
		Append("--start-maximized").
		Append("--disable-dev-shm-usage").
		Append("--disable-gpu").
		Append("--disable-extensions").
		Append("--disable-web-security").
		Append("--allow-running-insecure-content")

	// 防反爬配置
	l.Set("disable-blink-features", "AutomationControlled")
	l.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

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
	// 扩展选择器列表，适配更多分页组件
	nextSelectors := []string{
		".el-pagination .btn-next",
		".el-pager li:last-child",
		".ant-pagination-next",
		".ant-pagination-item-active + li",
		".pagination-next",
		"li[title='下一页']",
		"li[aria-label='Next Page']",
		"//*[contains(text(), '下一页')]",
		"//*[contains(@class, 'next')]",
	}

	for _, selector := range nextSelectors {
		var element *rod.Element
		var err error

		if strings.HasPrefix(selector, "//") {
			element, err = page.Timeout(3 * time.Second).ElementX(selector)
		} else {
			element, err = page.Timeout(3 * time.Second).Element(selector)
		}

		if err == nil {
			// 检查是否可点击（是否禁用）
			if disabled, _ := element.Attribute("disabled"); disabled != nil && *disabled == "true" {
				logInfo(ctx, taskId, "⚠️ 下一页按钮已禁用，已是最后一页")
				return false
			}

			// 检查是否是当前页（active 状态）
			if class, _ := element.Attribute("class"); class != nil && strings.Contains(*class, "active") {
				logInfo(ctx, taskId, "⚠️ 下一页按钮是 active 状态，已是最后一页")
				return false
			}

			if err := element.Click(proto.InputMouseButtonLeft, 1); err != nil {
				logDebug(ctx, taskId, fmt.Sprintf("⚠️ 点击失败：%v", err))
				continue
			}
			logInfo(ctx, taskId, "✅ 成功点击下一页")
			return true
		}
	}

	// 如果所有选择器都失败，尝试通过 URL 参数翻页
	logWarn(ctx, taskId, "⚠️ 未找到下一页按钮，尝试 URL 翻页")
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

	// 创建工作池：10个并发工作者（借鉴 Python 爬虫的 10 线程并发）
	pool := NewWorkerPool(10)

	startTime := time.Now()
	logInfo(ctx, taskId, fmt.Sprintf("🚀 开始并发获取商品详情（并发数=10，总数=%d）", len(products)))

	for i, product := range products {
		// 复制循环变量，避免闭包问题
		index := i
		p := product

		pool.Submit(func() {
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

	// 2. 构造详情 URL：优先用排行页提供的 URL，其次用 product_id + shop_id 模板
	detailURL := product.ProductURL
	if detailURL == "" && product.ProductID != "" {
		detailURL = buildDetailURL(product.ProductID, product.ShopID)
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

	// 4. 注册反检测拦截（配额 + 显示规则）
	router := page.HijackRequests()
	if err := router.Add("*get_view_quota*", "", func(h *rod.Hijack) {
		fakeResponse := `{"data":{"remaining_cnt":999,"target_id_viewable":true},"msg":"","st":0}`
		h.Response.SetBody([]byte(fakeResponse))
		h.Response.SetHeader("Content-Type", "application/json")
		h.ContinueRequest(&proto.FetchContinueRequest{})
	}); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 注册配额API拦截失败：%v", err))
	}
	if err := router.Add("*get_display_rule*", "", func(h *rod.Hijack) {
		fakeResponse := `{"data":{"display_type":1},"msg":"","st":0}`
		h.Response.SetBody([]byte(fakeResponse))
		h.Response.SetHeader("Content-Type", "application/json")
		h.ContinueRequest(&proto.FetchContinueRequest{})
	}); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 注册显示规则API拦截失败：%v", err))
	}
	go router.Run()
	defer router.Stop()

	// 5. 导航到详情页
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

	// 创建工作池：10个并发工作者（借鉴 Python 爬虫的 10 线程并发）
	pool := NewWorkerPool(10)

	startTime := time.Now()
	logInfo(ctx, taskId, fmt.Sprintf("🚀 开始并发获取视频分析（并发数=10，总数=%d）", len(products)))

	for i, product := range products {
		// 复制循环变量
		index := i
		p := product

		pool.Submit(func() {
			videos := s.scrapeSingleProductVideosWithAPI(ctx, browser, p, taskId)
			if len(videos) > 0 {
				mu.Lock()
				allVideos = append(allVideos, videos...)
				mu.Unlock()
				logDebug(ctx, taskId, fmt.Sprintf("✅ [%d/%d] 视频分析完成：%s (%d 条视频)", index+1, len(products), p.ProductName, len(videos)))
			} else {
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
	flowMorePattern := "*flow_more*"
	if err := router.Add(flowMorePattern, "", func(h *rod.Hijack) {
		reqURL := h.Request.URL().String()
		// 确保是视频分析的 flow_more 请求
		if !strings.Contains(reqURL, "source_entrance=video") {
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
		h.ContinueRequest(&proto.FetchContinueRequest{})
	}); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 注册 flow_more API 拦截失败：%v", err))
		return videos
	}

	// 拦截配额API和显示规则API，避免查看次数限制
	if err := router.Add("*get_view_quota*", "", func(h *rod.Hijack) {
		// 返回伪造的高配额响应
		fakeResponse := `{"data":{"remaining_cnt":999,"target_id_viewable":true},"msg":"","st":0}`
		h.Response.SetBody([]byte(fakeResponse))
		h.Response.SetHeader("Content-Type", "application/json")
		h.ContinueRequest(&proto.FetchContinueRequest{})
	}); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 注册配额API拦截失败：%v", err))
	}

	if err := router.Add("*get_display_rule*", "", func(h *rod.Hijack) {
		// 返回伪造的显示规则响应
		fakeResponse := `{"data":{"display_type":1},"msg":"","st":0}`
		h.Response.SetBody([]byte(fakeResponse))
		h.Response.SetHeader("Content-Type", "application/json")
		h.ContinueRequest(&proto.FetchContinueRequest{})
	}); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 注册显示规则API拦截失败：%v", err))
	}

	go router.Run()
	defer router.Stop()

	// 导航到视频分析页面
	analysisURL := fmt.Sprintf("https://compass.jinritemai.com/shop/product/product-detail?product_id=%s&tab=video", product.ProductID)
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
	if err := router.Add("*flow_more*", "", func(h *rod.Hijack) {
		// 记录请求 URL 和 Headers
		reqURL := h.Request.URL().String()
		if !strings.Contains(reqURL, "source_entrance=video") {
			h.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}

		// 提取 Headers
		headers := make(map[string]string)
		for k, v := range h.Request.Headers() {
			headers[k] = fmt.Sprintf("%v", v)
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

	// 拦截配额 API
	if err := router.Add("*get_view_quota*", "", func(h *rod.Hijack) {
		fakeResponse := `{"data":{"remaining_cnt":999,"target_id_viewable":true},"msg":"","st":0}`
		h.Response.SetBody([]byte(fakeResponse))
		h.Response.SetHeader("Content-Type", "application/json")
		h.ContinueRequest(&proto.FetchContinueRequest{})
	}); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 注册配额API拦截失败：%v", err))
	}

	go router.Run()
	defer router.Stop()

	// 打开第一个商品的详情页
	product := products[0]
	detailURL := fmt.Sprintf("https://compass.jinritemai.com/shop/product/product-detail?product_id=%s&tab=video", product.ProductID)

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
	Code int              `json:"code"`
	Data FlowMoreData     `json:"data"`
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
}

// fetchVideoPage 获取单页视频数据（使用正确的嵌套 JSON 结构 + 单位转换）
func (s *sCrawler) fetchVideoPage(
	template *FlowMoreTemplate,
	product ProductInfo,
	pageNo int,
) (*VideoPageResult, error) {
	// 替换 URL 中的 product_id 和 page_no
	requestURL := replaceURLParam(template.URLTemplate, "product_id", product.ProductID)
	requestURL = replaceURLParam(requestURL, "page_no", fmt.Sprintf("%d", pageNo))

	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, gerror.Wrap(err, "创建请求失败")
	}

	// 设置 Headers
	for k, v := range template.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Referer", fmt.Sprintf("https://compass.jinritemai.com/shop/product/product-detail?product_id=%s", product.ProductID))

	// 发送请求（带重试）
	client := &http.Client{Timeout: 30 * time.Second}
	var body []byte
	retryErr := s.retryWithBackoff(func() error {
		resp, reqErr := client.Do(req)
		if reqErr != nil {
			return reqErr
		}
		defer resp.Body.Close()
		body, err = io.ReadAll(resp.Body)
		return err
	}, 3)
	if retryErr != nil {
		return nil, gerror.Wrap(retryErr, "请求失败（重试3次后）")
	}

	// 解析嵌套 JSON 结构
	var resp FlowMoreResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, gerror.Wrap(err, "解析响应失败")
	}

	// 构建结果（使用正确的嵌套结构提取字段）
	result := &VideoPageResult{
		Total:    resp.Data.Total,
		PageSize: 10, // 默认每页 10 条
	}

	for _, item := range resp.Data.VideoList {
		video := VideoAnalysis{
			ProductID:         product.ProductID,
			ProductName:       product.ProductName,
			VideoId:           item.BaseInfo.VideoBaseInfo.VideoID,
			VideoDesc:         item.BaseInfo.VideoBaseInfo.VideoTitle,
			PaymentAmount:     item.Metrics.PayAmt.FormatRange(),
			ExposeUserCount:   item.Metrics.ShowUV.FormatRange(),
			ClickUserCount:    item.Metrics.DhProductClickUV.FormatRange(),
			ConvertUserCount:  item.Metrics.PayUcnt.FormatRange(),
			ClickRate:         item.Metrics.ClickRatio.FormatRange(),
			ClickConvertRate:  item.Metrics.ClickConversion.FormatRange(),
			ExposeConvertRate: item.Metrics.ShowConversion.FormatRange(),
			OrderCount:        item.Metrics.PayCnt.FormatRange(),
			SalesVolume:       item.Metrics.PayComboCnt.FormatRange(),
		}
		result.Videos = append(result.Videos, video)
	}

	return result, nil
}

// fetchVideosWithPagination 分页获取视频数据（借鉴 Python 爬虫的 _collect_flow_more_payloads）
func (s *sCrawler) fetchVideosWithPagination(
	template *FlowMoreTemplate,
	product ProductInfo,
) ([]VideoAnalysis, error) {
	var allVideos []VideoAnalysis

	// 请求第 1 页
	page1Result, err := s.fetchVideoPage(template, product, 1)
	if err != nil {
		return nil, err
	}
	allVideos = append(allVideos, page1Result.Videos...)

	// 如果没有分页信息或只有 1 页，直接返回
	if page1Result.Total == 0 || page1Result.PageSize == 0 {
		return allVideos, nil
	}

	// 计算总页数
	totalPages := (page1Result.Total + page1Result.PageSize - 1) / page1Result.PageSize
	if totalPages <= 1 {
		return allVideos, nil
	}

	// 请求剩余页
	for page := 2; page <= totalPages; page++ {
		pageResult, err := s.fetchVideoPage(template, product, page)
		if err != nil {
			// 出错停止分页，返回已获取的数据
			break
		}
		allVideos = append(allVideos, pageResult.Videos...)
		time.Sleep(700 * time.Millisecond) // 页间延迟，避免请求过快
	}

	return allVideos, nil
}

// scrapeVideoAnalysisOptimized 优化版视频分析抓取（借鉴 Python 爬虫的直接 HTTP 请求模式）
func (s *sCrawler) scrapeVideoAnalysisOptimized(ctx context.Context, browser *rod.Browser, products []ProductInfo, taskId string) []VideoAnalysis {
	if len(products) == 0 {
		return nil
	}

	// 1. 先用浏览器提取 API 模板
	logInfo(ctx, taskId, "🔧 正在提取 flow_more API 模板...")
	template, err := s.bootstrapFlowMoreTemplate(ctx, browser, products, taskId)
	if err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 提取 API 模板失败：%v，回退到浏览器模式", err))
		return s.scrapeVideoAnalysis(ctx, browser, products, taskId)
	}

	logInfo(ctx, taskId, "✅ API 模板提取成功，开始直接 HTTP 请求")

	// 2. 使用直接 HTTP 请求并发获取视频数据
	var allVideos []VideoAnalysis
	var mu sync.Mutex

	pool := NewWorkerPool(10)
	startTime := time.Now()

	for i, product := range products {
		index := i
		p := product

		pool.Submit(func() {
			videos, err := s.fetchVideosWithPagination(template, p)
			if err != nil {
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ [%d/%d] HTTP 请求失败：%s，回退到浏览器", index+1, len(products), err))
				// 回退到浏览器方式
				videos = s.scrapeSingleProductVideosWithAPI(ctx, browser, p, taskId)
			}

			if len(videos) > 0 {
				mu.Lock()
				allVideos = append(allVideos, videos...)
				mu.Unlock()
				logDebug(ctx, taskId, fmt.Sprintf("✅ [%d/%d] 视频分析完成：%s (%d 条视频)", index+1, len(products), p.ProductName, len(videos)))
			}
		})
	}

	pool.Wait()

	elapsed := time.Since(startTime)
	logInfo(ctx, taskId, fmt.Sprintf("✅ 视频分析获取完成：%d 条，耗时 %.1f 秒（HTTP 直接请求模式）",
		len(allVideos), elapsed.Seconds()))

	return allVideos
}

// parseVideoAnalysisResponse 解析视频分析 API 响应 JSON
func parseVideoAnalysisResponse(body string, product ProductInfo) []VideoAnalysis {
	var videos []VideoAnalysis

	// 打印响应体前1000字符，了解API响应结构
	if len(body) > 1000 {
		fmt.Printf("[DEBUG] 视频分析API响应前1000字符: %s\n", body[:1000])
	} else {
		fmt.Printf("[DEBUG] 视频分析API响应: %s\n", body)
	}

	// 这里需要根据实际的API响应结构进行解析
	// 以下是一个示例解析逻辑，需要根据实际API响应进行调整
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
	}

	type response struct {
		Data struct {
			VideoList []videoItem `json:"video_list"`
		} `json:"data"`
	}

	var resp response
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		fmt.Printf("[DEBUG] 解析视频分析API响应失败：%v\n", err)
		return videos
	}

	for _, item := range resp.Data.VideoList {
		video := VideoAnalysis{
			ProductID:         product.ProductID,
			ProductName:       product.ProductName,
			VideoId:           item.VideoId,
			VideoDesc:         item.VideoDesc,
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

	return videos
}

// scrapeSingleProductVideos 获取单个商品的视频分析数据（改进版）
func (s *sCrawler) scrapeSingleProductVideos(ctx context.Context, browser *rod.Browser, product ProductInfo, taskId string) []VideoAnalysis {
	var videos []VideoAnalysis

	// 重试打开页面
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 无法打开浏览器页面：%v", err))
		return videos
	}
	defer page.Close()

	// 导航到视频分析页面（带重试）
	analysisURL := fmt.Sprintf("https://compass.jinritemai.com/shop/product/product-detail?product_id=%s&tab=video", product.ProductID)
	if err := s.retryWithBackoff(func() error {
		return page.Navigate(analysisURL)
	}, 3); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 打开视频分析失败：%v", err))
		return videos
	}

	page.WaitLoad()
	time.Sleep(2 * time.Second) // 减少等待时间（从3秒改为2秒）

	// 获取核心数据
	coreData := s.scrapeVideoCoreData(ctx, page, taskId)

	// 获取短视频列表
	videoList := s.scrapeVideoList(ctx, page, taskId, product.ProductName)
	for _, v := range videoList {
		v.PaymentAmount = coreData.PaymentAmount
		v.ProductID = product.ProductID
		v.ProductName = product.ProductName
		videos = append(videos, v)
	}

	return videos
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
	var videos []VideoAnalysis

	// 查找视频列表
	videoRows, err := page.Elements(".video-item, .video-row, .ant-table-row")
	if err != nil {
		return videos
	}

	for i, row := range videoRows {
		video := VideoAnalysis{
			VideoId:   fmt.Sprintf("video_%d", i),
			VideoDesc: fmt.Sprintf("%s-视频%d", productName, i+1),
		}

		// 提取视频数据
		cells, err := row.Elements(".video-info, .ant-table-cell, td")
		if err == nil && len(cells) > 0 {
			if text, err := cells[0].Text(); err == nil {
				video.VideoDesc = strings.TrimSpace(text)
			}
		}

		videos = append(videos, video)
	}

	return videos
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

// setupAPIInterceptor 注册 API 拦截器（必须在页面导航和 Tab 点击之前调用）
// 返回 router 用于 defer 关闭，pageCh 用于接收每页数据
func (s *sCrawler) setupAPIInterceptor(ctx context.Context, page *rod.Page, taskId string) (*rod.HijackRouter, chan []ProductInfo) {
	pageCh := make(chan []ProductInfo, 1)

	router := page.HijackRequests()
	// 注意：不指定 ResourceType（传空字符串），否则只匹配 Fetch 类型，
	// 而抖店的 API 可能通过 XHR（XMLHttpRequest）发出，导致完全匹配不到
	if err := router.Add("*video_bring_good*", "", func(h *rod.Hijack) {
		if err := h.LoadResponse(&http.Client{}, true); err != nil {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ LoadResponse 失败：%v", err))
			h.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}
		body := h.Response.Body()
		if len(body) == 0 {
			// 响应体为空，可能是 Inspector Cache 被驱逐
			// 已通过 NetworkEnable.MaxTotalBufferSize 扩大缓存来预防此问题
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 响应体为空（可能 Inspector Cache 被驱逐），URL：%s", h.Request.URL().String()))
			h.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}
		logInfo(ctx, taskId, fmt.Sprintf("🔍 拦截到 API 响应，body 长度：%d", len(body)))
		products := parseProductRankResponse(body)
		logInfo(ctx, taskId, fmt.Sprintf("🔍 解析到 %d 条商品数据", len(products)))
		// 非阻塞发送，避免主循环已超时时 goroutine 泄漏
		select {
		case pageCh <- products:
		default:
			logWarn(ctx, taskId, "⚠️ pageCh 已满，丢弃数据（主循环未及时消费）")
		}
	}); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 注册 hijack 失败：%v", err))
	}

	return router, pageCh
}

// collectInterceptedData 从 pageCh 收集拦截到的数据并翻页
func (s *sCrawler) collectInterceptedData(ctx context.Context, page *rod.Page, taskId string, checkpointManager *service.CheckpointManager, startPage int, pageCh chan []ProductInfo) []ProductInfo {
	var allProducts []ProductInfo
	totalPages := 20
	currentPage := startPage

	for currentPage <= totalPages {
		logInfo(ctx, taskId, fmt.Sprintf("📄 正在处理第 %d/%d 页", currentPage, totalPages))

		// 等待当前页数据，最多 20 秒
		var pageProducts []ProductInfo
		select {
		case pageProducts = <-pageCh:
		case <-time.After(20 * time.Second):
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 第 %d 页等待超时，跳过", currentPage))
		}

		if len(pageProducts) > 0 {
			allProducts = append(allProducts, pageProducts...)
			logInfo(ctx, taskId, fmt.Sprintf("✅ 第 %d 页解析到 %d 条，立即保存", currentPage, len(pageProducts)))
			s.savePageData(ctx, taskId, pageProducts, currentPage)
		} else {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 第 %d 页未获取到数据", currentPage))
		}

		checkpointManager.UpdateProgress(ctx, taskId, 6, currentPage, len(allProducts))

		if currentPage < totalPages {
			if !s.clickNextPage(ctx, page, taskId) {
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ 无法点击下一页，停止翻页（当前第 %d 页）", currentPage))
				break
			}
		}

		currentPage++
	}

	allProducts = s.uniqueProducts(allProducts)
	logInfo(ctx, taskId, fmt.Sprintf("📊 API 拦截完成，共获取 %d 条商品（去重后）", len(allProducts)))
	return allProducts
}

// interceptProductRankAPI 已废弃，逻辑拆分到 setupAPIInterceptor + collectInterceptedData
// 保留此方法签名避免外部编译报错
func (s *sCrawler) interceptProductRankAPI(ctx context.Context, page *rod.Page, taskId string, checkpointManager *service.CheckpointManager, startPage int) []ProductInfo {
	return s.collectInterceptedData(ctx, page, taskId, checkpointManager, startPage, make(chan []ProductInfo, 1))
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
