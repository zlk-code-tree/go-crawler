package promise_QueryBenefitTractionByProductId_request

import (
	"doudian.com/open/sdk_golang/api/promise_QueryBenefitTractionByProductId/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type PromiseQueryBenefitTractionByProductIdRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *PromiseQueryBenefitTractionByProductIdParam 
}
func (c *PromiseQueryBenefitTractionByProductIdRequest) GetUrlPath() string{
	return "/promise/QueryBenefitTractionByProductId"
}


func New() *PromiseQueryBenefitTractionByProductIdRequest{
	request := &PromiseQueryBenefitTractionByProductIdRequest{
		Param: &PromiseQueryBenefitTractionByProductIdParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *PromiseQueryBenefitTractionByProductIdRequest) Execute(accessToken *doudian_sdk.AccessToken) (*promise_QueryBenefitTractionByProductId_response.PromiseQueryBenefitTractionByProductIdResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &promise_QueryBenefitTractionByProductId_response.PromiseQueryBenefitTractionByProductIdResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *PromiseQueryBenefitTractionByProductIdRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *PromiseQueryBenefitTractionByProductIdRequest) GetParams() *PromiseQueryBenefitTractionByProductIdParam{
	return c.Param
}


type PromiseQueryBenefitTractionByProductIdParam struct {
	// 商品 ID
	ProductId int64 `json:"product_id"`
	// true=返回商品和 SKU 维度库存建议增量
	NeedStockIncrementSuggestion *bool `json:"need_stock_increment_suggestion"`
}
