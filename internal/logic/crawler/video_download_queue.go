package crawler

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

var ErrVideoManagerStopped = errors.New("视频解析服务未启动或正在关闭")

type videoURLResolver interface {
	ResolveWithStrategy(context.Context, string) (string, string, error)
	Invalidate(string)
}

type videoResolution struct {
	done     chan struct{}
	url      string
	strategy string
	err      error
	skipped  bool
}

type VideoDownloadStats struct {
	Workers       int
	QueueCapacity int
	Queued        int
	Running       int
	Completed     int64
	Failed        int64
	Retries       int64
	Elapsed       time.Duration
}

func (m *VideoDownloadManager) runningLocked() error {
	if !m.started || m.stopping || m.ctx.Err() != nil {
		return ErrVideoManagerStopped
	}
	return nil
}

// Start is called after database initialization, with the service lifetime.
// A stopped manager is not reused; a new service instance constructs a new one.
func (m *VideoDownloadManager) Start(ctx context.Context, options ...VideoDownloadOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	o := DefaultVideoDownloadOptions()
	if len(options) > 1 {
		return fmt.Errorf("只能提供一组视频解析配置")
	}
	if len(options) == 1 {
		o = options[0]
	}
	if err := o.validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopping {
		return ErrVideoManagerStopped
	}
	if m.started {
		if m.options != o {
			return fmt.Errorf("运行中的视频解析配置不能修改")
		}
		return m.runningLocked()
	}
	m.options = o
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.queue = make(chan string, o.QueueCapacity)
	m.slots = make(chan struct{}, o.Workers)
	if downloader, ok := m.downloader.(*VideoDownloader); ok {
		downloader.client.Timeout = o.HTTPTimeout
	}
	m.started = true
	m.wg.Add(o.Workers + 1)
	for i := 0; i < o.Workers; i++ {
		go m.worker()
	}
	go m.refill()
	return nil
}

// Shutdown stops admission before cancellation. The queue is deliberately not
// closed: concurrent callers can never send to a closed channel. Database jobs
// left queued are recovered by the next service process.
func (m *VideoDownloadManager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return nil
	}
	if !m.stopping {
		m.stopping = true
		m.cancel()
		go func() {
			m.wg.Wait()
			m.active.Range(func(key, _ any) bool { m.active.Delete(key); return true })
			close(m.done)
		}()
	}
	done := m.done
	m.mu.Unlock()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *VideoDownloadManager) Stats() VideoDownloadStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return VideoDownloadStats{
		Workers: m.options.Workers, QueueCapacity: m.options.QueueCapacity,
		Queued: len(m.queue), Running: len(m.slots), Completed: m.completed.Load(),
		Failed: m.failed.Load(), Retries: m.retries.Load(), Elapsed: time.Duration(m.elapsedNS.Load()),
	}
}

func (m *VideoDownloadManager) worker() {
	defer m.wg.Done()
	for {
		if m.ctx.Err() != nil {
			return
		}
		select {
		case <-m.ctx.Done():
			return
		case id := <-m.queue:
			func() {
				defer m.active.Delete(id)
				// The actual work uses the service context; callers only wait for it.
				_, _, err := m.resolveVideo(m.ctx, id, true)
				if err != nil && m.ctx.Err() == nil {
					g.Log().Warningf(m.ctx, "[VideoDownload] video_id=%s 解析未完成: %v", id, err)
				}
			}()
		}
	}
}

func (m *VideoDownloadManager) refill() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.options.RefillInterval)
	defer ticker.Stop()
	for {
		if _, err := m.ResumePending(m.ctx, m.options.QueueCapacity); err != nil && m.ctx.Err() == nil {
			g.Log().Warningf(m.ctx, "[VideoDownload] 补投待办解析失败: %v", err)
		}
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			s := m.Stats()
			if s.Queued > 0 || s.Running > 0 {
				g.Log().Infof(m.ctx, "[VideoDownload] queued=%d running=%d completed=%d failed=%d retries=%d elapsed=%s",
					s.Queued, s.Running, s.Completed, s.Failed, s.Retries, s.Elapsed)
			}
		}
	}
}

// ResolveNow preserves synchronous API behavior. Unique jobs take a permit
// before spawning work; duplicate callers share a result. A caller cancellation
// abandons its wait, not the shared job needed by other callers or the queue.
var errVideoAdmissionCanceled = errors.New("解析任务派发被取消")

func (m *VideoDownloadManager) ResolveNow(ctx context.Context, videoID string) (string, string, error) {
	return m.resolveVideo(ctx, videoID, false)
}

