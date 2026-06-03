package freightTemplate_create_request

import (
	"doudian.com/open/sdk_golang/api/freightTemplate_create/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type FreightTemplateCreateRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *FreightTemplateCreateParam 
}
func (c *FreightTemplateCreateRequest) GetUrlPath() string{
	return "/freightTemplate/create"
}


func New() *FreightTemplateCreateRequest{
	request := &FreightTemplateCreateRequest{
		Param: &FreightTemplateCreateParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *FreightTemplateCreateRequest) Execute(accessToken *doudian_sdk.AccessToken) (*freightTemplate_create_response.FreightTemplateCreateResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &freightTemplate_create_response.FreightTemplateCreateResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *FreightTemplateCreateRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *FreightTemplateCreateRequest) GetParams() *FreightTemplateCreateParam{
	return c.Param
}


type ColumnsItem struct {
	// 0:阶梯计价；1:固定运费；2:卖家包邮（仅在多种履约模式下生效该字段）
	RuleType *int64 `json:"rule_type"`
	// 固定运费（仅在多种履约模式下生效该字段）
	FixedAmount *int64 `json:"fixed_amount"`
	// 2:次日达；3:全国送；(仅在多种履约模式下生效该字段)；100中转规则
	DeliveryFulfillmentMode *int64 `json:"delivery_fulfillment_mode"`
	// 限售解除的时间戳，秒级
	EndTime *int64 `json:"end_time"`
	// 限售原因。枚举值distance_shipping_cost_high：因配送距离导致运费过高weight_shipping_cost_high：因商品重量导致运费过高force_majeure：因不可抗力（如会议赛事、自然灾害）不配送cooperation_express_not_deliver：合作快递不配送cooperation_express_poor_service：合作快递该区域服务差other：其他
	Reason *string `json:"reason"`
	// 首重(单位:kg) 按重量计价必填 0.1-999.9之间的小数，小数点后一位
	FirstWeight *float64 `json:"first_weight"`
	// 首重价格(单位:元) 按重量计价必填 0.00-100.00之间的小数，小数点后两位
	FirstWeightPrice *float64 `json:"first_weight_price"`
	// 首件数量(单位:个) 按数量计价必填 1-999的整数
	FirstNum *int64 `json:"first_num"`
	// 首件价格(单位:元)按数量计价必填 0.00-100.00之间的小数，小数点后两位
	FirstNumPrice *float64 `json:"first_num_price"`
	// 续重(单位:kg) 按重量计价必填 0.1-999.9之间的小数，小数点后一位
	AddWeight *float64 `json:"add_weight"`
	// 续重价格(单位:元) 按重量计价必填 0.00-100.00之间的小数，小数点后两位
	AddWeightPrice *float64 `json:"add_weight_price"`
	// 续件(单位:个) 按数量计价必填 1-999的整数
	AddNum *int64 `json:"add_num"`
	// 续件价格(单位:元) 按数量计价必填 0.00-100.00之间的小数，小数点后两位
	AddNumPrice *float64 `json:"add_num_price"`
	// 是否默认计价方式(1:是；0:不是)
	IsDefault *int64 `json:"is_default"`
	// 是否限运规则
	IsLimited *bool `json:"is_limited"`
	// 当前规则生效的地址，非默认规则必填。map<i64, map<i64, map<i64, list<i64>>>>的json格式，省->市->区->街道，填至选择到的层级即可，仅限售规则支持四级街道
	RuleAddress *string `json:"rule_address"`
	// 是否包邮规则
	IsOverFree *bool `json:"is_over_free"`
	// 满xx重量包邮(单位:kg)0.1-10.0之间的小数，小数点后一位
	OverWeight *float64 `json:"over_weight"`
	// 满xx金额包邮(单位:分)10-99900的整数
	OverAmount *int64 `json:"over_amount"`
	// 满xx件包邮 1-10之间的整数
	OverNum *int64 `json:"over_num"`
	// 最小金额限制，单位分，不限制填-1
	MinSkuAmount *int64 `json:"min_sku_amount"`
	// 最大金额限制，单位分，不限制填-1
	MaxSkuAmount *int64 `json:"max_sku_amount"`
	// 当前规则生效的地址，统一以List<Struct>结构返回，该结构为嵌套结构。对应的json格式为[{"id":"32","children":[{"id":"320500","children":[{"id":"320508","children":[{"id":"320508014"},{"id":"320508004"}]}]}]}] 注意：返回的为最新的四级地址版本（地址存储升级变更的可能，以最新的返回）
	ProvinceInfos []ProvinceInfosItem `json:"province_infos"`
}
type FreightTemplateCreateParam struct {
	// 运费模板信息
	Template *Template `json:"template"`
	// 运费模板规则信息；每种类型模板可创建的规则类型: 阶梯计价模板-默认规则，普通计价规则，包邮规则，限运规则;固定运费模板-包邮规则，限运规则;固定运费模板-包邮规则，限运规则;包邮模板-限运规则;货到付款模板-限运规则
	Columns []ColumnsItem `json:"columns"`
	// 是否插入中转规则
	UpsertTransferRule *bool `json:"upsert_transfer_rule"`
}
type Template struct {
	// 模板名称
	TemplateName string `json:"template_name"`
	// 发货省份id
	ProductProvince int64 `json:"product_province"`
	// 发货城市id
	ProductCity int64 `json:"product_city"`
	// 计价方式-1.按重量 2.按数量；模板类型为1、2、3时，计价类型传2
	CalculateType int64 `json:"calculate_type"`
	// 快递方式-1.快递 目前仅支持1
	TransferType int64 `json:"transfer_type"`
	// 模板类型-0:阶梯计价 1:固定运费 2:卖家包邮 3:货到付款
	RuleType int64 `json:"rule_type"`
	// 固定运费金额(单位:分) 固定运费模板必填 1-9900之间的整数
	FixedAmount *int64 `json:"fixed_amount"`
}
type ChildrenItem struct {
	// 地址id，第一级是省份、第二级是城市、第三级是区、第四级是街道
	Id *int64 `json:"id"`
}
type ChildrenItem_5 struct {
	// 地址id，第一级是省份、第二级是城市、第三级是区、第四级是街道
	Id *int64 `json:"id"`
	// 下一级地址信息
	Children []ChildrenItem `json:"children"`
}
type ChildrenItem_4 struct {
	// 地址id，第一级是省份、第二级是城市、第三级是区、第四级是街道
	Id *int64 `json:"id"`
	// 下一级地址信息
	Children []ChildrenItem_5 `json:"children"`
}
type ProvinceInfosItem struct {
	// 地址id，第一级是省份、第二级是城市、第三级是区、第四级是街道
	Id *int64 `json:"id"`
	// 下一级地址信息
	Children []ChildrenItem_4 `json:"children"`
}
