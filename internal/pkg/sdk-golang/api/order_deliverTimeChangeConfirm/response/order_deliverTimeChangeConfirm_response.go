package order_deliverTimeChangeConfirm_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderDeliverTimeChangeConfirmResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderDeliverTimeChangeConfirmData `json:"data"`
}
type OrderDeliverTimeChangeConfirmData struct {
}
