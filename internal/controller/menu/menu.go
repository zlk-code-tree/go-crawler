package menu

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/service"
)

type Controller struct {
	menuSvc *service.MenuService
}

func New() *Controller {
	return &Controller{
		menuSvc: service.NewMenuService(),
	}
}

// List 获取菜单树（管理用）
func (c *Controller) List(ctx context.Context, req *v1.MenuListReq) (res *v1.MenuListRes, err error) {
	r := g.RequestFromCtx(ctx)
	tree, e := c.menuSvc.GetMenuTree(ctx)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": tree})
	return
}

// Detail 获取菜单详情
func (c *Controller) Detail(ctx context.Context, req *v1.MenuDetailReq) (res *v1.MenuDetailRes, err error) {
	r := g.RequestFromCtx(ctx)
	detail, e := c.menuSvc.GetDetail(ctx, req.Id)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	if detail == nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "msg": "菜单不存在", "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": detail})
	return
}

// Create 创建菜单
func (c *Controller) Create(ctx context.Context, req *v1.MenuCreateReq) (res *v1.MenuCreateRes, err error) {
	r := g.RequestFromCtx(ctx)
	id, e := c.menuSvc.Create(ctx, g.Map{
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
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": g.Map{"id": id}})
	return
}

// Update 更新菜单
func (c *Controller) Update(ctx context.Context, req *v1.MenuUpdateReq) (res *v1.MenuUpdateRes, err error) {
	r := g.RequestFromCtx(ctx)
	update := g.Map{
		"name":      req.Name,
		"code":      req.Code,
		"path":      req.Path,
		"component": req.Component,
		"icon":      req.Icon,
		"method":    req.Method,
		"api":       req.Api,
		"ext_url":   req.ExtUrl,
	}
	if req.Type != nil {
		update["type"] = *req.Type
	}
	if req.ParentId != nil {
		update["parent_id"] = *req.ParentId
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
	if e := c.menuSvc.Update(ctx, req.Id, update); e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": nil})
	return
}

// Delete 删除菜单
func (c *Controller) Delete(ctx context.Context, req *v1.MenuDeleteReq) (res *v1.MenuDeleteRes, err error) {
	r := g.RequestFromCtx(ctx)
	if e := c.menuSvc.Delete(ctx, req.Id); e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": nil})
	return
}
