package order_batchBindSerialNumber_request

import (
	"doudian.com/open/sdk_golang/api/order_batchBindSerialNumber/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderBatchBindSerialNumberRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderBatchBindSerialNumberParam 
}
func (c *OrderBatchBindSerialNumberRequest) GetUrlPath() string{
	return "/order/batchBindSerialNumber"
}


func New() *OrderBatchBindSerialNumberRequest{
	request := &OrderBatchBindSerialNumberRequest{
		Param: &OrderBatchBindSerialNumberParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderBatchBindSerialNumberRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_batchBindSerialNumber_response.OrderBatchBindSerialNumberResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_batchBindSerialNumber_response.OrderBatchBindSerialNumberResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderBatchBindSerialNumberRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderBatchBindSerialNumberRequest) GetParams() *OrderBatchBindSerialNumberParam{
	return c.Param
}


type BindListItem struct {
	// 订单号;
	ShopOrderId *string `json:"shop_order_id"`
	// Sku单号;
	SkuOrderId *string `json:"sku_order_id"`
	// 序列号信息;
	SerialNoList []SerialNoListItem `json:"serial_no_list"`
}
type OrderBatchBindSerialNumberParam struct {
	// 绑定列表;
	BindList []BindListItem `json:"bind_list"`
	// 绑定场景 1 // 发货后 2 // 发货前
	BindScene *int32 `json:"bind_scene"`
}
type SerialNoListItem struct {
	// 序列号相关的信息;
	Value []string `json:"value"`
	// 序列号类型,SN - SN码，69 - 69码，IMEI - IMEI码;
	Key *string `json:"key"`
}
