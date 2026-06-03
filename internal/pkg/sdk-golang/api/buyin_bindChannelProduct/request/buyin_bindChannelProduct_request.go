package buyin_bindChannelProduct_request

import (
	"doudian.com/open/sdk_golang/api/buyin_bindChannelProduct/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type BuyinBindChannelProductRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *BuyinBindChannelProductParam 
}
func (c *BuyinBindChannelProductRequest) GetUrlPath() string{
	return "/buyin/bindChannelProduct"
}


func New() *BuyinBindChannelProductRequest{
	request := &BuyinBindChannelProductRequest{
		Param: &BuyinBindChannelProductParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *BuyinBindChannelProductRequest) Execute(accessToken *doudian_sdk.AccessToken) (*buyin_bindChannelProduct_response.BuyinBindChannelProductResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &buyin_bindChannelProduct_response.BuyinBindChannelProductResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *BuyinBindChannelProductRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *BuyinBindChannelProductRequest) GetParams() *BuyinBindChannelProductParam{
	return c.Param
}


type BuyinBindChannelProductParam struct {
	// 商品ID
	ProductId int64 `json:"product_id"`
	// 渠道ID
	ChannelId int64 `json:"channel_id"`
	// 渠道类型：1: 店铺自卖，2：达人带货
	ChannelType int32 `json:"channel_type"`
	// 是否指定全部达人/全部店播达人
	IsAll *bool `json:"is_all"`
	// 需要绑定的达人uid 指定特定达人（is_all参数为false）时，该参数生效
	AuthorIds []int64 `json:"author_ids"`
	// 需要绑定的达人百应Id 指定特定达人（is_all参数为false）时，该参数生效
	BuyinIds []int64 `json:"buyin_ids"`
}
