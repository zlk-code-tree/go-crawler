package video

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/logic/crawler"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type Controller struct{}

var downloadManager = crawler.GetVideoDownloadManager()
var validVideoID = regexp.MustCompile(`^[0-9]{6,32}$`)

func New() *Controller { return &Controller{} }

func (c *Controller) DownloadTask(ctx context.Context, req *v1.VideoDownloadTaskReq) (res *v1.VideoDownloadTaskRes, err error) {
	r := g.RequestFromCtx(ctx)
	if !validVideoID.MatchString(req.VideoId) {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "video_id 格式不正确", "data": nil})
		return
	}
	status, queueErr := downloadManager.Enqueue(ctx, req.VideoId)
	if queueErr != nil {
		r.Response.WriteJsonExit(g.Map{"code": 1, "msg": queueErr.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "直链解析任务已接收", "data": g.Map{
		"video_id": req.VideoId, "download_status": status,
	}})
	return
}

func (c *Controller) DownloadTicket(ctx context.Context, req *v1.VideoDownloadTicketReq) (res *v1.VideoDownloadTicketRes, err error) {
	r := g.RequestFromCtx(ctx)
	if !validVideoID.MatchString(req.VideoId) {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "video_id 格式不正确", "data": nil})
		return
	}
	ticket, ticketErr := createDownloadTicket(ctx, req.VideoId)
	if ticketErr != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": ticketErr.Error(), "data": nil})
		return
	}
	downloadURL := fmt.Sprintf("/api/videos/%s/file-public?ticket=%s", url.PathEscape(req.VideoId), url.QueryEscape(ticket))
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": g.Map{
		"video_id": req.VideoId,
		"ticket":   ticket,
		"url":      downloadURL,
		"expires":  time.Now().Add(10 * time.Minute).Unix(),
	}})
	return
}

func (c *Controller) RetryFailed(ctx context.Context, req *v1.VideoDownloadRetryFailedReq) (res *v1.VideoDownloadRetryFailedRes, err error) {
	r := g.RequestFromCtx(ctx)
	result, retryErr := downloadManager.RetryFailed(ctx, strings.TrimSpace(req.TaskId), req.Limit)
	if retryErr != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": retryErr.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "失败视频直链解析已重新入队", "data": g.Map{
		"queued":    result.Queued,
		"completed": result.Completed,
		"skipped":   result.Skipped,
		"failed":    result.Failed,
	}})
	return
}

func (c *Controller) Import(ctx context.Context, req *v1.VideoImportReq) (res *v1.VideoImportRes, err error) {
	r := g.RequestFromCtx(ctx)
	candidate := strings.TrimSpace(req.VideoId)
	if candidate == "" {
		candidate = strings.TrimSpace(req.SourceURL)
	}
	videoID, normalizeErr := crawler.NormalizeVideoID(candidate)
	if normalizeErr != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": normalizeErr.Error(), "data": nil})
		return
	}
	sourceURL := strings.TrimSpace(req.SourceURL)
	if sourceURL == "" {
		sourceURL = "https://www.douyin.com/video/" + videoID
	}

	video := crawler.VideoAnalysis{
		ProductID:   strings.TrimSpace(req.ProductId),
		ProductName: strings.TrimSpace(req.ProductName),
		VideoId:     videoID,
		VideoDesc:   strings.TrimSpace(req.VideoDesc),
		SourceURL:   sourceURL,
	}
	_, ensureErr := downloadManager.EnsureVideoRecord(ctx, video, "manual")
	if ensureErr != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": ensureErr.Error(), "data": nil})
		return
	}

	status := "pending"
	shouldResolve := req.ResolveNow == nil || *req.ResolveNow
	if shouldResolve {
		status, ensureErr = downloadManager.Enqueue(ctx, videoID)
		if ensureErr != nil {
			r.Response.WriteJsonExit(g.Map{"code": 1, "msg": ensureErr.Error(), "data": nil})
			return
		}
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "视频已补录", "data": g.Map{
		"video_id": videoID, "source_url": sourceURL, "download_status": status,
	}})
	return
}

