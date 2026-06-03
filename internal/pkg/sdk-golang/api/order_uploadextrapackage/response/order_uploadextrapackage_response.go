package order_uploadextrapackage_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderUploadextrapackageResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderUploadextrapackageData `json:"data"`
}
type OrderUploadextrapackageData struct {
	// 此次发货行为对应的包裹ID
	DeliverId string `json:"deliver_id"`
}
