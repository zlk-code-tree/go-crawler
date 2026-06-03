package address_addAftersaleStrategy_request

import (
	"doudian.com/open/sdk_golang/api/address_addAftersaleStrategy/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type AddressAddAftersaleStrategyRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *AddressAddAftersaleStrategyParam 
}
func (c *AddressAddAftersaleStrategyRequest) GetUrlPath() string{
	return "/address/addAftersaleStrategy"
}


func New() *AddressAddAftersaleStrategyRequest{
	request := &AddressAddAftersaleStrategyRequest{
		Param: &AddressAddAftersaleStrategyParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *AddressAddAftersaleStrategyRequest) Execute(accessToken *doudian_sdk.AccessToken) (*address_addAftersaleStrategy_response.AddressAddAftersaleStrategyResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &address_addAftersaleStrategy_response.AddressAddAftersaleStrategyResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *AddressAddAftersaleStrategyRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *AddressAddAftersaleStrategyRequest) GetParams() *AddressAddAftersaleStrategyParam{
	return c.Param
}


type AddressItem struct {
	// 必传。商家通过抖店后台-订单管理-物流工具-地址库管理或者通过三方服务商平台在抖店后台创建地址生成的ID，可以通过接入/address/list获取商家地址列表。
	Id *int64 `json:"id"`
	// 非必传。动态策略类型必传，静态策略类型不传。省ID，消费者收货地址对应的省ID，需要使用标准四级地址对应的省ID，通过/address/provinceList获取。
	ProvinceId *int64 `json:"province_id"`
	// 非必传。动态策略类型必传，静态策略类型不传。城市ID，消费者收货地址对应的市ID，需要使用标准四级地址对应的市ID，通过/address/getAreasByProvince获取是ID。
	CityId *int64 `json:"city_id"`
}
type AftersaleStrategy struct {
	// 非必传。添加策略时需绑定的商品ID列表，单次请求中不允超过100个商品ID。
	ToAddProductId []int64 `json:"to_add_product_id"`
	// 必传。策略类型。静态策略类型和用户收货地址无关，只是将商品和退货地址绑定，无论消费者收货地址在哪里，该策略绑定的商品全部退往同一地址。动态策略类型绑定的商品可以指定不同的消费者收货区域退往不同的退货地址，比如指定消费者收货地址在北京、天津、河北的订单退往华北仓，指定消费者收货地址在山东、江苏的订单退往华东仓。静态类型策略和动态类型策略都可以绑定多个商品。
	StrategyType *int32 `json:"strategy_type"`
	// 必传。策略名称，不允许超过10个字。
	StrategyName *string `json:"strategy_name"`
	// 必传。售后策略绑定的地址列表，
	Address []AddressItem `json:"address"`
}
type AddressAddAftersaleStrategyParam struct {
	// 售后策略信息
	AftersaleStrategy *AftersaleStrategy `json:"aftersale_strategy"`
}
