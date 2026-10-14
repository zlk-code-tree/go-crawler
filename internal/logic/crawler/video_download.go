package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
	"github.com/gogf/gf/v2/errors/gerror"
)

// ===== 视频下载地址解析器 =====
// 三层降级策略：移动端 HTTP → 桌面浏览器主页面 → 桌面浏览器分享页
// 参考 Python 版本的 PublicVideoDownloadResolver

// VideoDownloader 视频下载地址解析器
type VideoDownloader struct {
	client    *http.Client
	cache     map[string]*cachedURL
	cacheLock sync.RWMutex
}

type cachedURL struct {
	URL       string
	ExpiredAt time.Time
}

// NewVideoDownloader 创建视频下载器
func NewVideoDownloader() *VideoDownloader {
	return &VideoDownloader{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		cache: make(map[string]*cachedURL),
	}
}

// Resolve 解析视频下载地址（三层降级策略）
func (d *VideoDownloader) Resolve(ctx context.Context, videoID string) (string, error) {
	url, _, err := d.ResolveWithStrategy(ctx, videoID)
	return url, err
}

// ResolveWithStrategy 解析视频地址并返回实际成功的策略。
func (d *VideoDownloader) ResolveWithStrategy(ctx context.Context, videoID string) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	// 0. 检查缓存
	if url := d.getCache(videoID); url != "" {
		return url, "cache", nil
	}

	var failures []string
	// 1. 移动端 HTTP（最快，成功率高）
	url, err := d.resolveByMobileHTTP(ctx, videoID)
	if ctx.Err() != nil {
		return "", "", ctx.Err()
	}
	if err == nil && url != "" {
		d.setCache(videoID, url)
		return url, "mobile_http", nil
	}
	if err != nil {
		failures = append(failures, "mobile_http: "+err.Error())
	}

	// 2. 桌面端浏览器（主页面）。该方法内部会收集多个候选并校验，避免把播放器占位 mp4 当作结果。
	url, err = d.resolveByDesktopBrowser(ctx, videoID, false)
	if ctx.Err() != nil {
		return "", "", ctx.Err()
	}
	if err == nil && url != "" {
		d.setCache(videoID, url)
		return url, "desktop_browser", nil
	}
	if err != nil {
		failures = append(failures, "desktop_browser: "+err.Error())
	}

	// 3. 桌面端浏览器（分享页兜底）。同样在方法内部完成候选校验。
	url, err = d.resolveByDesktopBrowser(ctx, videoID, true)
	if ctx.Err() != nil {
		return "", "", ctx.Err()
	}
	if err == nil && url != "" {
		d.setCache(videoID, url)
		return url, "share_browser", nil
	}
	if err != nil {
		failures = append(failures, "share_browser: "+err.Error())
	}

	return "", "", gerror.Newf("视频 %s 所有解析策略均失败: %s", videoID, strings.Join(failures, "; "))
}

// getCache 从缓存获取下载地址
func (d *VideoDownloader) getCache(videoID string) string {
	d.cacheLock.RLock()
	if entry, ok := d.cache[videoID]; ok {
		if time.Now().Before(entry.ExpiredAt) {
			d.cacheLock.RUnlock()
			return entry.URL
		}
	}
	d.cacheLock.RUnlock()

	d.cacheLock.Lock()
	if entry, ok := d.cache[videoID]; ok && !time.Now().Before(entry.ExpiredAt) {
		delete(d.cache, videoID)
	}
	d.cacheLock.Unlock()
	return ""
}

// setCache 设置缓存
func (d *VideoDownloader) setCache(videoID, url string) {
	d.cacheLock.Lock()
	defer d.cacheLock.Unlock()
	d.cache[videoID] = &cachedURL{
		URL:       url,
		ExpiredAt: time.Now().Add(20 * time.Minute),
	}
}

func (d *VideoDownloader) Invalidate(videoID string) {
	d.cacheLock.Lock()
	delete(d.cache, videoID)
	d.cacheLock.Unlock()
}

// ===== 策略 1: 移动端 HTTP 请求 =====

const mobileUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1"

