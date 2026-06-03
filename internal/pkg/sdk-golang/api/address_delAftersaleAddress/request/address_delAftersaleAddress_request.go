package address_delAftersaleAddress_request

import (
	"doudian.com/open/sdk_golang/api/address_delAftersaleAddress/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type AddressDelAftersaleAddressRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *AddressDelAftersaleAddressParam 
}
func (c *AddressDelAftersaleAddressRequest) GetUrlPath() string{
	return "/address/delAftersaleAddress"
}


func New() *AddressDelAftersaleAddressRequest{
	request := &AddressDelAftersaleAddressRequest{
		Param: &AddressDelAftersaleAddressParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *AddressDelAftersaleAddressRequest) Execute(accessToken *doudian_sdk.AccessToken) (*address_delAftersaleAddress_response.AddressDelAftersaleAddressResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &address_delAftersaleAddress_response.AddressDelAftersaleAddressResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *AddressDelAftersaleAddressRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *AddressDelAftersaleAddressRequest) GetParams() *AddressDelAftersaleAddressParam{
	return c.Param
}


type AddressDelAftersaleAddressParam struct {
	// 地址ID，通过/address/list获取到的地址信息对应的address_id字段
	AddressId int64 `json:"address_id"`
}
