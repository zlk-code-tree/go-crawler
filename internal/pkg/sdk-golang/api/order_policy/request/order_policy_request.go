package order_policy_request

import (
	"doudian.com/open/sdk_golang/api/order_policy/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderPolicyRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderPolicyParam 
}
func (c *OrderPolicyRequest) GetUrlPath() string{
	return "/order/policy"
}


func New() *OrderPolicyRequest{
	request := &OrderPolicyRequest{
		Param: &OrderPolicyParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderPolicyRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_policy_response.OrderPolicyResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_policy_response.OrderPolicyResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderPolicyRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderPolicyRequest) GetParams() *OrderPolicyParam{
	return c.Param
}


type OrderPolicyParam struct {
	// 订单id
	OrderId string `json:"order_id"`
	// 保单类型枚举值：returnfreight2020v1-运费险；allergyinsurance2021-过敏险；plant7dinsurance2021-绿植养死包赔7天； plant15dinsurance2021-绿植养死包赔15天；insopenbox2022v12b-开箱无忧；foodsafety2022ecom2b-食安保；
	InsProductId string `json:"ins_product_id"`
}
