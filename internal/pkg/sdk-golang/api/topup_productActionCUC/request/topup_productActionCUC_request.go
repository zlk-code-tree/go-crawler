package topup_productActionCUC_request

import (
	"doudian.com/open/sdk_golang/api/topup_productActionCUC/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type TopupProductActionCUCRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *TopupProductActionCUCParam 
}
func (c *TopupProductActionCUCRequest) GetUrlPath() string{
	return "/topup/productActionCUC"
}


func New() *TopupProductActionCUCRequest{
	request := &TopupProductActionCUCRequest{
		Param: &TopupProductActionCUCParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *TopupProductActionCUCRequest) Execute(accessToken *doudian_sdk.AccessToken) (*topup_productActionCUC_response.TopupProductActionCUCResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &topup_productActionCUC_response.TopupProductActionCUCResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *TopupProductActionCUCRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *TopupProductActionCUCRequest) GetParams() *TopupProductActionCUCParam{
	return c.Param
}


type TopupProductActionCUCParam struct {
	// 商品编码
	GoodCode *string `json:"goodCode"`
	// 1-上架，2-下架
	ActionType *string `json:"actionType"`
}
