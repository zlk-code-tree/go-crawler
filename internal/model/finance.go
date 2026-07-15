package model

import "time"

// ============================================
// 财务管理 - 达人佣金统计相关模型
// ============================================

// FinanceDailyStats 日佣金统计数据
type FinanceDailyStats struct {
	Id                   int64     `json:"id"`
	StatDate             string    `json:"statDate"`
	CompanyId            int64     `json:"companyId"`
	DepartmentId         int64     `json:"departmentId"`
	TraderId             int64     `json:"traderId"`
	AccountId            int       `json:"accountId"`
	Platform             string    `json:"platform"`
	TotalOrderCount      int       `json:"totalOrderCount"`
	OrderCount           int       `json:"orderCount"`
	InvalidOrderCount    int       `json:"invalidOrderCount"`
	ProductNum           int       `json:"productNum"`
	PayAmount            float64   `json:"payAmount"`
	CostAmount           float64   `json:"costAmount"`
	ReserveAmount        float64   `json:"reserveAmount"`
	Commission           float64   `json:"commission"`
	RefundCostAmount     float64   `json:"refundCostAmount"`
	RefundOrderCount     int       `json:"refundOrderCount"`
	ShopServiceFee       float64   `json:"shopServiceFee"`
	AdCostAmount         float64   `json:"adCostAmount"`
	ValidOrderCostMatch  int       `json:"validOrderCostMatch"`
	CommissionOrderCount int       `json:"commissionOrderCount"`
	RealtimeProfit       float64   `json:"realtimeProfit"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// ============================================
// 请求结构
// ============================================

// FinanceStatsReq 佣金统计请求
type FinanceStatsReq struct {
	TimeType  string `json:"timeType"`              // 时间类型：today/yesterday/week/month/custom
	StartDate string `json:"startDate"`             // 开始日期（custom时必填）
	EndDate   string `json:"endDate"`               // 结束日期（custom时必填）
	NodeId    int64  `json:"nodeId"`                // 组织节点ID
	NodeType  string `json:"nodeType" v:"required"` // 节点类型：company/department/trader
	Platform  string `json:"platform"`              // 平台筛选：douyin/kuaishou/shipinhao（空=全部）
}

// FinanceChildrenReq 获取下级节点请求
type FinanceChildrenReq struct {
	NodeId    int64  `json:"nodeId" v:"required"`
	NodeType  string `json:"nodeType" v:"required|in:company,department,trader"`
	TimeType  string `json:"timeType"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Platform  string `json:"platform"` // 平台筛选：douyin/kuaishou/shipinhao（空=全部）
}

