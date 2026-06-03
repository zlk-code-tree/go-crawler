package coupons_verifyV2_request

import (
	"doudian.com/open/sdk_golang/api/coupons_verifyV2/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type CouponsVerifyV2Request struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *CouponsVerifyV2Param 
}
func (c *CouponsVerifyV2Request) GetUrlPath() string{
	return "/coupons/verifyV2"
}


func New() *CouponsVerifyV2Request{
	request := &CouponsVerifyV2Request{
		Param: &CouponsVerifyV2Param{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *CouponsVerifyV2Request) Execute(accessToken *doudian_sdk.AccessToken) (*coupons_verifyV2_response.CouponsVerifyV2Response, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &coupons_verifyV2_response.CouponsVerifyV2Response{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *CouponsVerifyV2Request) GetParamObject() interface{}{
	return c.Param
}


func (c *CouponsVerifyV2Request) GetParams() *CouponsVerifyV2Param{
	return c.Param
}


type CouponsVerifyV2Param struct {
	// 卡券号码
	CertNo string `json:"cert_no"`
	// 核销时间
	VerifyTime *string `json:"verify_time"`
	// 幂等请求id
	BatchNo *string `json:"batch_no"`
	// 当次核销的次数
	VerifyCount *int64 `json:"verify_count"`
	// 卡券密码
	CertKey *string `json:"cert_key"`
	// 卡券类型：0-电商三方卡券，1-即时零售提货券/电商服饰自提/电商到店安装场景（请传入1），不传默认是0（电商三方卡券）核销方式
	CertType *int64 `json:"cert_type"`
	// 门店Id（提货券核销必传）
	StoreId *int64 `json:"store_id"`
	// 三方订单Id
	VerifyOrderId *string `json:"verify_order_id"`
	// 是否检查订单Id（verify_order_id）和卡券号码（cert_no）的关联关系
	CheckVerifyOrderId *bool `json:"check_verify_order_id"`
	// 车架码（电动车自提）
	VehicleIdentity *string `json:"vehicle_identity"`
}
