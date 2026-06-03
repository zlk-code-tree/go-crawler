package address_updateAftersaleStrategy_request

import (
	"doudian.com/open/sdk_golang/api/address_updateAftersaleStrategy/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type AddressUpdateAftersaleStrategyRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *AddressUpdateAftersaleStrategyParam 
}
func (c *AddressUpdateAftersaleStrategyRequest) GetUrlPath() string{
	return "/address/updateAftersaleStrategy"
}


func New() *AddressUpdateAftersaleStrategyRequest{
	request := &AddressUpdateAftersaleStrategyRequest{
		Param: &AddressUpdateAftersaleStrategyParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *AddressUpdateAftersaleStrategyRequest) Execute(accessToken *doudian_sdk.AccessToken) (*address_updateAftersaleStrategy_response.AddressUpdateAftersaleStrategyResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &address_updateAftersaleStrategy_response.AddressUpdateAftersaleStrategyResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *AddressUpdateAftersaleStrategyRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *AddressUpdateAftersaleStrategyRequest) GetParams() *AddressUpdateAftersaleStrategyParam{
	return c.Param
}


type AddressItem struct {
	// 非必传，消费者收货地址的市ID。如果operation=0，并且是动态退货策略，则必传；如果是operation=0，但是是静态策略，不是必传。
	CityId *int64 `json:"city_id"`
	// 非必传，省ID。如果operation=0，并且是动态退货策略，则必传；如果是operation=0，但是是静态策略，不是必传。通过/address/getAreasByProvince获取是ID。
	ProvinceId *int64 `json:"province_id"`
	// 非必传，地址ID。商家通过抖店后台-订单管理-物流工具-地址库管理或者通过三方服务商平台在抖店后台创建地址生成的ID，可以通过接入/address/list获取商家地址列表。如果是operation=0，则必传。通过/address/provinceList获取。
	Id *int64 `json:"id"`
}
type AftersaleStrategy struct {
	// 必传。策略类型。静态策略类型和用户收货地址无关，只是将商品和退货地址绑定，无论消费者收货地址在哪里，该策略绑定的商品全部退往同一地址。动态策略类型绑定的商品可以指定不同的消费者收货区域退往不同的退货地址，比如指定消费者收货地址在北京、天津、河北的订单退往华北仓，指定消费者收货地址在山东、江苏的订单退往华东仓。静态类型策略和动态类型策略都可以绑定多个商品。
	StrategyType *int32 `json:"strategy_type"`
	// 必传，策略id，需要更改的策略ID。
	StrategyId *string `json:"strategy_id"`
	// 售后策略绑定的地址列表;
	Address []AddressItem `json:"address"`
	// 策略名称，不能超过10个字。
	StrategyName *string `json:"strategy_name"`
	// 需绑定的商品列表
	ToAddProductId []int64 `json:"to_add_product_id"`
	// 需解绑的商品列表
	ToDelProductId []int64 `json:"to_del_product_id"`
}
type AddressUpdateAftersaleStrategyParam struct {
	// 售后策略;
	AftersaleStrategy *AftersaleStrategy `json:"aftersale_strategy"`
	// 操作类型，必传。0-全覆盖地址策略，会用传递过来的请求中的信息覆盖已有的信息；1-只是增加绑定商品；2-只是删除绑定商品。
	Operation *int32 `json:"operation"`
}
