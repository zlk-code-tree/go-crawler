package crawler

import (
	"context"
	"fmt"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/logic/crawler"
)

// ControllerV1 是 V1 版本的控制器
type ControllerV1 struct{}

func New() *ControllerV1 {
	return &ControllerV1{}
}

// Crawl 处理爬虫请求
func (c *ControllerV1) Crawl(ctx context.Context, req *v1.CrawlReq) (res *v1.CrawlRes, err error) {
	fmt.Printf("[Crawl Request Received] Params: %+v\n", req)

	// 直接调用 Logic 层执行核心逻辑
	logic := crawler.New()
	
	// 转换请求参数
	crawlReq := &crawler.CrawlReq{
		TargetUrl: req.TargetUrl,
		LoginType: "qrcode", // 默认扫码登录
		Username:  req.Username,
		Password:  req.Password,
	}

	logicData, err := logic.RunBrowserTask(ctx, crawlReq)
	if err != nil {
		fmt.Printf("[Crawl Error] %v\n", err)
		return &v1.CrawlRes{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// 转换结果类型：logic.Result -> v1.Result
	apiData := make([]v1.Result, len(logicData))
	for i, item := range logicData {
		apiData[i] = v1.Result{
			Title:   item.Title,
			Content: item.Content,
			URL:     item.URL,
		}
	}

	fmt.Printf("[Crawl Success] Result: %+v\n", apiData)

	return &v1.CrawlRes{
		Success: true,
		Message: "抓取成功",
		Data:    apiData,
	}, nil
}
