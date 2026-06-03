package instantShopping_trade_freightList_response

import (
	"doudian.com/open/sdk_golang/core"
)

type InstantShoppingTradeFreightListResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *InstantShoppingTradeFreightListData `json:"data"`
}
type Template struct {
	// 模板id
	Id int64 `json:"id"`
	// 模板名称
	TemplateName string `json:"template_name"`
	// 解析模板详情对应字段：固定运费金额（单位：分）
	FixedAmount int64 `json:"fixed_amount"`
	// 发货省id
	ProductProvince int64 `json:"product_province"`
	// 计价方式 6 混合履约模式
	RuleType int64 `json:"rule_type"`
	// 发货地市id
	ProductCity int64 `json:"product_city"`
	// 解析模板详情对应字段
	CalculateType int64 `json:"calculate_type"`
}
type ListItem struct {
	// 模板
	Template *Template `json:"template"`
}
type InstantShoppingTradeFreightListData struct {
	// 返回结果
	List []ListItem `json:"List"`
	// 模板数量
	Count int64 `json:"Count"`
}
