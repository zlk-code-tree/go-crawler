package task

import (
	"os"
	"path/filepath"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/grand"

	"go-crawler/internal/service"
)

type Controller struct{}

func New() *Controller { return &Controller{} }

// List GET /api/task/list
func (c *Controller) List(r *ghttp.Request) {
	page := r.GetQuery("page", 1).Int()
	pageSize := r.GetQuery("pageSize", 20).Int()
	status := r.GetQuery("status").String()
	keyword := r.GetQuery("keyword").String()
	if page < 1 { page = 1 }
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	m := g.DB().Model("crawl_tasks").OrderDesc("created_at")
	if status != "" { m = m.Where("status", status) }
	if keyword != "" { m = m.WhereLike("task_id", "%"+keyword+"%") }

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

// Detail GET /api/task/detail/{id}
func (c *Controller) Detail(r *ghttp.Request) {
	taskId := r.GetRouter("id").String()
	if taskId == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "task_id required", "data": nil})
		return
	}
	row, err := g.DB().Model("crawl_tasks").Where("task_id", taskId).One()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	if row.IsEmpty() {
		r.Response.WriteJsonExit(g.Map{"code": 404, "message": "task not found", "data": nil})
		return
	}
	cm := service.NewCheckpointManager("checkpoints")
	checkpoint, _ := cm.LoadCheckpoint(r.Context(), taskId)
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{
		"task": row, "checkpoint": checkpoint,
	}})
}

// Create POST /api/task/create
func (c *Controller) Create(r *ghttp.Request) {
	var req struct {
		TargetUrl string `json:"target_url" v:"required#target_url required"`
		LoginType string `json:"login_type"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": err.Error(), "data": nil})
		return
	}
	if req.LoginType == "" { req.LoginType = "qrcode" }

	taskId := grand.S(16)
	_, err := g.DB().Model("crawl_tasks").Insert(g.Map{
		"task_id": taskId, "status": "pending",
		"target_url": req.TargetUrl, "login_type": req.LoginType,
		"start_time": time.Now(), "created_at": time.Now(),
	})
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "task created", "data": g.Map{"task_id": taskId}})
}

// Stop POST /api/task/stop/{id}
func (c *Controller) Stop(r *ghttp.Request) {
	taskId := r.GetRouter("id").String()
	if taskId == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "task_id required", "data": nil})
		return
	}
	_, err := g.DB().Model("crawl_tasks").
		Where("task_id", taskId).Where("status", "running").
		Update(g.Map{"status": "stopped", "end_time": time.Now()})
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
}

// Delete DELETE /api/task/delete/{id}
func (c *Controller) Delete(r *ghttp.Request) {
	taskId := r.GetRouter("id").String()
	if taskId == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "task_id required", "data": nil})
		return
	}
	_, err := g.DB().Model("crawl_tasks").Where("task_id", taskId).Delete()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	_ = os.Remove(filepath.Join("checkpoints", "task_"+taskId+".json"))
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
}
