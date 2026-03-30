package v1

import "github.com/gogf/gf/v2/frame/g"

// CrawlReq 请求参数
type CrawlReq struct {
	g.Meta    `path:"/crawler" method:"post" tags:"Crawler" summary:"启动浏览器爬虫"`
	TargetUrl string `json:"targetUrl" dc:"目标网站URL" v:"required"`
	Username  string `json:"username" dc:"模拟登录用户名" `
	Password  string `json:"password" dc:"模拟登录密码"`
}

// CrawlRes 响应结果
type CrawlRes struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Data    []Result `json:"data"`
}

// Result 单条抓取数据
type Result struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	URL     string `json:"url"`
}
