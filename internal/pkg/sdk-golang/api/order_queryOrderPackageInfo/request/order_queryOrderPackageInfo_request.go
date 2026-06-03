package order_queryOrderPackageInfo_request

import (
	"doudian.com/open/sdk_golang/api/order_queryOrderPackageInfo/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderQueryOrderPackageInfoRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderQueryOrderPackageInfoParam 
}
func (c *OrderQueryOrderPackageInfoRequest) GetUrlPath() string{
	return "/order/queryOrderPackageInfo"
}


func New() *OrderQueryOrderPackageInfoRequest{
	request := &OrderQueryOrderPackageInfoRequest{
		Param: &OrderQueryOrderPackageInfoParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderQueryOrderPackageInfoRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_queryOrderPackageInfo_response.OrderQueryOrderPackageInfoResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_queryOrderPackageInfo_response.OrderQueryOrderPackageInfoResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderQueryOrderPackageInfoRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderQueryOrderPackageInfoRequest) GetParams() *OrderQueryOrderPackageInfoParam{
	return c.Param
}


type OrderQueryOrderPackageInfoParam struct {
	// 订单id
	OrderId string `json:"order_id"`
}
