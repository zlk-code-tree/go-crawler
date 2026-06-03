package address_delAftersaleStrategy_request

import (
	"doudian.com/open/sdk_golang/api/address_delAftersaleStrategy/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type AddressDelAftersaleStrategyRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *AddressDelAftersaleStrategyParam 
}
func (c *AddressDelAftersaleStrategyRequest) GetUrlPath() string{
	return "/address/delAftersaleStrategy"
}


func New() *AddressDelAftersaleStrategyRequest{
	request := &AddressDelAftersaleStrategyRequest{
		Param: &AddressDelAftersaleStrategyParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *AddressDelAftersaleStrategyRequest) Execute(accessToken *doudian_sdk.AccessToken) (*address_delAftersaleStrategy_response.AddressDelAftersaleStrategyResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &address_delAftersaleStrategy_response.AddressDelAftersaleStrategyResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *AddressDelAftersaleStrategyRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *AddressDelAftersaleStrategyRequest) GetParams() *AddressDelAftersaleStrategyParam{
	return c.Param
}


type AddressDelAftersaleStrategyParam struct {
	// 必填，售后地址策略。
	StrategyId *string `json:"strategy_id"`
}
