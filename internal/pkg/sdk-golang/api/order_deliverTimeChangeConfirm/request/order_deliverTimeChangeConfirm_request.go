package order_deliverTimeChangeConfirm_request

import (
	"doudian.com/open/sdk_golang/api/order_deliverTimeChangeConfirm/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderDeliverTimeChangeConfirmRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderDeliverTimeChangeConfirmParam 
}
func (c *OrderDeliverTimeChangeConfirmRequest) GetUrlPath() string{
	return "/order/deliverTimeChangeConfirm"
}


func New() *OrderDeliverTimeChangeConfirmRequest{
	request := &OrderDeliverTimeChangeConfirmRequest{
		Param: &OrderDeliverTimeChangeConfirmParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderDeliverTimeChangeConfirmRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_deliverTimeChangeConfirm_response.OrderDeliverTimeChangeConfirmResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_deliverTimeChangeConfirm_response.OrderDeliverTimeChangeConfirmResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderDeliverTimeChangeConfirmRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderDeliverTimeChangeConfirmRequest) GetParams() *OrderDeliverTimeChangeConfirmParam{
	return c.Param
}


type OrderDeliverTimeChangeConfirmParam struct {
	// 主订单ID
	ShopOrderId string `json:"shop_order_id"`
	// 任务单ID
	TaskId string `json:"task_id"`
	// 0:同意，400001审核拒绝
	RejectCode int64 `json:"reject_code"`
	// 审核拒绝原因，非必传
	RejectReason *string `json:"reject_reason"`
}
