package v1

import "github.com/gogf/gf/v2/frame/g"

// --- Permission CRUD APIs ---

type PermissionListReq struct {
	g.Meta `path:"/system/permission/list" method:"get" tags:"Permission" summary:"权限列表(树形)"`
}

type PermissionListRes struct {
	g.Meta `mime:"application/json"`
}

type PermissionDetailReq struct {
	g.Meta `path:"/system/permission/detail" method:"get" tags:"Permission" summary:"权限详情"`
	Id     int64 `json:"id" in:"query" v:"required" dc:"权限ID"`
}

type PermissionDetailRes struct {
	g.Meta `mime:"application/json"`
}

type PermissionCreateReq struct {
	g.Meta    `path:"/system/permission/create" method:"post" tags:"Permission" summary:"创建权限"`
	Name      string `json:"name" v:"required" dc:"权限名称"`
	Code      string `json:"code" v:"required" dc:"权限编码(唯一)"`
	Type      int    `json:"type" d:"3" dc:"类型：1-菜单 2-按钮 3-接口"`
	ParentId  int64  `json:"parent_id" d:"0" dc:"父权限ID"`
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

type PermissionCreateRes struct {
	g.Meta `mime:"application/json"`
	Id     int64 `json:"id" dc:"权限ID"`
}

type PermissionUpdateReq struct {
	g.Meta    `path:"/system/permission/update" method:"put" tags:"Permission" summary:"更新权限"`
	Id        int64  `json:"id" v:"required" dc:"权限ID"`
	Name      string `json:"name" dc:"权限名称"`
	Code      string `json:"code" dc:"权限编码"`
	Type      *int   `json:"type" dc:"类型：1-菜单 2-按钮 3-接口"`
	ParentId  *int64 `json:"parent_id" dc:"父权限ID"`
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

type PermissionUpdateRes struct {
	g.Meta `mime:"application/json"`
}

type PermissionDeleteReq struct {
	g.Meta `path:"/system/permission/delete" method:"delete" tags:"Permission" summary:"删除权限"`
	Id     int64 `json:"id" in:"query" v:"required" dc:"权限ID"`
}

type PermissionDeleteRes struct {
	g.Meta `mime:"application/json"`
}
