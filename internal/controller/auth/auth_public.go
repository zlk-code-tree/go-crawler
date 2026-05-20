package auth

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"golang.org/x/crypto/bcrypt"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/service"
	"go-crawler/utility"
)

// PublicController 公开认证接口（无需 JWT）
type PublicController struct {
	authSvc *service.AuthService
	menuSvc *service.MenuService
}

func NewPublic() *PublicController {
	return &PublicController{
		authSvc: service.NewAuthService(),
		menuSvc: service.NewMenuService(),
	}
}

// Login 用户登录
func (c *PublicController) Login(ctx context.Context, req *v1.AuthLoginReq) (res *v1.AuthLoginRes, err error) {
	r := g.RequestFromCtx(ctx)

	user, e := c.authSvc.GetUserByUsername(ctx, req.Username)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "系统错误", "data": nil})
		return
	}
	if user == nil {
		c.authSvc.RecordLoginLog(ctx, req.Username, 0, r.GetClientIp(), 2, "用户不存在", r.UserAgent())
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "用户名或密码错误", "data": nil})
		return
	}

	if e = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); e != nil {
		c.authSvc.RecordLoginLog(ctx, req.Username, user.Id, r.GetClientIp(), 2, "密码错误", r.UserAgent())
		r.Response.WriteJsonExit(g.Map{"code": 400, "msg": "用户名或密码错误", "data": nil})
		return
	}

	secret := getJwtSecret(ctx)
	token, e := utility.GenerateToken(user.Id, user.Username, user.CompanyId, secret, 86400)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "生成token失败", "data": nil})
		return
	}

	_ = c.authSvc.UpdateLastLogin(ctx, user.Id, r.GetClientIp())
	c.authSvc.RecordLoginLog(ctx, req.Username, user.Id, r.GetClientIp(), 1, "登录成功", r.UserAgent())

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "ok",
		"data": g.Map{
			"accessToken": token,
		},
	})
	return
}

// Logout 退出登录
func (c *PublicController) Logout(ctx context.Context, req *v1.AuthLogoutReq) (res *v1.AuthLogoutRes, err error) {
	r := g.RequestFromCtx(ctx)
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": nil})
	return
}

// PublicMenus 获取公开菜单树（无需认证）
func (c *PublicController) PublicMenus(ctx context.Context, req *v1.AuthPublicMenusReq) (res *v1.AuthMenusRes, err error) {
	r := g.RequestFromCtx(ctx)

	// 获取完整菜单树（不包含权限过滤）
	menus, e := c.menuSvc.GetMenuTree(ctx)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": "获取菜单失败", "data": nil})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code":    0,
		"message": "ok",
		"data":    menus,
	})
	return
}

// MenuTreeRoot 根路径菜单树接口（无需认证）
func (c *PublicController) MenuTreeRoot(ctx context.Context, req *v1.AuthMenuTreeReq) (res *v1.AuthMenusRes, err error) {
	r := g.RequestFromCtx(ctx)

	// 获取完整菜单树（不包含权限过滤）
	menus, e := c.menuSvc.GetMenuTree(ctx)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "message": "获取菜单失败", "data": nil})
		return
	}

	r.Response.WriteJsonExit(g.Map{
		"code":    0,
		"message": "ok",
		"data":    menus,
	})
	return
}
