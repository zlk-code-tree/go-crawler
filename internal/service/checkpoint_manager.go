package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/gogf/gf/v2/os/glog"
)

// CrawlTask 爬取任务状态
type CrawlTask struct {
	TaskId              string    `json:"task_id"`
	TaskType            string    `json:"task_type,omitempty"`
	SourceTaskId        string    `json:"source_task_id,omitempty"` // 视频池恢复时固定使用原榜单快照
	Status              string    `json:"status"`                   // pending, running, paused, completed, failed
	CurrentStep         int       `json:"current_step"`
	CurrentPage         int       `json:"current_page"`
	TotalPages          int       `json:"total_pages"`
	ScrapedCount        int       `json:"scraped_count"`
	CurrentProductIndex int       `json:"current_product_index"`
	ProductCount        int       `json:"product_count"`
	VideoCount          int       `json:"video_count"`
	StopReason          string    `json:"stop_reason,omitempty"`
	StartTime           time.Time `json:"start_time"`
	EndTime             time.Time `json:"end_time"`
	Error               string    `json:"error,omitempty"`
}

// CheckpointManager 断点管理器
type CheckpointManager struct {
	checkpointDir string
}

var checkpointTaskIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func (cm *CheckpointManager) checkpointFilename(taskId string) (string, error) {
	if !checkpointTaskIDPattern.MatchString(taskId) {
		return "", fmt.Errorf("无效的任务 ID")
	}
	return filepath.Join(cm.checkpointDir, "task_"+taskId+".json"), nil
}

// NewCheckpointManager 创建断点管理器
func NewCheckpointManager(checkpointDir string) *CheckpointManager {
	checkpointDir = filepath.Join(checkpointDir)
	if err := os.MkdirAll(checkpointDir, 0755); err != nil {
		panic(fmt.Sprintf("创建断点目录失败：%v", err))
	}
	return &CheckpointManager{
		checkpointDir: checkpointDir,
	}
}

// SaveCheckpoint 保存任务进度
func (cm *CheckpointManager) SaveCheckpoint(ctx context.Context, task *CrawlTask) error {
	if task == nil {
		return fmt.Errorf("任务断点为空")
	}
	filename, err := cm.checkpointFilename(task.TaskId)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化任务失败：%w", err)
	}

	// 读者只能看到完整的旧断点或新断点，进程中断不会截断已有文件。
	file, err := os.CreateTemp(cm.checkpointDir, ".checkpoint-*")
	if err != nil {
		return fmt.Errorf("创建断点临时文件失败：%w", err)
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return fmt.Errorf("写入断点文件失败：%w", err)
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Rename(file.Name(), filename); err != nil {
		return fmt.Errorf("替换断点文件失败：%w", err)
	}

	glog.Info(ctx, fmt.Sprintf("[%s] 💾 保存任务进度：步骤=%d, 页码=%d/%d, 已抓取=%d 条",
		task.TaskId, task.CurrentStep, task.CurrentPage, task.TotalPages, task.ScrapedCount))
	return nil
}

// LoadCheckpoint 加载任务进度
func (cm *CheckpointManager) LoadCheckpoint(ctx context.Context, taskId string) (*CrawlTask, error) {
	filename, err := cm.checkpointFilename(taskId)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, nil // 没有断点
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取断点文件失败：%w", err)
	}

	var task CrawlTask
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("解析断点文件失败：%w", err)
	}

	// 检查任务是否可以恢复
	if task.Status == "completed" {
		glog.Info(ctx, fmt.Sprintf("[%s] ✅ 任务已完成，无需恢复", taskId))
		return nil, nil
	}

	if task.Status == "failed" {
		glog.Info(ctx, fmt.Sprintf("[%s] ⚠️ 任务之前失败：%s", taskId, task.Error))
	}

	glog.Info(ctx, fmt.Sprintf("[%s] 🔄 发现未完成的任务：步骤=%d, 页码=%d/%d, 已抓取=%d 条",
		taskId, task.CurrentStep, task.CurrentPage, task.TotalPages, task.ScrapedCount))

	return &task, nil
}

// FindLatestResumableCheckpoint 查找最近一个可恢复的断点任务。
func (cm *CheckpointManager) FindLatestResumableCheckpoint(ctx context.Context) (*CrawlTask, error) {
	return cm.findLatestResumableCheckpoint(ctx, "")
}

// FindLatestVideoPoolCheckpoint 不与手动完整采集任务共用恢复位置。
func (cm *CheckpointManager) FindLatestVideoPoolCheckpoint(ctx context.Context) (*CrawlTask, error) {
	return cm.findLatestResumableCheckpoint(ctx, string(TaskTypeVideoPool))
}

