package v1

import "github.com/gogf/gf/v2/frame/g"

// --- Auth APIs ---

type AuthLoginReq struct {
	g.Meta   `path:"/auth/login" method:"post" tags:"Auth" summary:"用户登录"`
	Username string `json:"username" v:"required#用户名不能为空"`
	Password string `json:"password" v:"required#密码不能为空"`
}

type AuthLoginRes struct {
	g.Meta      `mime:"application/json"`
	AccessToken string `json:"accessToken"`
}

type AuthLogoutReq struct {
	g.Meta `path:"/auth/logout" method:"post" tags:"Auth" summary:"退出登录"`
}

type AuthLogoutRes struct {
	g.Meta `mime:"application/json"`
}

type AuthUserInfoReq struct {
	g.Meta `path:"/user/info" method:"get" tags:"Auth" summary:"获取当前用户信息"`
}

type AuthUserInfoRes struct {
	g.Meta   `mime:"application/json"`
	UserId   int64    `json:"userId"`
	Username string   `json:"username"`
	RealName string   `json:"realName"`
	Avatar   string   `json:"avatar"`
	Roles    []string `json:"roles"`
	HomePath string   `json:"homePath"`
	Desc     string   `json:"desc"`
}

type AuthAccessCodesReq struct {
	g.Meta `path:"/auth/codes" method:"get" tags:"Auth" summary:"获取用户权限码"`
}

type AuthAccessCodesRes struct {
	g.Meta `mime:"application/json"`
	Codes  []string `json:"codes"`
}

// --- Auth APIs ---

type AuthMenusReq struct {
	g.Meta `path:"/menu/all" method:"get" tags:"Auth" summary:"获取用户菜单"`
}

type AuthPublicMenusReq struct {
	g.Meta `path:"/menu/public" method:"get" tags:"Auth" summary:"获取公开菜单树"`
}

type AuthMenuTreeReq struct {
	g.Meta `path:"/menu-tree" method:"get" tags:"Auth" summary:"获取菜单树"`
}

type AuthMenusRes struct {
	g.Meta `mime:"application/json"`
	Menus  interface{} `json:"menus"`
}
