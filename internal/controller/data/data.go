package data

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gogf/gf/v2/frame/g"

	v1 "go-crawler/api/hello/v1"
)

type Controller struct{}

func New() *Controller { return &Controller{} }

// Overview GET /api/data/overview
func (c *Controller) Overview(ctx context.Context, req *v1.DataOverviewReq) (res *v1.DataOverviewRes, err error) {
	r := g.RequestFromCtx(ctx)
	taskTotal, _ := g.DB().Model("crawl_tasks").Count()
	taskRunning, _ := g.DB().Model("crawl_tasks").Where("status", "running").Count()
	taskCompleted, _ := g.DB().Model("crawl_tasks").Where("status", "completed").Count()
	rankingTotal, _ := g.DB().Model("tiktok_ranking_list").Count()
	videoTotal, _ := g.DB().Model("video_analysis").Count()
	trendRows, _ := g.DB().GetAll(ctx, `SELECT DATE(created_at) as date, COUNT(*) as count FROM crawl_tasks WHERE created_at >= DATE_SUB(CURDATE(), INTERVAL 7 DAY) GROUP BY DATE(created_at) ORDER BY date ASC`)
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "ok",
		"data": g.Map{
			"task_total":     taskTotal,
			"task_running":   taskRunning,
			"task_completed": taskCompleted,
			"ranking_total":  rankingTotal,
			"video_total":    videoTotal,
			"task_trend":     trendRows,
		},
	})
	return
}

// RankingList GET /api/data/ranking
func (c *Controller) RankingList(ctx context.Context, req *v1.DataRankingListReq) (res *v1.DataRankingListRes, err error) {
	r := g.RequestFromCtx(ctx)
	page := req.Page
	pageSize := req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	m := g.DB().Model("tiktok_ranking_list").OrderAsc("rank")
	if req.TaskId != "" {
		m = m.Where("task_id", req.TaskId)
	}
	if req.Keyword != "" {
		m = m.WhereLike("product_name", "%"+req.Keyword+"%")
	}
	if req.FirstList == "1" {
		m = m.Where("first_list", 1)
	}

	total, e := m.Count()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	list, e := m.Page(page, pageSize).All()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{
		"code":  0,
		"msg":   "ok",
		"data":  list,
		"total": total,
	})
	return
}

// RankingStats GET /api/data/ranking/stats
func (c *Controller) RankingStats(ctx context.Context, req *v1.DataRankingStatsReq) (res *v1.DataRankingStatsRes, err error) {
	r := g.RequestFromCtx(ctx)
	m := g.DB().Model("tiktok_ranking_list")
	if req.TaskId != "" {
		m = m.Where("task_id", req.TaskId)
	}
	total, _ := m.Count()
	firstListCount, _ := m.Clone().Where("first_list", 1).Count()
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "ok",
		"data": g.Map{
			"total":            total,
			"first_list_count": firstListCount,
		},
	})
	return
}

// VideoList GET /api/data/video
func (c *Controller) VideoList(ctx context.Context, req *v1.DataVideoListReq) (res *v1.DataVideoListRes, err error) {
	r := g.RequestFromCtx(ctx)
	page := req.Page
	pageSize := req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	m := g.DB().Model("video_analysis").OrderDesc("payment_amount")
	if req.TaskId != "" {
		m = m.Where("task_id", req.TaskId)
	}
	if req.ProductId != "" {
		m = m.Where("product_id", req.ProductId)
	}
	if req.Keyword != "" {
		m = m.WhereLike("product_name", "%"+req.Keyword+"%")
	}

	total, e := m.Count()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	list, e := m.Page(page, pageSize).All()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": g.Map{
		"list": list, "total": total, "page": page, "pageSize": pageSize,
	}})
	return
}

// VideoStats GET /api/data/video/stats
func (c *Controller) VideoStats(ctx context.Context, req *v1.DataVideoStatsReq) (res *v1.DataVideoStatsRes, err error) {
	r := g.RequestFromCtx(ctx)
	m := g.DB().Model("video_analysis")
	if req.TaskId != "" {
		m = m.Where("task_id", req.TaskId)
	}
	total, _ := m.Count()
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "ok",
		"data": g.Map{
			"total": total,
		},
	})
	return
}

// Export GET /api/data/export/{taskId}
func (c *Controller) Export(ctx context.Context, req *v1.DataExportReq) (res *v1.DataExportRes, err error) {
	r := g.RequestFromCtx(ctx)
	matches, e := filepath.Glob(fmt.Sprintf("exports/*%s*.csv", req.TaskId))
	if e != nil || len(matches) == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 404, "msg": "export file not found", "data": nil})
		return
	}
	filePath := matches[0]
	fileData, e := os.ReadFile(filePath)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "read file failed", "data": nil})
		return
	}
	r.Response.Header().Set("Content-Type", "text/csv; charset=utf-8")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(filePath)))
	r.Response.WriteHeader(http.StatusOK)
	r.Response.Write(fileData)
	return
}

// TaskIds GET /api/data/task_ids
func (c *Controller) TaskIds(ctx context.Context, req *v1.DataTaskIdsReq) (res *v1.DataTaskIdsRes, err error) {
	r := g.RequestFromCtx(ctx)
	rows, e := g.DB().GetAll(ctx, `SELECT DISTINCT r.task_id, t.start_time, t.status FROM tiktok_ranking_list r LEFT JOIN crawl_tasks t ON r.task_id = t.task_id ORDER BY t.start_time DESC LIMIT 50`)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "ok",
		"data": rows,
	})
	return
}
