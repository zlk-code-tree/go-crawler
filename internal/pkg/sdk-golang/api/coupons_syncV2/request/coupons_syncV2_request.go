package coupons_syncV2_request

import (
	"doudian.com/open/sdk_golang/api/coupons_syncV2/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type CouponsSyncV2Request struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *CouponsSyncV2Param 
}
func (c *CouponsSyncV2Request) GetUrlPath() string{
	return "/coupons/syncV2"
}


func New() *CouponsSyncV2Request{
	request := &CouponsSyncV2Request{
		Param: &CouponsSyncV2Param{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *CouponsSyncV2Request) Execute(accessToken *doudian_sdk.AccessToken) (*coupons_syncV2_response.CouponsSyncV2Response, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &coupons_syncV2_response.CouponsSyncV2Response{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *CouponsSyncV2Request) GetParamObject() interface{}{
	return c.Param
}


func (c *CouponsSyncV2Request) GetParams() *CouponsSyncV2Param{
	return c.Param
}


type CertListItem struct {
	// 卡券跳转链接（腾讯游戏商家必传字段）
	CertLink *string `json:"cert_link"`
	// 外部卡券id
	OutCertId *string `json:"out_cert_id"`
	// 卡券券码
	CertNo string `json:"cert_no"`
	// 卡密（三方卡券订单无需回传）
	CertKey *string `json:"cert_key"`
	// 发券时间
	GrantTime string `json:"grant_time"`
	// 可延期次数
	CanExtendValidCount *int64 `json:"can_extend_valid_count"`
	// 状态：1未激活 2未使用
	Status *int64 `json:"status"`
	// 有效开始时间，大闸蟹场景必传
	ValidStart *string `json:"valid_start"`
	// 有效结束时间，大闸蟹场景必传
	ValidEnd *string `json:"valid_end"`
}
type CouponsSyncV2Param struct {
	// 子订单id
	OrderId string `json:"order_id"`
	// skuid
	SkuId *int64 `json:"sku_id"`
	// 卡券信息列表
	CertList []CertListItem `json:"cert_list"`
}
