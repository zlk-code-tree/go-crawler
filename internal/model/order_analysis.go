package model

// OrderAnalysisReq 订单分析请求
type OrderAnalysisReq struct {
	TimeType string `json:"timeType" dc:"时间类型：today/yesterday/week/month"` // today=今日, yesterday=昨日, week=近7日, month=本月
	GroupId  int    `json:"groupId" dc:"小组ID，店铺维度筛选用"`                  // 店铺维度筛选用
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
	Rank       int     `json:"rank" dc:"排名"`
	Name       string  `json:"name" dc:"名称（小组名或店铺名）"`
	GroupName  string  `json:"groupName,omitempty" dc:"所属小组（仅店铺排名）"`
	OrderCount int     `json:"orderCount" dc:"订单数"`
	GoodsCount int     `json:"goodsCount" dc:"商品数"`
	PayAmount  float64 `json:"payAmount" dc:"销售额"`
	Commission float64 `json:"commission" dc:"佣金"`
}

// ShopInfo 店铺信息
type ShopInfo struct {
	ShopId    int    `json:"shopId" dc:"店铺ID"`
	ShopName  string `json:"shopName" dc:"店铺名称"`
	GroupName string `json:"groupName" dc:"所属小组"`
}