// resolveByMobileHTTP 用移动端 UA 请求分享页，从 HTML 中提取 videoInfoRes JSON
func (d *VideoDownloader) resolveByMobileHTTP(ctx context.Context, videoID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	shareURL := fmt.Sprintf("https://www.iesdouyin.com/share/video/%s/", videoID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, shareURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", mobileUA)
	req.Header.Set("Referer", "https://www.douyin.com/")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return "", gerror.Newf("分享页返回 HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		return "", err
	}

	videoInfo := extractVideoInfoFromHTML(string(body))
	if videoInfo == nil {
		return "", gerror.New("no videoInfoRes found in HTML")
	}

	candidates := pickVideoURLCandidates(videoInfo)
	if len(candidates) == 0 {
		return "", gerror.New("no video URL found in videoInfoRes")
	}

	return d.finalizeDownloadURLCandidates(ctx, candidates)
}

func (d *VideoDownloader) finalizeDownloadURLCandidates(ctx context.Context, candidates []string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var failures []string
	seen := make(map[string]struct{}, len(candidates)*2)
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		finalURL, err := d.finalizeDownloadURL(ctx, candidate)
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if err == nil {
			return finalURL, nil
		}
		failures = append(failures, err.Error())
	}
	if len(failures) == 0 {
		return "", gerror.New("empty video URL candidates")
	}
	return "", gerror.New(strings.Join(failures, "; "))
}

func (d *VideoDownloader) finalizeDownloadURL(ctx context.Context, candidate string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return "", gerror.New("empty video URL")
	}

	candidates := make([]string, 0, 2)
	if noWatermark := removeWatermark(candidate); noWatermark != candidate {
		candidates = append(candidates, noWatermark)
	}
	candidates = append(candidates, candidate)

	var failures []string
	for _, item := range candidates {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		finalURL, err := d.verifyDownloadURL(ctx, item)
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if err == nil {
			return finalURL, nil
		}
		failures = append(failures, err.Error())
	}
	return "", gerror.New(strings.Join(failures, "; "))
}

func (d *VideoDownloader) verifyDownloadURL(ctx context.Context, candidate string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, candidate, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", mobileUA)
	req.Header.Set("Referer", "https://www.douyin.com/")
	req.Header.Set("Accept", "video/*,*/*;q=0.8")
	req.Header.Set("Range", "bytes=0-1023")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	prefix, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		return "", fmt.Errorf("读取视频流失败: %w", err)
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	finalURL := resp.Request.URL.String()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return "", gerror.Newf("候选下载地址不可用 HTTP %d content_type=%s url=%s", resp.StatusCode, contentType, limitURLForLog(finalURL))
	}
	if isKnownPlaceholderVideoURL(strings.ToLower(finalURL)) {
		return "", gerror.Newf("候选地址是播放器占位视频 content_type=%s url=%s", contentType, limitURLForLog(finalURL))
	}
	if isDownloadableVideoResponse(contentType, finalURL, prefix) {
		return finalURL, nil
	}
	return "", gerror.Newf("候选地址不是视频流 content_type=%s url=%s", contentType, limitURLForLog(finalURL))
}

func isDownloadableVideoResponse(contentType, finalURL string, prefix []byte) bool {
	lowerURL := strings.ToLower(finalURL)
	if isKnownPlaceholderVideoURL(lowerURL) {
		return false
	}
	if strings.HasPrefix(contentType, "video/") ||
		strings.Contains(contentType, "application/octet-stream") ||
		strings.Contains(lowerURL, "/video/tos/") ||
		strings.Contains(lowerURL, "mime_type=video") {
		return true
	}
	if len(prefix) >= 12 && string(prefix[4:8]) == "ftyp" {
		return true
	}
	return false
}

func isKnownPlaceholderVideoURL(lowerURL string) bool {
	return strings.Contains(lowerURL, "douyin-pc-web/uuu_265.mp4")
}

// extractVideoInfoFromHTML 用大括号匹配算法从 HTML 中提取 videoInfoRes JSON
func extractVideoInfoFromHTML(html string) map[string]interface{} {
	marker := "videoInfoRes"
	idx := strings.Index(html, marker)
	if idx < 0 {
		return extractRenderDataFromHTML(html)
	}

	// 从 marker 位置找到第一个 '{'
	start := strings.Index(html[idx:], "{")
	if start < 0 {
		return nil
	}
	start += idx

	// 大括号深度匹配，同时忽略 JSON 字符串内的括号。
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(html); i++ {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if html[i] == '\\' {
				escaped = true
				continue
			}
			if html[i] == '"' {
				inString = false
			}
			continue
		}
		if html[i] == '"' {
			inString = true
			continue
		}
		switch html[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(html[start:i+1]), &result); err != nil {
					return nil
				}
				return result
			}
		}
	}
	return nil
}

