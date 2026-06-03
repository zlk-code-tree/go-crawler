package order_PackageProcessFail_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderPackageProcessFailResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderPackageProcessFailData `json:"data"`
}
type OrderPackageProcessFailData struct {
	// 返回状态
	RetStatus string `json:"ret_status"`
	// 返回码
	RetCode string `json:"ret_code"`
	// 返回信息
	RetMessage string `json:"ret_message"`
}
