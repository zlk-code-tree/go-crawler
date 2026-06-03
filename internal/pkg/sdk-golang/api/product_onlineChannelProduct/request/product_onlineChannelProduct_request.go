package product_onlineChannelProduct_request

import (
	"doudian.com/open/sdk_golang/api/product_onlineChannelProduct/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type ProductOnlineChannelProductRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *ProductOnlineChannelProductParam 
}
func (c *ProductOnlineChannelProductRequest) GetUrlPath() string{
	return "/product/onlineChannelProduct"
}


func New() *ProductOnlineChannelProductRequest{
	request := &ProductOnlineChannelProductRequest{
		Param: &ProductOnlineChannelProductParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *ProductOnlineChannelProductRequest) Execute(accessToken *doudian_sdk.AccessToken) (*product_onlineChannelProduct_response.ProductOnlineChannelProductResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &product_onlineChannelProduct_response.ProductOnlineChannelProductResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *ProductOnlineChannelProductRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *ProductOnlineChannelProductRequest) GetParams() *ProductOnlineChannelProductParam{
	return c.Param
}


type ProductOnlineChannelProductParam struct {
	// 主商品ID
	MainProductId int64 `json:"main_product_id"`
	// 渠道ID
	ChannelId int64 `json:"channel_id"`
	// 渠道类型。不支持修改。3：商家自卖4：达人带货
	ChannelType int64 `json:"channel_type"`
}
