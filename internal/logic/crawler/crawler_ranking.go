package crawler

import (
	"context"
	"encoding/base64"
	"fmt"
	netUrl "net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"

	"go-crawler/internal/service"
)

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
			return waitForCrawler(ctx, 2*time.Second)
		}
	}

	// 如果直接点击失败，尝试通过 URL 导航
	shortVideoURL := "https://compass.jinritemai.com/shop/chance/product-rank?rankType=short_video"
	logInfo(ctx, taskId, fmt.Sprintf("🔄 Tab 点击失败，尝试直接导航：%s", shortVideoURL))
	if err := page.Navigate(shortVideoURL); err != nil {
		return fmt.Errorf("导航到短视频榜失败：%w", err)
	}
	page.WaitLoad()
	return waitForCrawler(ctx, 2*time.Second)
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
	return waitForCrawler(ctx, 1500*time.Millisecond) == nil
}

// collectInterceptedData 从 pageCh 收集拦截到的数据并翻页
func (s *sCrawler) collectInterceptedData(ctx context.Context, page *rod.Page, taskId string, checkpointManager *service.CheckpointManager, startPage int, pageCh chan browserRankingPage) ([]ProductInfo, error) {
	var allProducts []ProductInfo
	totalPages := 20
	currentPage := startPage

	for currentPage <= totalPages {
		if ctx.Err() != nil {
			return s.uniqueProducts(allProducts), ctx.Err()
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
			case response, ok := <-pageCh:
				if !ok {
					responseTimer.Stop()
					return s.uniqueProducts(allProducts), fmt.Errorf("榜单响应监听已关闭")
				}
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
				return s.uniqueProducts(allProducts), ctx.Err()
			case <-responseTimer.C:
				recoverAttempts++
				if recoverAttempts <= 2 && s.triggerRankingPageRefresh(ctx, page, taskId, currentPage) {
					logWarn(ctx, taskId, fmt.Sprintf("⚠️ 第 %d 页等待浏览器响应超时，已触发第 %d 次刷新重试", currentPage, recoverAttempts))
					continue
				}
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ 第 %d 页等待浏览器响应超时，停止翻页", currentPage))
				allProducts = s.uniqueProducts(allProducts)
				logInfo(ctx, taskId, fmt.Sprintf("📊 浏览器榜单响应采集完成，共获取 %d 条商品（去重后）", len(allProducts)))
				return allProducts, fmt.Errorf("第 %d 页榜单响应超时，保留已保存数据", currentPage)
			}
		}

		if len(pageProducts) > 0 {
			allProducts = append(allProducts, pageProducts...)
			logInfo(ctx, taskId, fmt.Sprintf("✅ 第 %d 页解析到 %d 条，立即保存", currentPage, len(pageProducts)))
			if err := s.savePageData(ctx, taskId, pageProducts, currentPage); err != nil {
				return s.uniqueProducts(allProducts), err
			}
		}

		if checkpointManager != nil {
			if err := checkpointManager.UpdateProgress(ctx, taskId, 6, currentPage, len(allProducts)); err != nil {
				return s.uniqueProducts(allProducts), err
			}
		}

		if currentPage < totalPages {
			if !s.clickNextPage(ctx, page, taskId) {
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ 无法点击下一页，停止翻页（当前第 %d 页）", currentPage))
				return s.uniqueProducts(allProducts), fmt.Errorf("第 %d 页后无法翻页，榜单采集尚未完成", currentPage)
			}
		}

		currentPage++
	}

	allProducts = s.uniqueProducts(allProducts)
	logInfo(ctx, taskId, fmt.Sprintf("📊 浏览器榜单响应采集完成，共获取 %d 条商品（去重后）", len(allProducts)))
	return allProducts, nil
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
