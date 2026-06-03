package order_queryOrderLogistics_request

import (
	"doudian.com/open/sdk_golang/api/order_queryOrderLogistics/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderQueryOrderLogisticsRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderQueryOrderLogisticsParam 
}
func (c *OrderQueryOrderLogisticsRequest) GetUrlPath() string{
	return "/order/queryOrderLogistics"
}


func New() *OrderQueryOrderLogisticsRequest{
	request := &OrderQueryOrderLogisticsRequest{
		Param: &OrderQueryOrderLogisticsParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderQueryOrderLogisticsRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_queryOrderLogistics_response.OrderQueryOrderLogisticsResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_queryOrderLogistics_response.OrderQueryOrderLogisticsResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderQueryOrderLogisticsRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderQueryOrderLogisticsRequest) GetParams() *OrderQueryOrderLogisticsParam{
	return c.Param
}


type OrderQueryOrderLogisticsParam struct {
	// 店铺单号
	OrderId string `json:"order_id"`
}
