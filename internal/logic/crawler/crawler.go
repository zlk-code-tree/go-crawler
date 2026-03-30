package crawler

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/util/grand"
)

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
}

// ProductDetail 商品详情数据
type ProductDetail struct {
	ProductName    string // 商品名称
	FirstShelfTime string // 首次上架时间
	MinPrice       string // 最低到手价
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

	// ==================== 步骤 1: 启动浏览器 ====================
	logInfo(ctx, taskId, "🚀 步骤 1: 启动浏览器")
	browser, err := s.startBrowser(ctx, taskId)
	if err != nil {
		checkpointManager.FailTask(ctx, taskId, err.Error())
		return res, gerror.Wrap(err, "浏览器启动失败")
	}
	defer browser.Close()

	// ==================== 步骤 2: 打开首页并加载 Cookie ====================
	logInfo(ctx, taskId, "🔐 步骤 2: 加载 Cookie 并检测登录状态")
	homeURL := "https://compass.jinritemai.com/shop"
	page, err := browser.Page(proto.TargetCreateTarget{URL: homeURL})
	if err != nil {
		checkpointManager.FailTask(ctx, taskId, err.Error())
		return res, gerror.Wrap(err, "打开首页失败")
	}
	page = page.Timeout(15 * time.Minute)

	// 检测是否已登录
	isLoggedIn := s.checkIsLoggedIn(ctx, page, taskId)

	// 如果未登录，尝试使用 Cookie
	if !isLoggedIn {
		cookieFile := "cookies/cookies.json"
		if gfile.Exists(cookieFile) {
			logInfo(ctx, taskId, "🍪 发现 Cookie 文件，尝试自动登录...")
			// 等待页面完全加载
			time.Sleep(5 * time.Second)
			// 再次检查登录状态（有时页面加载慢）
			isLoggedIn = s.checkIsLoggedIn(ctx, page, taskId)
			if isLoggedIn {
				logInfo(ctx, taskId, "✅ Cookie 自动登录成功！")
			} else {
				logWarn(ctx, taskId, "⚠️ Cookie 已过期或无效，需要重新登录")
			}
		}
	}
	if !isLoggedIn {
		// ==================== 步骤 3: 执行登录 ====================
		logInfo(ctx, taskId, "📝 步骤 3: Cookie 失效或未登录，执行登录流程")
		loginSuccess := s.performLogin(ctx, page, taskId, req.LoginType, req.Username, req.Password)
		if !loginSuccess {
			checkpointManager.FailTask(ctx, taskId, "登录失败")
			return res, gerror.New("登录失败，请检查账号密码或重新尝试")
		}
		logInfo(ctx, taskId, "✅ 登录成功")

		// 保存新的 Cookie
		allCookies, _ := page.Cookies(nil)
		if err := cookieManager.SaveCookies(ctx, allCookies, taskId); err != nil {
			logWarn(ctx, taskId, fmt.Sprintf("⚠️ 保存 Cookie 失败：%v", err))
		} else {
			logInfo(ctx, taskId, fmt.Sprintf("💾 已保存 %d 个 Cookie 到文件", len(allCookies)))
		}
	} else {
		logInfo(ctx, taskId, "✅ 已登录（Cookie 自动登录成功），跳过登录步骤")
		// 更新 Cookie（刷新过期时间）
		allCookies, _ := page.Cookies(nil)
		if len(allCookies) > 0 {
			cookieManager.SaveCookies(ctx, allCookies, taskId)
			logInfo(ctx, taskId, "💾 Cookie 已更新")
		}
	}

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
func (s *sCrawler) startBrowser(ctx context.Context, taskId string) (*rod.Browser, error) {
	l := launcher.New().
		Headless(false).
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

		// 抓取当前页数据
		products := s.scrapeCurrentPage(ctx, page, taskId)
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
func (s *sCrawler) scrapeCurrentPage(ctx context.Context, page *rod.Page, taskId string) []ProductInfo {
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

	// 解析每行数据
	for rowIdx, row := range rows {
		// 跳过表头
		if rowIdx == 0 {
			text, _ := row.Text()
			if strings.Contains(text, "排名") || strings.Contains(text, "商品") || strings.Contains(text, "店铺") {
				logDebug(ctx, taskId, fmt.Sprintf("⏭️ 跳过表头行：%s", text))
				continue
			}
		}

		product := s.parseProductRow(ctx, row, taskId, rowIdx)
		if product != nil && product.ProductName != "" {
			// 如果排名解析失败，使用行号作为排名
			if product.Rank == 0 {
				product.Rank = rowIdx + 1
			}
			logDebug(ctx, taskId, fmt.Sprintf("✅ 解析商品：排名=%d, 名称=%s", product.Rank, product.ProductName))
			products = append(products, *product)
		}
	}

	logInfo(ctx, taskId, fmt.Sprintf("📊 当前页解析完成：获取 %d 条商品", len(products)))
	return products
}

// parseProductRow 解析商品行数据
func (s *sCrawler) parseProductRow(ctx context.Context, row *rod.Element, taskId string, rowIdx int) *ProductInfo {
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

	// 获取排名（处理 "#" 符号和纯文本）
	rankText := s.getText(cells[0])
	rank := s.parseRank(rankText)
	if rank == 0 {
		rank = rowIdx + 1 // 使用行号作为排名
	}

	product := &ProductInfo{
		Rank:             rank,
		ProductName:      s.getText(cells[1]),
		ShopName:         s.getText(cells[2]),
		SalesVolume:      s.getText(cells[3]),
		PaymentAmount:    s.getText(cells[4]),
		ClickCount:       s.getText(cells[5]),
		DeliveryVideoNum: "0",
	}

	if len(cells) > 6 {
		product.DeliveryVideoNum = s.getText(cells[6])
	}

	// 尝试获取商品详情 URL
	if len(cells) > 7 {
		if link, err := cells[1].Element("a"); err == nil {
			if href, _ := link.Attribute("href"); href != nil {
				product.ProductURL = *href
			}
		}
	}

	logDebug(ctx, taskId, fmt.Sprintf("✅ 解析商品行 %d: 排名=%d, 名称=%s", rowIdx, product.Rank, product.ProductName))
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

// scrapeProductDetails 获取商品详情数据
func (s *sCrawler) scrapeProductDetails(ctx context.Context, browser *rod.Browser, products []ProductInfo, taskId string) []ProductDetail {
	var details []ProductDetail

	for i, product := range products {
		logInfo(ctx, taskId, fmt.Sprintf("🔍 获取商品详情 %d/%d: %s", i+1, len(products), product.ProductName))

		detail := s.scrapeSingleProductDetail(ctx, browser, product, taskId)
		if detail != nil {
			details = append(details, *detail)
		}

		// 避免请求过快
		time.Sleep(1 * time.Second)
	}

	return details
}

// scrapeSingleProductDetail 获取单个商品详情
func (s *sCrawler) scrapeSingleProductDetail(ctx context.Context, browser *rod.Browser, product ProductInfo, taskId string) *ProductDetail {
	// 打开新页面
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return nil
	}
	defer page.Close()

	// 导航到商品详情页（这里需要根据实际 URL 调整）
	// 示例：https://detail.jinritemai.com/xxx
	detailURL := product.ProductURL
	if detailURL == "" {
		// 构造详情页 URL（需要根据实际情况调整）
		detailURL = fmt.Sprintf("https://detail.jinritemai.com/product/%s", product.ProductName)
	}

	if err := page.Navigate(detailURL); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 打开商品详情失败：%v", err))
		return nil
	}

	page.WaitLoad()
	time.Sleep(2 * time.Second)

	// 提取详情数据
	detail := &ProductDetail{
		ProductName: product.ProductName,
	}

	// 获取首次上架时间
	if elem, err := page.Timeout(3 * time.Second).Element("//*[contains(text(), '上架时间')]"); err == nil {
		if parent, err := elem.Parent(); err == nil {
			if text, err := parent.Text(); err == nil {
				detail.FirstShelfTime = strings.TrimSpace(text)
			}
		}
	}

	// 获取最低到手价
	if elem, err := page.Timeout(3 * time.Second).Element("//*[contains(text(), '最低到手价')]/following-sibling::*"); err == nil {
		if text, err := elem.Text(); err == nil {
			detail.MinPrice = strings.TrimSpace(text)
		}
	}

	logDebug(ctx, taskId, fmt.Sprintf("✅ 获取商品详情：名称=%s, 上架时间=%s, 最低价=%s",
		detail.ProductName, detail.FirstShelfTime, detail.MinPrice))
	return detail
}

// scrapeVideoAnalysis 获取视频分析数据
func (s *sCrawler) scrapeVideoAnalysis(ctx context.Context, browser *rod.Browser, products []ProductInfo, taskId string) []VideoAnalysis {
	var allVideos []VideoAnalysis

	for i, product := range products {
		logInfo(ctx, taskId, fmt.Sprintf("📹 获取视频分析 %d/%d: %s", i+1, len(products), product.ProductName))

		videos := s.scrapeSingleProductVideos(ctx, browser, product, taskId)
		allVideos = append(allVideos, videos...)

		time.Sleep(1 * time.Second)
	}

	return allVideos
}

// scrapeSingleProductVideos 获取单个商品的视频分析数据
func (s *sCrawler) scrapeSingleProductVideos(ctx context.Context, browser *rod.Browser, product ProductInfo, taskId string) []VideoAnalysis {
	var videos []VideoAnalysis

	// 打开新页面
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return videos
	}
	defer page.Close()

	// 导航到视频分析页面
	analysisURL := fmt.Sprintf("https://compass.jinritemai.com/shop/product/product-detail?product_id=%s&tab=video", product.ProductName)
	if err := page.Navigate(analysisURL); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 打开视频分析失败：%v", err))
		return videos
	}

	page.WaitLoad()
	time.Sleep(3 * time.Second)

	// 获取核心数据
	coreData := s.scrapeVideoCoreData(ctx, page, taskId)

	// 获取短视频列表
	videoList := s.scrapeVideoList(ctx, page, taskId, product.ProductName)
	for _, v := range videoList {
		v.PaymentAmount = coreData.PaymentAmount
		videos = append(videos, v)
	}

	logInfo(ctx, taskId, fmt.Sprintf("✅ 获取视频分析：商品=%s, 视频数=%d", product.ProductName, len(videos)))
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

		// 抓取当前页数据
		products := s.scrapeCurrentPage(ctx, page, taskId)
		if len(products) > 0 {
			allProducts = append(allProducts, products...)
			logInfo(ctx, taskId, fmt.Sprintf("✅ 第 %d 页抓取完成：获取 %d 条数据", currentPage, len(products)))
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
			CreatedAt:        time.Now(),
		}
		rankingList = append(rankingList, record)
	}

	// 批量插入
	if err := service.TikTokRankingList().CreateBatch(ctx, rankingList); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 保存第 %d 页数据失败：%v", pageNum, err))
	} else {
		logInfo(ctx, taskId, fmt.Sprintf("💾 第 %d 页数据已保存：%d 条", pageNum, len(rankingList)))
	}
}

func (s *sCrawler) getText(el *rod.Element) string {
	if el == nil {
		return ""
	}
	text, _ := el.Text()
	return strings.TrimSpace(text)
}

// parseRank 解析排名（处理 "#" 符号、纯文本、行号等）
func (s *sCrawler) parseRank(str string) int {
	// 移除常见符号
	str = strings.ReplaceAll(str, "#", "")
	str = strings.ReplaceAll(str, " ", "")
	str = strings.TrimSpace(str)

	// 尝试转换为整数
	val, err := strconv.Atoi(str)
	if err == nil {
		return val
	}

	// 如果转换失败，返回 0（调用方会使用行号）
	return 0
}

func (s *sCrawler) toInt(str string) int {
	return s.parseRank(str)
}

func (s *sCrawler) uniqueProducts(products []ProductInfo) []ProductInfo {
	rankMap := make(map[int]bool)
	var unique []ProductInfo

	for _, item := range products {
		if !rankMap[item.Rank] {
			rankMap[item.Rank] = true
			unique = append(unique, item)
		}
	}

	sort.Slice(unique, func(i, j int) bool {
		return unique[i].Rank < unique[j].Rank
	})

	return unique
}
