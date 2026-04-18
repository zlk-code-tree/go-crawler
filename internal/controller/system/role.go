package system

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/service"
)

// RoleController 角色管理相关接口
// 路径前缀：/system/role（由 g.Meta path 定义）

type RoleController struct{}

func NewRoleController() *RoleController {
	return &RoleController{}
}

// RoleList GET /system/role/list
func (c *RoleController) RoleList(ctx context.Context, req *v1.SystemRoleListReq) (res *v1.SystemRoleListRes, err error) {
	r := g.RequestFromCtx(ctx)
	list, total, e := service.NewRoleService().List(ctx, req.Page, req.PageSize, req.Keyword)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + e.Error()})
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

// Save POST /system/role/save
// id=0 新增，id>0 编辑，同时写入 sys_role_permission
func (c *RoleController) Save(ctx context.Context, req *v1.SystemRoleSaveReq) (res *v1.SystemRoleSaveRes, err error) {
	r := g.RequestFromCtx(ctx)
	roleSvc := service.NewRoleService()

	if req.Id == 0 {
		roleId, e := roleSvc.Create(ctx, req.Name, req.Code, req.OrderNum, req.Status, req.Remark, req.MenuIds)
		if e != nil {
			r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "创建失败: " + e.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"id": roleId}, "msg": "ok"})
	} else {
		e := roleSvc.Update(ctx, req.Id, req.Name, req.Code, req.OrderNum, req.Status, req.Remark, req.MenuIds)
		if e != nil {
			r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "更新失败: " + e.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"id": req.Id}, "msg": "ok"})
	}
	return
}

// Detail GET /system/role/detail
// 返回角色基本信息 + 已分配 menuIds
func (c *RoleController) Detail(ctx context.Context, req *v1.SystemRoleDetailReq) (res *v1.SystemRoleDetailRes, err error) {
	r := g.RequestFromCtx(ctx)
	roleSvc := service.NewRoleService()

	role, e := roleSvc.GetById(ctx, req.Id)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + e.Error()})
		return
	}
	if role == nil {
		r.Response.WriteJsonExit(g.Map{"code": 404, "data": nil, "msg": "角色不存在"})
		return
	}

	menuIds, e := roleSvc.GetMenuIds(ctx, req.Id)
	if e != nil || menuIds == nil {
		menuIds = []int64{}
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"data": g.Map{
			"id":       role.Id,
			"name":     role.Name,
			"code":     role.Code,
			"orderNum": role.Sort,
			"status":   role.Status,
			"remark":   role.Remark,
			"menuIds":  menuIds,
		},
		"msg": "ok",
	})
	return
}

// Delete POST /system/role/delete
func (c *RoleController) Delete(ctx context.Context, req *v1.SystemRoleDeleteReq) (res *v1.SystemRoleDeleteRes, err error) {
	r := g.RequestFromCtx(ctx)
	e := service.NewRoleService().Delete(ctx, req.Id)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "删除失败: " + e.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
	return
}

// MenuIds GET /system/role/menu-ids
func (c *RoleController) MenuIds(ctx context.Context, req *v1.SystemRoleMenuIdsReq) (res *v1.SystemRoleMenuIdsRes, err error) {
	r := g.RequestFromCtx(ctx)
	menuIds, e := service.NewRoleService().GetMenuIds(ctx, req.RoleId)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败: " + e.Error()})
		return
	}
	if menuIds == nil {
		menuIds = []int64{}
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": menuIds, "msg": "ok"})
	return
}

// AssignMenus POST /system/role/assign-menus
func (c *RoleController) AssignMenus(ctx context.Context, req *v1.SystemRoleAssignMenusReq) (res *v1.SystemRoleAssignMenusRes, err error) {
	r := g.RequestFromCtx(ctx)
	e := service.NewRoleService().AssignMenus(ctx, req.RoleId, req.MenuIds)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "分配失败: " + e.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
	return
}
