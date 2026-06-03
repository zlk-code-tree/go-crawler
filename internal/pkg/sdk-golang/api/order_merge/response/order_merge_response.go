package order_merge_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderMergeResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderMergeData `json:"data"`
}
type OrderMergeData struct {
	// 合单结果。可合单的订单ID列表用逗号分隔的字符串表示。比如，1111,2222表示订单1111和订单2222可合并发货，3333表示无法合单。
	OrderList []string `json:"order_list"`
}
