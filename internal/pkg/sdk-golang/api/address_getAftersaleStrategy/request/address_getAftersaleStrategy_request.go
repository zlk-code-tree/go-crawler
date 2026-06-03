package address_getAftersaleStrategy_request

import (
	"doudian.com/open/sdk_golang/api/address_getAftersaleStrategy/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type AddressGetAftersaleStrategyRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *AddressGetAftersaleStrategyParam 
}
func (c *AddressGetAftersaleStrategyRequest) GetUrlPath() string{
	return "/address/getAftersaleStrategy"
}


func New() *AddressGetAftersaleStrategyRequest{
	request := &AddressGetAftersaleStrategyRequest{
		Param: &AddressGetAftersaleStrategyParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *AddressGetAftersaleStrategyRequest) Execute(accessToken *doudian_sdk.AccessToken) (*address_getAftersaleStrategy_response.AddressGetAftersaleStrategyResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &address_getAftersaleStrategy_response.AddressGetAftersaleStrategyResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *AddressGetAftersaleStrategyRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *AddressGetAftersaleStrategyRequest) GetParams() *AddressGetAftersaleStrategyParam{
	return c.Param
}


type AddressGetAftersaleStrategyParam struct {
	// 策略ID;
	StrategyId *string `json:"strategy_id"`
}
