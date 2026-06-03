package coupons_cancelVerify_request

import (
	"doudian.com/open/sdk_golang/api/coupons_cancelVerify/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type CouponsCancelVerifyRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *CouponsCancelVerifyParam 
}
func (c *CouponsCancelVerifyRequest) GetUrlPath() string{
	return "/coupons/cancelVerify"
}


func New() *CouponsCancelVerifyRequest{
	request := &CouponsCancelVerifyRequest{
		Param: &CouponsCancelVerifyParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *CouponsCancelVerifyRequest) Execute(accessToken *doudian_sdk.AccessToken) (*coupons_cancelVerify_response.CouponsCancelVerifyResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &coupons_cancelVerify_response.CouponsCancelVerifyResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *CouponsCancelVerifyRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *CouponsCancelVerifyRequest) GetParams() *CouponsCancelVerifyParam{
	return c.Param
}


type CouponsCancelVerifyParam struct {
	// 卡号券码
	CertNo string `json:"cert_no"`
	// 幂等请求编号
	BatchNo *string `json:"batch_no"`
	// 取消核销总次数
	CancelVerifyCount *int64 `json:"cancel_verify_count"`
}
