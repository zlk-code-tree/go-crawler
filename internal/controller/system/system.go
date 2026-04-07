package system

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"golang.org/x/crypto/bcrypt"

	v1 "go-crawler/api/hello/v1"
)

type Controller struct{}

func New() *Controller { return &Controller{} }

// UserList GET /api/system/user/list
func (c *Controller) UserList(ctx context.Context, req *v1.SystemUserListReq) (res *v1.SystemUserListRes, err error) {
	r := g.RequestFromCtx(ctx)
	page := req.Page
	pageSize := req.PageSize
	m := g.DB().Model("sys_user").OrderDesc("created_at")
	if req.Keyword != "" {
		m = m.WhereOrLike("username", "%"+req.Keyword+"%").WhereOrLike("real_name", "%"+req.Keyword+"%")
	}
	if req.Status != "" {
		m = m.Where("status", req.Status)
	}
	total, _ := m.Count()
	list, e := m.Fields("id,username,real_name,email,phone,avatar,status,last_login_at,created_at").
		Page(page, pageSize).All()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{
		"list": list, "total": total, "page": page, "pageSize": pageSize,
	}})
	return
}

// UserCreate POST /api/system/user/create
func (c *Controller) UserCreate(ctx context.Context, req *v1.SystemUserCreateReq) (res *v1.SystemUserCreateRes, err error) {
	r := g.RequestFromCtx(ctx)
	count, _ := g.DB().Model("sys_user").Where("username", req.Username).Count()
	if count > 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "username already exists", "data": nil})
		return
	}
	hash, e := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": "password hash failed", "data": nil})
		return
	}
	result, e := g.DB().Model("sys_user").Insert(g.Map{
		"username": req.Username, "password": string(hash),
		"real_name": req.RealName, "email": req.Email, "phone": req.Phone,
		"status": 1, "created_at": time.Now(),
	})
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	id, _ := result.LastInsertId()
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{"id": id}})
	return
}

// UserUpdate PUT /api/system/user/update
func (c *Controller) UserUpdate(ctx context.Context, req *v1.SystemUserUpdateReq) (res *v1.SystemUserUpdateRes, err error) {
	r := g.RequestFromCtx(ctx)
	update := g.Map{"real_name": req.RealName, "email": req.Email, "phone": req.Phone}
	if req.Status != nil {
		update["status"] = *req.Status
	}
	if _, e := g.DB().Model("sys_user").Where("id", req.Id).Update(update); e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
	return
}

// UserDelete DELETE /api/system/user/delete
func (c *Controller) UserDelete(ctx context.Context, req *v1.SystemUserDeleteReq) (res *v1.SystemUserDeleteRes, err error) {
	r := g.RequestFromCtx(ctx)
	if _, e := g.DB().Model("sys_user").Where("id", req.Id).Update(g.Map{"status": 2}); e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
	return
}

// UserResetPassword PUT /api/system/user/reset-password
func (c *Controller) UserResetPassword(ctx context.Context, req *v1.SystemUserResetPasswordReq) (res *v1.SystemUserResetPasswordRes, err error) {
	r := g.RequestFromCtx(ctx)
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if _, e := g.DB().Model("sys_user").Where("id", req.Id).Update(g.Map{"password": string(hash)}); e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
	return
}

// UserAssignRoles POST /api/system/user/assign-roles
func (c *Controller) UserAssignRoles(ctx context.Context, req *v1.SystemUserAssignRolesReq) (res *v1.SystemUserAssignRolesRes, err error) {
	r := g.RequestFromCtx(ctx)
	_, _ = g.DB().Model("sys_user_role").Where("user_id", req.UserId).Delete()
	for _, roleId := range req.RoleIds {
		if _, e := g.DB().Model("sys_user_role").Insert(g.Map{"user_id": req.UserId, "role_id": roleId}); e != nil {
			r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
			return
		}
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
	return
}

// RoleList GET /api/system/role/list
func (c *Controller) RoleList(ctx context.Context, req *v1.SystemRoleListReq) (res *v1.SystemRoleListRes, err error) {
	r := g.RequestFromCtx(ctx)
	list, e := g.DB().Model("sys_role").Where("status", 1).OrderAsc("sort").All()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	total, _ := g.DB().Model("sys_role").Where("status", 1).Count()
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{"list": list, "total": total}})
	return
}

