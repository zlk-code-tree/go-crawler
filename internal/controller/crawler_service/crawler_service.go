package crawler_service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/grand"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/logic/crawler"
	"go-crawler/internal/service"
)

// Controller is the crawler service controller for workspace widget.
type Controller struct{}

// New returns a Controller.
func New() *Controller { return &Controller{} }

// CrawlerServiceStatus handles GET /api/crawler-service/status
func (c *Controller) CrawlerServiceStatus(ctx context.Context, req *v1.CrawlerServiceStatusReq) (res *v1.CrawlerServiceStatusRes, err error) {
	r := g.RequestFromCtx(ctx)
	mgr := service.GetCrawlerStatusManager()
	status := mgr.GetStatus()

	startTimeStr := ""
	if status.IsOnline {
		startTimeStr = status.StartTime.Format("2006-01-02 15:04:05")
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "ok",
		"data": g.Map{
			"isOnline":  status.IsOnline,
			"taskId":    status.TaskId,
			"loginType": status.LoginType,
			"startTime": startTimeStr,
			"message":   status.Message,
		},
	})
	return
}

// CrawlerServiceLogin handles POST /api/crawler-service/login
func (c *Controller) CrawlerServiceLogin(ctx context.Context, req *v1.CrawlerServiceLoginReq) (res *v1.CrawlerServiceLoginRes, err error) {
	r := g.RequestFromCtx(ctx)
	mgr := service.GetCrawlerStatusManager()

	loginType := req.LoginType
	if loginType == "" {
		loginType = "qrcode"
	}

	if loginType == "password" && (req.Username == "" || req.Password == "") {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "账号密码登录模式需要提供用户名和密码", "data": nil})
		return
	}

	taskId := req.ResumeTaskId
	checkpointManager := service.NewCheckpointManager("checkpoints")
	resumed := false
	if taskId == "" {
		if checkpoint, checkpointErr := checkpointManager.FindLatestResumableCheckpoint(ctx); checkpointErr == nil && checkpoint != nil {
			taskId = checkpoint.TaskId
			resumed = true
		}
	} else if checkpoint, checkpointErr := checkpointManager.LoadCheckpoint(ctx, taskId); checkpointErr != nil || checkpoint == nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "指定的恢复任务不存在或不可恢复", "data": nil})
		return
	} else {
		resumed = true
	}

	newTask := taskId == ""
	if newTask {
		taskId = grand.S(16)
	}
	// Reserve before changing persistent task state or starting the goroutine.
	taskCtx, finish, acquireErr := mgr.AcquireTask(context.Background(), taskId, loginType)
	if acquireErr != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": acquireErr.Error(), "data": nil})
		return
	}
	started := false
	defer func() {
		if !started {
			finish()
		}
	}()

	if newTask {
		_, dbErr := g.DB().Model("crawl_tasks").Ctx(ctx).Insert(g.Map{
			"task_id":    taskId,
			"status":     "running",
			"login_type": loginType,
			"start_time": time.Now(),
			"created_at": time.Now(),
		})
		if dbErr != nil {
			r.Response.WriteJsonExit(g.Map{"code": 500, "msg": fmt.Sprintf("创建任务失败: %v", dbErr), "data": nil})
			return
		}
	} else {
		_, dbErr := g.DB().Model("crawl_tasks").Ctx(ctx).Where("task_id", taskId).Update(g.Map{
			"status":     "running",
			"login_type": loginType,
			"start_time": time.Now(),
			"end_time":   nil,
		})
		if dbErr != nil {
			r.Response.WriteJsonExit(g.Map{"code": 500, "msg": fmt.Sprintf("恢复任务失败: %v", dbErr), "data": nil})
			return
		}
	}

	mgr.SetMessage(taskCtx, "登录中...")

	go func() {
		defer finish()

		logic := crawler.New()
		crawlReq := &crawler.CrawlReq{
			TaskId:    taskId,
			LoginType: loginType,
			Username:  req.Username,
			Password:  req.Password,
		}

		mgr.SetMessage(taskCtx, "爬虫运行中")

		results, runErr := logic.RunBrowserTask(taskCtx, crawlReq)
		// Cancellation must not prevent recording the final stopped state.
		terminalCtx, terminalCancel := context.WithTimeout(context.WithoutCancel(taskCtx), 10*time.Second)
		defer terminalCancel()
		if runErr != nil || taskCtx.Err() != nil {
			status := "failed"
			if taskCtx.Err() != nil {
				status = "stopped"
			}
			g.Log().Infof(terminalCtx, "[CrawlerService] task %s %s: %v", taskId, status, runErr)
			_, _ = g.DB().Model("crawl_tasks").Ctx(terminalCtx).Where("task_id", taskId).Update(g.Map{
				"status":   status,
				"end_time": time.Now(),
			})
			return
		}

		status := "completed"
		var exportFiles map[string]string
		if len(results) > 0 {
			exportFiles = results[0].ExportFiles
			if results[0].Paused {
				status = "paused"
			}
		}
		productCount, _ := g.DB().Model("tiktok_ranking_list").Ctx(terminalCtx).Where("task_id", taskId).Count()
		videoCount, _ := g.DB().Model("video_analysis_data").Ctx(terminalCtx).Where("task_id", taskId).Count()
		updateData := g.Map{
			"status":        status,
			"product_count": productCount,
			"video_count":   videoCount,
			"end_time":      time.Now(),
		}
		if len(exportFiles) > 0 {
			if encoded, err := json.Marshal(exportFiles); err == nil {
				updateData["export_files"] = string(encoded)
			}
		}
		_, _ = g.DB().Model("crawl_tasks").Ctx(terminalCtx).Where("task_id", taskId).Update(updateData)
		g.Log().Infof(taskCtx, "[CrawlerService] task %s %s", taskId, status)
	}()
	started = true

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "爬虫服务已启动",
		"data": g.Map{"taskId": taskId, "resumed": resumed},
	})
	return
}

