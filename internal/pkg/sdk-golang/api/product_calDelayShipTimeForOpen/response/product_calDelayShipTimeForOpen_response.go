package product_calDelayShipTimeForOpen_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductCalDelayShipTimeForOpenResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductCalDelayShipTimeForOpenData `json:"data"`
}
type ProductCalDelayShipTimeForOpenData struct {
	// 不能计算发货时效的错误码
	NotCanCalculateCode int64 `json:"not_can_calculate_code"`
	// 不能计算发货时效的错误文案
	NotCanCalculateMsg string `json:"not_can_calculate_msg"`
	// 发货时效计算说明文案
	DelayShipTimeTip string `json:"delay_ship_time_tip"`
	// 发货时效计算结果列表
	DelayShipTimeResults []DelayShipTimeResultsItem `json:"delay_ship_time_results"`
}
type DelayShipTimeDescItemsItem struct {
	// 支付周期描述
	PaymentPeriod string `json:"payment_period"`
	// 最晚发货时间描述
	ShipTimeDesc string `json:"ship_time_desc"`
	// 时效规则描述
	ShipTimeRuleDesc string `json:"ship_time_rule_desc"`
}
type DelayShipTimeResultsItem struct {
	// 发货时效描述标题
	DelayShipTimeDescTitle string `json:"delay_ship_time_desc_title"`
	// 时效计算结果项列表
	DelayShipTimeDescItems []DelayShipTimeDescItemsItem `json:"delay_ship_time_desc_items"`
}
