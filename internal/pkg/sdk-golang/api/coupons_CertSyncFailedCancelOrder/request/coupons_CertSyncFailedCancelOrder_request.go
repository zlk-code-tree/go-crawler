package coupons_CertSyncFailedCancelOrder_request

import (
	"doudian.com/open/sdk_golang/api/coupons_CertSyncFailedCancelOrder/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type CouponsCertSyncFailedCancelOrderRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *CouponsCertSyncFailedCancelOrderParam 
}
func (c *CouponsCertSyncFailedCancelOrderRequest) GetUrlPath() string{
	return "/coupons/CertSyncFailedCancelOrder"
}


func New() *CouponsCertSyncFailedCancelOrderRequest{
	request := &CouponsCertSyncFailedCancelOrderRequest{
		Param: &CouponsCertSyncFailedCancelOrderParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *CouponsCertSyncFailedCancelOrderRequest) Execute(accessToken *doudian_sdk.AccessToken) (*coupons_CertSyncFailedCancelOrder_response.CouponsCertSyncFailedCancelOrderResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &coupons_CertSyncFailedCancelOrder_response.CouponsCertSyncFailedCancelOrderResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *CouponsCertSyncFailedCancelOrderRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *CouponsCertSyncFailedCancelOrderRequest) GetParams() *CouponsCertSyncFailedCancelOrderParam{
	return c.Param
}


type Op struct {
	// 操作平台枚举（选填，暂时用不上）
	Platform *int32 `json:"Platform"`
	// 操作人uid（选填，暂时用不上）
	OpUid *int64 `json:"OpUid"`
}
type CouponsCertSyncFailedCancelOrderParam struct {
	// 订单号
	OrderId string `json:"OrderId"`
	// 风控相关信息
	Op *Op `json:"Op"`
}
