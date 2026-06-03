package product_batchGetProductLocks_request

import (
	"doudian.com/open/sdk_golang/api/product_batchGetProductLocks/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductBatchGetProductLocksRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductBatchGetProductLocksParam 
}
func (c *ProductBatchGetProductLocksRequest) GetUrlPath() string{
	return "/product/batchGetProductLocks"
}


func New() *ProductBatchGetProductLocksRequest{
	request := &ProductBatchGetProductLocksRequest{
		Param: &ProductBatchGetProductLocksParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductBatchGetProductLocksRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_batchGetProductLocks_response.ProductBatchGetProductLocksResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_batchGetProductLocks_response.ProductBatchGetProductLocksResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductBatchGetProductLocksRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductBatchGetProductLocksRequest) GetParams() *ProductBatchGetProductLocksParam{
	return c.Param
}


type ProductBatchGetProductLocksParam struct {
	// 商品ID数组
	ProductIds []int64 `json:"product_ids"`
}
