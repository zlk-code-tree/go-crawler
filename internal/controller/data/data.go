package data

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type Controller struct{}

func New() *Controller { return &Controller{} }

// Overview GET /api/data/overview
func (c *Controller) Overview(r *ghttp.Request) {
	taskTotal, _ := g.DB().Model("crawl_tasks").Count()
	taskRunning, _ := g.DB().Model("crawl_tasks").Where("status", "running").Count()
	taskCompleted, _ := g.DB().Model("crawl_tasks").Where("status", "completed").Count()
	rankingTotal, _ := g.DB().Model("tiktok_ranking_list").Count()
	videoTotal, _ := g.DB().Model("video_analysis").Count()
	trendRows, _ := g.DB().GetAll(r.Context(), `SELECT DATE(created_at) as date, COUNT(*) as count FROM crawl_tasks WHERE created_at >= DATE_SUB(CURDATE(), INTERVAL 7 DAY) GROUP BY DATE(created_at) ORDER BY date ASC`)
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{
		"task_total": taskTotal, "task_running": taskRunning, "task_completed": taskCompleted,
		"ranking_total": rankingTotal, "video_total": videoTotal, "task_trend": trendRows,
	}})
}

// RankingList GET /api/data/ranking
func (c *Controller) RankingList(r *ghttp.Request) {
	page := r.GetQuery("page", 1).Int()
	pageSize := r.GetQuery("pageSize", 20).Int()
	taskId := r.GetQuery("task_id").String()
	keyword := r.GetQuery("keyword").String()
	firstList := r.GetQuery("first_list").String()
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	m := g.DB().Model("tiktok_ranking_list").OrderAsc("rank")
	if taskId != "" {
		m = m.Where("task_id", taskId)
	}
	if keyword != "" {
		m = m.WhereLike("product_name", "%"+keyword+"%")
	}
	if firstList == "1" {
		m = m.Where("first_list", 1)
	}

	total, err := m.Count()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	list, err := m.Page(page, pageSize).All()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{
		"list": list, "total": total, "page": page, "pageSize": pageSize,
	}})
}

// RankingStats GET /api/data/ranking/stats
func (c *Controller) RankingStats(r *ghttp.Request) {
	taskId := r.GetQuery("task_id").String()
	m := g.DB().Model("tiktok_ranking_list")
	if taskId != "" {
		m = m.Where("task_id", taskId)
	}
	total, _ := m.Count()
	firstListCount, _ := m.Clone().Where("first_list", 1).Count()
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{
		"total": total, "first_list_count": firstListCount,
	}})
}

// VideoList GET /api/data/video
func (c *Controller) VideoList(r *ghttp.Request) {
	page := r.GetQuery("page", 1).Int()
	pageSize := r.GetQuery("pageSize", 20).Int()
	taskId := r.GetQuery("task_id").String()
	productId := r.GetQuery("product_id").String()
	keyword := r.GetQuery("keyword").String()
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	m := g.DB().Model("video_analysis").OrderDesc("payment_amount")
	if taskId != "" {
		m = m.Where("task_id", taskId)
	}
	if productId != "" {
		m = m.Where("product_id", productId)
	}
	if keyword != "" {
		m = m.WhereLike("product_name", "%"+keyword+"%")
	}

	total, err := m.Count()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	list, err := m.Page(page, pageSize).All()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{
		"list": list, "total": total, "page": page, "pageSize": pageSize,
	}})
}

// VideoStats GET /api/data/video/stats
func (c *Controller) VideoStats(r *ghttp.Request) {
	taskId := r.GetQuery("task_id").String()
	m := g.DB().Model("video_analysis")
	if taskId != "" {
		m = m.Where("task_id", taskId)
	}
	total, _ := m.Count()
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{"total": total}})
}

// Export GET /api/data/export/{taskId}
func (c *Controller) Export(r *ghttp.Request) {
	taskId := r.GetRouter("taskId").String()
	if taskId == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "taskId required", "data": nil})
		return
	}
	matches, err := filepath.Glob(fmt.Sprintf("exports/*%s*.csv", taskId))
	if err != nil || len(matches) == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 404, "message": "export file not found", "data": nil})
		return
	}
	filePath := matches[0]
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": "read file failed", "data": nil})
		return
	}
	r.Response.Header().Set("Content-Type", "text/csv; charset=utf-8")
	r.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(filePath)))
	r.Response.WriteHeader(http.StatusOK)
	r.Response.Write(fileData)
}

// TaskIds GET /api/data/task_ids
func (c *Controller) TaskIds(r *ghttp.Request) {
	rows, err := g.DB().GetAll(r.Context(), `SELECT DISTINCT r.task_id, t.start_time, t.status FROM tiktok_ranking_list r LEFT JOIN crawl_tasks t ON r.task_id = t.task_id ORDER BY t.start_time DESC LIMIT 50`)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": rows})
}
