package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
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
func (d *VideoDownloader) Resolve(videoID string) (string, error) {
	// 0. 检查缓存
	if url := d.getCache(videoID); url != "" {
		return url, nil
	}

	// 1. 移动端 HTTP（最快，成功率高）
	url, err := d.resolveByMobileHTTP(videoID)
	if err == nil && url != "" {
		d.setCache(videoID, url)
		return url, nil
	}

	// 2. 桌面端浏览器（主页面）
	url, err = d.resolveByDesktopBrowser(videoID, false)
	if err == nil && url != "" {
		d.setCache(videoID, url)
		return url, nil
	}

	// 3. 桌面端浏览器（分享页兜底）
	url, err = d.resolveByDesktopBrowser(videoID, true)
	if err == nil && url != "" {
		d.setCache(videoID, url)
		return url, nil
	}

	return "", gerror.Newf("all strategies failed for video %s", videoID)
}

// getCache 从缓存获取下载地址
func (d *VideoDownloader) getCache(videoID string) string {
	d.cacheLock.RLock()
	defer d.cacheLock.RUnlock()
	if entry, ok := d.cache[videoID]; ok {
		if time.Now().Before(entry.ExpiredAt) {
			return entry.URL
		}
		delete(d.cache, videoID)
	}
	return ""
}

// setCache 设置缓存
func (d *VideoDownloader) setCache(videoID, url string) {
	d.cacheLock.Lock()
	defer d.cacheLock.Unlock()
	d.cache[videoID] = &cachedURL{
		URL:       url,
		ExpiredAt: time.Now().Add(2 * time.Hour),
	}
}

// ===== 策略 1: 移动端 HTTP 请求 =====

const mobileUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1"

// resolveByMobileHTTP 用移动端 UA 请求分享页，从 HTML 中提取 videoInfoRes JSON
func (d *VideoDownloader) resolveByMobileHTTP(videoID string) (string, error) {
	shareURL := fmt.Sprintf("https://www.iesdouyin.com/share/video/%s/", videoID)
	req, err := http.NewRequest("GET", shareURL, nil)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	videoInfo := extractVideoInfoFromHTML(string(body))
	if videoInfo == nil {
		return "", gerror.New("no videoInfoRes found in HTML")
	}

	url := pickBestVideoURL(videoInfo)
	if url == "" {
		return "", gerror.New("no video URL found in videoInfoRes")
	}

	return url, nil
}

// extractVideoInfoFromHTML 用大括号匹配算法从 HTML 中提取 videoInfoRes JSON
func extractVideoInfoFromHTML(html string) map[string]interface{} {
	marker := "videoInfoRes"
	idx := strings.Index(html, marker)
	if idx < 0 {
		return nil
	}

	// 从 marker 位置找到第一个 '{'
	start := strings.Index(html[idx:], "{")
	if start < 0 {
		return nil
	}
	start += idx

	// 大括号深度匹配
	depth := 0
	for i := start; i < len(html); i++ {
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

// pickBestVideoURL 从 videoInfoRes 中提取最佳视频 URL
// 优先级: play_addr_265 > play_addr > bit_rate[0] > download_addr
func pickBestVideoURL(info map[string]interface{}) string {
	// 尝试 video_list
	if videoList, ok := info["video_list"].([]interface{}); ok && len(videoList) > 0 {
		if firstVideo, ok := videoList[0].(map[string]interface{}); ok {
			// play_addr_265
			if url := extractPlayAddr(firstVideo, "play_addr_265"); url != "" {
				return url
			}
			// play_addr
			if url := extractPlayAddr(firstVideo, "play_addr"); url != "" {
				return url
			}
			// bit_rate
			if bitRate, ok := firstVideo["bit_rate"].([]interface{}); ok && len(bitRate) > 0 {
				if br, ok := bitRate[0].(map[string]interface{}); ok {
					if url := extractPlayAddr(br, "play_addr"); url != "" {
						return url
					}
				}
			}
			// download_addr
			if url := extractPlayAddr(firstVideo, "download_addr"); url != "" {
				return url
			}
		}
	}
	return ""
}

// extractPlayAddr 从 play_addr 结构中提取 URL
func extractPlayAddr(parent map[string]interface{}, key string) string {
	addr, ok := parent[key].(map[string]interface{})
	if !ok {
		return ""
	}
	if urlList, ok := addr["url_list"].([]interface{}); ok && len(urlList) > 0 {
		if url, ok := urlList[0].(string); ok {
			return url
		}
	}
	// 有些情况 url 直接在 addr 下
	if url, ok := addr["url"].(string); ok {
		return url
	}
	return ""
}

// ===== 策略 2: 桌面端浏览器 + 网络拦截 =====

// resolveByDesktopBrowser 用桌面浏览器打开视频页，拦截网络请求找视频 URL
func (d *VideoDownloader) resolveByDesktopBrowser(videoID string, useShareURL bool) (string, error) {
	var pageURL string
	if useShareURL {
		pageURL = fmt.Sprintf("https://www.iesdouyin.com/share/video/%s/", videoID)
	} else {
		pageURL = fmt.Sprintf("https://www.douyin.com/video/%s", videoID)
	}

	// 使用 rod 连接到现有浏览器实例
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	page := browser.MustPage(pageURL)
	defer page.Close()

	// 设置请求拦截，查找视频 URL
	var videoURL string
	router := page.HijackRequests()
	if err := router.Add("*", "", func(h *rod.Hijack) {
		url := h.Request.URL().String()
		if isVideoURL(url) {
			videoURL = url
			h.Response.SetBody([]byte(""))
			h.Response.SetHeader("Content-Type", "video/mp4")
		} else {
			h.ContinueRequest(&proto.FetchContinueRequest{})
		}
	}); err != nil {
		return "", gerror.Wrap(err, "注册请求拦截失败")
	}
	go router.Run()
	defer router.Stop()

	// 等待页面加载
	page.MustWaitLoad()
	time.Sleep(3 * time.Second)

	// 从 DOM 的 <video> 标签提取
	if videoURL == "" {
		if video, err := page.Timeout(5 * time.Second).Element("video"); err == nil {
			if src, err := video.Property("src"); err == nil && src.String() != "" {
				videoURL = src.String()
			}
			if videoURL == "" {
				if sources, err := video.Elements("source"); err == nil {
					for _, s := range sources {
						if src, err := s.Property("src"); err == nil && src.String() != "" {
							videoURL = src.String()
							break
						}
					}
				}
			}
		}
	}

	if videoURL == "" {
		return "", gerror.New("no video URL found via desktop browser")
	}

	return removeWatermark(videoURL), nil
}

// isVideoURL 判断 URL 是否是视频资源
func isVideoURL(url string) bool {
	lower := strings.ToLower(url)
	return (strings.Contains(lower, ".mp4") ||
		strings.Contains(lower, ".m3u8") ||
		strings.Contains(lower, "/aweme/v1/play") ||
		strings.Contains(lower, "video")) &&
		!strings.Contains(lower, ".js") &&
		!strings.Contains(lower, ".css")
}

// removeWatermark 去水印：/aweme/v1/playwm/ → /aweme/v1/play/
func removeWatermark(url string) string {
	url = strings.Replace(url, "/playwm/", "/play/", 1)
	if !strings.Contains(url, "download=1") {
		if strings.Contains(url, "?") {
			url += "&download=1"
		} else {
			url += "?download=1"
		}
	}
	return url
}
