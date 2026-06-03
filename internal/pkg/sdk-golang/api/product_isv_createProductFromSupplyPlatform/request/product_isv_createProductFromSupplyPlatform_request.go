package product_isv_createProductFromSupplyPlatform_request

import (
	"doudian.com/open/sdk_golang/api/product_isv_createProductFromSupplyPlatform/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductIsvCreateProductFromSupplyPlatformRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductIsvCreateProductFromSupplyPlatformParam 
}
func (c *ProductIsvCreateProductFromSupplyPlatformRequest) GetUrlPath() string{
	return "/product/isv/createProductFromSupplyPlatform"
}


func New() *ProductIsvCreateProductFromSupplyPlatformRequest{
	request := &ProductIsvCreateProductFromSupplyPlatformRequest{
		Param: &ProductIsvCreateProductFromSupplyPlatformParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductIsvCreateProductFromSupplyPlatformRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_isv_createProductFromSupplyPlatform_response.ProductIsvCreateProductFromSupplyPlatformResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_isv_createProductFromSupplyPlatform_response.ProductIsvCreateProductFromSupplyPlatformResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductIsvCreateProductFromSupplyPlatformRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductIsvCreateProductFromSupplyPlatformRequest) GetParams() *ProductIsvCreateProductFromSupplyPlatformParam{
	return c.Param
}


type ProductIsvCreateProductFromSupplyPlatformParam struct {
	// 搜索词id
	QueryId *string `json:"query_id"`
	// 商品id
	ProductId int64 `json:"product_id"`
	// 来源
	Origin int32 `json:"origin"`
	// 线索id
	ClueId *int64 `json:"clue_id"`
}
