package v1

import "github.com/gogf/gf/v2/frame/g"

// --- System User APIs ---

type SystemUserListReq struct {
	g.Meta   `path:"/system/user/list" method:"get" tags:"System-User" summary:"用户列表"`
	Page     int    `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"20" dc:"每页条数"`
	Keyword  string `json:"keyword" in:"query" dc:"搜索关键词"`
	Status   string `json:"status" in:"query" dc:"状态"`
}

type SystemUserListRes struct {
	g.Meta   `mime:"application/json"`
	List     interface{} `json:"list" dc:"用户列表"`
	Total    int         `json:"total" dc:"总数"`
	Page     int         `json:"page" dc:"当前页"`
	PageSize int         `json:"pageSize" dc:"每页条数"`
}

type SystemUserCreateReq struct {
	g.Meta   `path:"/system/user/create" method:"post" tags:"System-User" summary:"创建用户"`
	Username string `json:"username" v:"required" dc:"用户名"`
	Password string `json:"password" v:"required|min-length:6" dc:"密码"`
	RealName string `json:"real_name" dc:"真实姓名"`
	Email    string `json:"email" dc:"邮箱"`
	Phone    string `json:"phone" dc:"手机号"`
}

type SystemUserCreateRes struct {
	g.Meta `mime:"application/json"`
	Id     int64 `json:"id" dc:"用户ID"`
}

type SystemUserUpdateReq struct {
	g.Meta   `path:"/system/user/update" method:"put" tags:"System-User" summary:"更新用户"`
	Id       int64  `json:"id" v:"required" dc:"用户ID"`
	RealName string `json:"real_name" dc:"真实姓名"`
	Email    string `json:"email" dc:"邮箱"`
	Phone    string `json:"phone" dc:"手机号"`
	Status   *int   `json:"status" dc:"状态"`
}

type SystemUserUpdateRes struct {
	g.Meta `mime:"application/json"`
}

type SystemUserDeleteReq struct {
	g.Meta `path:"/system/user/delete" method:"delete" tags:"System-User" summary:"删除用户"`
	Id     int64 `json:"id" in:"query" v:"required" dc:"用户ID"`
}

type SystemUserDeleteRes struct {
	g.Meta `mime:"application/json"`
}

type SystemUserResetPasswordReq struct {
	g.Meta      `path:"/system/user/reset-password" method:"put" tags:"System-User" summary:"重置密码"`
	Id          int64  `json:"id" v:"required" dc:"用户ID"`
	NewPassword string `json:"new_password" v:"required|min-length:6" dc:"新密码"`
}

type SystemUserResetPasswordRes struct {
	g.Meta `mime:"application/json"`
}

type SystemUserAssignRolesReq struct {
	g.Meta  `path:"/system/user/assign-roles" method:"post" tags:"System-User" summary:"分配角色"`
	UserId  int64   `json:"user_id" v:"required" dc:"用户ID"`
	RoleIds []int64 `json:"role_ids" dc:"角色ID列表"`
}

type SystemUserAssignRolesRes struct {
	g.Meta `mime:"application/json"`
}

// --- System Role APIs ---

type SystemRoleListReq struct {
	g.Meta   `path:"/system/role/list" method:"get" tags:"System-Role" summary:"角色列表"`
	Page     int    `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"10" dc:"每页条数"`
	Keyword  string `json:"keyword" in:"query" dc:"搜索关键词"`
}

type SystemRoleListRes struct {
	g.Meta `mime:"application/json"`
}

type SystemRoleSaveReq struct {
	g.Meta   `path:"/system/role/save" method:"post" tags:"System-Role" summary:"新增/编辑角色（id=0新增，id>0编辑）"`
	Id       int64   `json:"id" dc:"角色ID，编辑时必填"`
	Name     string  `json:"name" v:"required" dc:"角色名称"`
	Code     string  `json:"code" v:"required" dc:"角色编码"`
	OrderNum int     `json:"orderNum" dc:"排序号"`
	Status   int     `json:"status" d:"1" dc:"状态：1启用 0停用"`
	MenuIds  []int64 `json:"menuIds" dc:"菜单权限ID列表，同步写入 sys_role_permission"`
	Remark   string  `json:"remark" dc:"备注"`
}

