package order_orderCancel_request

import (
	"doudian.com/open/sdk_golang/api/order_orderCancel/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderOrderCancelRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderOrderCancelParam 
}
func (c *OrderOrderCancelRequest) GetUrlPath() string{
	return "/order/orderCancel"
}


func New() *OrderOrderCancelRequest{
	request := &OrderOrderCancelRequest{
		Param: &OrderOrderCancelParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderOrderCancelRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_orderCancel_response.OrderOrderCancelResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_orderCancel_response.OrderOrderCancelResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderOrderCancelRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderOrderCancelRequest) GetParams() *OrderOrderCancelParam{
	return c.Param
}


type OrderOrderCancelParam struct {
	// 店铺主订单号
	ShopOrderId string `json:"shop_order_id"`
	// 取消原因必须是如下枚举之一：①：用户重复下单②：用户不想买了③：用户买错了/信息填错了④重新下单、联系不上用户⑥：快递无法配送至用户地址⑦：其他；注意请使用枚举值信息，如“用户重复下单”必须要一致，开发者请勿自定义或带有空格等信息，否则无法通过接口校验；
	CancelReason string `json:"cancel_reason"`
}
