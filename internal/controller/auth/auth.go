package auth

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	v1 "go-crawler/api/hello/v1"
	"go-crawler/internal/service"
)

type Controller struct {
	authSvc *service.AuthService
	menuSvc *service.MenuService
}

func New() *Controller {
	return &Controller{
		authSvc: service.NewAuthService(),
		menuSvc: service.NewMenuService(),
	}
}

// UserInfo 获取用户信息
func (c *Controller) UserInfo(ctx context.Context, req *v1.AuthUserInfoReq) (res *v1.AuthUserInfoRes, err error) {
	r := g.RequestFromCtx(ctx)
	userId := r.GetCtxVar("userId").Int64()
	username := r.GetCtxVar("username").String()

	user, e := c.authSvc.GetUserByUsername(ctx, username)
	if e != nil || user == nil {
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
			"homePath": "/dashboard/workspace",
			"desc":     "管理员",
			"token":    "",
		},
	})
	return
}

// AccessCodes 获取权限码
func (c *Controller) AccessCodes(ctx context.Context, req *v1.AuthAccessCodesReq) (res *v1.AuthAccessCodesRes, err error) {
	r := g.RequestFromCtx(ctx)
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
	return
}

// Menus 获取菜单
func (c *Controller) Menus(ctx context.Context, req *v1.AuthMenusReq) (res *v1.AuthMenusRes, err error) {
	r := g.RequestFromCtx(ctx)
	userId := r.GetCtxVar("userId").Int64()

	menus, e := c.menuSvc.GetUserMenuTree(ctx, userId)
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

func getJwtSecret(ctx context.Context) string {
	secret, _ := g.Cfg().Get(ctx, "jwt.secret", "go-crawler-jwt-secret-2024")
	return secret.String()
}
