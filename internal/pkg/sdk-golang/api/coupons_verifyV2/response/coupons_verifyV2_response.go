package coupons_verifyV2_response

import (
	"doudian.com/open/sdk_golang/core"
)

type CouponsVerifyV2Response struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *CouponsVerifyV2Data `json:"data"`
}
type CouponsVerifyV2Data struct {
}
