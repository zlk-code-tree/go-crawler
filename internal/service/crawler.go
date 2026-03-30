package service

import (
	"context"

	v1 "go-crawler/api/hello/v1"
)

// ICrawler 定义接口（保留接口定义，但不再由 service 层实现）
type ICrawler interface {
	RunBrowserTask(ctx context.Context, req *v1.CrawlReq) ([]v1.Result, error)
}

// 注意：具体的爬虫实现在 internal/logic/crawler 中
// Controller 层应该直接调用 logic 层，而不是通过 service 层
// 这样可以避免循环依赖问题
