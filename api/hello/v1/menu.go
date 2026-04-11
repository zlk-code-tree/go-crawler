package v1

import "github.com/gogf/gf/v2/frame/g"

// --- Menu CRUD APIs ---

type MenuListReq struct {
	g.Meta `path:"/system/menu/list" method:"get" tags:"Menu" summary:"获取菜单列表(树形)"`
}

type MenuListRes struct {
	g.Meta `mime:"application/json"`
}

type MenuDetailReq struct {
	g.Meta `path:"/system/menu/detail" method:"get" tags:"Menu" summary:"获取菜单详情"`
	Id     int64 `json:"id" in:"query" v:"required" dc:"菜单ID"`
}

type MenuDetailRes struct {
	g.Meta `mime:"application/json"`
}

type MenuCreateReq struct {
	g.Meta    `path:"/system/menu/create" method:"post" tags:"Menu" summary:"创建菜单"`
	Name      string `json:"name" v:"required" dc:"菜单名称"`
	Code      string `json:"code" v:"required" dc:"权限编码(唯一)"`
	Type      int    `json:"type" d:"1" dc:"类型：1-菜单 2-按钮 3-接口"`
	ParentId  int64  `json:"parent_id" d:"0" dc:"父菜单ID"`
	Path      string `json:"path" dc:"路由路径"`
	Component string `json:"component" dc:"组件路径"`
	Icon      string `json:"icon" dc:"图标"`
	Sort      int    `json:"sort" d:"0" dc:"排序值"`
	Status    int    `json:"status" d:"1" dc:"状态：0-禁用 1-启用"`
	Hide      int    `json:"hide" d:"0" dc:"是否隐藏：0-显示 1-隐藏"`
	Method    string `json:"method" dc:"HTTP方法(按钮/接口类型)"`
	Api       string `json:"api" dc:"API路径(接口类型)"`
	ExtUrl    string `json:"extUrl" dc:"外链地址"`
}

type MenuCreateRes struct {
	g.Meta `mime:"application/json"`
	Id     int64 `json:"id" dc:"菜单ID"`
}

type MenuUpdateReq struct {
	g.Meta    `path:"/system/menu/update" method:"put" tags:"Menu" summary:"更新菜单"`
	Id        int64  `json:"id" v:"required" dc:"菜单ID"`
	Name      string `json:"name" dc:"菜单名称"`
	Code      string `json:"code" dc:"权限编码"`
	Type      *int   `json:"type" dc:"类型：1-菜单 2-按钮 3-接口"`
	ParentId  *int64 `json:"parent_id" dc:"父菜单ID"`
	Path      string `json:"path" dc:"路由路径"`
	Component string `json:"component" dc:"组件路径"`
	Icon      string `json:"icon" dc:"图标"`
	Sort      *int   `json:"sort" dc:"排序值"`
	Status    *int   `json:"status" dc:"状态：0-禁用 1-启用"`
	Hide      *int   `json:"hide" dc:"是否隐藏：0-显示 1-隐藏"`
	Method    string `json:"method" dc:"HTTP方法"`
	Api       string `json:"api" dc:"API路径"`
	ExtUrl    string `json:"extUrl" dc:"外链地址"`
}

type MenuUpdateRes struct {
	g.Meta `mime:"application/json"`
}

type MenuDeleteReq struct {
	g.Meta `path:"/system/menu/delete" method:"delete" tags:"Menu" summary:"删除菜单"`
	Id     int64 `json:"id" in:"query" v:"required" dc:"菜单ID"`
}

type MenuDeleteRes struct {
	g.Meta `mime:"application/json"`
}
