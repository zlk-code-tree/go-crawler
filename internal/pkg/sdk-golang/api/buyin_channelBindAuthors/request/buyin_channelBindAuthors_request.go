package buyin_channelBindAuthors_request

import (
	"doudian.com/open/sdk_golang/api/buyin_channelBindAuthors/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type BuyinChannelBindAuthorsRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *BuyinChannelBindAuthorsParam 
}
func (c *BuyinChannelBindAuthorsRequest) GetUrlPath() string{
	return "/buyin/channelBindAuthors"
}


func New() *BuyinChannelBindAuthorsRequest{
	request := &BuyinChannelBindAuthorsRequest{
		Param: &BuyinChannelBindAuthorsParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *BuyinChannelBindAuthorsRequest) Execute(accessToken *doudian_sdk.AccessToken) (*buyin_channelBindAuthors_response.BuyinChannelBindAuthorsResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &buyin_channelBindAuthors_response.BuyinChannelBindAuthorsResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *BuyinChannelBindAuthorsRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *BuyinChannelBindAuthorsRequest) GetParams() *BuyinChannelBindAuthorsParam{
	return c.Param
}


type BuyinChannelBindAuthorsParam struct {
	// 渠道id
	ChannelId int64 `json:"channel_id"`
	// 渠道绑定类型 1: 店铺自卖，2：达人带货
	ChannelType int32 `json:"channel_type"`
	// 商品id
	ProductId int64 `json:"product_id"`
	// 分页 从1开始
	Page *int64 `json:"page"`
	// 分页大小 最大值50
	PageSize *int64 `json:"page_size"`
}
