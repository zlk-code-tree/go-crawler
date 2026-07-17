package v1

import "github.com/gogf/gf/v2/frame/g"

// VideoDownloadLinkReq 获取视频下载地址请求
type VideoDownloadLinkReq struct {
	g.Meta  `path:"/videos/:videoId/download-link" tags:"Video" method:"get" summary:"获取视频下载地址"`
	VideoId string `json:"videoId" v:"required" dc:"视频 ID"`
}

// VideoDownloadLinkRes 获取视频下载地址响应
type VideoDownloadLinkRes struct {
	VideoId     string `json:"video_id" dc:"视频 ID"`
	DownloadURL string `json:"download_url" dc:"下载地址"`
	Strategy    string `json:"strategy" dc:"解析策略：mobile_http/desktop_browser/share_browser"`
}

// VideoDownloadProxyReq 代理下载视频请求
type VideoDownloadProxyReq struct {
	g.Meta  `path:"/videos/:videoId/download" tags:"Video" method:"get" summary:"获取视频下载直链"`
	VideoId string `json:"videoId" v:"required" dc:"视频 ID"`
}

// VideoDownloadProxyRes 获取视频下载直链响应
type VideoDownloadProxyRes struct{}

// VideoDownloadFileReq 下载视频文件请求
type VideoDownloadFileReq struct {
	g.Meta  `path:"/videos/:videoId/file" tags:"Video" method:"get" summary:"下载视频文件"`
	VideoId string `json:"videoId" v:"required" dc:"视频 ID"`
}

// VideoDownloadFileRes 下载视频文件响应
type VideoDownloadFileRes struct{}

// VideoDownloadTicketReq 创建视频下载票据请求
type VideoDownloadTicketReq struct {
	g.Meta  `path:"/videos/:videoId/download-ticket" tags:"Video" method:"post" summary:"创建视频下载票据"`
	VideoId string `json:"videoId" v:"required" dc:"视频 ID"`
}

// VideoDownloadTicketRes 创建视频下载票据响应
type VideoDownloadTicketRes struct{}

type VideoDownloadTaskReq struct {
	g.Meta  `path:"/videos/:videoId/download-task" tags:"Video" method:"post" summary:"创建视频直链解析任务"`
	VideoId string `json:"videoId" v:"required"`
}

type VideoDownloadTaskRes struct{}

type VideoDownloadRetryFailedReq struct {
	g.Meta `path:"/videos/download-retry-failed" tags:"Video" method:"post" summary:"批量重试失败的视频直链解析"`
	TaskId string `json:"task_id" dc:"可选；只重试指定爬虫任务的视频"`
	Limit  int    `json:"limit" d:"100" dc:"本次最多重试数量"`
}

type VideoDownloadRetryFailedRes struct{}

type VideoImportReq struct {
	g.Meta      `path:"/videos/import" tags:"Video" method:"post" summary:"手动补录已授权视频并解析直链"`
	VideoId     string `json:"video_id" dc:"视频 ID，可为空；为空时从 source_url 提取"`
	SourceURL   string `json:"source_url" dc:"已授权可访问的视频页链接"`
	ProductId   string `json:"product_id"`
	ProductName string `json:"product_name"`
	VideoDesc   string `json:"video_desc"`
	ResolveNow  *bool  `json:"resolve_now" dc:"是否立即排队解析下载直链；默认 true"`
}

type VideoImportRes struct{}

type VideoListReq struct {
	g.Meta         `path:"/videos" tags:"Video" method:"get" summary:"查询已发现的短视频"`
	Page           int    `json:"page" d:"1"`
	PageSize       int    `json:"page_size" d:"20"`
	TaskId         string `json:"task_id"`
	ProductId      string `json:"product_id"`
	Keyword        string `json:"keyword"`
	DownloadStatus string `json:"download_status"`
}

type VideoListRes struct{}
