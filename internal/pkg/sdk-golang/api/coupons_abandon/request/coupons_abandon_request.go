package coupons_abandon_request

import (
	"doudian.com/open/sdk_golang/api/coupons_abandon/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type CouponsAbandonRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *CouponsAbandonParam 
}
func (c *CouponsAbandonRequest) GetUrlPath() string{
	return "/coupons/abandon"
}


func New() *CouponsAbandonRequest{
	request := &CouponsAbandonRequest{
		Param: &CouponsAbandonParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *CouponsAbandonRequest) Execute(accessToken *doudian_sdk.AccessToken) (*coupons_abandon_response.CouponsAbandonResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &coupons_abandon_response.CouponsAbandonResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *CouponsAbandonRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *CouponsAbandonRequest) GetParams() *CouponsAbandonParam{
	return c.Param
}


type CouponsAbandonParam struct {
	// 卡号券码
	CertNo string `json:"cert_no"`
}
