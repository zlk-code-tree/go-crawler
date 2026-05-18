package video

import (
	"context"
	"io"
	"net/http"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/logic/crawler"

	"github.com/gogf/gf/v2/frame/g"
)

type Controller struct{}

func New() *Controller { return &Controller{} }

// DownloadLink GET /api/videos/:videoId/download-link
func (c *Controller) DownloadLink(ctx context.Context, req *v1.VideoDownloadLinkReq) (res *v1.VideoDownloadLinkRes, err error) {
	r := g.RequestFromCtx(ctx)

	videoId := req.VideoId
	if videoId == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "video_id is required", "data": nil})
		return
	}

	downloader := crawler.NewVideoDownloader()
	downloadURL, err := downloader.Resolve(videoId)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "msg": err.Error(), "data": nil})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "ok",
		"data": g.Map{
			"video_id":     videoId,
			"download_url": downloadURL,
		},
	})
	return
}

// DownloadProxy GET /api/videos/:videoId/download
func (c *Controller) DownloadProxy(ctx context.Context, req *v1.VideoDownloadProxyReq) (res *v1.VideoDownloadProxyRes, err error) {
	r := g.RequestFromCtx(ctx)

	videoId := req.VideoId
	if videoId == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "video_id is required", "data": nil})
		return
	}

	downloader := crawler.NewVideoDownloader()
	downloadURL, err := downloader.Resolve(videoId)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "msg": err.Error(), "data": nil})
		return
	}

	// 代理请求视频流
	client := &http.Client{}
	proxyReq, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "create proxy request failed", "data": nil})
		return
	}
	proxyReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	proxyReq.Header.Set("Referer", "https://www.douyin.com/")

	proxyResp, err := client.Do(proxyReq)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 502, "msg": "proxy request failed", "data": nil})
		return
	}
	defer proxyResp.Body.Close()

	// 转发响应头
	r.Response.Header().Set("Content-Type", proxyResp.Header.Get("Content-Type"))
	if proxyResp.Header.Get("Content-Length") != "" {
		r.Response.Header().Set("Content-Length", proxyResp.Header.Get("Content-Length"))
	}
	r.Response.Header().Set("Content-Disposition", "attachment; filename=\""+videoId+".mp4\"")
	r.Response.WriteHeader(proxyResp.StatusCode)

	// 转发响应体
	_, _ = io.Copy(r.Response.Writer, proxyResp.Body)
	return
}
