package coupons_extendCertValidEndByOrder_response

import (
	"doudian.com/open/sdk_golang/core"
)

type CouponsExtendCertValidEndByOrderResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *CouponsExtendCertValidEndByOrderData `json:"data"`
}
type CouponsExtendCertValidEndByOrderData struct {
	// 是否成功
	RetStatus string `json:"ret_status"`
	// 错误码
	RetCode string `json:"ret_code"`
	// 错误描述
	RetMessage string `json:"ret_message"`
}
