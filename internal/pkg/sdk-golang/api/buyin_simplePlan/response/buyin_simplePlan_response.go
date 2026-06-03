package buyin_simplePlan_response

import (
	"doudian.com/open/sdk_golang/core"
)

type BuyinSimplePlanResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *BuyinSimplePlanData `json:"data"`
}
type BuyinSimplePlanData struct {
	// 创建结果
	Result []ResultItem `json:"result"`
	// 操作的商品数量
	Total int64 `json:"total"`
}
type ResultItem struct {
	// 商品ID
	ProductId int64 `json:"product_id"`
	// 操作结果是否成功，佣金率下调次日生效
	IsSuccess bool `json:"is_success"`
	// 当前商品操作错误码，成功为0
	ErrorCode int64 `json:"error_code"`
	// 当前商品操作错误原因，成功为success
	ErrorMsg string `json:"error_msg"`
}
