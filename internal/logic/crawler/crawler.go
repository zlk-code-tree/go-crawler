package crawler

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/util/grand"
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
	ProductName      string // 商品名称
	ShopName         string // 店铺名称
	PaymentAmount    string // 短视频用户支付金额
	ClickCount       string // 短视频点击次数
	SalesVolume      string // 短视频成交件数
	DeliveryVideoNum string // 带货视频数
	ProductURL       string // 商品详情页 URL
	FirstList        int    // 是否首次上榜：0-否 1-是
	PriceRange       string // 价格带
}

// ProductDetail 商品详情数据
type ProductDetail struct {
	ProductName    string // 商品名称
	FirstShelfTime string // 首次上架时间
	MinPrice       string // 最低到手价
	PriceRange     string // 价格带
}

// VideoAnalysis 视频分析数据
type VideoAnalysis struct {
	VideoID           string // 视频 ID
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

	// ==================== 步骤 4: 导航到商品榜单页面 ====================
	logInfo(ctx, taskId, "📊 步骤 4: 导航到商品榜单页面")
	productRankURL := "https://compass.jinritemai.com/shop/chance/product-rank"
	if err := page.Navigate(productRankURL); err != nil {
		checkpointManager.FailTask(ctx, taskId, err.Error())
		return res, gerror.Wrap(err, "跳转到商品榜单失败")
	}
	page.WaitLoad()
	time.Sleep(3 * time.Second)

	// ==================== 步骤 5: 点击短视频榜 Tab ====================
	logInfo(ctx, taskId, "🎬 步骤 5: 点击短视频榜 Tab")
	if err := s.clickShortVideoTab(ctx, page, taskId); err != nil {
		checkpointManager.FailTask(ctx, taskId, err.Error())
		return res, gerror.Wrap(err, "点击短视频榜失败")
	}
	time.Sleep(2 * time.Second)

	// ==================== 步骤 6: 分页抓取 200 条数据（支持断点续爬） ====================
	logInfo(ctx, taskId, "📈 步骤 6: 分页抓取商品榜单数据（20 页×10 条=200 条）")

	var products []ProductInfo
	startPage := 1
	if existingTask != nil && existingTask.Status == "running" {
		startPage = existingTask.CurrentPage
		logInfo(ctx, taskId, fmt.Sprintf("🔄 从第 %d 页继续爬取", startPage))
	}

	products = s.scrapeAllPagesWithCheckpoint(ctx, page, taskId, checkpointManager, startPage)
	if len(products) == 0 {
		checkpointManager.FailTask(ctx, taskId, "未抓取到数据")
		logWarn(ctx, taskId, "⚠️ 未抓取到有效数据")
		return res, gerror.New("未抓取到商品数据")
	}
	logInfo(ctx, taskId, fmt.Sprintf("✅ 抓取完成，共获取 %d 条商品数据", len(products)))

	// ==================== 步骤 7: 获取商品详情数据 ====================
	logInfo(ctx, taskId, "🔍 步骤 7: 获取商品详情数据（首次上架时间、最低到手价）")
	productDetails := s.scrapeProductDetails(ctx, browser, products, taskId)
	logInfo(ctx, taskId, fmt.Sprintf("✅ 详情抓取完成，共获取 %d 条详情数据", len(productDetails)))

	// ==================== 步骤 8: 获取视频分析数据 ====================
	logInfo(ctx, taskId, "📹 步骤 8: 获取视频分析数据")
	videoAnalysisList := s.scrapeVideoAnalysis(ctx, browser, products, taskId)
	logInfo(ctx, taskId, fmt.Sprintf("✅ 视频分析抓取完成，共获取 %d 条视频数据", len(videoAnalysisList)))

	// ==================== 步骤 9: 保存数据到数据库 ====================
	logInfo(ctx, taskId, "💾 步骤 9: 保存数据到数据库")
	if err := s.saveAllData(ctx, taskId, products, productDetails, videoAnalysisList); err != nil {
		checkpointManager.FailTask(ctx, taskId, err.Error())
		logError(ctx, taskId, fmt.Sprintf("❌ 数据入库失败：%v", err))
		return res, gerror.Wrap(err, "保存数据失败")
	}
	logInfo(ctx, taskId, "✅ 数据入库成功")

	// ==================== 步骤 10: 导出数据文件 ====================
	logInfo(ctx, taskId, "📥 步骤 10: 导出数据文件（CSV/Excel）")
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

	// 创建工作池：5个并发工作者
	pool := NewWorkerPool(5)

	startTime := time.Now()
	logInfo(ctx, taskId, fmt.Sprintf("🚀 开始并发获取商品详情（并发数=5，总数=%d）", len(products)))

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

// scrapeSingleProductDetail 获取单个商品详情（改进版）
func (s *sCrawler) scrapeSingleProductDetail(ctx context.Context, browser *rod.Browser, product ProductInfo, taskId string) *ProductDetail {
	// 如果没有有效的详情 URL，跳过
	if product.ProductURL == "" {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 商品 [%s] 无详情 URL，跳过详情获取", product.ProductName))
		return &ProductDetail{ProductName: product.ProductName}
	}

	var page *rod.Page

	// 重试打开页面（最多3次）
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

	// 导航到商品详情页（使用从排行榜提取的真实 URL）
	detailURL := product.ProductURL
	logDebug(ctx, taskId, fmt.Sprintf("🔗 正在打开商品详情：%s", detailURL))

	if err := s.retryWithBackoff(func() error {
		return page.Navigate(detailURL)
	}, 3); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 打开商品详情失败：%v", err))
		return nil
	}

	page.WaitLoad()
	time.Sleep(2 * time.Second)

	// 提取详情数据
	detail := &ProductDetail{
		ProductName: product.ProductName,
	}

	// 获取首次上架时间（多选择器降级）
	shelfTimeSelectors := []string{
		"//*[contains(text(), '首次上架')]/following-sibling::*[1]",
		"//*[contains(text(), '上架时间')]/following-sibling::*[1]",
		"//span[contains(text(), '首次上架')]/../following-sibling::*[1]",
		"//div[contains(text(), '上架时间')]/following-sibling::*",
	}
	for _, selector := range shelfTimeSelectors {
		var elem *rod.Element
		var err error
		if strings.HasPrefix(selector, "//") {
			elem, err = page.Timeout(2 * time.Second).ElementX(selector)
		} else {
			elem, err = page.Timeout(2 * time.Second).Element(selector)
		}
		if err == nil {
			if text, textErr := elem.Text(); textErr == nil && strings.TrimSpace(text) != "" {
				detail.FirstShelfTime = strings.TrimSpace(text)
				logDebug(ctx, taskId, fmt.Sprintf("📅 获取到上架时间：%s", detail.FirstShelfTime))
				break
			}
		}
	}

	// 获取最低到手价（多选择器降级）
	priceSelectors := []string{
		"//*[contains(text(), '最低到手价')]/following-sibling::*[1]",
		"//*[contains(text(), '到手价')]/following-sibling::*[1]",
		"//*[contains(text(), '最低价')]/following-sibling::*[1]",
		"//span[contains(text(), '最低到手价')]/../following-sibling::*[1]",
	}
	for _, selector := range priceSelectors {
		var elem *rod.Element
		var err error
		if strings.HasPrefix(selector, "//") {
			elem, err = page.Timeout(2 * time.Second).ElementX(selector)
		} else {
			elem, err = page.Timeout(2 * time.Second).Element(selector)
		}
		if err == nil {
			if text, textErr := elem.Text(); textErr == nil && strings.TrimSpace(text) != "" {
				detail.MinPrice = strings.TrimSpace(text)
				logDebug(ctx, taskId, fmt.Sprintf("💰 获取到最低到手价：%s", detail.MinPrice))
				break
			}
		}
	}

	if detail.FirstShelfTime == "" && detail.MinPrice == "" {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 商品 [%s] 详情页未提取到数据，可能需要更新选择器", product.ProductName))
	}

	return detail
}

