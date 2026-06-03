package address_addAftersaleStrategy_response

import (
	"doudian.com/open/sdk_golang/core"
)

type AddressAddAftersaleStrategyResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *AddressAddAftersaleStrategyData `json:"data"`
}
type FailProductListItem struct {
	// 商品ID;
	ProductId string `json:"product_id"`
	// 绑定失败的具体原因，开发者可以依据返回的结果对用户展示为什么绑定失败。
	FailReason string `json:"fail_reason"`
}
type AddressAddAftersaleStrategyData struct {
	// 绑定失败商品id列表
	FailProductList []FailProductListItem `json:"fail_product_list"`
	// 绑定成功的商品ID;
	SuccessProductList []string `json:"success_product_list"`
	// 策略ID;
	StrategyId string `json:"strategy_id"`
}
