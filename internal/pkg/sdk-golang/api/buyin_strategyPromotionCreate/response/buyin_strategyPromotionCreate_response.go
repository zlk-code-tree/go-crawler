package buyin_strategyPromotionCreate_response

import (
	"doudian.com/open/sdk_golang/core"
)

type BuyinStrategyPromotionCreateResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *BuyinStrategyPromotionCreateData `json:"data"`
}
type ResultItem struct {
	// 是否操作成功
	Success bool `json:"success"`
	// 商品id
	ProductId int64 `json:"product_id"`
	// 商品名称
	ProductName string `json:"product_name"`
	// 绑定失败原因
	Reason string `json:"reason"`
}
type BuyinStrategyPromotionCreateData struct {
	// 绑定结果
	Result []ResultItem `json:"result"`
}
