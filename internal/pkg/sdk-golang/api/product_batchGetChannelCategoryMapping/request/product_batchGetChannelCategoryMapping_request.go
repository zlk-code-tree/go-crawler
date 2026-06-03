package product_batchGetChannelCategoryMapping_request

import (
	"doudian.com/open/sdk_golang/api/product_batchGetChannelCategoryMapping/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductBatchGetChannelCategoryMappingRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductBatchGetChannelCategoryMappingParam 
}
func (c *ProductBatchGetChannelCategoryMappingRequest) GetUrlPath() string{
	return "/product/batchGetChannelCategoryMapping"
}


func New() *ProductBatchGetChannelCategoryMappingRequest{
	request := &ProductBatchGetChannelCategoryMappingRequest{
		Param: &ProductBatchGetChannelCategoryMappingParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductBatchGetChannelCategoryMappingRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_batchGetChannelCategoryMapping_response.ProductBatchGetChannelCategoryMappingResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_batchGetChannelCategoryMapping_response.ProductBatchGetChannelCategoryMappingResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductBatchGetChannelCategoryMappingRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductBatchGetChannelCategoryMappingRequest) GetParams() *ProductBatchGetChannelCategoryMappingParam{
	return c.Param
}


type ReqItemsItem struct {
	// 类目ID
	CategoryId *int64 `json:"category_id"`
}
type ProductBatchGetChannelCategoryMappingParam struct {
	// 一次查询最多50个
	ReqItems []ReqItemsItem `json:"req_items"`
}
