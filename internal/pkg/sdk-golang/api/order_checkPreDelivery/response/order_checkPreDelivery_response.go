package order_checkPreDelivery_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderCheckPreDeliveryResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderCheckPreDeliveryData `json:"data"`
}
type OrderCheckPreDeliveryData struct {
	// 是否允许预发货
	CanPreDelivery bool `json:"can_pre_delivery"`
	// 原因说明
	Reason string `json:"reason"`
}
