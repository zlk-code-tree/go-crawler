package instantShopping_trade_freightList_request

import (
	"doudian.com/open/sdk_golang/api/instantShopping_trade_freightList/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type InstantShoppingTradeFreightListRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *InstantShoppingTradeFreightListParam 
}
func (c *InstantShoppingTradeFreightListRequest) GetUrlPath() string{
	return "/instantShopping/trade/freightList"
}


func New() *InstantShoppingTradeFreightListRequest{
	request := &InstantShoppingTradeFreightListRequest{
		Param: &InstantShoppingTradeFreightListParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *InstantShoppingTradeFreightListRequest) Execute(accessToken *doudian_sdk.AccessToken) (*instantShopping_trade_freightList_response.InstantShoppingTradeFreightListResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &instantShopping_trade_freightList_response.InstantShoppingTradeFreightListResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *InstantShoppingTradeFreightListRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *InstantShoppingTradeFreightListRequest) GetParams() *InstantShoppingTradeFreightListParam{
	return c.Param
}


type InstantShoppingTradeFreightListParam struct {
	// 每页返回数量
	Size *string `json:"size"`
	// 模板名称搜索
	Name *string `json:"name"`
	// 页码
	Page *string `json:"page"`
}
