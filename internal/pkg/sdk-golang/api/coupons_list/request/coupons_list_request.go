package coupons_list_request

import (
	"doudian.com/open/sdk_golang/api/coupons_list/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type CouponsListRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *CouponsListParam 
}
func (c *CouponsListRequest) GetUrlPath() string{
	return "/coupons/list"
}


func New() *CouponsListRequest{
	request := &CouponsListRequest{
		Param: &CouponsListParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *CouponsListRequest) Execute(accessToken *doudian_sdk.AccessToken) (*coupons_list_response.CouponsListResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &coupons_list_response.CouponsListResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *CouponsListRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *CouponsListRequest) GetParams() *CouponsListParam{
	return c.Param
}


type CouponsListParam struct {
	// 子订单id
	OrderId *string `json:"order_id"`
	// 卡号券码
	CertNo *string `json:"cert_no"`
	// 卡券类型（0-三方卡券；1-平台卡券）
	CertType int64 `json:"cert_type"`
}
