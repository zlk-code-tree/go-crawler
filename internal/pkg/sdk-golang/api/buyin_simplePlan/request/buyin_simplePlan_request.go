package buyin_simplePlan_request

import (
	"doudian.com/open/sdk_golang/api/buyin_simplePlan/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type BuyinSimplePlanRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *BuyinSimplePlanParam 
}
func (c *BuyinSimplePlanRequest) GetUrlPath() string{
	return "/buyin/simplePlan"
}


func New() *BuyinSimplePlanRequest{
	request := &BuyinSimplePlanRequest{
		Param: &BuyinSimplePlanParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *BuyinSimplePlanRequest) Execute(accessToken *doudian_sdk.AccessToken) (*buyin_simplePlan_response.BuyinSimplePlanResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &buyin_simplePlan_response.BuyinSimplePlanResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *BuyinSimplePlanRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *BuyinSimplePlanRequest) GetParams() *BuyinSimplePlanParam{
	return c.Param
}


type ProductsItem struct {
	// 商品ID
	ProductId int64 `json:"product_id"`
	// 佣金比例（创建推广计划时必填）普通计划范围：[1, 50]。每个商品的佣金率下调次日0点生效，最小下调1%；上调最高到50%，立即生效
	CosRatio *int64 `json:"cos_ratio"`
}
type BuyinSimplePlanParam struct {
	// 是否创建
	IsCreate bool `json:"is_create"`
	// 状态，修改时必填，创建推广时默认开启。1：开启普通计划；2：关闭普通计划
	Status *int32 `json:"status"`
	// 总个数不超过20，不可重复
	Products []ProductsItem `json:"products"`
}