type SystemRoleSaveRes struct {
	g.Meta `mime:"application/json"`
}

type SystemRoleDetailReq struct {
	g.Meta `path:"/system/role/detail" method:"get" tags:"System-Role" summary:"角色详情（含已分配menuIds）"`
	Id     int64 `json:"id" in:"query" v:"required" dc:"角色ID"`
}

type SystemRoleDetailRes struct {
	g.Meta `mime:"application/json"`
}

type SystemRoleDeleteReq struct {
	g.Meta `path:"/system/role/delete" method:"post" tags:"System-Role" summary:"删除角色"`
	Id     int64 `json:"id" v:"required" dc:"角色ID"`
}

type SystemRoleDeleteRes struct {
	g.Meta `mime:"application/json"`
}

type SystemRoleMenuIdsReq struct {
	g.Meta `path:"/system/role/menu-ids" method:"get" tags:"System-Role" summary:"获取角色已分配菜单ID列表"`
	RoleId int64 `json:"roleId" in:"query" v:"required" dc:"角色ID"`
}

type SystemRoleMenuIdsRes struct {
	g.Meta `mime:"application/json"`
}

type SystemRoleAssignMenusReq struct {
	g.Meta  `path:"/system/role/assign-menus" method:"post" tags:"System-Role" summary:"分配菜单权限"`
	RoleId  int64   `json:"roleId" v:"required" dc:"角色ID"`
	MenuIds []int64 `json:"menuIds" dc:"菜单权限ID列表"`
}

type SystemRoleAssignMenusRes struct {
	g.Meta `mime:"application/json"`
}

// --- System Permission APIs ---

type SystemPermissionTreeReq struct {
	g.Meta `path:"/system/permission/tree" method:"get" tags:"System-Permission" summary:"权限树"`
}

type SystemPermissionTreeRes struct {
	g.Meta `mime:"application/json"`
}

// --- System Log APIs ---

type SystemOperationLogListReq struct {
	g.Meta   `path:"/system/log/operation" method:"get" tags:"System-Log" summary:"操作日志"`
	Page     int    `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"20" dc:"每页条数"`
	Keyword  string `json:"keyword" in:"query" dc:"搜索关键词"`
}

type SystemOperationLogListRes struct {
	g.Meta   `mime:"application/json"`
	List     interface{} `json:"list" dc:"日志列表"`
	Total    int         `json:"total" dc:"总数"`
	Page     int         `json:"page" dc:"当前页"`
	PageSize int         `json:"pageSize" dc:"每页条数"`
}

type SystemOperationLogDetailReq struct {
	g.Meta `path:"/system/log/operation/detail" method:"get" tags:"System-Log" summary:"操作日志详情"`
	Id     int64 `json:"id" in:"query" v:"required" dc:"日志ID"`
}

type SystemOperationLogDetailRes struct {
	g.Meta `mime:"application/json"`
	Data   interface{} `json:"data" dc:"日志详情"`
}

type SystemLoginLogListReq struct {
	g.Meta   `path:"/system/log/login" method:"get" tags:"System-Log" summary:"登录日志"`
	Page     int `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize int `json:"pageSize" in:"query" d:"20" dc:"每页条数"`
}

type SystemLoginLogListRes struct {
	g.Meta   `mime:"application/json"`
	List     interface{} `json:"list" dc:"日志列表"`
	Total    int         `json:"total" dc:"总数"`
	Page     int         `json:"page" dc:"当前页"`
	PageSize int         `json:"pageSize" dc:"每页条数"`
}