func extractRenderDataFromHTML(pageHTML string) map[string]interface{} {
	idx := strings.Index(pageHTML, `id="RENDER_DATA"`)
	if idx < 0 {
		idx = strings.Index(pageHTML, `id='RENDER_DATA'`)
	}
	if idx < 0 {
		return nil
	}
	start := strings.Index(pageHTML[idx:], ">")
	if start < 0 {
		return nil
	}
	start += idx + 1
	end := strings.Index(pageHTML[start:], "</script>")
	if end < 0 {
		return nil
	}
	raw := strings.TrimSpace(html.UnescapeString(pageHTML[start : start+end]))
	if decoded, err := url.QueryUnescape(raw); err == nil {
		raw = decoded
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil
	}
	return result
}

// pickBestVideoURL 从 videoInfoRes 中提取最佳视频 URL
// 优先级: play_addr_265 > play_addr > bit_rate[0] > download_addr
func pickBestVideoURL(info map[string]interface{}) string {
	candidates := pickVideoURLCandidates(info)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}

func pickVideoURLCandidates(info map[string]interface{}) []string {
	candidates := make([]string, 0)
	seen := make(map[string]struct{})
	appendCandidate := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}
	pickVideoURLCandidatesValue(info, 0, appendCandidate)
	return candidates
}

func pickVideoURLCandidatesValue(value interface{}, depth int, appendCandidate func(string)) {
	if depth > 20 {
		return
	}
	info, ok := value.(map[string]interface{})
	if !ok {
		if list, listOK := value.([]interface{}); listOK {
			for _, item := range list {
				pickVideoURLCandidatesValue(item, depth+1, appendCandidate)
			}
		}
		return
	}

	// 尝试 video_list
	if videoList, ok := info["video_list"].([]interface{}); ok && len(videoList) > 0 {
		if firstVideo, ok := videoList[0].(map[string]interface{}); ok {
			// play_addr_265
			for _, url := range extractPlayAddrCandidates(firstVideo, "play_addr_265") {
				appendCandidate(url)
			}
			// play_addr
			for _, url := range extractPlayAddrCandidates(firstVideo, "play_addr") {
				appendCandidate(url)
			}
			// bit_rate
			if bitRate, ok := firstVideo["bit_rate"].([]interface{}); ok && len(bitRate) > 0 {
				for _, bitRateItem := range bitRate {
					if br, ok := bitRateItem.(map[string]interface{}); ok {
						for _, url := range extractPlayAddrCandidates(br, "play_addr") {
							appendCandidate(url)
						}
					}
				}
			}
			// download_addr
			for _, url := range extractPlayAddrCandidates(firstVideo, "download_addr") {
				appendCandidate(url)
			}
		}
	}
	for _, key := range []string{"play_addr_265", "play_addr", "download_addr"} {
		for _, found := range extractPlayAddrCandidates(info, key) {
			appendCandidate(found)
		}
	}
	for _, child := range info {
		pickVideoURLCandidatesValue(child, depth+1, appendCandidate)
	}
}

// extractPlayAddr 从 play_addr 结构中提取 URL
func extractPlayAddr(parent map[string]interface{}, key string) string {
	candidates := extractPlayAddrCandidates(parent, key)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}

func extractPlayAddrCandidates(parent map[string]interface{}, key string) []string {
	addr, ok := parent[key].(map[string]interface{})
	if !ok {
		return nil
	}
	candidates := make([]string, 0, 4)
	seen := make(map[string]struct{})
	appendCandidate := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}
	if urlList, ok := addr["url_list"].([]interface{}); ok && len(urlList) > 0 {
		for _, item := range urlList {
			if url, ok := item.(string); ok {
				appendCandidate(url)
			}
		}
	}
	// 有些情况 url 直接在 addr 下
	if url, ok := addr["url"].(string); ok {
		appendCandidate(url)
	}
	return candidates
}

