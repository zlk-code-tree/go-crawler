package address_listStrategyBindProducts_request

import (
	"doudian.com/open/sdk_golang/api/address_listStrategyBindProducts/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type AddressListStrategyBindProductsRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *AddressListStrategyBindProductsParam 
}
func (c *AddressListStrategyBindProductsRequest) GetUrlPath() string{
	return "/address/listStrategyBindProducts"
}


func New() *AddressListStrategyBindProductsRequest{
	request := &AddressListStrategyBindProductsRequest{
		Param: &AddressListStrategyBindProductsParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *AddressListStrategyBindProductsRequest) Execute(accessToken *doudian_sdk.AccessToken) (*address_listStrategyBindProducts_response.AddressListStrategyBindProductsResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &address_listStrategyBindProducts_response.AddressListStrategyBindProductsResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *AddressListStrategyBindProductsRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *AddressListStrategyBindProductsRequest) GetParams() *AddressListStrategyBindProductsParam{
	return c.Param
}


type AddressListStrategyBindProductsParam struct {
	// 策略id，必传，多地址策略对应的ID.
	StrategyId *string `json:"strategy_id"`
	// 页码,从1开始，
	Page *int64 `json:"page"`
	// 每页数目,最大100。地址策略目前最多支持拉取1w条记录，翻页数量超过1w会被系统拦截。
	PageSize *int64 `json:"page_size"`
}
