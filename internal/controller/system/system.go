package system

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"golang.org/x/crypto/bcrypt"
)

type Controller struct{}

func New() *Controller { return &Controller{} }

// UserList GET /api/system/user/list
func (c *Controller) UserList(r *ghttp.Request) {
	page := r.GetQuery("page", 1).Int()
	pageSize := r.GetQuery("pageSize", 20).Int()
	keyword := r.GetQuery("keyword").String()
	status := r.GetQuery("status").String()
	m := g.DB().Model("sys_user").OrderDesc("created_at")
	if keyword != "" {
		m = m.WhereOrLike("username", "%"+keyword+"%").WhereOrLike("real_name", "%"+keyword+"%")
	}
	if status != "" {
		m = m.Where("status", status)
	}
	total, _ := m.Count()
	list, err := m.Fields("id,username,real_name,email,phone,avatar,status,last_login_at,created_at").
		Page(page, pageSize).All()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{
		"list": list, "total": total, "page": page, "pageSize": pageSize,
	}})
}

// UserCreate POST /api/system/user/create
func (c *Controller) UserCreate(r *ghttp.Request) {
	var req struct {
		Username string `json:"username" v:"required"`
		Password string `json:"password" v:"required|min-length:6"`
		RealName string `json:"real_name"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": err.Error(), "data": nil})
		return
	}
	count, _ := g.DB().Model("sys_user").Where("username", req.Username).Count()
	if count > 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "username already exists", "data": nil})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": "password hash failed", "data": nil})
		return
	}
	result, err := g.DB().Model("sys_user").Insert(g.Map{
		"username": req.Username, "password": string(hash),
		"real_name": req.RealName, "email": req.Email, "phone": req.Phone,
		"status": 1, "created_at": time.Now(),
	})
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	id, _ := result.LastInsertId()
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{"id": id}})
}

// UserUpdate PUT /api/system/user/update
func (c *Controller) UserUpdate(r *ghttp.Request) {
	var req struct {
		Id       int64  `json:"id" v:"required"`
		RealName string `json:"real_name"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Status   *int   `json:"status"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": err.Error(), "data": nil})
		return
	}
	update := g.Map{"real_name": req.RealName, "email": req.Email, "phone": req.Phone}
	if req.Status != nil {
		update["status"] = *req.Status
	}
	if _, err := g.DB().Model("sys_user").Where("id", req.Id).Update(update); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
}

// UserDelete DELETE /api/system/user/delete
func (c *Controller) UserDelete(r *ghttp.Request) {
	id := r.GetQuery("id").Int64()
	if id == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "id required", "data": nil})
		return
	}
	if _, err := g.DB().Model("sys_user").Where("id", id).Update(g.Map{"status": 2}); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
}

// UserResetPassword PUT /api/system/user/reset-password
func (c *Controller) UserResetPassword(r *ghttp.Request) {
	var req struct {
		Id          int64  `json:"id" v:"required"`
		NewPassword string `json:"new_password" v:"required|min-length:6"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": err.Error(), "data": nil})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if _, err := g.DB().Model("sys_user").Where("id", req.Id).Update(g.Map{"password": string(hash)}); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
}

// UserAssignRoles POST /api/system/user/assign-roles
func (c *Controller) UserAssignRoles(r *ghttp.Request) {
	var req struct {
		UserId  int64   `json:"user_id" v:"required"`
		RoleIds []int64 `json:"role_ids"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": err.Error(), "data": nil})
		return
	}
	_, _ = g.DB().Model("sys_user_role").Where("user_id", req.UserId).Delete()
	for _, roleId := range req.RoleIds {
		if _, err := g.DB().Model("sys_user_role").Insert(g.Map{"user_id": req.UserId, "role_id": roleId}); err != nil {
			r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
			return
		}
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
}

// RoleList GET /api/system/role/list
func (c *Controller) RoleList(r *ghttp.Request) {
	list, err := g.DB().Model("sys_role").Where("status", 1).OrderAsc("sort").All()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	total, _ := g.DB().Model("sys_role").Where("status", 1).Count()
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{"list": list, "total": total}})
}

// RoleCreate POST /api/system/role/create
func (c *Controller) RoleCreate(r *ghttp.Request) {
	var req struct {
		Name   string `json:"name" v:"required"`
		Code   string `json:"code" v:"required"`
		Remark string `json:"remark"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": err.Error(), "data": nil})
		return
	}
	result, err := g.DB().Model("sys_role").Insert(g.Map{"name": req.Name, "code": req.Code, "remark": req.Remark, "status": 1})
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	id, _ := result.LastInsertId()
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{"id": id}})
}

// RoleUpdate PUT /api/system/role/update
func (c *Controller) RoleUpdate(r *ghttp.Request) {
	var req struct {
		Id     int64  `json:"id" v:"required"`
		Name   string `json:"name"`
		Remark string `json:"remark"`
		Status *int   `json:"status"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": err.Error(), "data": nil})
		return
	}
	update := g.Map{"name": req.Name, "remark": req.Remark}
	if req.Status != nil {
		update["status"] = *req.Status
	}
	if _, err := g.DB().Model("sys_role").Where("id", req.Id).Update(update); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
}

// RoleDelete DELETE /api/system/role/delete
func (c *Controller) RoleDelete(r *ghttp.Request) {
	id := r.GetQuery("id").Int64()
	if id == 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "id required", "data": nil})
		return
	}
	if _, err := g.DB().Model("sys_role").Where("id", id).Update(g.Map{"status": 0}); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
}

// RoleAssignPermissions POST /api/system/role/assign-permissions
func (c *Controller) RoleAssignPermissions(r *ghttp.Request) {
	var req struct {
		RoleId        int64   `json:"role_id" v:"required"`
		PermissionIds []int64 `json:"permission_ids"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": err.Error(), "data": nil})
		return
	}
	_, _ = g.DB().Model("sys_role_permission").Where("role_id", req.RoleId).Delete()
	for _, permId := range req.PermissionIds {
		if _, err := g.DB().Model("sys_role_permission").Insert(g.Map{"role_id": req.RoleId, "permission_id": permId}); err != nil {
			r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
			return
		}
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
}

// PermissionTree GET /api/system/permission/tree
func (c *Controller) PermissionTree(r *ghttp.Request) {
	list, err := g.DB().Model("sys_permission").Where("status", 1).OrderAsc("sort").All()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": list})
}

// OperationLogList GET /api/system/log/operation
func (c *Controller) OperationLogList(r *ghttp.Request) {
	page := r.GetQuery("page", 1).Int()
	pageSize := r.GetQuery("pageSize", 20).Int()
	total, _ := g.DB().Model("sys_operation_log").Count()
	list, err := g.DB().Model("sys_operation_log").OrderDesc("created_at").Page(page, pageSize).All()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{
		"list": list, "total": total, "page": page, "pageSize": pageSize,
	}})
}

// LoginLogList GET /api/system/log/login
func (c *Controller) LoginLogList(r *ghttp.Request) {
	page := r.GetQuery("page", 1).Int()
	pageSize := r.GetQuery("pageSize", 20).Int()
	total, _ := g.DB().Model("sys_login_log").Count()
	list, err := g.DB().Model("sys_login_log").OrderDesc("created_at").Page(page, pageSize).All()
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": err.Error(), "data": nil})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": g.Map{
		"list": list, "total": total, "page": page, "pageSize": pageSize,
	}})
}
