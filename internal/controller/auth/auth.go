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
		r.Response.WriteJsonExit(g.Map{"code": 401, "msg": "用户不存在", "data": nil})
		return
	}

	roles, _ := c.authSvc.GetUserRoles(ctx, userId)
	if roles == nil {
		roles = []string{}
	}

	r.Response.WriteJsonExit(g.Map{
		"code": 0,
		"msg":  "ok",
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
	username := r.GetCtxVar("username").String()

	// 白名单用户返回通配符，前端视为拥有所有权限
	for _, u := range getPermissionWhitelist(ctx) {
		if u == username {
			r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": []string{"*"}})
			return
		}
	}

	codes, _ := c.authSvc.GetUserPermissionCodes(ctx, userId)
	if codes == nil {
		codes = []string{}
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": codes})
	return
}

// Menus 获取菜单 + 按钮权限码
func (c *Controller) Menus(ctx context.Context, req *v1.AuthMenusReq) (res *v1.AuthMenusRes, err error) {
	r := g.RequestFromCtx(ctx)
	userId := r.GetCtxVar("userId").Int64()
	username := r.GetCtxVar("username").String()

	// 白名单用户返回全量菜单和全量按钮权限码
	for _, u := range getPermissionWhitelist(ctx) {
		if u == username {
			menus, e := c.menuSvc.GetAllMenusAsVbenTree(ctx)
			if e != nil {
				r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "获取菜单失败", "data": nil})
				return
			}
			buttons, _ := c.menuSvc.GetAllButtonCodes(ctx)
			if buttons == nil {
				buttons = []string{}
			}
			r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": g.Map{
				"menus":   menus,
				"buttons": buttons,
			}})
			return
		}
	}

	menus, e := c.menuSvc.GetUserMenuTree(ctx, userId)
	if e != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "msg": "获取菜单失败", "data": nil})
		return
	}
	buttons, _ := c.menuSvc.GetUserButtonCodes(ctx, userId)
	if buttons == nil {
		buttons = []string{}
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "msg": "ok", "data": g.Map{
		"menus":   menus,
		"buttons": buttons,
	}})
	return
}

func getJwtSecret(ctx context.Context) string {
	secret, _ := g.Cfg().Get(ctx, "jwt.secret", "go-crawler-jwt-secret-2024")
	return secret.String()
}

// getPermissionWhitelist 读取配置白名单
func getPermissionWhitelist(ctx context.Context) []string {
	cfg, err := g.Cfg().Get(ctx, "permission.whitelist")
	if err == nil {
		list := cfg.Strings()
		if len(list) > 0 {
			return list
		}
	}
	return []string{"admin"}
}
