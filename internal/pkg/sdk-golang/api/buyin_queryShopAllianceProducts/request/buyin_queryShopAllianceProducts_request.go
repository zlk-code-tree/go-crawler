package buyin_queryShopAllianceProducts_request

import (
	"doudian.com/open/sdk_golang/api/buyin_queryShopAllianceProducts/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type BuyinQueryShopAllianceProductsRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *BuyinQueryShopAllianceProductsParam 
}
func (c *BuyinQueryShopAllianceProductsRequest) GetUrlPath() string{
	return "/buyin/queryShopAllianceProducts"
}


func New() *BuyinQueryShopAllianceProductsRequest{
	request := &BuyinQueryShopAllianceProductsRequest{
		Param: &BuyinQueryShopAllianceProductsParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *BuyinQueryShopAllianceProductsRequest) Execute(accessToken *doudian_sdk.AccessToken) (*buyin_queryShopAllianceProducts_response.BuyinQueryShopAllianceProductsResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &buyin_queryShopAllianceProducts_response.BuyinQueryShopAllianceProductsResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *BuyinQueryShopAllianceProductsRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *BuyinQueryShopAllianceProductsRequest) GetParams() *BuyinQueryShopAllianceProductsParam{
	return c.Param
}


type BuyinQueryShopAllianceProductsParam struct {
	// 分页页数，从1开始
	Page int64 `json:"page"`
	// 分页大小，最大20
	Size int64 `json:"size"`
	// 商品ID列表，用“,”分隔
	ProductIds []int64 `json:"product_ids"`
	// 商品状态。2：商品在架；3：商家关闭推广；4：商品下架；6：平台治理关闭；99：商家删除
	Status *int64 `json:"status"`
	// 商品名称
	Title *string `json:"title"`
	// 是否有库存
	InStock *bool `json:"in_stock"`
	// 审核状态。 0：未审核；2：审核未通过；3：审核通过
	CheckStatus []int64 `json:"check_status"`
	// 排序字段。 update_time: 更新时间; cos_ratio: 佣金率
	SortBy *string `json:"sort_by"`
	// 排序方式（Desc或者Asc）。Desc：降序；Asc：生序
	SortOrder *string `json:"sort_order"`
	// 加载更多的字段，如 is_exclusive
	Fields []string `json:"fields"`
	// 0或者不填：原联盟计划商品 1：查询新投放商品
	Launch int64 `json:"launch"`
}
