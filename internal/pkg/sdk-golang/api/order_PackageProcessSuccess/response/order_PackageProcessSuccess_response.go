package order_PackageProcessSuccess_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderPackageProcessSuccessResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderPackageProcessSuccessData `json:"data"`
}
type OrderPackageProcessSuccessData struct {
	// 返回信息
	RetMessage string `json:"ret_message"`
	// 返回状态,依据状态判定是否需要重试
	RetStatus string `json:"ret_status"`
	// 返回码
	RetCode string `json:"ret_code"`
}
