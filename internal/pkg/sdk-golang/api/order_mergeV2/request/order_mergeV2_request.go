package order_mergeV2_request

import (
	"doudian.com/open/sdk_golang/api/order_mergeV2/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderMergeV2Request struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderMergeV2Param 
}
func (c *OrderMergeV2Request) GetUrlPath() string{
	return "/order/mergeV2"
}


func New() *OrderMergeV2Request{
	request := &OrderMergeV2Request{
		Param: &OrderMergeV2Param{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderMergeV2Request) Execute(accessToken *doudian_sdk.AccessToken) (*order_mergeV2_response.OrderMergeV2Response, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_mergeV2_response.OrderMergeV2Response{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderMergeV2Request) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderMergeV2Request) GetParams() *OrderMergeV2Param{
	return c.Param
}


type OrderMergeV2Param struct {
	// 合单列表
	MergeList []MergeListItem `json:"merge_list"`
}
type MergeListItem struct {
	// 店铺单id
	OrderId *string `json:"order_id"`
	// open address id
	OpenAddressId *string `json:"open_address_id"`
}
