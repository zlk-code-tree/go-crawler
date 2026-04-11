package permission

import (
	"context"
	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
)

type Controller struct {
	permissionSvc *service.PermissionService
}

func New() *Controller {
	return &Controller{
		permissionSvc: service.NewPermissionService(),
	}
}

// List 权限树接口 GET /system/permission/list
func (c *Controller) List(ctx context.Context, req *v1.PermissionListReq) (res *v1.PermissionListRes, err error) {
	r := g.RequestFromCtx(ctx)
	tree, e := c.permissionSvc.GetPermissionTree(ctx)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": tree})
	return
}

// Detail 权限详情 GET /system/permission/detail
func (c *Controller) Detail(ctx context.Context, req *v1.PermissionDetailReq) (res *v1.PermissionDetailRes, err error) {
	r := g.RequestFromCtx(ctx)
	detail, e := c.permissionSvc.GetDetail(ctx, req.Id)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	if detail == nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "message": "权限不存在", "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": detail})
	return
}

// Create 创建权限 POST /system/permission/create
func (c *Controller) Create(ctx context.Context, req *v1.PermissionCreateReq) (res *v1.PermissionCreateRes, err error) {
	r := g.RequestFromCtx(ctx)
	id, e := c.permissionSvc.Create(ctx, g.Map{
		"name":      req.Name,
		"code":      req.Code,
		"type":      req.Type,
		"parent_id": req.ParentId,
		"path":      req.Path,
		"component": req.Component,
		"icon":      req.Icon,
		"sort":      req.Sort,
		"status":    req.Status,
		"hide":      req.Hide,
		"method":    req.Method,
		"api":       req.Api,
		"ext_url":   req.ExtUrl,
	})
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{"id": id}})
	return
}

// Update 更新权限 PUT /system/permission/update
func (c *Controller) Update(ctx context.Context, req *v1.PermissionUpdateReq) (res *v1.PermissionUpdateRes, err error) {
	r := g.RequestFromCtx(ctx)
	update := g.Map{}
	if req.Name != "" {
		update["name"] = req.Name
	}
	if req.Code != "" {
		update["code"] = req.Code
	}
	if req.Type != nil {
		update["type"] = *req.Type
	}
	if req.ParentId != nil {
		update["parent_id"] = *req.ParentId
	}
	if req.Path != "" {
		update["path"] = req.Path
	}
	if req.Component != "" {
		update["component"] = req.Component
	}
	if req.Icon != "" {
		update["icon"] = req.Icon
	}
	if req.Sort != nil {
		update["sort"] = *req.Sort
	}
	if req.Status != nil {
		update["status"] = *req.Status
	}
	if req.Hide != nil {
		update["hide"] = *req.Hide
	}
	if req.Method != "" {
		update["method"] = req.Method
	}
	if req.Api != "" {
		update["api"] = req.Api
	}
	if req.ExtUrl != "" {
		update["ext_url"] = req.ExtUrl
	}
	if len(update) == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "无更新内容", "data": nil})
		return
	}
	if e := c.permissionSvc.Update(ctx, req.Id, update); e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
	return
}

// Delete 删除权限 DELETE /system/permission/delete
func (c *Controller) Delete(ctx context.Context, req *v1.PermissionDeleteReq) (res *v1.PermissionDeleteRes, err error) {
	r := g.RequestFromCtx(ctx)
	if e := c.permissionSvc.Delete(ctx, req.Id); e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
	return
}
