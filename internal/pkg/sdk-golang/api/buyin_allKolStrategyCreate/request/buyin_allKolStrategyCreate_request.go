package buyin_allKolStrategyCreate_request

import (
	"doudian.com/open/sdk_golang/api/buyin_allKolStrategyCreate/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type BuyinAllKolStrategyCreateRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *BuyinAllKolStrategyCreateParam 
}
func (c *BuyinAllKolStrategyCreateRequest) GetUrlPath() string{
	return "/buyin/allKolStrategyCreate"
}


func New() *BuyinAllKolStrategyCreateRequest{
	request := &BuyinAllKolStrategyCreateRequest{
		Param: &BuyinAllKolStrategyCreateParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *BuyinAllKolStrategyCreateRequest) Execute(accessToken *doudian_sdk.AccessToken) (*buyin_allKolStrategyCreate_response.BuyinAllKolStrategyCreateResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &buyin_allKolStrategyCreate_response.BuyinAllKolStrategyCreateResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *BuyinAllKolStrategyCreateRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *BuyinAllKolStrategyCreateRequest) GetParams() *BuyinAllKolStrategyCreateParam{
	return c.Param
}


type BuyinAllKolStrategyCreateParam struct {
	// 商品id
	ProductId int64 `json:"product_id"`
	// 佣金率
	CosRatio int64 `json:"cos_ratio"`
}
