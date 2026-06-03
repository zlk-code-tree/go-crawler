package coupons_certVerifyUpdate_request

import (
	"doudian.com/open/sdk_golang/api/coupons_certVerifyUpdate/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type CouponsCertVerifyUpdateRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *CouponsCertVerifyUpdateParam 
}
func (c *CouponsCertVerifyUpdateRequest) GetUrlPath() string{
	return "/coupons/certVerifyUpdate"
}


func New() *CouponsCertVerifyUpdateRequest{
	request := &CouponsCertVerifyUpdateRequest{
		Param: &CouponsCertVerifyUpdateParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *CouponsCertVerifyUpdateRequest) Execute(accessToken *doudian_sdk.AccessToken) (*coupons_certVerifyUpdate_response.CouponsCertVerifyUpdateResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &coupons_certVerifyUpdate_response.CouponsCertVerifyUpdateResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *CouponsCertVerifyUpdateRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *CouponsCertVerifyUpdateRequest) GetParams() *CouponsCertVerifyUpdateParam{
	return c.Param
}


type CouponsCertVerifyUpdateParam struct {
	// 卡号券码
	CertNo string `json:"cert_no"`
	// 更新时间
	UpdateTime string `json:"update_time"`
	// 更新核销次数
	UpdateVerifyCount int64 `json:"update_verify_count"`
	// 幂等请求编号
	BatchNo string `json:"batch_no"`
	// 额外信息
	Extra *string `json:"extra"`
}
