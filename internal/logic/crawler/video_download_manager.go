package crawler

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

var numericVideoID = regexp.MustCompile(`^[0-9]{6,32}$`)
var douyinVideoURLID = regexp.MustCompile(`(?:douyin\.com/(?:share/)?video/|/video/)([0-9]{6,32})`)

// VideoDownloadManager 是爬虫流水线和 HTTP 接口共用的视频直链解析队列。
type VideoDownloadManager struct {
	downloader *VideoDownloader
	queue      chan string
	active     sync.Map
}

type VideoEnqueueResult struct {
	Queued    int
	Completed int
	Skipped   int
	Failed    int
}

func NormalizeVideoID(candidate string) (string, error) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return "", fmt.Errorf("video_id 或视频链接不能为空")
	}
	if numericVideoID.MatchString(candidate) {
		return candidate, nil
	}
	if match := douyinVideoURLID.FindStringSubmatch(candidate); len(match) > 1 {
		return match[1], nil
	}
	return "", fmt.Errorf("未识别到有效抖音 video_id")
}

var (
	defaultVideoDownloadManager     *VideoDownloadManager
	defaultVideoDownloadManagerOnce sync.Once
)

func GetVideoDownloadManager() *VideoDownloadManager {
	defaultVideoDownloadManagerOnce.Do(func() {
		defaultVideoDownloadManager = NewVideoDownloadManager(NewVideoDownloader())
	})
	return defaultVideoDownloadManager
}

func NewVideoDownloadManager(downloader *VideoDownloader) *VideoDownloadManager {
	if downloader == nil {
		downloader = NewVideoDownloader()
	}
	m := &VideoDownloadManager{
		downloader: downloader,
		queue:      make(chan string, 1000),
	}
	for i := 0; i < 2; i++ {
		go m.worker()
	}
	return m
}

func (m *VideoDownloadManager) Enqueue(ctx context.Context, videoID string) (string, error) {
	videoID = strings.TrimSpace(videoID)
	if !numericVideoID.MatchString(videoID) {
		return "", fmt.Errorf("video_id 格式不正确")
	}
	row, err := g.DB().Model("video_analysis_data").Ctx(ctx).Where("video_id", videoID).OrderDesc("created_at").One()
	if err != nil {
		return "", err
	}
	if row.IsEmpty() {
		return "", fmt.Errorf("视频 %s 尚未入库，请先执行视频发现", videoID)
	}
	if row["download_status"].String() == "resolved" && row["download_url"].String() != "" {
		return "resolved", nil
	}
	if _, loaded := m.active.LoadOrStore(videoID, struct{}{}); loaded {
		return "queued", nil
	}
	_, err = g.DB().Model("video_analysis_data").Ctx(ctx).Where("video_id", videoID).Update(g.Map{
		"download_status": "queued", "last_error": "",
	})
	if err != nil {
		m.active.Delete(videoID)
		return "", err
	}
	select {
	case m.queue <- videoID:
		return "queued", nil
	default:
		m.active.Delete(videoID)
		_, _ = g.DB().Model("video_analysis_data").Ctx(ctx).Where("video_id", videoID).Update(g.Map{
			"download_status": "failed", "last_error": "解析队列已满",
		})
		return "", fmt.Errorf("解析队列已满")
	}
}

func (m *VideoDownloadManager) EnsureVideoRecord(ctx context.Context, video VideoAnalysis, taskID string) (string, error) {
	videoID, err := NormalizeVideoID(video.VideoId)
	if err != nil {
		return "", err
	}
	video.VideoId = videoID
	if strings.TrimSpace(taskID) == "" {
		taskID = "manual"
	}
	if strings.TrimSpace(video.ProductName) == "" {
		video.ProductName = "手动补录视频"
	}
	if strings.TrimSpace(video.VideoDesc) == "" {
		video.VideoDesc = video.ProductName
	}
	if strings.TrimSpace(video.SourceURL) == "" {
		video.SourceURL = "https://www.douyin.com/video/" + videoID
	}

	row, err := g.DB().Model("video_analysis_data").Ctx(ctx).Where("video_id", videoID).OrderDesc("created_at").One()
	if err != nil {
		return "", err
	}
	if row.IsEmpty() {
		_, err = g.DB().Model("video_analysis_data").Ctx(ctx).Insert(g.Map{
			"task_id":             taskID,
			"product_id":          video.ProductID,
			"product_name":        video.ProductName,
			"video_id":            videoID,
			"video_desc":          video.VideoDesc,
			"author_name":         video.AuthorName,
			"cover_url":           video.CoverURL,
			"publish_time":        video.PublishTime,
			"source_url":          video.SourceURL,
			"download_status":     "pending",
			"download_url":        "",
			"resolve_strategy":    "",
			"payment_amount":      video.PaymentAmount,
			"expose_user_count":   video.ExposeUserCount,
			"click_user_count":    video.ClickUserCount,
			"convert_user_count":  video.ConvertUserCount,
			"click_rate":          video.ClickRate,
			"click_convert_rate":  video.ClickConvertRate,
			"expose_convert_rate": video.ExposeConvertRate,
			"order_count":         video.OrderCount,
			"sales_volume":        video.SalesVolume,
			"created_at":          time.Now(),
		})
		return videoID, err
	}

	updates := g.Map{}
	if strings.TrimSpace(row["source_url"].String()) == "" && video.SourceURL != "" {
		updates["source_url"] = video.SourceURL
	}
	if strings.TrimSpace(row["video_desc"].String()) == "" && video.VideoDesc != "" {
		updates["video_desc"] = video.VideoDesc
	}
	if strings.TrimSpace(row["product_name"].String()) == "" && video.ProductName != "" {
		updates["product_name"] = video.ProductName
	}
	if strings.TrimSpace(row["download_status"].String()) == "" {
		updates["download_status"] = "pending"
	}
	if len(updates) > 0 {
		_, err = g.DB().Model("video_analysis_data").Ctx(ctx).Where("id", row["id"].Int64()).Update(updates)
	}
	return videoID, err
}