func (cm *CheckpointManager) findLatestResumableCheckpoint(ctx context.Context, taskType string) (*CrawlTask, error) {
	entries, err := os.ReadDir(cm.checkpointDir)
	if err != nil {
		return nil, err
	}

	var latest *CrawlTask
	var latestMod time.Time
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join(cm.checkpointDir, entry.Name()))
		if err != nil {
			continue
		}
		var task CrawlTask
		if err := json.Unmarshal(data, &task); err != nil {
			continue
		}
		if task.TaskId == "" || task.Status == "completed" || task.TaskType != taskType {
			continue
		}
		// 榜单任务可以在分页阶段恢复；视频池任务需要已建立视频断点。
		if taskType == string(TaskTypeVideoPool) && (task.CurrentStep < 8 || task.CurrentProductIndex < 0) {
			continue
		}
		if latest == nil || info.ModTime().After(latestMod) {
			copied := task
			latest = &copied
			latestMod = info.ModTime()
		}
	}
	if latest != nil {
		glog.Info(ctx, fmt.Sprintf("[%s] 🔄 找到最近可恢复的视频采集断点：商品索引=%d，视频=%d 条",
			latest.TaskId, latest.CurrentProductIndex, latest.VideoCount))
	}
	return latest, nil
}

// UpdateProgress 更新任务进度
func (cm *CheckpointManager) UpdateProgress(ctx context.Context, taskId string, step, page, scrapedCount int) error {
	task, err := cm.LoadCheckpoint(ctx, taskId)
	if err != nil {
		return err
	}

	if task == nil {
		task = &CrawlTask{
			TaskId:     taskId,
			Status:     "running",
			StartTime:  time.Now(),
			TotalPages: 20,
		}
	}

	task.CurrentStep = step
	task.CurrentPage = page
	task.ScrapedCount = scrapedCount
	task.Status = "running"

	return cm.SaveCheckpoint(ctx, task)
}

// UpdateVideoProgress 保存视频采集阶段断点。
func (cm *CheckpointManager) UpdateVideoProgress(ctx context.Context, taskId string, nextProductIndex, productCount, videoCount int, status, reason string) error {
	task, err := cm.LoadCheckpoint(ctx, taskId)
	if err != nil {
		return err
	}

	if task == nil {
		task = &CrawlTask{
			TaskId:     taskId,
			StartTime:  time.Now(),
			TotalPages: 20,
		}
	}
	if status == "" {
		status = "running"
	}
	task.CurrentStep = 8
	task.CurrentPage = 20
	task.TotalPages = 20
	task.ScrapedCount = productCount
	task.CurrentProductIndex = nextProductIndex
	task.ProductCount = productCount
	task.VideoCount = videoCount
	task.Status = status
	task.StopReason = reason
	task.Error = ""
	task.EndTime = time.Time{}
	if status == "paused" {
		task.EndTime = time.Now()
	}

	return cm.SaveCheckpoint(ctx, task)
}

// CompleteTask 标记任务完成
func (cm *CheckpointManager) CompleteTask(ctx context.Context, taskId string, scrapedCount int) error {
	task, err := cm.LoadCheckpoint(ctx, taskId)
	if err != nil {
		return err
	}

	if task == nil {
		task = &CrawlTask{
			TaskId: taskId,
		}
	}

	task.Status = "completed"
	task.ScrapedCount = scrapedCount
	task.EndTime = time.Now()

	// 保存完成后删除断点文件
	filename, err := cm.checkpointFilename(taskId)
	if err != nil {
		return err
	}
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除完成任务断点失败：%w", err)
	}

	glog.Info(ctx, fmt.Sprintf("[%s] ✅ 任务完成，共抓取 %d 条数据", taskId, scrapedCount))
	return nil
}

// FailTask 标记任务失败
func (cm *CheckpointManager) FailTask(ctx context.Context, taskId string, errorMsg string) error {
	task, err := cm.LoadCheckpoint(ctx, taskId)
	if err != nil {
		return err
	}

	if task == nil {
		task = &CrawlTask{
			TaskId: taskId,
		}
	}

	task.Status = "failed"
	task.Error = errorMsg
	task.EndTime = time.Now()

	return cm.SaveCheckpoint(ctx, task)
}

// CleanOldTasks 清理旧任务（超过指定时间）
func (cm *CheckpointManager) CleanOldTasks(ctx context.Context, maxAgeHours int) (int, error) {
	cleaned := 0
	threshold := time.Now().Add(-time.Duration(maxAgeHours) * time.Hour)

	entries, err := os.ReadDir(cm.checkpointDir)
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".json" {
			filePath := filepath.Join(cm.checkpointDir, entry.Name())
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(threshold) {
				if err := os.Remove(filePath); err != nil {
					glog.Warning(ctx, fmt.Sprintf("删除旧任务文件失败：%s", filePath))
					continue
				}
				cleaned++
				glog.Info(ctx, fmt.Sprintf("🗑️ 清理旧任务：%s", filePath))
			}
		}
	}

	return cleaned, nil
}
