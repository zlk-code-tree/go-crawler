package order_deliverTimeChangeTaskInfo_request

import (
	"doudian.com/open/sdk_golang/api/order_deliverTimeChangeTaskInfo/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderDeliverTimeChangeTaskInfoRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderDeliverTimeChangeTaskInfoParam 
}
func (c *OrderDeliverTimeChangeTaskInfoRequest) GetUrlPath() string{
	return "/order/deliverTimeChangeTaskInfo"
}


func New() *OrderDeliverTimeChangeTaskInfoRequest{
	request := &OrderDeliverTimeChangeTaskInfoRequest{
		Param: &OrderDeliverTimeChangeTaskInfoParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderDeliverTimeChangeTaskInfoRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_deliverTimeChangeTaskInfo_response.OrderDeliverTimeChangeTaskInfoResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_deliverTimeChangeTaskInfo_response.OrderDeliverTimeChangeTaskInfoResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderDeliverTimeChangeTaskInfoRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderDeliverTimeChangeTaskInfoRequest) GetParams() *OrderDeliverTimeChangeTaskInfoParam{
	return c.Param
}


type OrderDeliverTimeChangeTaskInfoParam struct {
	// 店铺单号
	ShopOrderId string `json:"shop_order_id"`
}
