package crawler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	netUrl "net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/gogf/gf/v2/frame/g"

	"go-crawler/internal/service"
)

func (s *sCrawler) scrapeVideoAnalysisDOMWithSinkFrom(
	ctx context.Context, browser *rod.Browser, products []ProductInfo, taskId string,
	startIndex int, checkpointManager *service.CheckpointManager,
	onDiscovered func([]VideoAnalysis) error,
) videoScrapeResult {
	collection := videoCollection{
		maxViews: g.Cfg().MustGet(ctx, "crawler.video.maxDetailViews", 10).Int(),
		maxEmpty: g.Cfg().MustGet(ctx, "crawler.video.maxEmptyStreak", 4).Int(),
		fetch: func(product ProductInfo) videoPageResult {
			logInfo(ctx, taskId, fmt.Sprintf("🎬 通过 UI 打开视频分析：%s", product.ProductName))
			return s.scrapeSingleProductVideosWithStatus(ctx, browser, product, taskId)
		},
		persist: onDiscovered,
		wait:    func(ctx context.Context) error { return waitForCrawler(ctx, 1200*time.Millisecond) },
	}
	if checkpointManager != nil {
		collection.checkpoint = func(nextIndex int, status, reason string) error {
			videoCount, err := s.countVideoRowsChecked(ctx, taskId)
			if err != nil {
				return fmt.Errorf("查询已保存视频数量失败：%w", err)
			}
			return checkpointManager.UpdateVideoProgress(ctx, taskId, nextIndex, len(products), videoCount, status, reason)
		}
	}
	return collection.run(ctx, products, startIndex)
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
	logInfo(ctx, taskId, fmt.Sprintf("🎬 视频已入库并安排解析下载直链：新排队=%d，待补投=%d，已解析=%d，跳过=%d，失败=%d",
		result.Queued, result.Pending, result.Completed, result.Skipped, result.Failed))
	return nil
}

