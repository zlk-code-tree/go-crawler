package product_openPropAutoOptimize_request

import (
	"doudian.com/open/sdk_golang/api/product_openPropAutoOptimize/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductOpenPropAutoOptimizeRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductOpenPropAutoOptimizeParam 
}
func (c *ProductOpenPropAutoOptimizeRequest) GetUrlPath() string{
	return "/product/openPropAutoOptimize"
}


func New() *ProductOpenPropAutoOptimizeRequest{
	request := &ProductOpenPropAutoOptimizeRequest{
		Param: &ProductOpenPropAutoOptimizeParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductOpenPropAutoOptimizeRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_openPropAutoOptimize_response.ProductOpenPropAutoOptimizeResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_openPropAutoOptimize_response.ProductOpenPropAutoOptimizeResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductOpenPropAutoOptimizeRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductOpenPropAutoOptimizeRequest) GetParams() *ProductOpenPropAutoOptimizeParam{
	return c.Param
}


type ProductOpenPropAutoOptimizeParam struct {
}
