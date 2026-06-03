package instantShopping_trade_freightUpdate_request

import (
	"doudian.com/open/sdk_golang/api/instantShopping_trade_freightUpdate/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type InstantShoppingTradeFreightUpdateRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *InstantShoppingTradeFreightUpdateParam 
}
func (c *InstantShoppingTradeFreightUpdateRequest) GetUrlPath() string{
	return "/instantShopping/trade/freightUpdate"
}


func New() *InstantShoppingTradeFreightUpdateRequest{
	request := &InstantShoppingTradeFreightUpdateRequest{
		Param: &InstantShoppingTradeFreightUpdateParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *InstantShoppingTradeFreightUpdateRequest) Execute(accessToken *doudian_sdk.AccessToken) (*instantShopping_trade_freightUpdate_response.InstantShoppingTradeFreightUpdateResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &instantShopping_trade_freightUpdate_response.InstantShoppingTradeFreightUpdateResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *InstantShoppingTradeFreightUpdateRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *InstantShoppingTradeFreightUpdateRequest) GetParams() *InstantShoppingTradeFreightUpdateParam{
	return c.Param
}


type Template struct {
	// 模板名称
	TemplateName *string `json:"template_name"`
	// 解析column中calculate_type字段：计价方式-1.按重量计价 2.按数量计价
	CalculateType *int64 `json:"calculate_type"`
	// 发货城市id
	ProductCity *int64 `json:"product_city"`
	// 发货省份id
	ProductProvince *int64 `json:"product_province"`
	// 固定运费金额（单位：分）
	FixedAmount *int64 `json:"fixed_amount"`
	// 计价方式 0:阶梯计价 1:固定运费 2:卖家包邮 6:混合履约模式；固定值传6
	RuleType *int64 `json:"rule_type"`
	// 模板id
	Id *int64 `json:"id"`
	// 固定值1
	TransferType *int64 `json:"transfer_type"`
}
type ChildrenItem struct {
	// 地址id，第一级是省份、第二级是城市、第三级是区、第四级是街道
	Id *int64 `json:"id"`
}
type ChildrenItem_5 struct {
	// 当前规则生效的地址，统一以List<Struct>结构返回，该结构为嵌套结构。对应的json格式为[{"id":"32","children":[{"id":"320500","children":[{"id":"320508","children":[{"id":"320508014"},{"id":"320508004"}]}]}]}] 注意：返回的为最新的四级地址版本（地址存储升级变更的可能，以最新的返回）
	Children []ChildrenItem `json:"children"`
	// 地址id，第一级是省份、第二级是城市、第三级是区、第四级是街道
	Id *int64 `json:"id"`
}
type ChildrenItem_4 struct {
	// 当前规则生效的地址，统一以List<Struct>结构返回，该结构为嵌套结构。对应的json格式为[{"id":"32","children":[{"id":"320500","children":[{"id":"320508","children":[{"id":"320508014"},{"id":"320508004"}]}]}]}] 注意：返回的为最新的四级地址版本（地址存储升级变更的可能，以最新的返回）
	Children []ChildrenItem_5 `json:"children"`
	// 地址id，第一级是省份、第二级是城市、第三级是区、第四级是街道
	Id *int64 `json:"id"`
}
type ProvinceInfosItem struct {
	// 地址id，第一级是省份、第二级是城市、第三级是区、第四级是街道
	Id *int64 `json:"id"`
	// 当前规则生效的地址，统一以List<Struct>结构返回，该结构为嵌套结构。对应的json格式为[{"id":"32","children":[{"id":"320500","children":[{"id":"320508","children":[{"id":"320508014"},{"id":"320508004"}]}]}]}] 注意：返回的为最新的四级地址版本（地址存储升级变更的可能，以最新的返回）
	Children []ChildrenItem_4 `json:"children"`
}
type ColumnsItem struct {
	// 续重价格(单位:元) 按重量计价必填 0.00-30.00之间的小数，小数点后两位
	AddWeightPrice *float64 `json:"add_weight_price"`
	// 续件(单位:个) 按数量计价必填 1-999的整数
	AddNum *int64 `json:"add_num"`
	// 满xx重量包邮 单位：kg
	OverWeight *float64 `json:"over_weight"`
	// 当前规则生效的地址，统一以List<Struct>结构返回，该结构为嵌套结构。对应的json格式为[{"id":"32","children":[{"id":"320500","children":[{"id":"320508","children":[{"id":"320508014"},{"id":"320508004"}]}]}]}] 注意：返回的为最新的四级地址版本（地址存储升级变更的可能，以最新的返回）
	ProvinceInfos []ProvinceInfosItem `json:"province_infos"`
	// 限售自动解除时间戳，单位秒
	EndTime *int64 `json:"end_time"`
	// 首件数量(单位:个) 按数量计价必填 1-999的整数
	FirstNum *int64 `json:"first_num"`
	// 首件价格(单位:元)按数量计价必填 0.00-30.00之间的小数，小数点后两位
	FirstNumPrice *float64 `json:"first_num_price"`
	// 是否默认计价方式(1:是；0:不是)
	IsDefault *int64 `json:"is_default"`
	// 是否限运规则
	IsLimited *bool `json:"is_limited"`
	// 最小商品金额限制条件
	MinSkuAmount *int64 `json:"min_sku_amount"`
	// 限售原因。枚举值 distance_shipping_cost_high：因配送距离导致运费过高 weight_shipping_cost_high：因商品重量导致运费过高 force_majeure：因不可抗力（如会议赛事、自然灾害）不配送 cooperation_express_not_deliver：合作快递不配送 cooperation_express_poor_service：合作快递该区域服务差 other：其他
	Reason *string `json:"reason"`
	// 计价方式-1.按重量计价 2.按数量计价
	CalculateType *int64 `json:"calculate_type"`
	// 最大商品金额限制条件
	MaxSkuAmount *int64 `json:"max_sku_amount"`
	// 运费规则类型，同template.rule_type字段;0:阶梯计价；1:固定运费；2:卖家包邮
	RuleType *int64 `json:"rule_type"`
	// 首重(单位:kg) 按重量计价必填 0.1-999.9之间的小数，小数点后一位
	FirstWeight *float64 `json:"first_weight"`
	// 续重(单位:kg) 按重量计价必填 0.1-999.9之间的小数，小数点后一位
	AddWeight *float64 `json:"add_weight"`
	// 固定运费金额（单位：分）
	FixedAmount *int64 `json:"fixed_amount"`
	// 是否条件包邮
	IsOverFree *bool `json:"is_over_free"`
	// 首重价格(单位:元) 按重量计价必填 0.00-30.00之间的小数，小数点后两位
	FirstWeightPrice *float64 `json:"first_weight_price"`
	// 续件价格(单位:元) 按数量计价必填 0.00-30.00之间的小数，小数点后两位
	AddNumPrice *float64 `json:"add_num_price"`
	// 满xx件包邮
	OverNum *int64 `json:"over_num"`
	// 满xx金额包邮 单位：分
	OverAmount *int64 `json:"over_amount"`
	// 履约模式：1：即配，2：次日达，3：全国送
	DeliveryFulfillmentMode int64 `json:"delivery_fulfillment_mode"`
}
type InstantShoppingTradeFreightUpdateParam struct {
	// 运费模板
	Template *Template `json:"template"`
	// 运费规则
	Columns []ColumnsItem `json:"columns"`
}