// List GET /api/videos
func (c *Controller) List(ctx context.Context, req *v1.VideoListReq) (res *v1.VideoListRes, err error) {
	r := g.RequestFromCtx(ctx)
	page, pageSize := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	m := g.DB().Model("video_analysis_data").Ctx(ctx)
	if req.TaskId != "" {
		m = m.Where("task_id", req.TaskId)
	}
	if req.ProductId != "" {
		m = m.Where("product_id", req.ProductId)
	}
	if req.DownloadStatus != "" {
		m = m.Where("download_status", req.DownloadStatus)
	}
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		m = m.Where("(video_desc LIKE ? OR product_name LIKE ? OR author_name LIKE ?)", like, like, like)
	}

	total, queryErr := m.Count()
	if queryErr != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": queryErr.Error(), "data": nil})
		return
	}
	rows, queryErr := m.OrderDesc("created_at").Page(page, pageSize).All()
	if queryErr != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": queryErr.Error(), "data": nil})
		return
	}

	items := make([]g.Map, 0, len(rows))
	for _, row := range rows {
		videoId := row["video_id"].String()
		items = append(items, g.Map{
			"id": row["id"].Int64(), "task_id": row["task_id"].String(),
			"product_id": row["product_id"].String(), "product_name": row["product_name"].String(),
			"video_id": videoId, "video_desc": row["video_desc"].String(),
			"author_name": row["author_name"].String(), "cover_url": row["cover_url"].String(),
			"publish_time": row["publish_time"].String(), "source_url": row["source_url"].String(),
			"payment_amount": row["payment_amount"].String(), "sales_volume": row["sales_volume"].String(),
			"expose_user_count": row["expose_user_count"].String(), "click_user_count": row["click_user_count"].String(),
			"convert_user_count": row["convert_user_count"].String(), "click_rate": row["click_rate"].String(),
			"click_convert_rate": row["click_convert_rate"].String(), "expose_convert_rate": row["expose_convert_rate"].String(),
			"order_count":     row["order_count"].String(),
			"download_status": row["download_status"].String(), "storage_path": row["storage_path"].String(),
			"download_url": row["download_url"].String(), "resolve_strategy": row["resolve_strategy"].String(),
			"last_error": row["last_error"].String(), "retry_count": row["retry_count"].Int(),
			"download_endpoint":      "/api/videos/" + url.PathEscape(videoId) + "/download",
			"download_task_endpoint": "/api/videos/" + url.PathEscape(videoId) + "/download-task",
			"created_at":             row["created_at"].String(),
		})
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": g.Map{
		"list": items, "total": total, "page": page, "page_size": pageSize,
	}})
	return
}

// DownloadLink GET /api/videos/:videoId/download-link
func (c *Controller) DownloadLink(ctx context.Context, req *v1.VideoDownloadLinkReq) (res *v1.VideoDownloadLinkRes, err error) {
	r := g.RequestFromCtx(ctx)

	videoId := req.VideoId
	if !validVideoID.MatchString(videoId) {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "video_id 格式不正确", "data": nil})
		return
	}

	downloadURL, strategy, err := downloadManager.ResolveNow(ctx, videoId)
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
			"strategy":     strategy,
		},
	})
	return
}

// DownloadProxy GET /api/videos/:videoId/download
func (c *Controller) DownloadProxy(ctx context.Context, req *v1.VideoDownloadProxyReq) (res *v1.VideoDownloadProxyRes, err error) {
	r := g.RequestFromCtx(ctx)

	videoId := req.VideoId
	if !validVideoID.MatchString(videoId) {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "video_id 格式不正确", "data": nil})
		return
	}
	downloadURL, strategy, err := downloadManager.ResolveNow(ctx, videoId)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "msg": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": g.Map{
		"video_id": videoId, "download_url": downloadURL, "strategy": strategy,
	}})
	return
}

// DownloadFile GET /api/videos/:videoId/file
// 将已解析的下载直链通过后端转发为附件下载，避免浏览器跨域时忽略 download 属性。
func (c *Controller) DownloadFile(ctx context.Context, req *v1.VideoDownloadFileReq) (res *v1.VideoDownloadFileRes, err error) {
	r := g.RequestFromCtx(ctx)
	videoId := req.VideoId
	if !validVideoID.MatchString(videoId) {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "video_id 格式不正确", "data": nil})
		return
	}

	writeVideoFile(ctx, r.Response, videoId)
	return
}

