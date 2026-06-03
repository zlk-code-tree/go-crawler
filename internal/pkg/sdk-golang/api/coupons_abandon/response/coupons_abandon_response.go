package coupons_abandon_response

import (
	"doudian.com/open/sdk_golang/core"
)

type CouponsAbandonResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *CouponsAbandonData `json:"data"`
}
type CouponsAbandonData struct {
}
