package product_getCategoryPropertyValue_request

import (
	"doudian.com/open/sdk_golang/api/product_getCategoryPropertyValue/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductGetCategoryPropertyValueRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductGetCategoryPropertyValueParam 
}
func (c *ProductGetCategoryPropertyValueRequest) GetUrlPath() string{
	return "/product/getCategoryPropertyValue"
}


func New() *ProductGetCategoryPropertyValueRequest{
	request := &ProductGetCategoryPropertyValueRequest{
		Param: &ProductGetCategoryPropertyValueParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductGetCategoryPropertyValueRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_getCategoryPropertyValue_response.ProductGetCategoryPropertyValueResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_getCategoryPropertyValue_response.ProductGetCategoryPropertyValueResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductGetCategoryPropertyValueRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductGetCategoryPropertyValueRequest) GetParams() *ProductGetCategoryPropertyValueParam{
	return c.Param
}


type ProductGetCategoryPropertyValueParam struct {
	// 类目id
	CategoryId int64 `json:"category_id"`
	// 属性项id
	PropertyId int64 `json:"property_id"`
}
