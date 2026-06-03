package product_previewChannelProduct_request

import (
	"doudian.com/open/sdk_golang/api/product_previewChannelProduct/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductPreviewChannelProductRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductPreviewChannelProductParam 
}
func (c *ProductPreviewChannelProductRequest) GetUrlPath() string{
	return "/product/previewChannelProduct"
}


func New() *ProductPreviewChannelProductRequest{
	request := &ProductPreviewChannelProductRequest{
		Param: &ProductPreviewChannelProductParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductPreviewChannelProductRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_previewChannelProduct_response.ProductPreviewChannelProductResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_previewChannelProduct_response.ProductPreviewChannelProductResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductPreviewChannelProductRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductPreviewChannelProductRequest) GetParams() *ProductPreviewChannelProductParam{
	return c.Param
}


type ProductPreviewChannelProductParam struct {
	// 主商品ID
	MainProductId int64 `json:"main_product_id"`
	// 渠道ID
	ChannelId int64 `json:"channel_id"`
	// 渠道类型
	ChannelType int64 `json:"channel_type"`
}
