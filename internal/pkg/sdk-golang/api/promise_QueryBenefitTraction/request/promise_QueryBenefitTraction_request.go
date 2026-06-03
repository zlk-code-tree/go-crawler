package promise_QueryBenefitTraction_request

import (
	"doudian.com/open/sdk_golang/api/promise_QueryBenefitTraction/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type PromiseQueryBenefitTractionRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *PromiseQueryBenefitTractionParam 
}
func (c *PromiseQueryBenefitTractionRequest) GetUrlPath() string{
	return "/promise/QueryBenefitTraction"
}


func New() *PromiseQueryBenefitTractionRequest{
	request := &PromiseQueryBenefitTractionRequest{
		Param: &PromiseQueryBenefitTractionParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *PromiseQueryBenefitTractionRequest) Execute(accessToken *doudian_sdk.AccessToken) (*promise_QueryBenefitTraction_response.PromiseQueryBenefitTractionResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &promise_QueryBenefitTraction_response.PromiseQueryBenefitTractionResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *PromiseQueryBenefitTractionRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *PromiseQueryBenefitTractionRequest) GetParams() *PromiseQueryBenefitTractionParam{
	return c.Param
}


type PromiseQueryBenefitTractionParam struct {
	// true=店铺维度聚合利益点
	NeedAggregationBenefitByShop *bool `json:"need_aggregation_benefit_by_shop"`
	// true=返回近 14 天预售订单量
	NeedQuerySalesVolumeLast14Days *bool `json:"need_query_sales_volume_last14_days"`
	// true=返回商品和 SKU 维度库存建议增量
	NeedStockIncrementSuggestion *bool `json:"need_stock_increment_suggestion"`
	// true=按商品库存建议值倒排
	NeedSortByProductRecommendation *bool `json:"need_sort_by_product_recommendation"`
}
