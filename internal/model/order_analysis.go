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
	TimeType string            `json:"timeType" dc:"时间类型"`
	StartDate string          `json:"startDate" dc:"开始日期"`
	EndDate   string          `json:"endDate" dc:"结束日期"`
	GroupStats []GroupStatItem `json:"groupStats" dc:"小组统计列表"`
	ShopStats  []ShopStatItem  `json:"shopStats" dc:"店铺统计列表"`
}
