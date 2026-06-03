package order_reportPreDeliveryOrder_request

import (
	"doudian.com/open/sdk_golang/api/order_reportPreDeliveryOrder/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderReportPreDeliveryOrderRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderReportPreDeliveryOrderParam 
}
func (c *OrderReportPreDeliveryOrderRequest) GetUrlPath() string{
	return "/order/reportPreDeliveryOrder"
}


func New() *OrderReportPreDeliveryOrderRequest{
	request := &OrderReportPreDeliveryOrderRequest{
		Param: &OrderReportPreDeliveryOrderParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderReportPreDeliveryOrderRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_reportPreDeliveryOrder_response.OrderReportPreDeliveryOrderResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_reportPreDeliveryOrder_response.OrderReportPreDeliveryOrderResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderReportPreDeliveryOrderRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderReportPreDeliveryOrderRequest) GetParams() *OrderReportPreDeliveryOrderParam{
	return c.Param
}


type OrderReportPreDeliveryOrderParam struct {
	// 订单ID
	ShopOrderId *string `json:"shop_order_id"`
	// 扩展信息
	Extra *map[string]string `json:"extra"`
	// 商家每日审单次数
	DailyAuditTimes *int32 `json:"daily_audit_times"`
	// 仓模式, 枚举值：self 自有仓 cloud 云仓 hybrid 混合仓
	WareMode *string `json:"ware_mode"`
}
