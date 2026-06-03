package coupons_CertSyncFailedCancelOrder_response

import (
	"doudian.com/open/sdk_golang/core"
)

type CouponsCertSyncFailedCancelOrderResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *CouponsCertSyncFailedCancelOrderData `json:"data"`
}
type CouponsCertSyncFailedCancelOrderData struct {
	// 返回状态
	RetStatus string `json:"RetStatus"`
	// 返回码
	RetCode string `json:"RetCode"`
	// 返回信息
	RetMessage string `json:"RetMessage"`
}
