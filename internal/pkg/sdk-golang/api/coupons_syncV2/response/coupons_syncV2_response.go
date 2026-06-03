package coupons_syncV2_response

import (
	"doudian.com/open/sdk_golang/core"
)

type CouponsSyncV2Response struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *CouponsSyncV2Data `json:"data"`
}
type CouponsSyncV2Data struct {
}
