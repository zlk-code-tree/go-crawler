package order_orderCancel_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderOrderCancelResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderOrderCancelData `json:"data"`
}
type OrderOrderCancelData struct {
}
