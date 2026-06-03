package order_getMCToken_request

import (
	"doudian.com/open/sdk_golang/api/order_getMCToken/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type OrderGetMCTokenRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *OrderGetMCTokenParam 
}
func (c *OrderGetMCTokenRequest) GetUrlPath() string{
	return "/order/getMCToken"
}


func New() *OrderGetMCTokenRequest{
	request := &OrderGetMCTokenRequest{
		Param: &OrderGetMCTokenParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *OrderGetMCTokenRequest) Execute(accessToken *doudian_sdk.AccessToken) (*order_getMCToken_response.OrderGetMCTokenResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &order_getMCToken_response.OrderGetMCTokenResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *OrderGetMCTokenRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *OrderGetMCTokenRequest) GetParams() *OrderGetMCTokenParam{
	return c.Param
}


type OrderGetMCTokenParam struct {
}
