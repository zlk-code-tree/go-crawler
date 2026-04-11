package model

import "time"

// SysUser 系统用户
type SysUser struct {
	Id          int64      `json:"id"`
	Username    string     `json:"username"`
	Password    string     `json:"-"`
	RealName    string     `json:"realName"`
	Nickname    string     `json:"nickname"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	Avatar      string     `json:"avatar"`
	Gender      int        `json:"gender"`
	Status      int        `json:"status"`
	DeptId      int64      `json:"deptId"`
	CompanyId   int64      `json:"companyId"`
	Position    string     `json:"position"`
	RoleId      int64      `json:"roleId"`
	Remark      string     `json:"remark"`
	LastLoginAt *time.Time `json:"lastLoginAt"`
	LastLoginIp string     `json:"lastLoginIp"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// SysRole 角色
type SysRole struct {
	Id        int64     `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	CompanyId *int64    `json:"companyId"`
	Type      int       `json:"type"`
	DataScope int       `json:"dataScope"`
	Status    int       `json:"status"`
	Sort      int       `json:"sort"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"createdAt"`
}

// SysPermission 权限
type SysPermission struct {
	Id        int64     `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Type      int       `json:"type"`
	ParentId  int64     `json:"parentId"`
	Path      string    `json:"path"`
	Component string    `json:"component"`
	Icon      string    `json:"icon"`
	Method    string    `json:"method"`
	Api       string    `json:"api"`
	ExtUrl    string    `json:"extUrl"`
	Status    int       `json:"status"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"createdAt"`
}

// SysCompany 公司
type SysCompany struct {
	Id        int64     `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	ParentId  int64     `json:"parentId"`
	Level     int       `json:"level"`
	Status    int       `json:"status"`
	Sort      int       `json:"sort"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SysDept 部门
type SysDept struct {
	Id        int64     `json:"id"`
	Name      string    `json:"name"`
	ParentId  int64     `json:"parentId"`
	Level     int       `json:"level"`
	Status    int       `json:"status"`
	Sort      int       `json:"sort"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// LoginReq 登录请求
type LoginReq struct {
	Username string `json:"username" v:"required#用户名不能为空"`
	Password string `json:"password" v:"required#密码不能为空"`
}

// LoginRes 登录响应
type LoginRes struct {
	AccessToken string `json:"accessToken"`
}

// UserInfoRes 用户信息响应 (适配 Vben Admin)
type UserInfoRes struct {
	UserId   int64    `json:"userId"`
	Username string   `json:"username"`
	RealName string   `json:"realName"`
	Avatar   string   `json:"avatar"`
	Roles    []string `json:"roles"`
	HomePath string   `json:"homePath"`
}

// MenuMeta 菜单元信息 (适配 Vben Admin RouteRecordStringComponent)
type MenuMeta struct {
	Title            string   `json:"title"`
	Icon             string   `json:"icon,omitempty"`
	Order            int      `json:"order,omitempty"`
	HideInMenu       bool     `json:"hideInMenu,omitempty"`
	HideInBreadcrumb bool     `json:"hideInBreadcrumb,omitempty"`
	Authority        []string `json:"authority,omitempty"`
	Link             string   `json:"link,omitempty"`
}

// MenuItem 菜单项 (适配 Vben Admin)
type MenuItem struct {
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	Component string     `json:"component"`
	Meta      MenuMeta   `json:"meta"`
	Children  []MenuItem `json:"children,omitempty"`
}
