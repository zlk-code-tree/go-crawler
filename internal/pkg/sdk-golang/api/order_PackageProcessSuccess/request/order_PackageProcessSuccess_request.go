package order_PackageProcessSuccess_request

import (
	"doudian.com/open/sdk_golang/api/order_PackageProcessSuccess/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderPackageProcessSuccessRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderPackageProcessSuccessParam 
}
func (c *OrderPackageProcessSuccessRequest) GetUrlPath() string{
	return "/order/PackageProcessSuccess"
}


func New() *OrderPackageProcessSuccessRequest{
	request := &OrderPackageProcessSuccessRequest{
		Param: &OrderPackageProcessSuccessParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderPackageProcessSuccessRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_PackageProcessSuccess_response.OrderPackageProcessSuccessResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_PackageProcessSuccess_response.OrderPackageProcessSuccessResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderPackageProcessSuccessRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderPackageProcessSuccessRequest) GetParams() *OrderPackageProcessSuccessParam{
	return c.Param
}


type CertListItem struct {
	// 订单号与上面订单号保持一致
	CertNo string `json:"cert_no"`
	// 卡券状态
	Status int64 `json:"status"`
	// 发券时间,格式:时间戳,0则用当前时间
	GrantTime int64 `json:"grant_time"`
}
type Op struct {
	// 操作平台
	Platform *int32 `json:"platform"`
	// 操作人uid
	OpUid *int64 `json:"op_uid"`
}
type OrderPackageProcessSuccessParam struct {
	// 子订单id
	OrderId string `json:"order_id"`
	// skuid
	ComboId int64 `json:"combo_id"`
	// 卡券信息列表
	CertList []CertListItem `json:"cert_list"`
	// 风控相关信息
	Op *Op `json:"op"`
}
