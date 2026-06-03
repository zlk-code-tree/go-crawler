package product_removeChannelProduct_request

import (
	"doudian.com/open/sdk_golang/api/product_removeChannelProduct/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductRemoveChannelProductRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductRemoveChannelProductParam 
}
func (c *ProductRemoveChannelProductRequest) GetUrlPath() string{
	return "/product/removeChannelProduct"
}


func New() *ProductRemoveChannelProductRequest{
	request := &ProductRemoveChannelProductRequest{
		Param: &ProductRemoveChannelProductParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductRemoveChannelProductRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_removeChannelProduct_response.ProductRemoveChannelProductResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_removeChannelProduct_response.ProductRemoveChannelProductResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductRemoveChannelProductRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductRemoveChannelProductRequest) GetParams() *ProductRemoveChannelProductParam{
	return c.Param
}


type ProductRemoveChannelProductParam struct {
	// 主商品ID
	MainProductId int64 `json:"main_product_id"`
	// 渠道ID
	ChannelId int64 `json:"channel_id"`
	// 渠道类型3：商家自卖4：达人带货
	ChannelType int64 `json:"channel_type"`
}
