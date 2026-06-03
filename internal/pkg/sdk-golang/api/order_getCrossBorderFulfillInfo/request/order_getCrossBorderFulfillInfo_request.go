package order_getCrossBorderFulfillInfo_request

import (
	"doudian.com/open/sdk_golang/api/order_getCrossBorderFulfillInfo/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderGetCrossBorderFulfillInfoRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderGetCrossBorderFulfillInfoParam 
}
func (c *OrderGetCrossBorderFulfillInfoRequest) GetUrlPath() string{
	return "/order/getCrossBorderFulfillInfo"
}


func New() *OrderGetCrossBorderFulfillInfoRequest{
	request := &OrderGetCrossBorderFulfillInfoRequest{
		Param: &OrderGetCrossBorderFulfillInfoParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderGetCrossBorderFulfillInfoRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_getCrossBorderFulfillInfo_response.OrderGetCrossBorderFulfillInfoResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_getCrossBorderFulfillInfo_response.OrderGetCrossBorderFulfillInfoResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderGetCrossBorderFulfillInfoRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderGetCrossBorderFulfillInfoRequest) GetParams() *OrderGetCrossBorderFulfillInfoParam{
	return c.Param
}


type OrderGetCrossBorderFulfillInfoParam struct {
	// 主订单号(不带A)
	ShopOrderId string `json:"shop_order_id"`
}
