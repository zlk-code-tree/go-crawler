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
	fmt.Printf("[Crawl Request Received] task_id=%s login_type=%s target_url=%s\n", req.TaskId, req.LoginType, req.TargetUrl)

	// 直接调用 Logic 层执行核心逻辑
	logic := crawler.New()

	// 转换请求参数
	crawlReq := &crawler.CrawlReq{
		TaskId:    req.TaskId,
		TargetUrl: req.TargetUrl,
		LoginType: req.LoginType,
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
			Title:        item.Title,
			Content:      item.Content,
			URL:          item.URL,
			ProductCount: item.ProductCount,
			VideoCount:   item.VideoCount,
			ExportFiles:  item.ExportFiles,
		}
		for _, product := range item.Products {
			apiData[i].Products = append(apiData[i].Products, v1.ProductInfo{
				Rank: product.Rank, ProductID: product.ProductID, ProductName: product.ProductName,
				ShopID: product.ShopID, ShopName: product.ShopName, PaymentAmount: product.PaymentAmount,
				ClickCount: product.ClickCount, SalesVolume: product.SalesVolume,
				DeliveryVideoNum: product.DeliveryVideoNum, ProductURL: product.ProductURL, FirstList: product.FirstList,
			})
		}
		for _, detail := range item.ProductDetails {
			apiData[i].ProductDetails = append(apiData[i].ProductDetails, v1.ProductDetail{
				ProductID: detail.ProductID, ProductName: detail.ProductName,
				FirstShelfTime: detail.FirstShelfTime, MinPrice: detail.MinPrice,
			})
		}
		for _, video := range item.VideoAnalysis {
			apiData[i].VideoAnalysis = append(apiData[i].VideoAnalysis, v1.VideoAnalysis{
				VideoID: video.VideoId, VideoDesc: video.VideoDesc, PaymentAmount: video.PaymentAmount,
				AuthorName: video.AuthorName, CoverURL: video.CoverURL,
				PublishTime: video.PublishTime, SourceURL: video.SourceURL,
				ExposeUserCount: video.ExposeUserCount, ClickUserCount: video.ClickUserCount,
				ConvertUserCount: video.ConvertUserCount, ClickRate: video.ClickRate,
				ClickConvertRate: video.ClickConvertRate, ExposeConvertRate: video.ExposeConvertRate,
				OrderCount: video.OrderCount, SalesVolume: video.SalesVolume,
			})
		}
	}

	fmt.Printf("[Crawl Success] Result: %+v\n", apiData)

	return &v1.CrawlRes{
		Success: true,
		Message: "抓取成功",
		Data:    apiData,
	}, nil
}
