package product_queryPropAutoOptimizeConfig_request

import (
	"doudian.com/open/sdk_golang/api/product_queryPropAutoOptimizeConfig/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductQueryPropAutoOptimizeConfigRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductQueryPropAutoOptimizeConfigParam 
}
func (c *ProductQueryPropAutoOptimizeConfigRequest) GetUrlPath() string{
	return "/product/queryPropAutoOptimizeConfig"
}


func New() *ProductQueryPropAutoOptimizeConfigRequest{
	request := &ProductQueryPropAutoOptimizeConfigRequest{
		Param: &ProductQueryPropAutoOptimizeConfigParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductQueryPropAutoOptimizeConfigRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_queryPropAutoOptimizeConfig_response.ProductQueryPropAutoOptimizeConfigResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_queryPropAutoOptimizeConfig_response.ProductQueryPropAutoOptimizeConfigResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductQueryPropAutoOptimizeConfigRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductQueryPropAutoOptimizeConfigRequest) GetParams() *ProductQueryPropAutoOptimizeConfigParam{
	return c.Param
}


type ProductQueryPropAutoOptimizeConfigParam struct {
}