// scrapeVideoAnalysis 获取视频分析数据（并发版本）
func (s *sCrawler) scrapeVideoAnalysis(ctx context.Context, browser *rod.Browser, products []ProductInfo, taskId string) []VideoAnalysis {
	var allVideos []VideoAnalysis
	var mu sync.Mutex

	// 创建工作池：5个并发工作者
	pool := NewWorkerPool(5)

	startTime := time.Now()
	logInfo(ctx, taskId, fmt.Sprintf("🚀 开始并发获取视频分析（并发数=5，总数=%d）", len(products)))

	for i, product := range products {
		// 复制循环变量
		index := i
		p := product

		pool.Submit(func() {
			videos := s.scrapeSingleProductVideos(ctx, browser, p, taskId)
			if len(videos) > 0 {
				mu.Lock()
				allVideos = append(allVideos, videos...)
				mu.Unlock()
				logDebug(ctx, taskId, fmt.Sprintf("✅ [%d/%d] 视频分析完成：%s (%d 条视频)", index+1, len(products), p.ProductName, len(videos)))
			} else {
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ [%d/%d] 视频分析失败：%s", index+1, len(products), p.ProductName))
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
	analysisURL := fmt.Sprintf("https://compass.jinritemai.com/shop/product/product-detail?product_id=%s&tab=video", product.ProductName)
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
			VideoID:   fmt.Sprintf("video_%d", i),
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
	for i, p := range products {
		record := &model.TikTokRankingListInput{
			TaskId:           taskId,
			Rank:             p.Rank,
			ProductName:      p.ProductName,
			ShopName:         p.ShopName,
			PaymentAmount:    p.PaymentAmount,
			ClickCount:       p.ClickCount,
			SalesVolume:      p.SalesVolume,
			DeliveryVideoNum: p.DeliveryVideoNum,
			CreatedAt:        time.Now(),
		}

		// 合并详情数据
		if i < len(details) {
			record.FirstShelfTime = details[i].FirstShelfTime
			record.MinPrice = details[i].MinPrice
		}

		rankingList = append(rankingList, record)
	}

	// 商品数据已经每页保存，这里跳过批量插入
	if len(rankingList) > 0 {
		logInfo(ctx, taskId, fmt.Sprintf("ℹ️  商品数据已分页保存（共 %d 条），跳过批量插入", len(rankingList)))
	}

	// 保存视频分析数据
	var videoData []*model.VideoAnalysisData
	for _, v := range videos {
		record := &model.VideoAnalysisData{
			TaskId:            taskId,
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
		if err := service.VideoAnalysis().CreateBatch(ctx, videoData); err != nil {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 保存视频分析数据失败：%v", err))
		} else {
			logInfo(ctx, taskId, fmt.Sprintf("✅ 保存视频分析数据：%d 条", len(videoData)))
		}
	}

	return nil
}

// ==================== 工具函数 ====================

// convertToModelProducts 转换 ProductInfo 为 model.TikTokRankingListInput
func (s *sCrawler) convertToModelProducts(products []ProductInfo, details []ProductDetail) []model.TikTokRankingListInput {
	var result []model.TikTokRankingListInput
	for i, p := range products {
		record := model.TikTokRankingListInput{
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
		if i < len(details) {
			record.FirstShelfTime = details[i].FirstShelfTime
			record.MinPrice = details[i].MinPrice
		}
		result = append(result, record)
	}
	return result
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

	// 转换为数据库模型
	var rankingList []*model.TikTokRankingListInput
	for _, p := range products {
		record := &model.TikTokRankingListInput{
			TaskId:           taskId,
			Rank:             p.Rank,
			ProductName:      p.ProductName,
			ShopName:         p.ShopName,
			PaymentAmount:    p.PaymentAmount,
			ClickCount:       p.ClickCount,
			SalesVolume:      p.SalesVolume,
			DeliveryVideoNum: p.DeliveryVideoNum,
			FirstShelfTime:   "",
			MinPrice:         "",
			FirstList:        p.FirstList,  // 首次上榜标识
			PriceRange:       p.PriceRange, // 从表格提取的价格带
			CreatedAt:        time.Now(),
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
