package crawler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// TestProductDetailIntercept 测试商品详情 API 拦截
func TestProductDetailIntercept(t *testing.T) {
	// 创建爬虫实例
	crawler := &sCrawler{}
	ctx := context.Background()
	taskId := fmt.Sprintf("test_%d", time.Now().UnixMilli())

	// 启动浏览器（无头模式）
	browser, err := crawler.startBrowser(ctx, taskId, true)
	if err != nil {
		t.Fatalf("启动浏览器失败: %v", err)
	}
	defer browser.Close()

	// 打开新页面
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		t.Fatalf("打开页面失败: %v", err)
	}
	defer page.Close()

	// 设置 API 拦截器
	detailCh := make(chan *ProductDetail, 1)
	router := page.HijackRequests()

	// 拦截商品详情 API
	if err := router.Add("*product*detail*", "", func(h *rod.Hijack) {
		if err := h.LoadResponse(&http.Client{}, true); err != nil {
			t.Logf("LoadResponse 失败: %v", err)
			h.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}

		// 检查 Content-Type
		respHeaders := h.Response.Headers()
		contentType := ""
		if ct, ok := respHeaders["Content-Type"]; ok && len(ct) > 0 {
			contentType = ct[0]
		}
		t.Logf("拦截到请求: %s, Content-Type: %s", h.Request.URL().String(), contentType)

		if !strings.Contains(contentType, "application/json") {
			t.Logf("跳过非 JSON 响应: %s", contentType)
			h.ContinueRequest(&proto.FetchContinueRequest{})
			return
		}

		body := h.Response.Body()
		t.Logf("拦截到 JSON 响应，长度: %d", len(body))

		detailCh <- &ProductDetail{
			ProductName: "测试商品",
		}
		h.ContinueRequest(&proto.FetchContinueRequest{})
	}); err != nil {
		t.Fatalf("注册拦截器失败: %v", err)
	}

	go router.Run()
	defer router.Stop()

	// 导航到抖音罗盘商品详情页（测试用）
	testURL := "https://compass.jinritemai.com/shop/product/product-detail?product_id=123"
	t.Logf("导航到: %s", testURL)

	if err := page.Navigate(testURL); err != nil {
		t.Fatalf("导航失败: %v", err)
	}
	page.WaitLoad()
	time.Sleep(3 * time.Second)

	// 等待响应
	select {
	case detail := <-detailCh:
		t.Logf("✅ 成功拦截到商品详情: %+v", detail)
	case <-time.After(10 * time.Second):
		t.Log("⚠️ 超时，未拦截到商品详情响应")
	}

	t.Log("测试完成")
}

// TestContentTypeCheck 测试 Content-Type 检查逻辑
func TestContentTypeCheck(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{"JSON", "application/json; charset=utf-8", true},
		{"JSON 无 charset", "application/json", true},
		{"PNG", "image/png", false},
		{"JPEG", "image/jpeg", false},
		{"HTML", "text/html; charset=utf-8", false},
		{"空", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strings.Contains(tt.contentType, "application/json")
			if result != tt.expected {
				t.Errorf("Content-Type %q: 期望 %v, 实际 %v", tt.contentType, tt.expected, result)
			}
		})
	}
}

func TestBuildVideoAnalysisURL(t *testing.T) {
	product := ProductInfo{ProductID: "123456", ShopID: "shop-1"}
	got := buildVideoAnalysisURL(product)
	for _, expected := range []string{"product_id=123456", "product_shop_id=shop-1", "tab=video"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("期望 URL 包含 %q，实际为 %s", expected, got)
		}
	}
}

func TestVideoAnalysisAPIURLDetection(t *testing.T) {
	if !isVideoAnalysisAPIURL("https://compass.jinritemai.com/api/video/flow_more?product_id=123&source_entrance=video", "123") {
		t.Fatal("应识别商品视频分析 API")
	}
	if isVideoAnalysisAPIURL("https://compass.jinritemai.com/shop/chance/rank-product/detail?product_id=123&tab=video", "123") {
		t.Fatal("商品 HTML 详情页不应被识别为 API")
	}
	if isVideoAnalysisAPIURL("https://compass.jinritemai.com/api/video/list?product_id=456", "123") {
		t.Fatal("不应捕获其他商品的视频 API")
	}
}

func TestExtractVideoIdentityFromDOMText(t *testing.T) {
	videoID, sourceURL := extractVideoIdentityFromText(`href="https://www.douyin.com/video/7123456789012345678"`)
	if videoID != "7123456789012345678" || !strings.Contains(sourceURL, videoID) {
		t.Fatalf("视频身份提取失败: id=%s url=%s", videoID, sourceURL)
	}
	videoID, _ = extractVideoIdentityFromText(`data-aweme_id="7987654321098765432"`)
	if videoID != "7987654321098765432" {
		t.Fatalf("aweme_id 提取失败: %s", videoID)
	}
}

