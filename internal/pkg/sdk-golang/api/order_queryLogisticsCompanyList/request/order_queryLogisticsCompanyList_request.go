package order_queryLogisticsCompanyList_request

import (
	"doudian.com/open/sdk_golang/api/order_queryLogisticsCompanyList/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderQueryLogisticsCompanyListRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderQueryLogisticsCompanyListParam 
}
func (c *OrderQueryLogisticsCompanyListRequest) GetUrlPath() string{
	return "/order/queryLogisticsCompanyList"
}


func New() *OrderQueryLogisticsCompanyListRequest{
	request := &OrderQueryLogisticsCompanyListRequest{
		Param: &OrderQueryLogisticsCompanyListParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderQueryLogisticsCompanyListRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_queryLogisticsCompanyList_response.OrderQueryLogisticsCompanyListResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_queryLogisticsCompanyList_response.OrderQueryLogisticsCompanyListResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderQueryLogisticsCompanyListRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderQueryLogisticsCompanyListRequest) GetParams() *OrderQueryLogisticsCompanyListParam{
	return c.Param
}


type OrderQueryLogisticsCompanyListParam struct {
	// 订单号列表
	ShopOrderIds []string `json:"shop_order_ids"`
}
