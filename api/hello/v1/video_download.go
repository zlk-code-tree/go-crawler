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
	g.Meta  `path:"/videos/:videoId/download" tags:"Video" method:"get" summary:"代理下载视频"`
	VideoId string `json:"videoId" v:"required" dc:"视频 ID"`
}

// VideoDownloadProxyRes 代理下载视频响应
type VideoDownloadProxyRes struct{}
