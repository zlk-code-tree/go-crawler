package order_merge_request

import (
	"doudian.com/open/sdk_golang/api/order_merge/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderMergeRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderMergeParam 
}
func (c *OrderMergeRequest) GetUrlPath() string{
	return "/order/merge"
}


func New() *OrderMergeRequest{
	request := &OrderMergeRequest{
		Param: &OrderMergeParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderMergeRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_merge_response.OrderMergeResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_merge_response.OrderMergeResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderMergeRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderMergeRequest) GetParams() *OrderMergeParam{
	return c.Param
}


type MergeListItem struct {
	// 店铺订单id，不可重复。不支持售后单
	OrderId string `json:"order_id"`
	// 收件人唯一id，订单列表、订单详情及售后详情API会返回，长度256以内
	OpenAddressId string `json:"open_address_id"`
}
type OrderMergeParam struct {
	// 合并需求列表，最少支持2个，最多支持100个。order_id和open_address_id必须一一对应，不可传错
	MergeList []MergeListItem `json:"merge_list"`
}