func (m *VideoDownloadManager) resolveVideo(ctx context.Context, videoID string, queued bool) (string, string, error) {
	id := strings.TrimSpace(videoID)
	if !numericVideoID.MatchString(id) {
		return "", "", fmt.Errorf("video_id 格式不正确")
	}
	for {
		if err := ctx.Err(); err != nil {
			return "", "", err
		}
		m.mu.Lock()
		if err := m.runningLocked(); err != nil {
			m.mu.Unlock()
			return "", "", err
		}
		serviceCtx, slots := m.ctx, m.slots
		if job := m.flights[id]; job != nil {
			m.mu.Unlock()
			url, strategy, err, skipped := waitVideoResolution(ctx, serviceCtx, job)
			if errors.Is(err, errVideoAdmissionCanceled) || (!queued && skipped) {
				continue
			}
			return url, strategy, err
		}
		// Reserve the video before waiting for a permit. Concurrent callers join
		// even when every execution slot is busy; no waiting goroutine is spawned.
		job := &videoResolution{done: make(chan struct{})}
		m.flights[id] = job
		m.wg.Add(1)
		m.mu.Unlock()
		var admissionErr error
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			admissionErr = ctx.Err()
		case <-serviceCtx.Done():
			admissionErr = ErrVideoManagerStopped
		}
		if admissionErr == nil {
			m.mu.Lock()
			admissionErr = m.runningLocked()
			if admissionErr == nil {
				admissionErr = ctx.Err()
			}
			m.mu.Unlock()
			if admissionErr != nil {
				<-slots
			}
		}
		if admissionErr != nil {
			m.mu.Lock()
			delete(m.flights, id)
			job.err = errVideoAdmissionCanceled
			close(job.done)
			m.mu.Unlock()
			m.wg.Done()
			// Other live callers retry admission; cancelling this caller cannot
			// cancel their execution, including a queued background task.
			return "", "", admissionErr
		}
		go m.runResolution(serviceCtx, id, job, queued)
		url, strategy, err, _ := waitVideoResolution(ctx, serviceCtx, job)
		return url, strategy, err
	}
}

func waitVideoResolution(ctx, serviceCtx context.Context, job *videoResolution) (string, string, error, bool) {
	select {
	case <-ctx.Done():
		return "", "", ctx.Err(), false
	case <-serviceCtx.Done():
		return "", "", ErrVideoManagerStopped, false
	case <-job.done:
		return job.url, job.strategy, job.err, job.skipped
	}
}

func (m *VideoDownloadManager) runResolution(serviceCtx context.Context, id string, job *videoResolution, queued bool) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(serviceCtx, m.options.TaskTimeout)
	defer cancel()
	var attempts int
	var persistenceFailed bool
	var terminalExisting bool
	// Final notification and resource release are independent of persistence.
	// Even a panic in error recording cannot strand callers or Shutdown.
	defer m.wg.Done()
	defer func() { <-m.slots }()
	defer func() {
		if recovered := recover(); recovered != nil {
			job.err = errors.Join(job.err, fmt.Errorf("视频解析收尾异常: %v", recovered))
			g.Log().Errorf(context.Background(), "[VideoDownload] video_id=%s 收尾异常: %v\n%s", id, recovered, debug.Stack())
		}
		m.mu.Lock()
		delete(m.flights, id)
		close(job.done)
		m.mu.Unlock()
	}()
	defer func() {
		if recovered := recover(); recovered != nil {
			job.err = fmt.Errorf("视频解析异常: %v", recovered)
			g.Log().Errorf(context.Background(), "[VideoDownload] video_id=%s 解析异常: %v\n%s", id, recovered, debug.Stack())
		}
		if job.err != nil && !terminalExisting {
			status, retries := "failed", attempts
			if serviceCtx.Err() != nil || persistenceFailed {
				status, retries = "pending", -1
			} else {
				m.failed.Add(1)
			}
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := m.store.mark(cleanupCtx, id, status, retries, job.err.Error()); err != nil {
				g.Log().Errorf(cleanupCtx, "[VideoDownload] video_id=%s 状态写入失败: %v", id, err)
			}
			cleanupCancel()
		}
		m.elapsedNS.Add(time.Since(started).Nanoseconds())
	}()
	if queued {
		record, found, err := m.store.lookup(ctx, id)
		if err != nil {
			job.err, persistenceFailed = fmt.Errorf("读取解析状态失败：%w", err), true
			return
		}
		// A synchronous call may have finished after this ID entered the queue.
		// Stale queue entries do not repeat terminal work or overwrite strategy.
		if found && record.Status == "resolved" && record.URL != "" {
			job.url, job.strategy, job.skipped = record.URL, record.Strategy, true
			return
		}
		if found && record.Status == "failed" {
			terminalExisting = true
			job.skipped = true
			job.err = fmt.Errorf("视频 %s 的解析已失败，等待显式重试", id)
			return
		}
	}
	for attempt := 1; attempt <= m.options.RetryAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			job.err = err
			return
		}
		if err := m.store.mark(ctx, id, "resolving", attempt-1, ""); err != nil {
			job.err, persistenceFailed = fmt.Errorf("写入解析状态失败: %w", err), true
			return
		}
		attempts = attempt
		url, strategy, err := m.downloader.ResolveWithStrategy(ctx, id)
		if err == nil && strings.TrimSpace(url) == "" {
			err = errors.New("解析未返回有效视频直链")
		}
		if err == nil {
			if err = ctx.Err(); err != nil {
				job.err = err
				return
			}
			if err = m.store.complete(ctx, id, url, strategy); err != nil {
				job.err, persistenceFailed = fmt.Errorf("保存解析结果失败: %w", err), true
				return
			}
			job.url, job.strategy, job.err = url, strategy, nil
			m.completed.Add(1)
			g.Log().Infof(ctx, "[VideoDownload] 直链解析成功 video_id=%s strategy=%s url_prefix=%s", id, strategy, limitURLForLog(url))
			return
		}
		job.err = err
		if ctx.Err() != nil {
			job.err = ctx.Err()
			return
		}
		m.downloader.Invalidate(id)
		if attempt < m.options.RetryAttempts {
			m.retries.Add(1)
			if err := waitForCrawler(ctx, time.Duration(attempt)*m.options.RetryDelay); err != nil {
				job.err = err
				return
			}
		}
	}
}
