package system

import (
	"go-crawler/internal/model"
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// UserController 用户管理相关接口
// 路径前缀：/system/user

type UserController struct{}

func NewUserController() *UserController {
	return &UserController{}
}

// UserList 用户列表接口 GET /system/user/list
func (c *UserController) UserList(r *ghttp.Request) {
	// 解析分页参数
	page := r.Get("page").Int()
	if page <= 0 {
		page = 1
	}
	pageSize := r.Get("pageSize").Int()
	if pageSize <= 0 {
		pageSize = 10
	}
	keyword := r.Get("keyword").String()

	// 调用服务层方法查询用户列表
	userService := service.NewUserService()
	users, total, err := userService.List(r.Context(), page, pageSize, keyword)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败"})
		return
	}

	// 构建返回数据
	var userList []g.Map
	for _, user := range users {
		userList = append(userList, g.Map{
			"id":       user.Id,
			"username": user.Username,
			"realName": user.RealName,
			"status":   user.Status,
			"email":    user.Email,
			"phone":    user.Phone,
			"nickname": user.Nickname,
			"deptId":   user.DeptId,
			"position": user.Position,
			"remark":   user.Remark,
		})
	}

	data := g.Map{
		"list":  userList,
		"total": total,
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": data, "msg": "ok"})
}

// AssignRoles 分配角色 POST /system/user/assign-roles
func (c *UserController) AssignRoles(r *ghttp.Request) {
	// 解析请求参数
	var req struct {
		UserId  int64   `json:"userId"`
		RoleIds []int64 `json:"roleIds"`
	}

	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误"})
		return
	}

	// 调用服务层方法分配角色
	userService := service.NewUserService()
	err := userService.AssignRoles(r.Context(), req.UserId, req.RoleIds)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "分配角色失败"})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}

// UserSave 新增/编辑用户接口 POST /system/user/save
func (c *UserController) UserSave(r *ghttp.Request) {
	// 解析请求参数
	var req struct {
		Id       int64  `json:"id"`
		Nickname string `json:"nickname"`
		DeptId   int64  `json:"deptId"`
		Phone    string `json:"phone"`
		Email    string `json:"email"`
		Username string `json:"username"`
		Password string `json:"password"`
		Gender   int    `json:"gender"`
		Status   int    `json:"status"`
		Position string `json:"position"`
		RoleId   int64  `json:"roleId"`
		Remark   string `json:"remark"`
	}

	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误"})
		return
	}

	// 构建用户模型
	user := &model.SysUser{
		Id:       req.Id,
		Username: req.Username,
		Nickname: req.Nickname,
		Email:    req.Email,
		Phone:    req.Phone,
		Gender:   req.Gender,
		Status:   req.Status,
		DeptId:   req.DeptId,
		Position: req.Position,
		Remark:   req.Remark,
	}

	// 如果有密码，则设置密码
	if req.Password != "" {
		user.Password = req.Password
	}

	// 调用服务层方法保存用户
	userService := service.NewUserService()
	id, err := userService.Save(r.Context(), user)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "保存失败"})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"id": id}, "msg": "ok"})
}

// UserDelete 删除用户接口 POST /system/user/delete
func (c *UserController) UserDelete(r *ghttp.Request) {
	// 解析请求参数
	var req struct {
		Id int64 `json:"id"`
	}

	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "参数解析错误"})
		return
	}

	// 调用服务层方法删除用户
	userService := service.NewUserService()
	err := userService.Delete(r.Context(), req.Id)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "删除失败"})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}

// GetUser 获取用户详情接口 GET /system/user/{id}
func (c *UserController) GetUser(r *ghttp.Request) {
	// 解析用户ID
	userId := r.Get("id").Int64()
	if userId <= 0 {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "用户ID不能为空"})
		return
	}

	// 调用服务层方法获取用户详情
	userService := service.NewUserService()
	user, err := userService.Get(r.Context(), userId)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "获取用户详情失败"})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": user, "msg": "ok"})
}
