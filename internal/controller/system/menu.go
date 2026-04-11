package system

import (
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// MenuController 菜单管理相关接口
// 路径前缀：/system/menu

type MenuController struct{}

func NewMenuController() *MenuController {
	return &MenuController{}
}

// MenuTree 菜单树接口 GET /system/menu/tree
func (c *MenuController) MenuTree(r *ghttp.Request) {
	// 调用服务层方法获取菜单树
	menuService := service.NewMenuService()
	tree, err := menuService.GetMenuTree(r.Context())
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "获取菜单树失败"})
		return
	}
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": tree, "msg": "ok"})
}
