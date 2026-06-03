package buyin_strategyPromotionCreate_request

import (
	"doudian.com/open/sdk_golang/api/buyin_strategyPromotionCreate/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type BuyinStrategyPromotionCreateRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *BuyinStrategyPromotionCreateParam 
}
func (c *BuyinStrategyPromotionCreateRequest) GetUrlPath() string{
	return "/buyin/strategyPromotionCreate"
}


func New() *BuyinStrategyPromotionCreateRequest{
	request := &BuyinStrategyPromotionCreateRequest{
		Param: &BuyinStrategyPromotionCreateParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *BuyinStrategyPromotionCreateRequest) Execute(accessToken *doudian_sdk.AccessToken) (*buyin_strategyPromotionCreate_response.BuyinStrategyPromotionCreateResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &buyin_strategyPromotionCreate_response.BuyinStrategyPromotionCreateResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *BuyinStrategyPromotionCreateRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *BuyinStrategyPromotionCreateRequest) GetParams() *BuyinStrategyPromotionCreateParam{
	return c.Param
}


type BuyinStrategyPromotionCreateParam struct {
	// 商品id
	ProductIds []int64 `json:"product_ids"`
}
