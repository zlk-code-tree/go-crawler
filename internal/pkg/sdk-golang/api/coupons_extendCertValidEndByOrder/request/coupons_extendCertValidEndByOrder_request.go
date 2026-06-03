package coupons_extendCertValidEndByOrder_request

import (
	"doudian.com/open/sdk_golang/api/coupons_extendCertValidEndByOrder/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type CouponsExtendCertValidEndByOrderRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *CouponsExtendCertValidEndByOrderParam 
}
func (c *CouponsExtendCertValidEndByOrderRequest) GetUrlPath() string{
	return "/coupons/extendCertValidEndByOrder"
}


func New() *CouponsExtendCertValidEndByOrderRequest{
	request := &CouponsExtendCertValidEndByOrderRequest{
		Param: &CouponsExtendCertValidEndByOrderParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *CouponsExtendCertValidEndByOrderRequest) Execute(accessToken *doudian_sdk.AccessToken) (*coupons_extendCertValidEndByOrder_response.CouponsExtendCertValidEndByOrderResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &coupons_extendCertValidEndByOrder_response.CouponsExtendCertValidEndByOrderResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *CouponsExtendCertValidEndByOrderRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *CouponsExtendCertValidEndByOrderRequest) GetParams() *CouponsExtendCertValidEndByOrderParam{
	return c.Param
}


type CouponsExtendCertValidEndByOrderParam struct {
	// 商品单id
	OrderId string `json:"order_id"`
	// 请求唯一编号
	BatchNo string `json:"batch_no"`
	// 延期时间
	ExtendTime int64 `json:"extend_time"`
	// 延期原因。1-疫情原因，2-商家缺货，100-其他原因
	Reason int32 `json:"reason"`
	// 操作人信息
	Op *Op `json:"op"`
}
type Op struct {
	// 操作人平台
	Platform *int32 `json:"platform"`
	// 操作人id
	OpUid *int64 `json:"op_uid"`
}
