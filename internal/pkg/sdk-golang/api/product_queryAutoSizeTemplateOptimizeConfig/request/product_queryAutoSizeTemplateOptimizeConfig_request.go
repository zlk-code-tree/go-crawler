package product_queryAutoSizeTemplateOptimizeConfig_request

import (
	"doudian.com/open/sdk_golang/api/product_queryAutoSizeTemplateOptimizeConfig/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductQueryAutoSizeTemplateOptimizeConfigRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductQueryAutoSizeTemplateOptimizeConfigParam 
}
func (c *ProductQueryAutoSizeTemplateOptimizeConfigRequest) GetUrlPath() string{
	return "/product/queryAutoSizeTemplateOptimizeConfig"
}


func New() *ProductQueryAutoSizeTemplateOptimizeConfigRequest{
	request := &ProductQueryAutoSizeTemplateOptimizeConfigRequest{
		Param: &ProductQueryAutoSizeTemplateOptimizeConfigParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductQueryAutoSizeTemplateOptimizeConfigRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_queryAutoSizeTemplateOptimizeConfig_response.ProductQueryAutoSizeTemplateOptimizeConfigResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_queryAutoSizeTemplateOptimizeConfig_response.ProductQueryAutoSizeTemplateOptimizeConfigResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductQueryAutoSizeTemplateOptimizeConfigRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductQueryAutoSizeTemplateOptimizeConfigRequest) GetParams() *ProductQueryAutoSizeTemplateOptimizeConfigParam{
	return c.Param
}


type ProductQueryAutoSizeTemplateOptimizeConfigParam struct {
}
