package task

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/grand"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/service"
)

type Controller struct{}

func New() *Controller { return &Controller{} }

// List GET /api/task/list
func (c *Controller) List(ctx context.Context, req *v1.TaskListReq) (res *v1.TaskListRes, err error) {
	page := req.Page
	pageSize := req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	m := g.DB().Model("crawl_tasks").OrderDesc("created_at")
	if req.Status != "" {
		m = m.Where("status", req.Status)
	}
	if req.Keyword != "" {
		m = m.WhereLike("task_id", "%"+req.Keyword+"%")
	}

	r := g.RequestFromCtx(ctx)
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

// Detail GET /api/task/detail/{id}
func (c *Controller) Detail(ctx context.Context, req *v1.TaskDetailReq) (res *v1.TaskDetailRes, err error) {
	r := g.RequestFromCtx(ctx)
	row, e := g.DB().Model("crawl_tasks").Where("task_id", req.Id).One()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	if row.IsEmpty() {
		r.Response.WriteJsonExit(g.Map{"code": 404, "msg": "task not found", "data": nil})
		return
	}
	cm := service.NewCheckpointManager("checkpoints")
	checkpoint, _ := cm.LoadCheckpoint(ctx, req.Id)
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": g.Map{
		"task": row, "checkpoint": checkpoint,
	}})
	return
}

// Create POST /api/task/create
func (c *Controller) Create(ctx context.Context, req *v1.TaskCreateReq) (res *v1.TaskCreateRes, err error) {
	r := g.RequestFromCtx(ctx)
	taskId := grand.S(16)
	_, e := g.DB().Model("crawl_tasks").Insert(g.Map{
		"task_id": taskId, "status": "pending",
		"target_url": req.TargetUrl, "login_type": req.LoginType,
		"start_time": time.Now(), "created_at": time.Now(),
	})
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "task created", "data": g.Map{"task_id": taskId}})
	return
}

// Stop POST /api/task/stop/{id}
func (c *Controller) Stop(ctx context.Context, req *v1.TaskStopReq) (res *v1.TaskStopRes, err error) {
	r := g.RequestFromCtx(ctx)
	_, e := g.DB().Model("crawl_tasks").
		Where("task_id", req.Id).Where("status", "running").
		Update(g.Map{"status": "stopped", "end_time": time.Now()})
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": nil})
	return
}

// Delete DELETE /api/task/delete/{id}
func (c *Controller) Delete(ctx context.Context, req *v1.TaskDeleteReq) (res *v1.TaskDeleteRes, err error) {
	r := g.RequestFromCtx(ctx)
	_, e := g.DB().Model("crawl_tasks").Where("task_id", req.Id).Delete()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	_ = os.Remove(filepath.Join("checkpoints", "task_"+req.Id+".json"))
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": nil})
	return
}
