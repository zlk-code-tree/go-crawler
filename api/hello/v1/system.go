package v1

import (
	"go-crawler/internal/model"

	"github.com/gogf/gf/v2/frame/g"
)

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

// --- System Dept APIs ---

type SystemDeptDetailReq struct {
	g.Meta `path:"/system/dept/detail" method:"get" tags:"System-Dept" summary:"部门详情"`
	Id     int64 `json:"id" in:"query" v:"required" dc:"部门ID"`
}

type SystemDeptDetailRes struct {
	g.Meta `mime:"application/json"`
	Data   interface{} `json:"data" dc:"部门详情"`
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

// --- System Goods APIs ---

type SystemGoodsListReq struct {
	g.Meta   `path:"/system/goods/list" method:"get" tags:"System-Goods" summary:"商品列表"`
	Page     int    `json:"page" in:"query" d:"1" dc:"页码"`
	PageSize int    `json:"pageSize" in:"query" d:"10" dc:"每页条数"`
	Keyword  string `json:"keyword" in:"query" dc:"搜索关键词（商品名称/编码）"`
}

type SystemGoodsListRes struct {
	g.Meta `mime:"application/json"`
	List   interface{} `json:"list" dc:"商品列表"`
	Total  int         `json:"total" dc:"总数"`
	Page   int         `json:"page" dc:"当前页"`
	PageSize int       `json:"pageSize" dc:"每页条数"`
}

type SystemGoodsDetailReq struct {
	g.Meta `path:"/system/goods/detail" method:"get" tags:"System-Goods" summary:"商品详情"`
	Id     int `json:"id" in:"query" v:"required" dc:"商品ID"`
}

type SystemGoodsDetailRes struct {
	g.Meta `mime:"application/json"`
	Data   interface{} `json:"data" dc:"商品详情"`
}

type SystemGoodsSaveReq struct {
	g.Meta     `path:"/system/goods/save" method:"post" tags:"System-Goods" summary:"新增/编辑商品（id=0新增，id>0编辑）"`
	Id        int     `json:"id" dc:"商品ID，编辑时必填"`
	GoodsName string  `json:"goods_name" v:"required" dc:"商品名称"`
	GoodsCode string  `json:"goods_code" v:"required" dc:"商品编码"`
	BasePrice float64 `json:"base_price" d:"0.00" dc:"商品底价"`
	Status    int     `json:"status" d:"1" dc:"状态：1正常 2下架 3删除"`
}

type SystemGoodsSaveRes struct {
	g.Meta `mime:"application/json"`
	Id     int64 `json:"id" dc:"商品ID"`
}

type SystemGoodsDeleteReq struct {
	g.Meta `path:"/system/goods/delete" method:"delete" tags:"System-Goods" summary:"删除商品"`
	Id     int `json:"id" in:"query" v:"required" dc:"商品ID"`
}

type SystemGoodsDeleteRes struct {
	g.Meta `mime:"application/json"`
}

type SystemGoodsExportReq struct {
	g.Meta  `path:"/system/goods/export" method:"get" tags:"System-Goods" summary:"导出商品Excel"`
	Keyword string `json:"keyword" in:"query" dc:"搜索关键词"`
}

type SystemGoodsExportRes struct {
	g.Meta `mime:"application/json"`
}

type SystemGoodsImportReq struct {
	g.Meta `path:"/system/goods/import" method:"post" tags:"System-Goods" summary:"导入商品Excel"`
}

type SystemGoodsImportRes struct {
	g.Meta `mime:"application/json"`
}

// --- System Order Analysis APIs ---

type SystemOrderAnalysisGroupStatsReq struct {
	g.Meta   `path:"/order-analysis/group-stats" method:"get" tags:"System-OrderAnalysis" summary:"小组维度统计"`
	TimeType string `json:"timeType" in:"query" d:"today" dc:"时间类型: today/yesterday/week/month"`
}

type SystemOrderAnalysisGroupStatsRes struct {
	g.Meta   `mime:"application/json"`
	TimeType string        `json:"timeType" dc:"时间类型"`
	StartDate string       `json:"startDate" dc:"开始日期"`
	EndDate   string       `json:"endDate" dc:"结束日期"`
	Stats     interface{}  `json:"stats" dc:"统计数据"`
}

type SystemOrderAnalysisShopStatsReq struct {
	g.Meta   `path:"/order-analysis/shop-stats" method:"get" tags:"System-OrderAnalysis" summary:"店铺维度统计"`
	TimeType string `json:"timeType" in:"query" d:"today" dc:"时间类型: today/yesterday/week/month"`
	GroupId  int    `json:"groupId" in:"query" d:"0" dc:"小组ID，0表示全部"`
}

type SystemOrderAnalysisShopStatsRes struct {
	g.Meta   `mime:"application/json"`
	TimeType string        `json:"timeType" dc:"时间类型"`
	StartDate string       `json:"startDate" dc:"开始日期"`
	EndDate   string       `json:"endDate" dc:"结束日期"`
	Stats     interface{}  `json:"stats" dc:"统计数据"`
}

type SystemOrderAnalysisGroupListReq struct {
	g.Meta `path:"/order-analysis/groups" method:"get" tags:"System-OrderAnalysis" summary:"获取小组列表"`
}

type SystemOrderAnalysisGroupListRes struct {
	g.Meta `mime:"application/json"`
	Data   interface{} `json:"data" dc:"小组列表"`
}

type SystemOrderAnalysisShopListReq struct {
	g.Meta     `path:"/order-analysis/shops" method:"get" tags:"System-OrderAnalysis" summary:"获取店铺列表"`
	GroupName string `json:"groupName" in:"query" d:"" dc:"小组名称，为空则获取所有店铺"`
}

type SystemOrderAnalysisShopListRes struct {
	g.Meta `mime:"application/json"`
	Data   interface{} `json:"data" dc:"店铺列表"`
}

type SystemOrderAnalysisResultReq struct {
	g.Meta   `path:"/order-analysis/result" method:"get" tags:"System-OrderAnalysis" summary:"获取完整分析结果"`
	TimeType string `json:"timeType" in:"query" d:"today" dc:"时间类型: today/yesterday/week/month"`
	GroupId  int    `json:"groupId" in:"query" d:"0" dc:"小组ID，0表示全部"`
}

type SystemOrderAnalysisResultRes struct {
	g.Meta    `mime:"application/json"`
	TimeType  string       `json:"timeType" dc:"时间类型"`
	StartDate string       `json:"startDate" dc:"开始日期"`
	EndDate   string       `json:"endDate" dc:"结束日期"`
	GroupStats interface{} `json:"groupStats" dc:"小组统计"`
	ShopStats  interface{} `json:"shopStats" dc:"店铺统计"`
}

// --- System Order Analysis V2 APIs ---

// OverviewReq 统计概览请求
type SystemOrderAnalysisOverviewReq struct {
	g.Meta    `path:"/order-analysis/overview" method:"get" tags:"System-OrderAnalysis" summary:"统计概览"`
	Preset    string `json:"preset" in:"query" d:"today" dc:"预设时间: today/yesterday/last7/last30/thisMonth/all"`
	StartDate string `json:"startDate" in:"query" d:"" dc:"自定义开始日期(与preset二选一)"`
	EndDate   string `json:"endDate" in:"query" d:"" dc:"自定义结束日期(与preset二选一)"`
}

type SystemOrderAnalysisOverviewRes struct {
	g.Meta `mime:"application/json"`
	// 当前选中期间统计
	Period struct {
		OrderCount    int     `json:"orderCount" dc:"有效订单数"`
		PayAmount     float64 `json:"payAmount" dc:"订单应付金额"`
		CostAmount    float64 `json:"costAmount" dc:"货款"`
		Commission    float64 `json:"commission" dc:"佣金"`
		RefundCount   int     `json:"refundCount" dc:"退款订单数"`
		RefundAmount  float64 `json:"refundAmount" dc:"退款金额"`
	} `json:"period" dc:"期间统计"`
	// 历史累计
	Total struct {
		OrderCount int     `json:"orderCount" dc:"累计订单数"`
		PayAmount  float64 `json:"payAmount" dc:"累计应付金额"`
		Commission float64 `json:"commission" dc:"累计佣金"`
	} `json:"total" dc:"历史累计"`
}

// SalesTrendReq 销售额趋势请求
type SystemOrderAnalysisSalesTrendReq struct {
	g.Meta    `path:"/order-analysis/sales-trend" method:"get" tags:"System-OrderAnalysis" summary:"销售额趋势"`
	Preset    string `json:"preset" in:"query" d:"last7" dc:"预设时间: last7/last30/thisMonth"`
	StartDate string `json:"startDate" in:"query" d:"" dc:"自定义开始日期"`
	EndDate   string `json:"endDate" in:"query" d:"" dc:"自定义结束日期"`
}

type SystemOrderAnalysisSalesTrendRes struct {
	g.Meta `mime:"application/json"`
	// 当前期趋势
	Current   interface{} `json:"current" dc:"当前期数据"`
	// 对比期趋势(上月同期)
	Compare   interface{} `json:"compare" dc:"对比期数据"`
	StartDate string      `json:"startDate" dc:"当前期开始日期"`
	EndDate   string      `json:"endDate" dc:"当前期结束日期"`
}

// RankReq 排名榜单请求
type SystemOrderAnalysisRankReq struct {
	g.Meta    `path:"/order-analysis/rank" method:"get" tags:"System-OrderAnalysis" summary:"排名榜单"`
	Preset    string `json:"preset" in:"query" d:"thisMonth" dc:"预设时间: today/yesterday/last7/last30/thisMonth"`
	StartDate string `json:"startDate" in:"query" d:"" dc:"自定义开始日期"`
	EndDate   string `json:"endDate" in:"query" d:"" dc:"自定义结束日期"`
}

// RankItem 排名单项（小组/店铺）
type RankItem struct {
	Rank       int     `json:"rank" dc:"排名"`
	GroupName  string  `json:"groupName" dc:"小组名称"`
	ShopName   string  `json:"shopName" dc:"店铺名称"`
	ShopId     int     `json:"shopId,omitempty" dc:"店铺ID"`
	OrderCount int     `json:"orderCount" dc:"订单数"`
	GoodsCount int     `json:"goodsCount" dc:"商品数量"`
	PayAmount  float64 `json:"payAmount" dc:"应付金额"`
	Commission float64 `json:"commission" dc:"佣金"`
}

type SystemOrderAnalysisRankRes struct {
	g.Meta `mime:"application/json"`
	// 小组排名
	GroupRank interface{} `json:"groupRank" dc:"小组排名"`
	// 店铺排名
	ShopRank  interface{} `json:"shopRank" dc:"店铺排名"`
	StartDate string      `json:"startDate" dc:"开始日期"`
	EndDate   string      `json:"endDate" dc:"结束日期"`
}

// GroupExportReq 小组订单明细导出请求
type SystemOrderAnalysisGroupExportReq struct {
	g.Meta   `path:"/order-analysis/group-export" method:"get" tags:"System-OrderAnalysis" summary:"导出小组订单明细"`
	GroupName string `json:"groupName" in:"query" v:"required" dc:"小组名称"`
	TimeType  string `json:"timeType" in:"query" d:"today" dc:"时间类型: today/yesterday/last7/last30/thisMonth"`
	StartDate string `json:"startDate" in:"query" d:"" dc:"自定义开始日期（格式：2006-01-02），指定后timeType失效"`
	EndDate   string `json:"endDate" in:"query" d:"" dc:"自定义结束日期（格式：2006-01-02），指定后timeType失效"`
}

// GroupExportRes 小组订单明细导出响应
type SystemOrderAnalysisGroupExportRes struct {
	g.Meta `mime:"application/json"`
}

// ShopExportReq 店铺订单明细导出请求
type SystemOrderAnalysisShopExportReq struct {
	g.Meta   `path:"/order-analysis/export-shop-orders" method:"get" tags:"System-OrderAnalysis" summary:"导出店铺订单明细"`
	ShopId   int    `json:"shopId" in:"query" v:"required" dc:"店铺ID"`
	TimeType string `json:"timeType" in:"query" d:"today" dc:"时间类型: today/yesterday/last7/last30/thisMonth"`
}

// ShopExportRes 店铺订单明细导出响应
type SystemOrderAnalysisShopExportRes struct {
	g.Meta `mime:"application/json"`
}

// RankExportReq 排名数据导出请求
type SystemOrderAnalysisRankExportReq struct {
	g.Meta   `path:"/order-analysis/export-rank" method:"get" tags:"System-OrderAnalysis" summary:"导出排名数据"`
	Type     string `json:"type" in:"query" v:"required|in:group,shop" dc:"导出类型: group=小组排名,shop=店铺排名"`
	TimeType string `json:"timeType" in:"query" d:"thisMonth" dc:"时间类型: today/yesterday/last7/last30/thisMonth"`
}

// RankExportRes 排名数据导出响应
type SystemOrderAnalysisRankExportRes struct {
	g.Meta `mime:"application/json"`
}

// ShopTalentStatsReq 店铺达人明细请求
type SystemOrderAnalysisShopTalentStatsReq struct {
	g.Meta    `path:"/order-analysis/shop-talent-stats" method:"get" tags:"System-OrderAnalysis" summary:"店铺达人明细"`
	ShopName  string `json:"shopName" in:"query" v:"required" dc:"店铺名称"`
	Preset    string `json:"preset" in:"query" d:"thisMonth" dc:"预设时间: today/yesterday/last7/last30/thisMonth"`
	StartDate string `json:"startDate" in:"query" d:"" dc:"自定义开始日期"`
	EndDate   string `json:"endDate" in:"query" d:"" dc:"自定义结束日期"`
}

// ShopTalentStatsRes 店铺达人明细响应
type SystemOrderAnalysisShopTalentStatsRes struct {
	g.Meta    `mime:"application/json"`
	ShopName  string                     `json:"shopName" dc:"店铺名称"`
	GroupName string                     `json:"groupName" dc:"所属小组"`
	Stats     []model.ShopTalentStatItem `json:"stats" dc:"达人统计列表"`
	StartDate string                     `json:"startDate" dc:"开始日期"`
	EndDate   string                     `json:"endDate" dc:"结束日期"`
}

// --- System Order Analysis Org APIs ---

// OrgTreeReq 获取组织架构树请求
type SystemOrderAnalysisOrgTreeReq struct {
	g.Meta `path:"/order-analysis/org-tree" method:"get" tags:"System-OrderAnalysis" summary:"获取组织架构树"`
}

// OrgTreeRes 获取组织架构树响应
type SystemOrderAnalysisOrgTreeRes struct {
	g.Meta `mime:"application/json"`
	Tree   []model.OrgTreeNode `json:"tree" dc:"组织架构树"`
}

// OrgStatsReq 按组织统计请求
type SystemOrderAnalysisOrgStatsReq struct {
	g.Meta       `path:"/order-analysis/org-stats" method:"get" tags:"System-OrderAnalysis" summary:"按组织维度统计"`
	CompanyId    int64  `json:"companyId" in:"query" d:"0" dc:"公司ID（集团/分公司）"`
	DepartmentId int64  `json:"departmentId" in:"query" d:"0" dc:"部门ID"`
	LeaderName   string `json:"leaderName" in:"query" d:"" dc:"组长名称"`
	GroupType    string `json:"groupType" in:"query" d:"" dc:"组类型：invest/shop"`
	TalentId     int    `json:"talentId" in:"query" d:"0" dc:"达人ID（成员）"`
	Preset       string `json:"preset" in:"query" d:"today" dc:"预设时间: today/yesterday/last7/last30/thisMonth/all"`
	StartDate    string `json:"startDate" in:"query" d:"" dc:"自定义开始日期"`
	EndDate      string `json:"endDate" in:"query" d:"" dc:"自定义结束日期"`
}

// OrgStatsRes 按组织统计响应
type SystemOrderAnalysisOrgStatsRes struct {
	g.Meta       `mime:"application/json"`
	CurrentLevel string               `json:"currentLevel" dc:"当前筛选层级：company/department/leader/group/talent"`
	Stats        []model.OrgStatItem  `json:"stats" dc:"下级单位统计列表"`
	Summary      model.OrgStatItem    `json:"summary" dc:"当前层级汇总"`
}

