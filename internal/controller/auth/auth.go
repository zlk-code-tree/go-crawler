package auth

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"golang.org/x/crypto/bcrypt"

	"go-crawler/internal/service"
	"go-crawler/utility"
)

type Controller struct {
	authSvc *service.AuthService
}

func New() *Controller {
	return &Controller{
		authSvc: service.NewAuthService(),
	}
}

// Login 处理登录请求
func (c *Controller) Login(r *ghttp.Request) {
	var req struct {
		Username string `json:"username" v:"required#用户名不能为空"`
		Password string `json:"password" v:"required#密码不能为空"`
	}
	if err := r.Parse(&req); err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": err.Error(), "data": nil})
		return
	}

	ctx := r.Context()
	user, err := c.authSvc.GetUserByUsername(ctx, req.Username)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": "系统错误", "data": nil})
		return
	}
	if user == nil {
		c.authSvc.RecordLoginLog(ctx, req.Username, 0, r.GetClientIp(), 0, "用户不存在")
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "用户名或密码错误", "data": nil})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.authSvc.RecordLoginLog(ctx, req.Username, user.Id, r.GetClientIp(), 0, "密码错误")
		r.Response.WriteJsonExit(g.Map{"code": 400, "message": "用户名或密码错误", "data": nil})
		return
	}

	secret := getJwtSecret(r)
	token, err := utility.GenerateToken(user.Id, user.Username, secret, 86400)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": "生成token失败", "data": nil})
		return
	}

	_ = c.authSvc.UpdateLastLogin(ctx, user.Id, r.GetClientIp())
	c.authSvc.RecordLoginLog(ctx, req.Username, user.Id, r.GetClientIp(), 1, "登录成功")

	r.Response.WriteJsonExit(g.Map{
		"code":    0,
		"message": "ok",
		"data": g.Map{
			"accessToken": token,
		},
	})
}

// Logout 处理登出请求
func (c *Controller) Logout(r *ghttp.Request) {
	r.Response.WriteJsonExit(g.Map{"code": 0, "message": "ok", "data": nil})
}

// UserInfo 获取用户信息
func (c *Controller) UserInfo(r *ghttp.Request) {
	ctx := r.Context()
	userId := r.GetCtxVar("userId").Int64()
	username := r.GetCtxVar("username").String()

	user, err := c.authSvc.GetUserByUsername(ctx, username)
	if err != nil || user == nil {
		r.Response.WriteStatus(401)
		r.Response.WriteJsonExit(g.Map{"code": 401, "message": "用户不存在", "data": nil})
		return
	}

	roles, _ := c.authSvc.GetUserRoles(ctx, userId)
	if roles == nil {
		roles = []string{}
	}

	r.Response.WriteJsonExit(g.Map{
		"code":    0,
		"message": "ok",
		"data": g.Map{
			"userId":   user.Id,
			"username": user.Username,
			"realName": user.RealName,
			"avatar":   user.Avatar,
			"roles":    roles,
			"homePath": "/dashboard",
			"desc":     "管理员",
			"token":    "",
		},
	})
}

// AccessCodes 获取权限码
func (c *Controller) AccessCodes(r *ghttp.Request) {
	ctx := r.Context()
	userId := r.GetCtxVar("userId").Int64()

	codes, _ := c.authSvc.GetUserPermissionCodes(ctx, userId)
	if codes == nil {
		codes = []string{}
	}

	r.Response.WriteJsonExit(g.Map{
		"code":    0,
		"message": "ok",
		"data":    codes,
	})
}

// Menus 获取菜单
func (c *Controller) Menus(r *ghttp.Request) {
	ctx := r.Context()
	userId := r.GetCtxVar("userId").Int64()

	menus, err := c.authSvc.GetUserMenus(ctx, userId)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": "获取菜单失败", "data": nil})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code":    0,
		"message": "ok",
		"data":    menus,
	})
}

func getJwtSecret(r *ghttp.Request) string {
	secret, _ := g.Cfg().Get(r.Context(), "jwt.secret", "go-crawler-jwt-secret-2024")
	return secret.String()
}
