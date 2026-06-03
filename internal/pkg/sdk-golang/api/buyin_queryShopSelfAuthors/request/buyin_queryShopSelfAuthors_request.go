package buyin_queryShopSelfAuthors_request

import (
	"doudian.com/open/sdk_golang/api/buyin_queryShopSelfAuthors/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type BuyinQueryShopSelfAuthorsRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *BuyinQueryShopSelfAuthorsParam 
}
func (c *BuyinQueryShopSelfAuthorsRequest) GetUrlPath() string{
	return "/buyin/queryShopSelfAuthors"
}


func New() *BuyinQueryShopSelfAuthorsRequest{
	request := &BuyinQueryShopSelfAuthorsRequest{
		Param: &BuyinQueryShopSelfAuthorsParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *BuyinQueryShopSelfAuthorsRequest) Execute(accessToken *doudian_sdk.AccessToken) (*buyin_queryShopSelfAuthors_response.BuyinQueryShopSelfAuthorsResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &buyin_queryShopSelfAuthors_response.BuyinQueryShopSelfAuthorsResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *BuyinQueryShopSelfAuthorsRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *BuyinQueryShopSelfAuthorsRequest) GetParams() *BuyinQueryShopSelfAuthorsParam{
	return c.Param
}


type BuyinQueryShopSelfAuthorsParam struct {
	// 分页 从1开始
	Page *int64 `json:"page"`
	// 分页大小 最大20
	PageSize *int64 `json:"page_size"`
}
