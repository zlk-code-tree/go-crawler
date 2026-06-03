package order_batchBindSerialNumber_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderBatchBindSerialNumberResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderBatchBindSerialNumberData `json:"data"`
}
type SuccessBindListItem struct {
	// 店铺单单号：
	ShopOrderId string `json:"shop_order_id"`
	// sku单单号;
	SkuOrderId string `json:"sku_order_id"`
	// 失败原因;
	FailReason string `json:"fail_reason"`
	// 失败原因的code;
	FailCode string `json:"fail_code"`
}
type FailBindListItem struct {
	// 失败原因的code;
	FailCode string `json:"fail_code"`
	// 店铺单单号：
	ShopOrderId string `json:"shop_order_id"`
	// sku单单号;
	SkuOrderId string `json:"sku_order_id"`
	// 失败原因;
	FailReason string `json:"fail_reason"`
}
type OrderBatchBindSerialNumberData struct {
	// 绑定失败的数量;
	TotalFailBindCnt int64 `json:"total_fail_bind_cnt"`
	// 成功绑定的列表;
	SuccessBindList []SuccessBindListItem `json:"success_bind_list"`
	// 绑定失败的列表;
	FailBindList []FailBindListItem `json:"fail_bind_list"`
	// 请求绑定的数量;
	TotalRequestBindCnt int64 `json:"total_request_bind_cnt"`
	// 成功绑定的数量;
	TotalSuccessBindCnt int64 `json:"total_success_bind_cnt"`
}