// DownloadFilePublic GET /api/videos/{videoId}/file-public?ticket=xxx
// 公开下载入口只接受短期签名票据，用于浏览器原生下载，避免前端 fetch 等完整 blob。
func (c *Controller) DownloadFilePublic(r *ghttp.Request) {
	videoId := r.Get("videoId").String()
	if !validVideoID.MatchString(videoId) {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "video_id 格式不正确", "data": nil})
		return
	}
	if err := verifyDownloadTicket(r.Context(), videoId, r.GetQuery("ticket").String()); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 401, "msg": err.Error(), "data": nil})
		return
	}
	writeVideoFile(r.Context(), r.Response, videoId)
}

func writeVideoFile(ctx context.Context, response *ghttp.Response, videoId string) {
	downloadURL, resolveErr := getResolvedDownloadURL(ctx, videoId)
	if resolveErr != nil {
		response.WriteJsonExit(g.Map{"code": 404, "msg": resolveErr.Error(), "data": nil})
		return
	}

	resp, fetchErr := fetchVideoStream(ctx, downloadURL)
	if fetchErr != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		downloadURL, resolveErr = forceResolveDownloadURL(ctx, videoId)
		if resolveErr != nil {
			response.WriteJsonExit(g.Map{"code": 404, "msg": resolveErr.Error(), "data": nil})
			return
		}
		resp, fetchErr = fetchVideoStream(ctx, downloadURL)
	}
	if fetchErr != nil {
		response.WriteJsonExit(g.Map{"code": 502, "msg": fetchErr.Error(), "data": nil})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		response.WriteJsonExit(g.Map{"code": resp.StatusCode, "msg": fmt.Sprintf("视频源返回 HTTP %d", resp.StatusCode), "data": nil})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}
	response.Header().Set("Content-Type", contentType)
	response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="douyin_%s.mp4"`, videoId))
	if length := resp.Header.Get("Content-Length"); length != "" {
		response.Header().Set("Content-Length", length)
	}
	response.WriteHeader(http.StatusOK)
	_, _ = io.Copy(response.Writer, resp.Body)
}

func getResolvedDownloadURL(ctx context.Context, videoId string) (string, error) {
	row, err := g.DB().Model("video_analysis_data").Ctx(ctx).
		Fields("download_status, download_url").
		Where("video_id", videoId).
		OrderDesc("created_at").
		One()
	if err != nil {
		return "", err
	}
	if !row.IsEmpty() && row["download_status"].String() == "resolved" && strings.TrimSpace(row["download_url"].String()) != "" {
		return row["download_url"].String(), nil
	}
	return forceResolveDownloadURL(ctx, videoId)
}

func forceResolveDownloadURL(ctx context.Context, videoId string) (string, error) {
	downloadURL, _, err := downloadManager.ResolveNow(ctx, videoId)
	return downloadURL, err
}

func createDownloadTicket(ctx context.Context, videoId string) (string, error) {
	expires := time.Now().Add(10 * time.Minute).Unix()
	expireText := strconv.FormatInt(expires, 10)
	signature, err := signDownloadTicket(ctx, videoId, expireText)
	if err != nil {
		return "", err
	}
	return expireText + "." + signature, nil
}

func verifyDownloadTicket(ctx context.Context, videoId, ticket string) error {
	parts := strings.Split(ticket, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("下载票据无效")
	}
	expires, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() > expires {
		return fmt.Errorf("下载票据已过期")
	}
	expected, err := signDownloadTicket(ctx, videoId, parts[0])
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return fmt.Errorf("下载票据校验失败")
	}
	return nil
}

func signDownloadTicket(ctx context.Context, videoId, expires string) (string, error) {
	secret, err := g.Cfg().Get(ctx, "jwt.secret", "go-crawler-jwt-secret-2024")
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(secret.String()))
	_, _ = mac.Write([]byte(videoId + "." + expires))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func fetchVideoStream(ctx context.Context, downloadURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1")
	req.Header.Set("Referer", "https://www.douyin.com/")
	req.Header.Set("Accept", "video/*,*/*;q=0.8")
	return http.DefaultClient.Do(req)
}
