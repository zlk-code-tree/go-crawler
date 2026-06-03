package address_updateAftersaleStrategy_response

import (
	"doudian.com/open/sdk_golang/core"
)

type AddressUpdateAftersaleStrategyResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *AddressUpdateAftersaleStrategyData `json:"data"`
}
type FailAddProductListItem struct {
	// 商品ID;
	ProductId string `json:"product_id"`
	// 失败原因
	FailReason string `json:"fail_reason"`
	// 失败code
	FailCode int64 `json:"fail_code"`
}
type FailDelProductListItem struct {
	// 商品ID;
	ProductId string `json:"product_id"`
	// 失败原因
	FailReason string `json:"fail_reason"`
	// 失败code
	FailCode int64 `json:"fail_code"`
}
type AddressUpdateAftersaleStrategyData struct {
	// 成功添加的商品ID列表;
	SuccessAddProductList []string `json:"success_add_product_list"`
	// 成功删除的商品ID列表;
	SuccessDelProductList []string `json:"success_del_product_list"`
	// 添加失败的商品ID列表;
	FailAddProductList []FailAddProductListItem `json:"fail_add_product_list"`
	// 添加失败的商品ID列表;
	FailDelProductList []FailDelProductListItem `json:"fail_del_product_list"`
}
