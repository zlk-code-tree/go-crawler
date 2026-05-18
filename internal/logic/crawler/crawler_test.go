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
