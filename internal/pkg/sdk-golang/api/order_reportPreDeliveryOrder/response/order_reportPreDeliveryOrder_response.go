package order_reportPreDeliveryOrder_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderReportPreDeliveryOrderResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderReportPreDeliveryOrderData `json:"data"`
}
type OrderReportPreDeliveryOrderData struct {
	// 是否上报成功
	Success bool `json:"success"`
	// 原因说明
	Reason string `json:"reason"`
}
