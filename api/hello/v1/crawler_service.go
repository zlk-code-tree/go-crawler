package v1

import "github.com/gogf/gf/v2/frame/g"

// --- Crawler Service APIs (工作台爬虫服务管理) ---

type CrawlerServiceStatusReq struct {
	g.Meta `path:"/crawler-service/status" method:"get" tags:"CrawlerService" summary:"获取爬虫服务状态"`
}

type CrawlerServiceStatusRes struct {
	g.Meta    `mime:"application/json"`
	IsOnline  bool   `json:"isOnline" dc:"是否在线"`
	TaskId    string `json:"taskId" dc:"当前任务ID"`
	LoginType string `json:"loginType" dc:"登录方式"`
	StartTime string `json:"startTime" dc:"启动时间"`
	Message   string `json:"message" dc:"状态消息"`
}

type CrawlerServiceLoginReq struct {
	g.Meta    `path:"/crawler-service/login" method:"post" tags:"CrawlerService" summary:"登录并启动爬虫服务"`
	LoginType string `json:"login_type" d:"qrcode" dc:"登录方式：qrcode/password"`
	Username  string `json:"username" dc:"账号（password模式）"`
	Password  string `json:"password" dc:"密码（password模式）"`
}

type CrawlerServiceLoginRes struct {
	g.Meta `mime:"application/json"`
	TaskId string `json:"taskId" dc:"任务ID"`
}

type CrawlerServiceForceOfflineReq struct {
	g.Meta `path:"/crawler-service/force-offline" method:"post" tags:"CrawlerService" summary:"强制下线爬虫服务"`
}

type CrawlerServiceForceOfflineRes struct {
	g.Meta `mime:"application/json"`
}
