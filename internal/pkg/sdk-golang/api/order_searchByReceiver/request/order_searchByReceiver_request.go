package order_searchByReceiver_request

import (
	"doudian.com/open/sdk_golang/api/order_searchByReceiver/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderSearchByReceiverRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderSearchByReceiverParam 
}
func (c *OrderSearchByReceiverRequest) GetUrlPath() string{
	return "/order/searchByReceiver"
}


func New() *OrderSearchByReceiverRequest{
	request := &OrderSearchByReceiverRequest{
		Param: &OrderSearchByReceiverParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderSearchByReceiverRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_searchByReceiver_response.OrderSearchByReceiverResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_searchByReceiver_response.OrderSearchByReceiverResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderSearchByReceiverRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderSearchByReceiverRequest) GetParams() *OrderSearchByReceiverParam{
	return c.Param
}


type OrderSearchByReceiverParam struct {
	// 查询类型：0-收货人手机号，1-收货人姓名
	PostType int64 `json:"post_type"`
	// 收货人手机号或收货人姓名
	PostReceiver string `json:"post_receiver"`
	// 下单时间：开始，秒级时间戳。只允许查最近90天的订单，不传默认按90天前处理
	CreateTimeStart *int64 `json:"create_time_start"`
	// 下单时间：截止，秒级时间戳。不传默认按现在处理
	CreateTimeEnd *int64 `json:"create_time_end"`
	// 单页大小，限制100以内
	Size int64 `json:"size"`
	// 页码，0页开始
	Page int64 `json:"page"`
}
