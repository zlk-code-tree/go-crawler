package system

import (
	"go-crawler/internal/service"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// RoleController 角色管理相关接口
// 路径前缀：/system/role

type RoleController struct{}

func NewRoleController() *RoleController {
	return &RoleController{}
}

// RoleList 角色列表接口 GET /system/role/list
func (c *RoleController) RoleList(r *ghttp.Request) {
	page := r.Get("page", 1).Int()
	pageSize := r.Get("pageSize", 10).Int()
	keyword := r.Get("keyword", "").String()

	list, total, err := service.NewRoleService().List(r.Context(), page, pageSize, keyword)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败"})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"list": list, "total": total}, "msg": "ok"})
}

// Save 新增/编辑角色接口 POST /system/role/save
func (c *RoleController) Save(r *ghttp.Request) {
	name := r.Get("name", "").String()
	code := r.Get("code", "").String()
	orderNum := r.Get("orderNum", 0).Int()
	status := r.Get("status", "1").String()
	menuIdsParam := r.Get("menuIds", "").String()
	remark := r.Get("remark", "").String()

	if name == "" || code == "" {
		r.Response.WriteJsonExit(g.Map{"code": 400, "data": nil, "msg": "角色名称和编码不能为空"})
		return
	}

	statusInt := 1
	if status == "0" {
		statusInt = 0
	}

	var menuIds []int64
	if menuIdsParam != "" {
		for _, id := range r.Get("menuIds").Ints() {
			menuIds = append(menuIds, int64(id))
		}
	}

	roleId, err := service.NewRoleService().Create(r.Context(), name, code, orderNum, statusInt, remark, menuIds)
	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "创建失败: " + err.Error()})
		return
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"id": roleId}, "msg": "ok"})
}

// Delete 删除角色接口 POST /system/role/delete
func (c *RoleController) Delete(r *ghttp.Request) {
	// TODO: 删除角色，参数见前端
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}

// MenuIds 获取角色已分配菜单 GET /system/role/menu-ids/{roleId}
func (c *RoleController) MenuIds(r *ghttp.Request) {
	// TODO: 查询角色已分配菜单ID
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": []int{1, 2, 3}, "msg": "ok"})
}

// AssignMenus 分配菜单 POST /system/role/assign-menus
func (c *RoleController) AssignMenus(r *ghttp.Request) {
	// TODO: 保存角色菜单分配
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}

// DataScope 获取角色数据权限 GET /system/role/data-scope/{roleId}
func (c *RoleController) DataScope(r *ghttp.Request) {
	// TODO: 查询角色数据权限
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": g.Map{"deptIds": []int{1, 2}}, "msg": "ok"})
}

// AssignDataScope 分配数据权限 POST /system/role/assign-data-scope
func (c *RoleController) AssignDataScope(r *ghttp.Request) {
	// TODO: 保存角色数据权限
	r.Response.WriteJsonExit(g.Map{"code": 0, "data": nil, "msg": "ok"})
}
