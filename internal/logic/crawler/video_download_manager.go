package crawler

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

var numericVideoID = regexp.MustCompile(`^[0-9]{6,32}$`)
var douyinVideoURLID = regexp.MustCompile(`(?:douyin\.com/(?:share/)?video/|/video/)([0-9]{6,32})`)

// VideoDownloadManager owns a bounded queue and a shared limit for every
// resolution entry point. Constructing it does not start background goroutines.
type VideoDownloadManager struct {
	downloader   videoURLResolver
	store        videoTaskStore
	options      VideoDownloadOptions
	mu           sync.Mutex
	started      bool
	stopping     bool
	ctx          context.Context
	cancel       context.CancelFunc
	queue        chan string
	slots        chan struct{}
	done         chan struct{}
	wg           sync.WaitGroup
	active       sync.Map
	flights      map[string]*videoResolution
	refillMu     sync.Mutex
	refillCursor string
	completed    atomic.Int64
	failed       atomic.Int64
	retries      atomic.Int64
	elapsedNS    atomic.Int64
}

type VideoEnqueueResult struct {
	Queued    int
	Completed int
	Skipped   int
	Failed    int
	Pending   int
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
	return newVideoDownloadManager(downloader, sqlVideoTaskStore{})
}

func newVideoDownloadManager(downloader videoURLResolver, store videoTaskStore) *VideoDownloadManager {
	return &VideoDownloadManager{
		downloader: downloader, store: store, options: DefaultVideoDownloadOptions(),
		flights: make(map[string]*videoResolution), done: make(chan struct{}),
	}
}

func (m *VideoDownloadManager) Enqueue(ctx context.Context, videoID string) (string, error) {
	videoID = strings.TrimSpace(videoID)
	if !numericVideoID.MatchString(videoID) {
		return "", fmt.Errorf("video_id 格式不正确")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m.mu.Lock()
	err := m.runningLocked()
	m.mu.Unlock()
	if err != nil {
		return "", err
	}
	row, found, err := m.store.lookup(ctx, videoID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("视频 %s 尚未入库，请先执行视频发现", videoID)
	}
	if row.Status == "resolved" && row.URL != "" {
		return "resolved", nil
	}
	if _, loaded := m.active.LoadOrStore(videoID, struct{}{}); loaded {
		return "queued", nil
	}
	retained := false
	defer func() {
		if !retained {
			m.active.Delete(videoID)
		}
	}()
	if err := m.store.mark(ctx, videoID, "queued", -1, ""); err != nil {
		return "", err
	}
	m.mu.Lock()
	err = m.runningLocked()
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		select {
		case m.queue <- videoID:
			retained = true
		default:
		}
	}
	m.mu.Unlock()
	if retained {
		return "queued", nil
	}
	// Capacity pressure is scheduling state, not a failed video resolution.
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if markErr := m.store.mark(cleanupCtx, videoID, "pending", -1, "等待解析队列空位"); markErr != nil {
		return "", markErr
	}
	if err != nil {
		return "", err
	}
	return "pending", nil
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
		switch status {
		case "resolved":
			result.Completed++
		case "pending":
			result.Pending++
		default:
			result.Queued++
		}
	}
	return result
}

// ResumePending advances a keyset cursor, so active tasks at the front of the
// table cannot hide the rest of a backlog larger than the memory queue.
func (m *VideoDownloadManager) ResumePending(ctx context.Context, limit int) (VideoEnqueueResult, error) {
	m.refillMu.Lock()
	defer m.refillMu.Unlock()
	m.mu.Lock()
	err := m.runningLocked()
	capacity := m.options.QueueCapacity
	available := 0
	if err == nil {
		available = cap(m.queue) - len(m.queue)
	}
	m.mu.Unlock()
	if err != nil {
		return VideoEnqueueResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return VideoEnqueueResult{}, err
	}
	if available == 0 {
		return VideoEnqueueResult{}, nil
	}
	if limit <= 0 || limit > capacity {
		limit = capacity
	}
	ids, err := m.store.pending(ctx, m.refillCursor, limit)
	if err != nil {
		return VideoEnqueueResult{}, err
	}
	if len(ids) == 0 && m.refillCursor != "" {
		m.refillCursor = ""
		ids, err = m.store.pending(ctx, "", limit)
		if err != nil {
			return VideoEnqueueResult{}, err
		}
	}
	result := VideoEnqueueResult{}
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		status, err := m.Enqueue(ctx, id)
		m.refillCursor = id
		if err != nil {
			result.Failed++
			continue
		}
		switch status {
		case "resolved":
			result.Completed++
		case "pending":
			result.Pending++
			return result, nil
		default:
			result.Queued++
		}
	}
	return result, nil
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
			"download_status":     "discovered",
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
		updates["download_status"] = "discovered"
	}
	if len(updates) > 0 {
		_, err = g.DB().Model("video_analysis_data").Ctx(ctx).Where("id", row["id"].Int64()).Update(updates)
	}
	return videoID, err
}

// RetryFailed 将失败的视频直链解析任务重新放回队列。taskID 为空时重试最近的失败视频。
func (m *VideoDownloadManager) RetryFailed(ctx context.Context, taskID string, limit int) (VideoEnqueueResult, error) {
	m.mu.Lock()
	readyErr := m.runningLocked()
	capacity := m.options.QueueCapacity
	m.mu.Unlock()
	if readyErr != nil {
		return VideoEnqueueResult{}, readyErr
	}
	if limit <= 0 || limit > capacity {
		limit = capacity
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

func limitURLForLog(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) <= 80 {
		return raw
	}
	return raw[:80] + "..."
}
