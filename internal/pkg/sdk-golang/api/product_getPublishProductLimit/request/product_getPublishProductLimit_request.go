package product_getPublishProductLimit_request

import (
	"doudian.com/open/sdk_golang/api/product_getPublishProductLimit/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductGetPublishProductLimitRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductGetPublishProductLimitParam 
}
func (c *ProductGetPublishProductLimitRequest) GetUrlPath() string{
	return "/product/getPublishProductLimit"
}


func New() *ProductGetPublishProductLimitRequest{
	request := &ProductGetPublishProductLimitRequest{
		Param: &ProductGetPublishProductLimitParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductGetPublishProductLimitRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_getPublishProductLimit_response.ProductGetPublishProductLimitResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_getPublishProductLimit_response.ProductGetPublishProductLimitResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductGetPublishProductLimitRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductGetPublishProductLimitRequest) GetParams() *ProductGetPublishProductLimitParam{
	return c.Param
}


type ProductGetPublishProductLimitParam struct {
	// 一级类目id，可选
	FirstCategoryId *int64 `json:"first_category_id"`
}
