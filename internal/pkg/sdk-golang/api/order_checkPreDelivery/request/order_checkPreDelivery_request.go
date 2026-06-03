package order_checkPreDelivery_request

import (
	"doudian.com/open/sdk_golang/api/order_checkPreDelivery/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderCheckPreDeliveryRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderCheckPreDeliveryParam 
}
func (c *OrderCheckPreDeliveryRequest) GetUrlPath() string{
	return "/order/checkPreDelivery"
}


func New() *OrderCheckPreDeliveryRequest{
	request := &OrderCheckPreDeliveryRequest{
		Param: &OrderCheckPreDeliveryParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderCheckPreDeliveryRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_checkPreDelivery_response.OrderCheckPreDeliveryResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_checkPreDelivery_response.OrderCheckPreDeliveryResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderCheckPreDeliveryRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderCheckPreDeliveryRequest) GetParams() *OrderCheckPreDeliveryParam{
	return c.Param
}


type OrderCheckPreDeliveryParam struct {
	// 订单ID
	ShopOrderId *string `json:"shop_order_id"`
	// 扩展信息
	Extra *map[string]string `json:"extra"`
	// 仓模式, 枚举值：self 自有仓 cloud 云仓 hybrid 混合仓
	WareMode *string `json:"ware_mode"`
	// 商家每日审单次数
	DailyAuditTimes *int32 `json:"daily_audit_times"`
}
