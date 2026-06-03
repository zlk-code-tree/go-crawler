package product_getSchema_request

import (
	"doudian.com/open/sdk_golang/api/product_getSchema/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductGetSchemaRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductGetSchemaParam 
}
func (c *ProductGetSchemaRequest) GetUrlPath() string{
	return "/product/getSchema"
}


func New() *ProductGetSchemaRequest{
	request := &ProductGetSchemaRequest{
		Param: &ProductGetSchemaParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductGetSchemaRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_getSchema_response.ProductGetSchemaResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_getSchema_response.ProductGetSchemaResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductGetSchemaRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductGetSchemaRequest) GetParams() *ProductGetSchemaParam{
	return c.Param
}


type ProductGetSchemaParam struct {
	// 类目Id
	CategoryId *int64 `json:"category_id"`
	// 商品Id
	ProductId *int64 `json:"product_id"`
	// SPU id
	SpuId *int64 `json:"spu_id"`
	// schem数据、json格式
	Model *string `json:"model"`
}