func (m *VideoDownloadManager) EnqueueBatch(ctx context.Context, videoIDs []string) VideoEnqueueResult {
	result := VideoEnqueueResult{}
	seen := make(map[string]struct{}, len(videoIDs))
	for _, videoID := range videoIDs {
		if _, ok := seen[videoID]; ok {
			result.Skipped++
			continue
		}
		seen[videoID] = struct{}{}
		status, err := m.Enqueue(ctx, videoID)
		if err != nil {
			result.Failed++
			continue
		}
		if status == "resolved" {
			result.Completed++
		} else {
			result.Queued++
		}
	}
	return result
}

// ResumePending 恢复服务重启前已入库但未完成的直链解析。
func (m *VideoDownloadManager) ResumePending(ctx context.Context, limit int) (VideoEnqueueResult, error) {
	if limit <= 0 || limit > cap(m.queue) {
		limit = cap(m.queue)
	}
	rows, err := g.DB().Model("video_analysis_data").Ctx(ctx).
		Fields("video_id").
		WhereIn("download_status", []string{"pending", "queued", "resolving", "downloading"}).
		WhereNot("video_id", "").
		Group("video_id").Limit(limit).All()
	if err != nil {
		return VideoEnqueueResult{}, err
	}
	videoIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		videoIDs = append(videoIDs, row["video_id"].String())
	}
	return m.EnqueueBatch(ctx, videoIDs), nil
}

// RetryFailed 将失败的视频直链解析任务重新放回队列。taskID 为空时重试最近的失败视频。
func (m *VideoDownloadManager) RetryFailed(ctx context.Context, taskID string, limit int) (VideoEnqueueResult, error) {
	if limit <= 0 || limit > cap(m.queue) {
		limit = cap(m.queue)
	}
	query := g.DB().Model("video_analysis_data").Ctx(ctx).
		Fields("video_id, MAX(id) AS latest_id").
		Where("download_status", "failed").
		WhereNot("video_id", "")
	if strings.TrimSpace(taskID) != "" {
		query = query.Where("task_id", strings.TrimSpace(taskID))
	}
	rows, err := query.Group("video_id").OrderDesc("latest_id").Limit(limit).All()
	if err != nil {
		return VideoEnqueueResult{}, err
	}
	videoIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		videoIDs = append(videoIDs, row["video_id"].String())
	}
	return m.EnqueueBatch(ctx, videoIDs), nil
}

func (m *VideoDownloadManager) worker() {
	for videoID := range m.queue {
		m.resolveWithRetry(videoID)
		m.active.Delete(videoID)
	}
}

func (m *VideoDownloadManager) resolveWithRetry(videoID string) {
	ctx := context.Background()
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		_, _ = g.DB().Model("video_analysis_data").Ctx(ctx).Where("video_id", videoID).Update(g.Map{
			"download_status": "resolving", "retry_count": attempt - 1, "last_error": "",
		})
		if downloadURL, strategy, err := m.ResolveNow(ctx, videoID); err == nil {
			g.Log().Infof(ctx, "[VideoDownload] 直链解析成功 video_id=%s strategy=%s url_prefix=%s", videoID, strategy, limitURLForLog(downloadURL))
			return
		} else {
			lastErr = err
			g.Log().Warningf(ctx, "[VideoDownload] 直链解析失败 video_id=%s attempt=%d err=%v", videoID, attempt, err)
			m.downloader.Invalidate(videoID)
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	errorMessage := "未知下载错误"
	if lastErr != nil {
		errorMessage = lastErr.Error()
	}
	_, _ = g.DB().Model("video_analysis_data").Ctx(ctx).Where("video_id", videoID).Update(g.Map{
		"download_status": "failed", "retry_count": 3, "last_error": errorMessage,
	})
}

func limitURLForLog(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) <= 80 {
		return raw
	}
	return raw[:80] + "..."
}

func (m *VideoDownloadManager) ResolveNow(ctx context.Context, videoID string) (string, string, error) {
	videoID = strings.TrimSpace(videoID)
	if !numericVideoID.MatchString(videoID) {
		return "", "", fmt.Errorf("video_id 格式不正确")
	}
	downloadURL, strategy, err := m.downloader.ResolveWithStrategy(videoID)
	if err != nil {
		return "", "", err
	}
	_, err = g.DB().Model("video_analysis_data").Ctx(ctx).Where("video_id", videoID).Update(g.Map{
		"download_status":  "resolved",
		"download_url":     downloadURL,
		"resolve_strategy": strategy,
		"storage_path":     "",
		"file_size":        0,
		"last_error":       "",
	})
	if err != nil {
		return "", "", err
	}
	return downloadURL, strategy, nil
}
