package product_offlineChannelProduct_request

import (
	"doudian.com/open/sdk_golang/api/product_offlineChannelProduct/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductOfflineChannelProductRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductOfflineChannelProductParam 
}
func (c *ProductOfflineChannelProductRequest) GetUrlPath() string{
	return "/product/offlineChannelProduct"
}


func New() *ProductOfflineChannelProductRequest{
	request := &ProductOfflineChannelProductRequest{
		Param: &ProductOfflineChannelProductParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductOfflineChannelProductRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_offlineChannelProduct_response.ProductOfflineChannelProductResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_offlineChannelProduct_response.ProductOfflineChannelProductResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductOfflineChannelProductRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductOfflineChannelProductRequest) GetParams() *ProductOfflineChannelProductParam{
	return c.Param
}


type ProductOfflineChannelProductParam struct {
	// 主商品ID
	MainProductId int64 `json:"main_product_id"`
	// 渠道ID
	ChannelId int64 `json:"channel_id"`
	// 渠道类型3：商家自卖4：达人带货
	ChannelType int64 `json:"channel_type"`
}