// ===== 策略 2: 桌面端浏览器 + 网络拦截 =====

// resolveByDesktopBrowser 用桌面浏览器打开视频页，拦截网络请求找视频 URL。
// 注意：页面播放器通常会先请求一个固定占位 mp4，它的 Content-Type 也是 video/mp4；
// 因此这里必须收集多个候选并逐个校验，而不能拿到第一个 media 请求就返回。
func (d *VideoDownloader) resolveByDesktopBrowser(ctx context.Context, videoID string, useShareURL bool) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var pageURL string
	if useShareURL {
		pageURL = fmt.Sprintf("https://www.iesdouyin.com/share/video/%s/", videoID)
	} else {
		pageURL = fmt.Sprintf("https://www.douyin.com/video/%s", videoID)
	}

	l := launcher.New().Context(ctx).Headless(true).NoSandbox(true)
	launcherOwned := true
	defer func() {
		if launcherOwned {
			cleanupVideoResolverLauncher(l)
		}
	}()
	launchURL, err := l.Launch()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", gerror.Wrap(err, "启动视频解析浏览器失败")
	}
	// Ownership is kept in the context so closeBrowser can clean up with a fresh
	// context even when this resolution has already been cancelled.
	browserCtx := context.WithValue(ctx, browserLifecycleKey{}, &browserLifecycle{launcher: l})
	browser := rod.New().Context(browserCtx).ControlURL(launchURL)
	if err := browser.Connect(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", gerror.Wrap(err, "连接视频解析浏览器失败")
	}
	launcherOwned = false
	defer closeBrowser(browser)

	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return "", gerror.Wrap(err, "创建视频解析页面失败")
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = page.Context(closeCtx).Close()
	}()

	// 设置请求拦截，查找视频 URL
	videoCh := make(chan string, 32)
	routerCtx, cancelRouter := context.WithCancel(ctx)
	defer cancelRouter()
	router := page.Context(routerCtx).HijackRequests()
	defer func() {
		cancelRouter()
		_ = router.Stop()
	}()
	if err := router.Add("*", "", func(h *rod.Hijack) {
		requestURL := h.Request.URL().String()
		lowerURL := strings.ToLower(requestURL)
		isMediaVideo := h.Request.Type() == proto.NetworkResourceTypeMedia &&
			!strings.Contains(lowerURL, ".mp3") && !strings.Contains(lowerURL, "audio")
		if (isVideoURL(requestURL) || isMediaVideo) && !isKnownPlaceholderVideoURL(lowerURL) {
			select {
			case <-routerCtx.Done():
			case videoCh <- requestURL:
			default:
			}
		}
		h.ContinueRequest(&proto.FetchContinueRequest{})
	}); err != nil {
		return "", gerror.Wrap(err, "注册请求拦截失败")
	}
	routerDone := make(chan struct{})
	go func() {
		defer close(routerDone)
		router.Run()
	}()
	defer func() {
		cancelRouter()
		<-routerDone
	}()

	navigateCtx, cancelNavigate := context.WithTimeout(ctx, 15*time.Second)
	err = page.Context(navigateCtx).Navigate(pageURL)
	cancelNavigate()
	if err != nil {
		return "", gerror.Wrap(err, "打开视频页失败")
	}
	loadCtx, cancelLoad := context.WithTimeout(ctx, 5*time.Second)
	_ = page.Context(loadCtx).WaitLoad()
	cancelLoad()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	_ = page.Mouse.Scroll(0, 480, 3)
	playCtx, cancelPlay := context.WithTimeout(ctx, 2*time.Second)
	if playButton, err := page.Context(playCtx).ElementX("//*[contains(text(), '播放') or contains(@aria-label, '播放')]"); err == nil {
		_ = playButton.Click(proto.InputMouseButtonLeft, 1)
	} else if ctx.Err() == nil {
		cancelPlay()
		playCtx, cancelPlay = context.WithTimeout(ctx, 2*time.Second)
		if video, err := page.Context(playCtx).Element("video"); err == nil {
			_ = video.Click(proto.InputMouseButtonLeft, 1)
		}
	}
	cancelPlay()
	if err := ctx.Err(); err != nil {
		return "", err
	}

	candidates, err := collectVideoCandidates(ctx, videoCh, 5*time.Second)
	if err != nil {
		return "", err
	}
	seen := make(map[string]struct{}, 16)
	for _, candidate := range candidates {
		seen[candidate] = struct{}{}
	}
	appendCandidate := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		lower := strings.ToLower(candidate)
		if isKnownPlaceholderVideoURL(lower) {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}

	// 从 DOM 的 <video> 标签提取
	domCtx, cancelDOM := context.WithTimeout(ctx, 5*time.Second)
	if video, err := page.Context(domCtx).Element("video"); err == nil {
		if src, err := video.Property("src"); err == nil && src.String() != "" {
			appendCandidate(src.String())
		}
		if sources, err := video.Elements("source"); err == nil {
			for _, source := range sources {
				if src, err := source.Property("src"); err == nil && src.String() != "" {
					appendCandidate(src.String())
				}
			}
		}
	}
	cancelDOM()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	// 补充从浏览器 Performance API 中可见的媒体请求。部分播放器请求会在
	// hijack 启动前或 ServiceWorker 内部触发，直接读 performance entries
	// 能捞回这些已发生的请求，仍然不重放罗盘内部接口。
	performanceCtx, cancelPerformance := context.WithTimeout(ctx, 3*time.Second)
	if result, err := page.Context(performanceCtx).Evaluate(rod.Eval(`
		() => performance.getEntriesByType('resource')
			.map(item => item.name)
			.filter(Boolean)
	`)); err == nil {
		for _, item := range result.Value.Arr() {
			appendCandidate(item.String())
		}
	}
	cancelPerformance()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	// 从页面 HTML 中的 RENDER_DATA / videoInfoRes 兜底提取播放地址。
	htmlCtx, cancelHTML := context.WithTimeout(ctx, 3*time.Second)
	if pageHTML, err := page.Context(htmlCtx).HTML(); err == nil {
		if videoInfo := extractVideoInfoFromHTML(pageHTML); videoInfo != nil {
			for _, candidate := range pickVideoURLCandidates(videoInfo) {
				appendCandidate(candidate)
			}
		}
	}
	cancelHTML()
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if len(candidates) == 0 {
		return "", gerror.New("no video URL found via desktop browser")
	}

	if finalURL, err := d.finalizeDownloadURLCandidates(ctx, candidates); err == nil {
		return finalURL, nil
	} else {
		return "", gerror.Wrapf(err, "no downloadable video URL found via desktop browser candidates=%d", len(candidates))
	}
}

