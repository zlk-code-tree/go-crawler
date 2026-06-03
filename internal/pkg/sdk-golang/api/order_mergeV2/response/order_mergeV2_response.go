package order_mergeV2_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderMergeV2Response struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderMergeV2Data `json:"data"`
}
type OrderMergeV2Data struct {
	// 合单结果。可合单的订单ID列表用逗号分隔的字符串表示。比如，1111,2222表示订单1111和订单2222可合并发货。
	OrderList []string `json:"order_list"`
	// key是失败订单id，value是失败原因
	FailedOrder map[string]string `json:"failed_order"`
}
