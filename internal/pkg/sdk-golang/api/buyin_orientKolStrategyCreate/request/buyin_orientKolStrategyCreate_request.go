package buyin_orientKolStrategyCreate_request

import (
	"doudian.com/open/sdk_golang/api/buyin_orientKolStrategyCreate/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type BuyinOrientKolStrategyCreateRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *BuyinOrientKolStrategyCreateParam 
}
func (c *BuyinOrientKolStrategyCreateRequest) GetUrlPath() string{
	return "/buyin/orientKolStrategyCreate"
}


func New() *BuyinOrientKolStrategyCreateRequest{
	request := &BuyinOrientKolStrategyCreateRequest{
		Param: &BuyinOrientKolStrategyCreateParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *BuyinOrientKolStrategyCreateRequest) Execute(accessToken *doudian_sdk.AccessToken) (*buyin_orientKolStrategyCreate_response.BuyinOrientKolStrategyCreateResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &buyin_orientKolStrategyCreate_response.BuyinOrientKolStrategyCreateResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *BuyinOrientKolStrategyCreateRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *BuyinOrientKolStrategyCreateRequest) GetParams() *BuyinOrientKolStrategyCreateParam{
	return c.Param
}


type CosRatio struct {
	// 佣金率 如果是阶梯佣金 存第一阶梯的佣金率
	CosRatio int64 `json:"cos_ratio"`
	// 阶梯佣金率门槛 非阶梯为0
	LadderBar int64 `json:"ladder_bar"`
	// 阶梯佣金率 非阶梯为0
	LadderCosRatio int64 `json:"ladder_cos_ratio"`
}
type BuyinOrientKolStrategyCreateParam struct {
	// 商品id
	ProductId int64 `json:"product_id"`
	// 达人百应ID
	KolBuyinIds []int64 `json:"kol_buyin_ids"`
	// 佣金类型：1固定佣金率 2阶梯佣金率
	CosRatioType int32 `json:"cos_ratio_type"`
	// 佣金率
	CosRatio *CosRatio `json:"cos_ratio"`
	// 是否长期有效，和end_time互斥
	LongTerm *bool `json:"long_term"`
	// 结束时间 例：2020-05-19 00:00:00
	EndTime *string `json:"end_time"`
}
