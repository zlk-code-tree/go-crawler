package order_PackageProcessFail_request

import (
	"doudian.com/open/sdk_golang/api/order_PackageProcessFail/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderPackageProcessFailRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderPackageProcessFailParam 
}
func (c *OrderPackageProcessFailRequest) GetUrlPath() string{
	return "/order/PackageProcessFail"
}


func New() *OrderPackageProcessFailRequest{
	request := &OrderPackageProcessFailRequest{
		Param: &OrderPackageProcessFailParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderPackageProcessFailRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_PackageProcessFail_response.OrderPackageProcessFailResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_PackageProcessFail_response.OrderPackageProcessFailResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderPackageProcessFailRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderPackageProcessFailRequest) GetParams() *OrderPackageProcessFailParam{
	return c.Param
}


type Op struct {
	// 操作平台
	Platform *int32 `json:"platform"`
	// 操作人uid
	OpUid *int64 `json:"op_uid"`
}
type OrderPackageProcessFailParam struct {
	// 订单号
	OrderId string `json:"order_id"`
	// 风控相关信息
	Op *Op `json:"op"`
}
