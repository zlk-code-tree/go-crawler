package product_openAutoSizeTemplateOptimize_request

import (
	"doudian.com/open/sdk_golang/api/product_openAutoSizeTemplateOptimize/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductOpenAutoSizeTemplateOptimizeRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductOpenAutoSizeTemplateOptimizeParam 
}
func (c *ProductOpenAutoSizeTemplateOptimizeRequest) GetUrlPath() string{
	return "/product/openAutoSizeTemplateOptimize"
}


func New() *ProductOpenAutoSizeTemplateOptimizeRequest{
	request := &ProductOpenAutoSizeTemplateOptimizeRequest{
		Param: &ProductOpenAutoSizeTemplateOptimizeParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductOpenAutoSizeTemplateOptimizeRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_openAutoSizeTemplateOptimize_response.ProductOpenAutoSizeTemplateOptimizeResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_openAutoSizeTemplateOptimize_response.ProductOpenAutoSizeTemplateOptimizeResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductOpenAutoSizeTemplateOptimizeRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductOpenAutoSizeTemplateOptimizeRequest) GetParams() *ProductOpenAutoSizeTemplateOptimizeParam{
	return c.Param
}


type ProductOpenAutoSizeTemplateOptimizeParam struct {
}
