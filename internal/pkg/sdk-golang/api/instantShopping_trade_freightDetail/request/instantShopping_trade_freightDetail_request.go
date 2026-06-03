package instantShopping_trade_freightDetail_request

import (
	"doudian.com/open/sdk_golang/api/instantShopping_trade_freightDetail/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type InstantShoppingTradeFreightDetailRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *InstantShoppingTradeFreightDetailParam 
}
func (c *InstantShoppingTradeFreightDetailRequest) GetUrlPath() string{
	return "/instantShopping/trade/freightDetail"
}


func New() *InstantShoppingTradeFreightDetailRequest{
	request := &InstantShoppingTradeFreightDetailRequest{
		Param: &InstantShoppingTradeFreightDetailParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *InstantShoppingTradeFreightDetailRequest) Execute(accessToken *doudian_sdk.AccessToken) (*instantShopping_trade_freightDetail_response.InstantShoppingTradeFreightDetailResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &instantShopping_trade_freightDetail_response.InstantShoppingTradeFreightDetailResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *InstantShoppingTradeFreightDetailRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *InstantShoppingTradeFreightDetailRequest) GetParams() *InstantShoppingTradeFreightDetailParam{
	return c.Param
}


type InstantShoppingTradeFreightDetailParam struct {
	// 运费模板id
	FreightId *int64 `json:"freight_id"`
}
