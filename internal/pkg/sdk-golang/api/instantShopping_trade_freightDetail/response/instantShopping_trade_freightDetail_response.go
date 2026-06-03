package instantShopping_trade_freightDetail_response

import (
	"doudian.com/open/sdk_golang/core"
)

type InstantShoppingTradeFreightDetailResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *Data `json:"data"`
}
type Template struct {
	// 发货省份id
	ProductProvince int64 `json:"product_province"`
	// 固定值1
	TransferType int64 `json:"transfer_type"`
	// 解析column中calculate_type
	CalculateType int64 `json:"calculate_type"`
	// 解析column中对应的字段，固定运费金额（单位：分）
	FixedAmount int64 `json:"fixed_amount"`
	// 模板名称
	TemplateName string `json:"template_name"`
	// 发货城市id
	ProductCity int64 `json:"product_city"`
	// 计价方式:固定为6，解析col
	RuleType int64 `json:"rule_type"`
	// 模板id
	Id int64 `json:"id"`
}
type ChildrenItem struct {
	// 地址id，第一级是省份、第二级是城市、第三级是区、第四级是街道
	Id int64 `json:"id"`
}
type ChildrenItem_6 struct {
	// 当前规则生效的地址，统一以List<Struct>结构返回，该结构为嵌套结构。对应的json格式为[{"id":"32","children":[{"id":"320500","children":[{"id":"320508","children":[{"id":"320508014"},{"id":"320508004"}]}]}]}] 注意：返回的为最新的四级地址版本（地址存储升级变更的可能，以最新的返回）
	Children []ChildrenItem `json:"children"`
	// 地址id，第一级是省份、第二级是城市、第三级是区、第四级是街道
	Id int64 `json:"id"`
}
type ChildrenItem_5 struct {
	// 当前规则生效的地址，统一以List<Struct>结构返回，该结构为嵌套结构。对应的json格式为[{"id":"32","children":[{"id":"320500","children":[{"id":"320508","children":[{"id":"320508014"},{"id":"320508004"}]}]}]}] 注意：返回的为最新的四级地址版本（地址存储升级变更的可能，以最新的返回）
	Children []ChildrenItem_6 `json:"children"`
	// 地址id，第一级是省份、第二级是城市、第三级是区、第四级是街道
	Id int64 `json:"id"`
}
type ProvinceInfosItem struct {
	// 地址id，第一级是省份、第二级是城市、第三级是区、第四级是街道
	Id int64 `json:"id"`
	// 当前规则生效的地址，统一以List<Struct>结构返回，该结构为嵌套结构。对应的json格式为[{"id":"32","children":[{"id":"320500","children":[{"id":"320508","children":[{"id":"320508014"},{"id":"320508004"}]}]}]}] 注意：返回的为最新的四级地址版本（地址存储升级变更的可能，以最新的返回）
	Children []ChildrenItem_5 `json:"children"`
}
type ColumnsItem struct {
	// 满xx金额包邮 单位：分
	OverAmount int64 `json:"over_amount"`
	// 最小商品金额限制条件
	MinSkuAmount int64 `json:"min_sku_amount"`
	// 运费规则类型，同template.rule_type字段；模板类型-0:阶梯计价 1:固定运费 2:卖家包邮 3:货到付款
	RuleType int64 `json:"rule_type"`
	// 首件价格(单位:元)按数量计价必填 0.00-30.00之间的小数，小数点后两位
	FirstNumPrice float64 `json:"first_num_price"`
	// 满xx重量包邮 单位：kg
	OverWeight float64 `json:"over_weight"`
	// 最大商品金额限制条件
	MaxSkuAmount int64 `json:"max_sku_amount"`
	// 首件数量(单位:个) 按数量计价必填 1-999的整数
	FirstNum int64 `json:"first_num"`
	// 首重价格(单位:元) 按重量计价必填 0.00-30.00之间的小数，小数点后两位
	FirstWeightPrice float64 `json:"first_weight_price"`
	// 续件价格(单位:元) 按数量计价必填 0.00-30.00之间的小数，小数点后两位
	AddNumPrice float64 `json:"add_num_price"`
	// 是否限运规则
	IsLimited bool `json:"is_limited"`
	// 限售自动解除时间戳，单位秒
	EndTime int64 `json:"end_time"`
	// 首重(单位:kg) 按重量计价必填 0.1-999.9之间的小数，小数点后一位
	FirstWeight float64 `json:"first_weight"`
	// 续件(单位:个) 按数量计价必填 1-999的整数
	AddNum int64 `json:"add_num"`
	// 履约模式：1：即配，2：次日达，3：全国送
	DeliveryFulfillmentMode int64 `json:"delivery_fulfillment_mode"`
	// 限售原因。枚举值 distance_shipping_cost_high：因配送距离导致运费过高 weight_shipping_cost_high：因商品重量导致运费过高 force_majeure：因不可抗力（如会议赛事、自然灾害）不配送 cooperation_express_not_deliver：合作快递不配送 cooperation_express_poor_service：合作快递该区域服务差 other：其他
	Reason string `json:"reason"`
	// 续重(单位:kg) 按重量计价必填 0.1-999.9之间的小数，小数点后一位
	AddWeight float64 `json:"add_weight"`
	// 续重价格(单位:元) 按重量计价必填 0.00-30.00之间的小数，小数点后两位
	AddWeightPrice float64 `json:"add_weight_price"`
	// 计价方式-1.按重量计价 2.按数量计价
	CalculateType int64 `json:"calculate_type"`
	// 满xx件包邮
	OverNum int64 `json:"over_num"`
	// 是否条件包邮
	IsOverFree bool `json:"is_over_free"`
	// 当前规则生效的地址，统一以List<Struct>结构返回，该结构为嵌套结构。对应的json格式为[{"id":"32","children":[{"id":"320500","children":[{"id":"320508","children":[{"id":"320508014"},{"id":"320508004"}]}]}]}] 注意：返回的为最新的四级地址版本（地址存储升级变更的可能，以最新的返回）
	ProvinceInfos []ProvinceInfosItem `json:"province_infos"`
	// 是否默认计价方式(1:是；0:不是)
	IsDefault int64 `json:"is_default"`
	// 固定运费金额（单位：分）
	FixedAmount int64 `json:"fixed_amount"`
}
type Data struct {
	// 模板
	Template *Template `json:"template"`
	// 运费规则
	Columns []ColumnsItem `json:"columns"`
}
type InstantShoppingTradeFreightDetailData struct {
	// 返回值
	Data *Data `json:"data"`
}
