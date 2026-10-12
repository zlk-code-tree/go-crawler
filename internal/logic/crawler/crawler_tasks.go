package crawler

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"

	"go-crawler/internal/service"
)

// RunProductRankTask 商品榜单抓取任务（独立任务，用于调度器）
func (s *sCrawler) RunProductRankTask(ctx context.Context) error {
	taskId := fmt.Sprintf("product_rank_%d", time.Now().UnixMilli())
	logInfo(ctx, taskId, "🚀 开始执行商品榜单抓取任务")

	statusMgr := service.GetCrawlerStatusManager()
	ctx, finish, err := statusMgr.AcquireTask(ctx, taskId, "product_rank")
	if err != nil {
		return err
	}
	defer finish()
	checkpointManager := service.NewCheckpointManager("checkpoints")
	if err := checkpointManager.SaveCheckpoint(ctx, &service.CrawlTask{
		TaskId: taskId, TaskType: string(service.TaskTypeProductRank), Status: "running",
		StartTime: time.Now(), TotalPages: 20,
	}); err != nil {
		return err
	}

	// 启动浏览器
	browser, err := s.startBrowser(ctx, taskId, false)
	if err != nil {
		return gerror.Wrap(err, "启动浏览器失败")
	}
	defer closeBrowser(browser)

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
	statusMgr.SetMessage(ctx, "抓取商品榜单中")

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
	products, err := s.collectInterceptedData(ctx, page, taskId, checkpointManager, 1, pageCh)
	if err != nil {
		return err
	}
	if len(products) == 0 {
		return gerror.New("未抓取到商品数据")
	}

	logInfo(ctx, taskId, fmt.Sprintf("✅ 商品榜单抓取完成，共 %d 条", len(products)))
	return checkpointManager.CompleteTask(ctx, taskId, len(products))
}

// RunVideoPoolTask 视频池抓取任务（独立任务，用于调度器）
func (s *sCrawler) RunVideoPoolTask(ctx context.Context) error {
	checkpointManager := service.NewCheckpointManager("checkpoints")
	checkpoint, err := checkpointManager.FindLatestVideoPoolCheckpoint(ctx)
	if err != nil {
		return fmt.Errorf("读取视频池断点失败：%w", err)
	}
	taskId := fmt.Sprintf("video_pool_%d", time.Now().UnixMilli())
	if checkpoint != nil {
		taskId = checkpoint.TaskId
	}
	statusMgr := service.GetCrawlerStatusManager()
	ctx, finish, err := statusMgr.AcquireTask(ctx, taskId, "video_pool")
	if err != nil {
		return err
	}
	defer finish()
	if checkpoint == nil {
		latest, err := g.DB().Model("tiktok_ranking_list").Ctx(ctx).
			Fields("task_id").WhereNot("task_id", "").OrderDesc("created_at").One()
		if err != nil {
			return fmt.Errorf("读取榜单快照失败：%w", err)
		}
		if latest.IsEmpty() {
			return gerror.New("未找到商品榜单，请先抓取榜单")
		}
		checkpoint = &service.CrawlTask{
			TaskId: taskId, TaskType: string(service.TaskTypeVideoPool), SourceTaskId: latest["task_id"].String(),
			Status: "running", CurrentStep: 8, StartTime: time.Now(),
		}
	}
	if checkpoint.SourceTaskId == "" {
		return gerror.New("视频池断点缺少原榜单快照，无法安全恢复")
	}
	products, err := s.loadProductsFromDB(ctx, checkpoint.SourceTaskId)
	if err != nil {
		return fmt.Errorf("读取原榜单快照失败：%w", err)
	}
	if len(products) == 0 {
		return gerror.New("原榜单快照已不存在，无法恢复视频采集")
	}
	if checkpoint.ProductCount > 0 && checkpoint.ProductCount != len(products) {
		return gerror.New("原榜单商品数量已变化，暂停恢复以避免错过商品")
	}
	checkpoint.ProductCount = len(products)
	if err := checkpointManager.SaveCheckpoint(ctx, checkpoint); err != nil {
		return err
	}
	logInfo(ctx, taskId, fmt.Sprintf("📋 恢复榜单 %s 的视频采集：%d/%d", checkpoint.SourceTaskId, checkpoint.CurrentProductIndex, len(products)))
	browser, err := s.startBrowser(ctx, taskId, false)
	if err != nil {
		return err
	}
	defer closeBrowser(browser)
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return err
	}
	cookieManager := service.NewCookieManager("cookies")
	if !cookieManager.HasCompassCookies(ctx, taskId) || !s.injectCookies(ctx, page, cookieManager, taskId) {
		return gerror.New("没有可用 Cookie，请先登录")
	}
	if err := page.Navigate("https://compass.jinritemai.com/shop"); err != nil {
		return err
	}
	if err := page.Timeout(30 * time.Second).WaitLoad(); err != nil {
		return err
	}
	if !s.checkIsLoggedIn(ctx, page, taskId) {
		return gerror.New("Cookie 已过期，请重新登录")
	}
	statusMgr.SetMessage(ctx, "抓取视频数据中")
	result := s.scrapeVideoAnalysisDOMWithSinkFrom(ctx, browser, products, taskId,
		checkpoint.CurrentProductIndex, checkpointManager, func(videos []VideoAnalysis) error {
			return s.persistAndQueueVideos(ctx, taskId, videos)
		})
	if result.Err != nil {
		return result.Err
	}
	if result.Paused {
		logWarn(ctx, taskId, fmt.Sprintf("视频池暂停：%s，下次从商品索引 %d 继续", result.StopReason, result.NextProductIndex))
		return &service.TaskPausedError{Reason: result.StopReason}
	}
	if !result.Completed {
		return gerror.New("视频采集尚未完成")
	}
	if err := checkpointManager.CompleteTask(ctx, taskId, len(products)); err != nil {
		return err
	}
	logInfo(ctx, taskId, fmt.Sprintf("✅ 视频池抓取完成，本轮 %d 条视频", len(result.Videos)))
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
	ctx, finish, err := statusMgr.AcquireTask(ctx, taskId, "video_diagnostic")
	if err != nil {
		return nil, err
	}
	defer finish()
	statusMgr.SetMessage(ctx, "诊断单商品视频链路")

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
	defer closeBrowser(browser)
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