// A launcher that never started a process has no exit event, so Cleanup would
// block forever. Its generated profile directory can still be removed safely.
func cleanupVideoResolverLauncher(l *launcher.Launcher) {
	if l.PID() != 0 {
		l.Kill()
		l.Cleanup()
		return
	}
	if profileDir := l.Get(flags.UserDataDir); profileDir != "" {
		_ = os.RemoveAll(profileDir)
	}
}

func collectVideoCandidates(ctx context.Context, videoCh <-chan string, duration time.Duration) ([]string, error) {
	candidates := make([]string, 0, 16)
	seen := make(map[string]struct{}, 16)
	timer := time.NewTimer(duration)
	defer timer.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return candidates, nil
		case candidate, ok := <-videoCh:
			if !ok {
				return candidates, nil
			}
			candidate = strings.TrimSpace(candidate)
			if candidate == "" || isKnownPlaceholderVideoURL(strings.ToLower(candidate)) {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			candidates = append(candidates, candidate)
		}
	}
}

// isVideoURL 判断 URL 是否是视频资源
func isVideoURL(url string) bool {
	lower := strings.ToLower(url)
	return (strings.Contains(lower, ".mp4") ||
		strings.Contains(lower, "/aweme/v1/play") ||
		strings.Contains(lower, "/video/tos/") ||
		strings.Contains(lower, "mime_type=video")) &&
		!strings.Contains(lower, ".js") &&
		!strings.Contains(lower, ".css")
}

// removeWatermark 只替换已知的 playwm 路径，不向带签名的 CDN URL 追加参数。
func removeWatermark(url string) string {
	return strings.Replace(url, "/playwm/", "/play/", 1)
}
