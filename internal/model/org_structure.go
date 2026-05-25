package model

import "time"

// ============================================
// ID 编码：三表（sys_company / sys_department / trader）统一编码
// Company:  1 ~ 999,999（原 ID 不变）
// Department: 1,000,000 + 原 ID
// Trader:    2,000,000 + 原 ID
// ============================================
const (
	idOffsetDepartment = 1000000
	idOffsetTrader     = 2000000
)

// EncodeDeptID 将部门真实 ID 编码为全局唯一 ID
func EncodeDeptID(id int64) int64 { return id + idOffsetDepartment }

// EncodeTraderID 将投手真实 ID 编码为全局唯一 ID
func EncodeTraderID(id int64) int64 { return id + idOffsetTrader }

// DecodeDeptID 将全局 ID 还原为部门真实 ID
func DecodeDeptID(id int64) int64 { return id - idOffsetDepartment }

// DecodeTraderID 将全局 ID 还原为投手真实 ID
func DecodeTraderID(id int64) int64 { return id - idOffsetTrader }

// IsCompanyID 判断是否是公司节点 ID
func IsCompanyID(id int64) bool { return id > 0 && id < idOffsetDepartment }

// IsDeptID 判断是否是部门节点 ID
func IsDeptID(id int64) bool { return id >= idOffsetDepartment && id < idOffsetTrader }

// IsTraderID 判断是否是投手节点 ID
func IsTraderID(id int64) bool { return id >= idOffsetTrader }

// ============================================
// 节点类型常量
// ============================================
const (
	OrgTypeCompany  = "company"
	OrgTypeGroup    = "group"
	OrgTypeEmployee = "employee"
)

// OrgStructure 组织架构节点（对外统一响应结构）
type OrgStructure struct {
	Id        int64     `json:"id"`
	ParentId  int64     `json:"parentId"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // company/group/employee
	Code      string    `json:"code"`
	Sort      int       `json:"sort"`
	Status    int       `json:"status"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// OrgTreeNode 组织树节点（用于前端展示）
type OrgTreeNode struct {
	Id       int64         `json:"id"`
	ParentId int64         `json:"parentId"`
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	Code     string        `json:"code,omitempty"`
	Children []OrgTreeNode `json:"children,omitempty"`
}

// OrgListReq 组织列表请求
type OrgListReq struct {
	ParentId int64  `json:"parentId"` // 父节点ID，0表示根节点
	Type     string `json:"type"`     // 节点类型筛选
	Keyword  string `json:"keyword"`  // 搜索关键词
}

// OrgSaveReq 保存组织节点请求
type OrgSaveReq struct {
	Id       int64  `json:"id"`       // 节点ID，编辑时必填
	ParentId int64  `json:"parentId"` // 父节点ID
	Name     string `json:"name" v:"required#节点名称不能为空"`
	Type     string `json:"type" v:"required#节点类型不能为空"`
	Code     string `json:"code"`
	Sort     int    `json:"sort"`
	Remark   string `json:"remark"`
}

// ShopOrgRelation 店铺组织关联
type ShopOrgRelation struct {
	Id        int64     `json:"id"`
	ShopId    int       `json:"shopId"`
	OrgId     int64     `json:"orgId"`
	CreatedAt time.Time `json:"createdAt"`
}

// ShopOrgBindReq 店铺绑定组织请求
type ShopOrgBindReq struct {
	ShopId int   `json:"shopId" v:"required#店铺ID不能为空"`
	OrgId  int64 `json:"orgId" v:"required#组织节点ID不能为空"`
}

// OrgFilterReq 组织筛选请求（用于订单统计）
type OrgFilterReq struct {
	TimeType  string `json:"timeType"`  // 时间类型
	CompanyId int64  `json:"companyId"` // 公司ID
	BranchId  int64  `json:"branchId"`  // 分公司ID
	ManagerId int64  `json:"managerId"` // 组长ID
	GroupId   int64  `json:"groupId"`   // 组ID
	EmployeeId int64 `json:"employeeId"` // 成员ID
	PlatformId int64 `json:"platformId"` // 平台ID
	AccountId  int64 `json:"accountId"`  // 账号ID
	Preset    string `json:"preset"`    // 预设时间
	StartDate string `json:"startDate"` // 开始日期
	EndDate   string `json:"endDate"`   // 结束日期
}

// OrgStatItem 组织统计项（通用）
type OrgStatItem struct {
	OrgId         string  `json:"orgId"`         // 组织ID
	OrgName       string  `json:"orgName"`       // 组织名称
	OrgType       string  `json:"orgType"`       // 组织类型
	OrderCount    int     `json:"orderCount"`    // 有效订单数
	ProductNum    int     `json:"productNum"`    // 商品数量合计
	PayAmount     float64 `json:"payAmount"`     // 订单应付金额
	CostAmount    float64 `json:"costAmount"`    // 货款（代发价）
	ReserveAmount float64 `json:"reserveAmount"` // 留底
	Commission    float64 `json:"commission"`    // 佣金
}

// OrgStatsRes 按组织统计响应
type OrgStatsRes struct {
	CurrentLevel string        `json:"currentLevel"` // 当前筛选层级
	CurrentName  string        `json:"currentName"`  // 当前层级名称
	Stats        []OrgStatItem `json:"stats"`        // 下级单位统计列表
	Summary      OrgStatItem   `json:"summary"`      // 当前层级汇总
}

// StatsSummary 佣金统计汇总
type StatsSummary struct {
	OrderCount    int     `json:"orderCount" dc:"有效订单数"`
	PayAmount     float64 `json:"payAmount" dc:"订单应付金额（销售额）"`
	CostAmount    float64 `json:"costAmount" dc:"货款（代发价）"`
	ReserveAmount float64 `json:"reserveAmount" dc:"留底"`
	Commission    float64 `json:"commission" dc:"佣金"`
}

// OrgStatsNode 组织节点统计（树形结构）
type OrgStatsNode struct {
	OrgId    int64          `json:"orgId" dc:"节点ID"`
	OrgName  string         `json:"orgName" dc:"节点名称"`
	OrgType  string         `json:"orgType" dc:"节点类型"`
	Summary  StatsSummary   `json:"summary" dc:"该节点汇总统计"`
	Children []OrgStatsNode `json:"children,omitempty" dc:"直接子节点统计"`
}

// CommissionStatItem 达人佣金统计项
type CommissionStatItem struct {
	TraderId   int64   `json:"traderId" dc:"投手ID"`
	TraderName string  `json:"name" dc:"投手名称"`
	OrderCount int     `json:"orderCount" dc:"有效订单数"`
	ProductNum int     `json:"productCount" dc:"商品数量"`
	PayAmount  float64 `json:"payableAmount" dc:"应付金额（销售额）"`
	Commission float64 `json:"commission" dc:"佣金"`
}

// CommissionStatsRes 达人佣金统计响应
type CommissionStatsRes struct {
	OrgId     int64                `json:"orgId" dc:"节点ID"`
	OrgName   string               `json:"orgName" dc:"节点名称"`
	Summary   StatsSummary         `json:"summary" dc:"汇总统计"`
	Items     []CommissionStatItem `json:"items" dc:"达人佣金列表"`
	Total     int                  `json:"total" dc:"总数"`
	StartDate string               `json:"startDate" dc:"开始日期"`
	EndDate   string               `json:"endDate" dc:"结束日期"`
}