func (s *sCrawler) scrapeSingleProductVideosWithStatus(ctx context.Context, browser *rod.Browser, product ProductInfo, taskId string) videoPageResult {
	result := videoPageResult{Status: videoPageAccessAvailable}

	// 重试打开页面
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		result.Err = fmt.Errorf("打开视频页面失败：%w", err)
		logWarn(ctx, taskId, result.Err.Error())
		return result
	}
	defer page.Close()

	// 配额绕过：通过 CDP Fetch 域劫持拦截 get_view_quota / get_display_rule
	bypassRouter := s.setupQuotaBypass(ctx, page, taskId)
	if bypassRouter != nil {
		defer bypassRouter.Stop()
	}

	// 详情页的视频标识通常只存在于浏览器原生 flow_more 响应中，
	// DOM 未必包含 video_id。这里只读取页面已经收到的响应，不重放接口。
	stopObserver, videoResponseCh := s.setupBrowserVideoResponseObserver(ctx, page, taskId, product)
	defer stopObserver()

	// 导航到视频分析页面（带重试）
	analysisURL := buildVideoAnalysisURL(product)
	if err := s.retryWithBackoff(ctx, func() error {
		return page.Navigate(analysisURL)
	}, 3); err != nil {
		result.Err = fmt.Errorf("导航视频分析失败：%w", err)
		logWarn(ctx, taskId, result.Err.Error())
		return result
	}

	if err := page.Timeout(30 * time.Second).WaitLoad(); err != nil {
		result.Err = err
		return result
	}
	if err := waitForCrawler(ctx, 2*time.Second); err != nil {
		result.Err = err
		return result
	}

	// URL 参数未触发 Tab 时，通过可见文本点击“视频分析”。
	if tab, err := page.Timeout(3 * time.Second).ElementX("//*[contains(text(), '视频分析')]"); err == nil {
		_ = tab.Click(proto.InputMouseButtonLeft, 1)
		if err := waitForCrawler(ctx, 2*time.Second); err != nil {
			result.Err = err
			return result
		}
	}
	if !s.waitForRiskControl(ctx, page, taskId) {
		result.Err = fmt.Errorf("页面验证尚未完成，保留当前商品位置")
		return result
	}

	// 优先使用正常页面触发后，Chrome 已经接收到的原生响应。
	// 有些页面会同时渲染“竞对详情受限”遮罩和已加载的部分数据，
	// 因此先消费浏览器已经收到的响应，避免把可见/已返回的数据误判为空。
	select {
	case response := <-videoResponseCh:
		result.Diagnostics = append(result.Diagnostics, response.Diagnostics...)
		if response.Status == videoPageAccessDailyLimit {
			s.logBrowserDeliveredDataProbe(ctx, page, taskId, product, response.Note)
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
		s.logBrowserDeliveredDataProbe(ctx, page, taskId, product, note)
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
		fillVideoMetricDefaults(&v, coreData)
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
		video := VideoAnalysis{
			VideoId: videoID, VideoDesc: desc, CoverURL: coverURL, SourceURL: sourceURL,
		}
		applyVisibleVideoRowMetrics(&video, row)
		videos = append(videos, video)
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

func fillVideoMetricDefaults(video *VideoAnalysis, fallback VideoAnalysis) {
	if video.PaymentAmount == "" {
		video.PaymentAmount = fallback.PaymentAmount
	}
	if video.ExposeUserCount == "" {
		video.ExposeUserCount = fallback.ExposeUserCount
	}
	if video.ClickUserCount == "" {
		video.ClickUserCount = fallback.ClickUserCount
	}
	if video.ConvertUserCount == "" {
		video.ConvertUserCount = fallback.ConvertUserCount
	}
	if video.ClickRate == "" {
		video.ClickRate = fallback.ClickRate
	}
	if video.ClickConvertRate == "" {
		video.ClickConvertRate = fallback.ClickConvertRate
	}
	if video.ExposeConvertRate == "" {
		video.ExposeConvertRate = fallback.ExposeConvertRate
	}
	if video.OrderCount == "" {
		video.OrderCount = fallback.OrderCount
	}
	if video.SalesVolume == "" {
		video.SalesVolume = fallback.SalesVolume
	}
}

func applyVisibleVideoRowMetrics(video *VideoAnalysis, row *rod.Element) {
	texts := collectVisibleCellTexts(row, "td")
	if len(texts) < 6 {
		texts = collectVisibleCellTexts(row, ".ant-table-cell")
	}
	if len(texts) < 8 {
		return
	}

	start := -1
	switch {
	case len(texts) >= 10:
		start = 4
	case len(texts) >= 9:
		start = 3
	}
	if start < 0 || len(texts) <= start+5 {
		return
	}

	video.PaymentAmount = firstNonEmptyString(video.PaymentAmount, texts[start])
	video.ExposeUserCount = firstNonEmptyString(video.ExposeUserCount, texts[start+1])
	video.ClickUserCount = firstNonEmptyString(video.ClickUserCount, texts[start+2])
	video.ClickRate = firstNonEmptyString(video.ClickRate, texts[start+3])
	video.ClickConvertRate = firstNonEmptyString(video.ClickConvertRate, texts[start+4])
	video.SalesVolume = firstNonEmptyString(video.SalesVolume, texts[start+5])
}

func collectVisibleCellTexts(row *rod.Element, selector string) []string {
	cells, err := row.Elements(selector)
	if err != nil {
		return nil
	}
	texts := make([]string, 0, len(cells))
	for _, cell := range cells {
		text, err := cell.Text()
		if err != nil {
			continue
		}
		text = strings.TrimSpace(strings.Join(strings.Fields(text), " "))
		if text == "" {
			continue
		}
		texts = append(texts, text)
	}
	return texts
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
				if diagnosticIndicatesDailyLimit(diagnostic) {
					select {
					case resultCh <- videoPageResult{
						Status: videoPageAccessDailyLimit,
						Note: fmt.Sprintf(
							"官方查看额度接口返回不可看：remaining_cnt=%s,target_viewable=%s",
							diagnostic.RemainingCount,
							diagnostic.TargetViewable,
						),
						Diagnostics: []VideoResponseDiagnostic{diagnostic},
					}:
					default:
					}
				}
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

func (s *sCrawler) logBrowserDeliveredDataProbe(ctx context.Context, page *rod.Page, taskId string, product ProductInfo, reason string) {
	result, err := page.Timeout(2*time.Second).Eval(`(productId) => {
		const hasVideoToken = (value) => /(?:video_id|aweme_id|aweme_id_str|item_id|item_id_str|play_addr|download_url)/.test(String(value || ''));
		const safeString = (value, max = 200000) => {
			const text = String(value || '');
			return text.length > max ? text.slice(0, max) : text;
		};
			const scanStorage = (storage) => {
				const summary = {
					key_count: 0,
					value_bytes: 0,
					video_key_count: 0,
					video_value_count: 0,
					product_value_count: 0,
					matches: []
				};
				if (!storage) return summary;
				for (let i = 0; i < storage.length; i++) {
					const key = storage.key(i) || '';
					let value = '';
				try {
					value = storage.getItem(key) || '';
				} catch (_) {
					value = '';
				}
				summary.key_count++;
					summary.value_bytes += value.length;
					const keyHasVideo = hasVideoToken(key);
					const valueHasVideo = hasVideoToken(value);
					const valueHasProduct = Boolean(productId && value.includes(String(productId)));
					if (keyHasVideo) summary.video_key_count++;
					if (valueHasVideo) summary.video_value_count++;
					if (valueHasProduct) summary.product_value_count++;
					if ((keyHasVideo || valueHasVideo || valueHasProduct) && summary.matches.length < 12) {
						summary.matches.push({
							key,
							value_length: value.length,
							key_has_video_token: keyHasVideo,
							value_has_video_token: valueHasVideo,
							value_has_product_id: valueHasProduct
						});
					}
				}
				return summary;
			};
			const sanitizeURL = (raw) => {
				try {
					const url = new URL(raw, location.href);
					const keys = Array.from(url.searchParams.keys())
						.filter((key) => !/token|bogus|fp|verify|csrf|sign|cookie|session|lid/i.test(key))
						.slice(0, 12)
						.sort();
					return {host: url.host, path: url.pathname, query_keys: keys};
				} catch (_) {
					return {host: '', path: String(raw || '').slice(0, 120), query_keys: []};
				}
			};
			const html = safeString(document.documentElement ? document.documentElement.innerHTML : '');
			const text = safeString(document.body ? document.body.innerText : '');
			const scripts = Array.from(document.scripts || []).map((script) => script.textContent || '').join('\n').slice(0, 300000);
			const resources = performance.getEntriesByType('resource').map((item) => item.name || '');
			const videoResources = resources.filter((url) => hasVideoToken(url) || /video|aweme|flow_more|play/.test(url));
		const productResources = resources.filter((url) => productId && url.includes(String(productId)));
		const report = {
			page_url: location.href,
			body_has_limit_text: /查看(?:次数|个数).*上限|今日查看|明日再来/.test(text),
			dom_has_video_token: hasVideoToken(html),
			text_has_video_token: hasVideoToken(text),
			script_has_video_token: hasVideoToken(scripts),
			dom_has_product_id: productId ? html.includes(String(productId)) : false,
			storage: {
				local: scanStorage(window.localStorage),
				session: scanStorage(window.sessionStorage)
			},
				performance: {
					resource_count: resources.length,
					video_like_resource_count: videoResources.length,
					product_resource_count: productResources.length,
					video_like_resources: videoResources.slice(-12).map(sanitizeURL),
					product_resources: productResources.slice(-12).map(sanitizeURL)
				}
			};
		return JSON.stringify(report);
	}`, product.ProductID)
	if err != nil || result == nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 浏览器已下发数据探针执行失败：product_id=%s reason=%s err=%v", product.ProductID, reason, err))
		return
	}
	logInfo(ctx, taskId, fmt.Sprintf(
		"🧪 浏览器已下发数据探针：product_id=%s reason=%s report=%s",
		product.ProductID,
		reason,
		result.Value.String(),
	))
}

func diagnosticIndicatesDailyLimit(diagnostic VideoResponseDiagnostic) bool {
	if !strings.Contains(diagnostic.ResponsePath, "/compass_api/shop/common/competition/get_view_quota") {
		return false
	}
	return strings.TrimSpace(diagnostic.RemainingCount) == "0" && stringValueMeansFalse(diagnostic.TargetViewable)
}

func stringValueMeansFalse(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "false", "0", "否", "no":
		return true
	default:
		return false
	}
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
		"查看个数已达上限",
		"今日查看次数已达上限",
		"今日查看个数已达上限",
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
		"查看个数已达上限",
		"今日查看次数已达上限",
		"今日查看个数已达上限",
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
	service.GetCrawlerStatusManager().SetMessage(ctx, "检测到环境验证，等待手动处理")
	logWarn(ctx, taskId, "⚠️ 检测到环境异常或验证页，已暂停自动操作，请在浏览器中手动处理")
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(3 * time.Second):
		}
		if !hasRisk() {
			service.GetCrawlerStatusManager().SetMessage(ctx, "环境验证已完成，继续采集")
			return true
		}
	}
	logError(ctx, taskId, "❌ 等待手动处理环境验证超时")
	return false
}
