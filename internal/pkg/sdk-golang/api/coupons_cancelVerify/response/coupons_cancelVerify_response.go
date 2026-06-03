package coupons_cancelVerify_response

import (
	"doudian.com/open/sdk_golang/core"
)

type CouponsCancelVerifyResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *CouponsCancelVerifyData `json:"data"`
}
type CouponsCancelVerifyData struct {
}
