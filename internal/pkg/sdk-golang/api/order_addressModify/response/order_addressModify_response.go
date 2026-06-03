package order_addressModify_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderAddressModifyResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderAddressModifyData `json:"data"`
}
type OrderAddressModifyData struct {
	// true 表示当前改地址需要消费者确认，消费者确认之后才修改地址false 表示当前改地址不需要消费者确认，直接修改成功
	NeedUserReview bool `json:"need_user_review"`
}
