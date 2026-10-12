package crawler

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/grand"

	"go-crawler/internal/service"
)

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
	ctx, finish, err := statusMgr.AcquireTask(ctx, taskId, req.LoginType)
	if err != nil {
		return res, err
	}
	defer finish()

	// 检查是否有未完成的任务（断点续爬）
	existingTask, err := checkpointManager.LoadCheckpoint(ctx, taskId)
	if err != nil {
		return res, err
	}
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
			closeBrowser(browser)
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
			closeBrowser(browser)
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
			closeBrowser(headedBrowser)
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
			closeBrowser(headedBrowser)
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
	defer closeBrowser(browser)

	// 更新状态消息
	statusMgr.SetMessage(ctx, "爬取数据中")

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
	if existingTask != nil && existingTask.CurrentStep >= 8 && existingTask.CurrentProductIndex >= 0 &&
		(existingTask.Status == "paused" || existingTask.Status == "running" || existingTask.Status == "failed") {
		videoStartIndex = existingTask.CurrentProductIndex
		resumeVideoPhase = true
		logInfo(ctx, taskId, fmt.Sprintf("🔄 从视频采集断点继续：商品索引=%d，已抓视频=%d 条", videoStartIndex, existingTask.VideoCount))
	} else if existingTask != nil && existingTask.Status == "running" {
		startPage = 1 // 榜单重放采用任务内幂等写入，避免 UI 页码与断点不一致
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
	if err := waitForCrawler(ctx, 3*time.Second); err != nil {
		return res, err
	}

	// ==================== 步骤 6: 点击短视频榜 Tab ====================
	logInfo(ctx, taskId, "🎬 步骤 6: 点击短视频榜 Tab")
	if err := s.clickShortVideoTab(ctx, page, taskId); err != nil {
		checkpointManager.FailTask(ctx, taskId, err.Error())
		return res, gerror.Wrap(err, "点击短视频榜失败")
	}
	if err := waitForCrawler(ctx, 2*time.Second); err != nil {
		return res, err
	}

	if checkCancelled() {
		return res, gerror.New("任务已取消")
	}

	// ==================== 步骤 7: 分页抓取 200 条数据（支持断点续爬） ====================
	logInfo(ctx, taskId, "📈 步骤 7: 分页抓取商品榜单数据（20 页×10 条=200 条）")

	var products []ProductInfo
	if resumeVideoPhase {
		products, err = s.loadProductsFromDB(ctx, taskId)
		if err != nil {
			return res, fmt.Errorf("读取恢复任务商品快照失败：%w", err)
		}
		if len(products) > 0 {
			logInfo(ctx, taskId, fmt.Sprintf("✅ 从数据库加载 %d 条商品，跳过榜单重抓，直接恢复视频采集", len(products)))
		}
	}
	if resumeVideoPhase && len(products) == 0 {
		return res, gerror.New("恢复任务的商品快照不存在")
	}
	if len(products) == 0 {
		products, err = s.collectInterceptedData(ctx, page, taskId, checkpointManager, startPage, pageCh)
		if err != nil {
			return res, err
		}
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
	videoScrape := s.scrapeVideoAnalysisDOMWithSinkFrom(ctx, browser, products, taskId, videoStartIndex, checkpointManager, func(videos []VideoAnalysis) error {
		return s.persistAndQueueVideos(ctx, taskId, videos)
	})
	if videoScrape.Err != nil {
		return res, videoScrape.Err
	}
	videoAnalysisList := videoScrape.Videos
	totalVideoCount, err := s.countVideoRowsChecked(ctx, taskId)
	if err != nil {
		return res, fmt.Errorf("查询任务视频总数失败：%w", err)
	}
	logInfo(ctx, taskId, fmt.Sprintf("✅ 本轮视频分析抓取到 %d 条，任务累计 %d 条", len(videoAnalysisList), totalVideoCount))
	if videoScrape.Paused {
		logWarn(ctx, taskId, fmt.Sprintf("⏸️ 本轮视频采集已暂停：next_index=%d/%d，原因=%s", videoScrape.NextProductIndex, len(products), videoScrape.StopReason))

		modelProducts := s.convertToModelProducts(products, nil)
		modelVideos := s.convertToModelVideos(taskId, videoAnalysisList)
		exportFiles, exportErr := exportManager.ExportAll(ctx, taskId, modelProducts, modelVideos)
		if exportErr != nil {
			return res, fmt.Errorf("暂停前导出数据失败：%w", exportErr)
		}
		logInfo(ctx, taskId, fmt.Sprintf("✅ 暂停前已导出数据：%d 个文件", len(exportFiles)))
		if err := s.updateCrawlTaskSummary(ctx, taskId, "paused", exportFiles); err != nil {
			return res, err
		}
		res = append(res, Result{
			URL:           productRankURL,
			Title:         fmt.Sprintf("抖音商品短视频榜_暂停_%d 条商品_%d 条视频", len(products), totalVideoCount),
			Content:       fmt.Sprintf("视频采集已暂停：%s；下次将从第 %d 个商品继续", videoScrape.StopReason, videoScrape.NextProductIndex+1),
			ProductCount:  len(products),
			VideoCount:    totalVideoCount,
			Paused:        true,
			NextIndex:     videoScrape.NextProductIndex,
			StopReason:    videoScrape.StopReason,
			Products:      products,
			VideoAnalysis: videoAnalysisList,
			ExportFiles:   exportFiles,
		})
		return res, nil
	}

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
	if err := checkpointManager.UpdateProgress(ctx, taskId, 9, 20, len(products)); err != nil {
		return res, fmt.Errorf("保存商品详情阶段断点失败：%w", err)
	}

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
		return res, fmt.Errorf("数据导出失败：%w", err)
	}
	logInfo(ctx, taskId, fmt.Sprintf("✅ 数据导出成功：%d 个文件", len(exportFiles)))
	for key, file := range exportFiles {
		logInfo(ctx, taskId, fmt.Sprintf("  - %s: %s", key, file))
	}

	if err := s.updateCrawlTaskSummary(ctx, taskId, "completed", exportFiles); err != nil {
		return res, err
	}
	// 确认数据与导出状态都已保存，再移除断点。
	if err := checkpointManager.CompleteTask(ctx, taskId, len(products)); err != nil {
		return res, err
	}

	// 构造返回结果
	res = append(res, Result{
		URL:   productRankURL,
		Title: fmt.Sprintf("抖音商品短视频榜_%d 条商品_%d 条视频", len(products), totalVideoCount),
		Content: fmt.Sprintf("成功抓取%d条商品数据，%d条详情数据，累计%d条视频分析数据，已导出%d个文件",
			len(products), len(productDetails), totalVideoCount, len(exportFiles)),
		ProductCount:   len(products),
		VideoCount:     totalVideoCount,
		Products:       products,
		ProductDetails: productDetails,
		VideoAnalysis:  videoAnalysisList,
		ExportFiles:    exportFiles,
	})

	logInfo(ctx, taskId, fmt.Sprintf("🎉 爬虫任务全部完成！商品：%d 条，详情：%d 条，视频分析：%d 条",
		len(products), len(productDetails), totalVideoCount))
	return res, nil
}