// FinanceTraderDetailReq 投手详情统计请求
type FinanceTraderDetailReq struct {
	TraderId  int64  `json:"traderId" v:"required"`
	TimeType  string `json:"timeType"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

// FinanceShopDetailReq 账号明细统计请求
type FinanceShopDetailReq struct {
	AccountId int    `json:"accountId" v:"required"`
	TimeType  string `json:"timeType"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

// ============================================
// 响应结构
// ============================================

// FinanceStatsRes 佣金统计响应
type FinanceStatsRes struct {
	TimeType  string              `json:"timeType"`
	StartDate string              `json:"startDate"`
	EndDate   string              `json:"endDate"`
	Summary   FinanceStatsSummary `json:"summary"`
}

// FinanceStatsSummary 财务统计汇总
type FinanceStatsSummary struct {
	TotalOrderCount      int     `json:"totalOrderCount"`
	OrderCount           int     `json:"orderCount"`
	InvalidOrderCount    int     `json:"invalidOrderCount"`
	ProductNum           int     `json:"productNum"`
	PayAmount            float64 `json:"payAmount"`
	CostAmount           float64 `json:"costAmount"`
	ReserveAmount        float64 `json:"reserveAmount"`
	Commission           float64 `json:"commission"`
	RefundCostAmount     float64 `json:"refundCostAmount"`
	RefundOrderCount     int     `json:"refundOrderCount"`
	ShopServiceFee       float64 `json:"shopServiceFee"`
	AdCostAmount         float64 `json:"adCostAmount"`
	ValidOrderCostMatch  int     `json:"validOrderCostMatch"`
	CommissionOrderCount int     `json:"commissionOrderCount"`
	RealtimeProfit       float64 `json:"realtimeProfit"`
}

// FinanceChildrenRes 下级节点响应
type FinanceChildrenRes struct {
	CurrentNode NodeInfo        `json:"currentNode"`
	Children    []NodeStatsItem `json:"children"`
}

// NodeInfo 节点信息
type NodeInfo struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// NodeStatsItem 节点统计项
type NodeStatsItem struct {
	Id                int64   `json:"id"`
	Name              string  `json:"name"`
	Type              string  `json:"type"`
	DouyinNickname    string  `json:"douyinNickname"`
	Uid               string  `json:"uid"`
	TotalOrderCount   int     `json:"totalOrderCount"`
	OrderCount        int     `json:"orderCount"`
	InvalidOrderCount int     `json:"invalidOrderCount"`
	ProductNum        int     `json:"productNum"`
	PayAmount         float64 `json:"payAmount"`
	CostAmount        float64 `json:"costAmount"`
	ReserveAmount     float64 `json:"reserveAmount"`
	Commission        float64 `json:"commission"`
	RefundCostAmount  float64 `json:"refundCostAmount"`
	RefundOrderCount  int     `json:"refundOrderCount"`
	ShopServiceFee    float64 `json:"shopServiceFee"`
	AdCostAmount      float64 `json:"adCostAmount"`
	RealtimeProfit    float64 `json:"realtimeProfit"`
}

// FinanceTraderDetailRes 投手详情响应
type FinanceTraderDetailRes struct {
	TraderId    int64               `json:"traderId"`
	TraderName  string              `json:"traderName"`
	CompanyName string              `json:"companyName"`
	DeptName    string              `json:"deptName"`
	TimeType    string              `json:"timeType"`
	StartDate   string              `json:"startDate"`
	EndDate     string              `json:"endDate"`
	Summary     FinanceStatsSummary `json:"summary"`
	ShopStats   []ShopStatsItem     `json:"shopStats"`
}

// ShopStatsItem 账号统计项
type ShopStatsItem struct {
	AccountId         int     `json:"accountId"`
	ShopName          string  `json:"shopName"`
	Uid               string  `json:"uid"`
	Platform          string  `json:"platform"`
	TotalOrderCount   int     `json:"totalOrderCount"`
	OrderCount        int     `json:"orderCount"`
	InvalidOrderCount int     `json:"invalidOrderCount"`
	ProductNum        int     `json:"productNum"`
	PayAmount         float64 `json:"payAmount"`
	CostAmount        float64 `json:"costAmount"`
	ReserveAmount     float64 `json:"reserveAmount"`
	Commission        float64 `json:"commission"`
	RefundCostAmount  float64 `json:"refundCostAmount"`
	RefundOrderCount  int     `json:"refundOrderCount"`
	ShopServiceFee    float64 `json:"shopServiceFee"`
	AdCostAmount      float64 `json:"adCostAmount"`
	RealtimeProfit    float64 `json:"realtimeProfit"`
}

// FinanceShopDetailRes 账号明细响应
type FinanceShopDetailRes struct {
	AccountId int                 `json:"accountId"`
	ShopName  string              `json:"shopName"`
	Uid       string              `json:"uid"`
	Platform  string              `json:"platform"`
	TimeType  string              `json:"timeType"`
	StartDate string              `json:"startDate"`
	EndDate   string              `json:"endDate"`
	Summary   FinanceStatsSummary `json:"summary"`
}

// FinanceTraderPlatformStatsReq 投手平台统计明细请求
type FinanceTraderPlatformStatsReq struct {
	TraderId  int64  `json:"traderId" in:"query" v:"required#投手ID不能为空"`
	TimeType  string `json:"timeType" in:"query" d:"today"`
	StartDate string `json:"startDate" in:"query"`
	EndDate   string `json:"endDate" in:"query"`
}

// FinanceTraderPlatformStatsRes 投手平台统计明细响应
type FinanceTraderPlatformStatsRes struct {
	TraderId      int64               `json:"traderId"`
	TraderName    string              `json:"traderName"`
	TimeType      string              `json:"timeType"`
	StartDate     string              `json:"startDate"`
	EndDate       string              `json:"endDate"`
	Summary       FinanceStatsSummary `json:"summary"`
	PlatformStats []PlatformStats     `json:"platformStats"`
}

// PlatformStats 单平台统计
type PlatformStats struct {
	Platform          string  `json:"platform"`
	PlatformName      string  `json:"platformName"`
	TotalOrderCount   int     `json:"totalOrderCount"`
	OrderCount        int     `json:"orderCount"`
	InvalidOrderCount int     `json:"invalidOrderCount"`
	ProductNum        int     `json:"productNum"`
	PayAmount         float64 `json:"payAmount"`
	CostAmount        float64 `json:"costAmount"`
	ReserveAmount     float64 `json:"reserveAmount"`
	Commission        float64 `json:"commission"`
	RefundCostAmount  float64 `json:"refundCostAmount"`
	RefundOrderCount  int     `json:"refundOrderCount"`
	ShopServiceFee    float64 `json:"shopServiceFee"`
	RealtimeProfit    float64 `json:"realtimeProfit"`
}

// ============================================
// 投手订单列表
// ============================================

// FinanceTraderOrdersReq 投手订单列表请求
type FinanceTraderOrdersReq struct {
	TraderId  int64  `json:"traderId" v:"required"`
	TimeType  string `json:"timeType"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Page      int    `json:"page" d:"1"`
	PageSize  int    `json:"pageSize" d:"20"`
}

// FinanceTraderOrdersRes 投手订单列表响应
type FinanceTraderOrdersRes struct {
	TraderId   int64               `json:"traderId"`
	TraderName string              `json:"traderName"`
	TimeType   string              `json:"timeType"`
	StartDate  string              `json:"startDate"`
	EndDate    string              `json:"endDate"`
	Summary    FinanceStatsSummary `json:"summary"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"pageSize"`
	Orders     []FinanceOrderItem  `json:"orders"`
}

// FinanceOrderItem 订单明细项
type FinanceOrderItem struct {
	MainOrderNo     string  `json:"mainOrderNo"`
	ProductName     string  `json:"productName"`
	ProductNum      int     `json:"productNum"`
	ProductPrice    float64 `json:"productPrice"`
	OrderPayAmount  float64 `json:"orderPayAmount"`
	CostAmount      float64 `json:"costAmount"`
	ReserveAmount   float64 `json:"reserveAmount"`
	Commission      float64 `json:"commission"`
	Platform        string  `json:"platform"`
	ShopName        string  `json:"shopName"`
	OrderStatus     string  `json:"orderStatus"`
	AfterSaleStatus string  `json:"afterSaleStatus"`
	PayFinishTime   string  `json:"payFinishTime"`
}

// ============================================
// 账号订单列表
// ============================================

// FinanceAccountOrdersReq 账号订单列表请求
type FinanceAccountOrdersReq struct {
	AccountId int    `json:"accountId" v:"required"`
	TimeType  string `json:"timeType"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Page      int    `json:"page" d:"1"`
	PageSize  int    `json:"pageSize" d:"20"`
}

// FinanceAccountOrdersRes 账号订单列表响应
type FinanceAccountOrdersRes struct {
	AccountId   int                 `json:"accountId"`
	AccountName string              `json:"accountName"`
	TimeType    string              `json:"timeType"`
	StartDate   string              `json:"startDate"`
	EndDate     string              `json:"endDate"`
	Summary     FinanceStatsSummary `json:"summary"`
	Total       int                 `json:"total"`
	Page        int                 `json:"page"`
	PageSize    int                 `json:"pageSize"`
	Orders      []FinanceOrderItem  `json:"orders"`
}