// RoleCreate POST /api/system/role/create
func (c *Controller) RoleCreate(ctx context.Context, req *v1.SystemRoleCreateReq) (res *v1.SystemRoleCreateRes, err error) {
	r := g.RequestFromCtx(ctx)
	result, e := g.DB().Model("sys_role").Insert(g.Map{"name": req.Name, "code": req.Code, "remark": req.Remark, "status": 1})
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	id, _ := result.LastInsertId()
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{"id": id}})
	return
}

// RoleUpdate PUT /api/system/role/update
func (c *Controller) RoleUpdate(ctx context.Context, req *v1.SystemRoleUpdateReq) (res *v1.SystemRoleUpdateRes, err error) {
	r := g.RequestFromCtx(ctx)
	update := g.Map{"name": req.Name, "remark": req.Remark}
	if req.Status != nil {
		update["status"] = *req.Status
	}
	if _, e := g.DB().Model("sys_role").Where("id", req.Id).Update(update); e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
	return
}

// RoleDelete DELETE /api/system/role/delete
func (c *Controller) RoleDelete(ctx context.Context, req *v1.SystemRoleDeleteReq) (res *v1.SystemRoleDeleteRes, err error) {
	r := g.RequestFromCtx(ctx)
	if _, e := g.DB().Model("sys_role").Where("id", req.Id).Update(g.Map{"status": 0}); e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
	return
}

// RoleAssignPermissions POST /api/system/role/assign-permissions
func (c *Controller) RoleAssignPermissions(ctx context.Context, req *v1.SystemRoleAssignPermissionsReq) (res *v1.SystemRoleAssignPermissionsRes, err error) {
	r := g.RequestFromCtx(ctx)
	_, _ = g.DB().Model("sys_role_permission").Where("role_id", req.RoleId).Delete()
	for _, permId := range req.PermissionIds {
		if _, e := g.DB().Model("sys_role_permission").Insert(g.Map{"role_id": req.RoleId, "permission_id": permId}); e != nil {
			r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
			return
		}
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
	return
}

// PermissionTree GET /api/system/permission/tree
func (c *Controller) PermissionTree(ctx context.Context, req *v1.SystemPermissionTreeReq) (res *v1.SystemPermissionTreeRes, err error) {
	r := g.RequestFromCtx(ctx)
	list, e := g.DB().Model("sys_permission").Where("status", 1).OrderAsc("sort").All()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": list})
	return
}

// OperationLogList GET /api/system/log/operation
func (c *Controller) OperationLogList(ctx context.Context, req *v1.SystemOperationLogListReq) (res *v1.SystemOperationLogListRes, err error) {
	r := g.RequestFromCtx(ctx)
	page := req.Page
	pageSize := req.PageSize
	total, _ := g.DB().Model("sys_operation_log").Count()
	list, e := g.DB().Model("sys_operation_log").OrderDesc("created_at").Page(page, pageSize).All()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{
		"list": list, "total": total, "page": page, "pageSize": pageSize,
	}})
	return
}

// LoginLogList GET /api/system/log/login
func (c *Controller) LoginLogList(ctx context.Context, req *v1.SystemLoginLogListReq) (res *v1.SystemLoginLogListRes, err error) {
	r := g.RequestFromCtx(ctx)
	page := req.Page
	pageSize := req.PageSize
	total, _ := g.DB().Model("sys_login_log").Count()
	list, e := g.DB().Model("sys_login_log").OrderDesc("created_at").Page(page, pageSize).All()
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": e.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{
		"list": list, "total": total, "page": page, "pageSize": pageSize,
	}})
	return
}
