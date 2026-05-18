package model

// OrderAnalysisReq 订单分析请求
type OrderAnalysisReq struct {
	TimeType string `json:"timeType" dc:"时间类型：today/yesterday/week/month"` // today=今日, yesterday=昨日, week=近7日, month=本月
	GroupId  int    `json:"groupId" dc:"小组ID，店铺维度筛选用"`                  // 店铺维度筛选用
}

// OrgFilterReq 组织筛选请求
type OrgFilterReq struct {
	TimeType     string `json:"timeType"`     // 时间类型
	CompanyId    int64  `json:"companyId"`    // 公司ID（集团/分公司）
	DepartmentId int64  `json:"departmentId"` // 部门ID
	LeaderName   string `json:"leaderName"`   // 组长名称
	GroupType    string `json:"groupType"`    // 组类型：invest/shop
	TalentId     int    `json:"talentId"`     // 达人ID（成员）
	Preset       string `json:"preset"`       // 预设时间
	StartDate    string `json:"startDate"`    // 开始日期
	EndDate      string `json:"endDate"`      // 结束日期
}

// OrgTreeNode 组织树节点
type OrgTreeNode struct {
	Id       int           `json:"id"`
	Name     string        `json:"name"`
	Type     string        `json:"type"` // company/department/leader/invest/shop/member
	Children []OrgTreeNode `json:"children,omitempty"`
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
	CurrentLevel string       `json:"currentLevel"` // 当前筛选层级：company/department/leader/group/talent
	Stats        []OrgStatItem `json:"stats"`        // 下级单位统计列表
	Summary      OrgStatItem   `json:"summary"`      // 当前层级汇总
}

// GroupStatItem 小组统计项
type GroupStatItem struct {
	GroupName     string  `json:"groupName" dc:"小组名称"`
	OrderCount    int     `json:"orderCount" dc:"有效订单数"`
	ProductNum    int     `json:"productNum" dc:"商品数量合计"`
	PayAmount     float64 `json:"payAmount" dc:"订单应付金额"`
	CostAmount    float64 `json:"costAmount" dc:"货款（代发价）"`
	ReserveAmount float64 `json:"reserveAmount" dc:"留底"`
	Commission    float64 `json:"commission" dc:"佣金"`
}

// ShopStatItem 店铺统计项
type ShopStatItem struct {
	ShopId        int     `json:"shopId" dc:"店铺ID"`
	ShopName      string  `json:"shopName" dc:"店铺名称"`
	GroupName     string  `json:"groupName" dc:"所属小组"`
	OrderCount    int     `json:"orderCount" dc:"有效订单数"`
	ProductNum    int     `json:"productNum" dc:"商品数量合计"`
	PayAmount     float64 `json:"payAmount" dc:"订单应付金额"`
	CostAmount    float64 `json:"costAmount" dc:"货款（代发价）"`
	ReserveAmount float64 `json:"reserveAmount" dc:"留底"`
	Commission    float64 `json:"commission" dc:"佣金"`
}

// OrderAnalysisResult 订单分析结果
type OrderAnalysisResult struct {
	TimeType   string          `json:"timeType" dc:"时间类型"`
	StartDate  string          `json:"startDate" dc:"开始日期"`
	EndDate    string          `json:"endDate" dc:"结束日期"`
	GroupStats []GroupStatItem `json:"groupStats" dc:"小组统计列表"`
	ShopStats  []ShopStatItem  `json:"shopStats" dc:"店铺统计列表"`
}

// OverviewPeriod 概览期间统计
type OverviewPeriod struct {
	OrderCount   int     `json:"orderCount" dc:"订单数"`
	PayAmount    float64 `json:"payAmount" dc:"销售额"`
	CostAmount   float64 `json:"costAmount" dc:"成本"`
	Commission   float64 `json:"commission" dc:"佣金"`
	RefundCount  int     `json:"refundCount" dc:"退款订单数"`
	RefundAmount float64 `json:"refundAmount" dc:"退款金额"`
}

// OverviewTotal 概览累计统计
type OverviewTotal struct {
	OrderCount int     `json:"orderCount" dc:"累计订单数"`
	PayAmount  float64 `json:"payAmount" dc:"累计销售额"`
	Commission float64 `json:"commission" dc:"累计佣金"`
}

// DailyTrend 每日趋势数据
type DailyTrend struct {
	Date       string  `json:"date" dc:"日期"`
	OrderCount int     `json:"orderCount" dc:"订单数"`
	PayAmount  float64 `json:"payAmount" dc:"销售额"`
	Commission float64 `json:"commission" dc:"佣金"`
}

// RankItem 排名项
type RankItem struct {
	Rank       int                   `json:"rank" dc:"排名"`
	Name       string                `json:"name" dc:"名称（小组名或店铺名）"`
	GroupName  string                `json:"groupName,omitempty" dc:"所属小组（仅店铺排名）"`
	ShopId     int                   `json:"shopId,omitempty" dc:"店铺ID（仅店铺排名）"`
	OrderCount int                   `json:"orderCount" dc:"订单数"`
	GoodsCount int                   `json:"goodsCount" dc:"商品数"`
	PayAmount  float64               `json:"payAmount" dc:"销售额"`
	Commission float64               `json:"commission" dc:"佣金"`
	Details    []ShopTalentStatItem  `json:"details,omitempty" dc:"达人明细（仅店铺排名）"`
}

// ShopInfo 店铺信息
type ShopInfo struct {
	ShopId    int    `json:"shopId" dc:"店铺ID"`
	ShopName  string `json:"shopName" dc:"店铺名称"`
	GroupName string `json:"groupName" dc:"所属小组"`
}

// ShopTalentStatItem 店铺下达人统计项
type ShopTalentStatItem struct {
	TalentId    int     `json:"talentId" dc:"达人ID"`
	TalentName  string  `json:"talentName" dc:"达人昵称"`
	GroupName   string  `json:"groupName" dc:"所属小组"`
	OrderCount  int     `json:"orderCount" dc:"有效订单数"`
	GoodsCount  int     `json:"goodsCount" dc:"商品数量"`
	PayAmount   float64 `json:"payAmount" dc:"销售额"`
	Commission  float64 `json:"commission" dc:"佣金"`
}