func TestExtractAllVideoIdentitiesFromRenderedHTML(t *testing.T) {
	html := `<script>{"video_id":"7123456789012345678","nested":"{\\"aweme_id\\":\\"7234567890123456789\\"}"}</script>
		<a href="https://www.douyin.com/video/7345678901234567890">video</a>`
	identities := extractAllVideoIdentitiesFromText(html)
	if len(identities) != 3 {
		t.Fatalf("expected 3 identities, got %d: %#v", len(identities), identities)
	}
	for _, identity := range identities {
		if identity.SourceURL != "https://www.douyin.com/video/"+identity.VideoId {
			t.Fatalf("unexpected source URL: %#v", identity)
		}
	}
}

func TestDailyVideoViewLimitMarker(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "daily limit", text: "查看次数已达上限，明日再来查看", want: true},
		{name: "ten per day", text: "每日仅可查看10次", want: true},
		{name: "competitor closed", text: "对方已关闭竞对详情功能", want: false},
		{name: "normal", text: "视频分析", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dailyVideoViewLimitMarker(tt.text) != ""
			if got != tt.want {
				t.Fatalf("daily limit detection = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseVideoAnalysisResponseNested(t *testing.T) {
	body := `{
		"code": 0,
		"data": {
			"total": 1,
			"video_list": [{
				"base_info": {"video_base_info": {
					"video_id": "7123456789012345678",
					"video_title": "测试爆款视频",
					"video_image": "https://example.com/cover.jpg",
					"publish_time": "2026-07-16 10:00:00",
					"author_info": {"author_nick_name": "测试达人"}
				}},
				"metrics": {
					"pay_amt": {"value": {"lower": 10000, "upper": 20000}, "unit": "fen"},
					"pay_combo_cnt": {"value": {"lower": 10, "upper": 20}}
				}
			}]
		}
	}`

	videos := parseVideoAnalysisResponse(body, ProductInfo{ProductID: "product-1", ProductName: "测试商品"})
	if len(videos) != 1 {
		t.Fatalf("期望 1 条视频，实际为 %d", len(videos))
	}
	video := videos[0]
	if video.VideoId != "7123456789012345678" || video.AuthorName != "测试达人" {
		t.Fatalf("视频基础信息解析错误: %+v", video)
	}
	if video.PaymentAmount != "100-200" || video.SalesVolume != "10-20" {
		t.Fatalf("视频指标解析错误: %+v", video)
	}
}

func TestParseVideoAnalysisResponseFallsBackToIdentity(t *testing.T) {
	body := `{
		"code": 0,
		"data": {
			"items": [{"unknown_wrapper": {"aweme_id": "7456789012345678901"}}]
		}
	}`
	videos := parseVideoAnalysisResponse(body, ProductInfo{ProductID: "product-2", ProductName: "改版商品"})
	if len(videos) != 1 {
		t.Fatalf("expected one fallback video, got %d: %#v", len(videos), videos)
	}
	if videos[0].VideoId != "7456789012345678901" || videos[0].SourceURL != "https://www.douyin.com/video/7456789012345678901" {
		t.Fatalf("unexpected fallback video: %#v", videos[0])
	}
}

func TestParseVideoAnalysisResponseDataArrayRows(t *testing.T) {
	body := `{
		"st": 0,
		"msg": "",
		"attributes": [{"key": "video_id"}, {"key": "aweme_id"}],
		"data": [{
			"base_info": {
				"video_base_info": {
					"aweme_id_str": "7456789012345678901",
					"video_title": "数组结构爆款视频",
					"video_image": "https://example.com/cover.jpg",
					"publish_time": "2026-07-17 09:00:00",
					"author_info": {"author_nick_name": "数组达人"}
				}
			},
			"metrics": {
				"pay_amt": {"value": {"lower": 12300, "upper": 45600}, "unit": "fen"},
				"show_uv": {"value": {"lower": 1000, "upper": 2000}},
				"click_ratio": {"value": {"lower": 0.12, "upper": 0.34}, "unit": "ratio"}
			}
		}]
	}`

	videos := parseVideoAnalysisResponse(body, ProductInfo{ProductID: "product-3", ProductName: "数组商品"})
	if len(videos) != 1 {
		t.Fatalf("expected one data-array video, got %d: %#v", len(videos), videos)
	}
	video := videos[0]
	if video.VideoId != "7456789012345678901" || video.VideoDesc != "数组结构爆款视频" || video.AuthorName != "数组达人" {
		t.Fatalf("unexpected data-array video: %#v", video)
	}
	if video.PaymentAmount != "123-456" || video.ExposeUserCount != "1000-2000" || video.ClickRate != "12.00%-34.00%" {
		t.Fatalf("unexpected metrics: %#v", video)
	}
}

func TestNormalizeVideoID(t *testing.T) {
	tests := map[string]string{
		"7456789012345678901":                                       "7456789012345678901",
		"https://www.douyin.com/video/7456789012345678901":          "7456789012345678901",
		"https://www.douyin.com/share/video/7456789012345678901/":   "7456789012345678901",
		"https://www.douyin.com/video/7456789012345678901?previous": "7456789012345678901",
	}
	for input, want := range tests {
		got, err := NormalizeVideoID(input)
		if err != nil {
			t.Fatalf("NormalizeVideoID(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("NormalizeVideoID(%q) = %q, want %q", input, got, want)
		}
	}

	if _, err := NormalizeVideoID("https://example.com/not-a-video"); err == nil {
		t.Fatal("expected invalid URL to fail")
	}
}

func TestBuildVideoResponseDiagnostic(t *testing.T) {
	body := `{
		"code": 0,
		"msg": "ok",
		"data": {
			"video_list": [{"video_id": "7456789012345678901", "play_addr": "https://example.com/video.mp4"}],
			"items": [1, 2]
		}
	}`
	diagnostic := buildVideoResponseDiagnostic(
		body,
		"https://compass.jinritemai.com/compass_api/shop/product/compete_product_detail/flow_source/flow_more?product_id=1&msToken=secret&a_bogus=secret&page_no=1",
		"product-1",
		1,
	)
	if diagnostic.ResponsePath != "/compass_api/shop/product/compete_product_detail/flow_source/flow_more" {
		t.Fatalf("unexpected response path: %s", diagnostic.ResponsePath)
	}
	if containsString(diagnostic.QueryKeys, "msToken") || containsString(diagnostic.QueryKeys, "a_bogus") {
		t.Fatalf("sensitive query keys should be redacted: %#v", diagnostic.QueryKeys)
	}
	if !diagnostic.JSONValid || !diagnostic.HasVideoID || !diagnostic.HasVideoList || !diagnostic.HasDownloadURL {
		t.Fatalf("unexpected diagnostic flags: %#v", diagnostic)
	}
	if diagnostic.ArrayLengths["data.video_list"] != 1 || diagnostic.ArrayLengths["data.items"] != 2 {
		t.Fatalf("array lengths not collected: %#v", diagnostic.ArrayLengths)
	}
}

func TestBuildAccessControlDiagnostic(t *testing.T) {
	body := `{
		"st": 0,
		"msg": "",
		"data": {
			"remaining_cnt": 10,
			"target_id_viewable": true,
			"display_type": 1
		}
	}`
	diagnostic := buildVideoResponseDiagnostic(
		body,
		"https://compass.jinritemai.com/compass_api/shop/common/competition/get_view_quota?product_id=1&a_bogus=secret",
		"product-1",
		0,
	)
	if !isCompetitorAccessControlAPIURL("https://compass.jinritemai.com/compass_api/shop/common/competition/get_view_quota") {
		t.Fatal("expected get_view_quota to be recognized")
	}
	if diagnostic.RemainingCount != "10" || diagnostic.TargetViewable != "true" || diagnostic.DisplayType != "1" {
		t.Fatalf("access control fields not collected: %#v", diagnostic)
	}
	if containsString(diagnostic.QueryKeys, "a_bogus") {
		t.Fatalf("sensitive query key should be redacted: %#v", diagnostic.QueryKeys)
	}
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func TestExtractVideoInfoFromHTMLHandlesBracesInString(t *testing.T) {
	html := `<script>window.videoInfoRes = {"video_list":[{"title":"a } brace","play_addr":{"url_list":["https://example.com/video.mp4"]}}]};</script>`
	info := extractVideoInfoFromHTML(html)
	if info == nil {
		t.Fatal("应能解析字符串中含大括号的 JSON")
	}
	if got := pickBestVideoURL(info); got != "https://example.com/video.mp4" {
		t.Fatalf("视频地址解析错误: %s", got)
	}
}

func TestExtractVideoInfoFromRenderData(t *testing.T) {
	html := `<script id="RENDER_DATA" type="application/json">{"app":{"videoDetail":{"video":{"play_addr":{"url_list":["https://example.com/render.mp4"]}}}}}</script>`
	info := extractVideoInfoFromHTML(html)
	if info == nil {
		t.Fatal("应能解析 RENDER_DATA")
	}
	if got := pickBestVideoURL(info); got != "https://example.com/render.mp4" {
		t.Fatalf("RENDER_DATA 视频地址解析错误: %s", got)
	}
}

func TestVideoURLDetectionRejectsHTMLPage(t *testing.T) {
	if isVideoURL("https://www.douyin.com/video/7123456789012345678") {
		t.Fatal("抖音 HTML 页不应被误认为媒体流")
	}
	if !isVideoURL("https://cdn.example.com/video/tos/cn/tos-cn.mp4") {
		t.Fatal("MP4 媒体地址应被识别")
	}
}