// CrawlerServiceForceOffline handles POST /api/crawler-service/force-offline
func (c *Controller) CrawlerServiceForceOffline(ctx context.Context, req *v1.CrawlerServiceForceOfflineReq) (res *v1.CrawlerServiceForceOfflineRes, err error) {
	r := g.RequestFromCtx(ctx)
	mgr := service.GetCrawlerStatusManager()
	isRunning := mgr.IsOnline()

	if !isRunning {
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "爬虫服务未在运行", "data": nil})
		return
	}

	mgr.ForceOffline()

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "已请求停止，正在清理浏览器",
		"data": nil,
	})
	return
}

func (c *Controller) CrawlerVideoDiagnostic(ctx context.Context, req *v1.CrawlerVideoDiagnosticReq) (res *v1.CrawlerVideoDiagnosticRes, err error) {
	r := g.RequestFromCtx(ctx)
	result, runErr := crawler.New().RunVideoDiagnostic(ctx, req.ProductId)
	if runErr != nil {
		r.Response.WriteJsonExit(g.Map{"code": 1, "msg": runErr.Error(), "data": nil})
		return
	}
	videos := make([]g.Map, 0, len(result.Videos))
	for _, video := range result.Videos {
		videos = append(videos, g.Map{
			"video_id": video.VideoId, "video_desc": video.VideoDesc,
			"author_name": video.AuthorName, "cover_url": video.CoverURL,
			"publish_time": video.PublishTime, "source_url": video.SourceURL,
			"payment_amount": video.PaymentAmount, "sales_volume": video.SalesVolume,
		})
	}
	diagnostics := make([]g.Map, 0, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		diagnostics = append(diagnostics, g.Map{
			"product_id":         diagnostic.ProductID,
			"response_path":      diagnostic.ResponsePath,
			"query_keys":         diagnostic.QueryKeys,
			"body_size":          diagnostic.BodySize,
			"json_valid":         diagnostic.JSONValid,
			"business_code":      diagnostic.BusinessCode,
			"message":            diagnostic.Message,
			"top_level_keys":     diagnostic.TopLevelKeys,
			"data_keys":          diagnostic.DataKeys,
			"array_lengths":      diagnostic.ArrayLengths,
			"has_video_id":       diagnostic.HasVideoID,
			"has_aweme_id":       diagnostic.HasAwemeID,
			"has_video_list":     diagnostic.HasVideoList,
			"has_download_url":   diagnostic.HasDownloadURL,
			"restriction_note":   diagnostic.RestrictionNote,
			"parsed_video_count": diagnostic.ParsedVideoCount,
			"remaining_count":    diagnostic.RemainingCount,
			"target_viewable":    diagnostic.TargetViewable,
			"display_type":       diagnostic.DisplayType,
		})
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": g.Map{
		"task_id": result.TaskId, "product_id": result.Product.ProductID,
		"product_name": result.Product.ProductName, "analysis_url": result.AnalysisURL,
		"video_count": len(videos), "videos": videos, "diagnostics": diagnostics,
	}})
	return
}
