package order_searchByReceiver_response

import (
	"doudian.com/open/sdk_golang/core"
)

type OrderSearchByReceiverResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *OrderSearchByReceiverData `json:"data"`
}
type ShopOrderListItem struct {
	// 下单时间，时间戳，秒
	CreateTime string `json:"create_time"`
	// 订单状态：1-待确认/待支付（订单创建完毕），105-已支付，2 备货中，101-部分发货，3-已发货（全部发货），4-已取消，5-已完成（已收货）
	OrderStatus string `json:"order_status"`
	// 店铺订单号
	OrderId string `json:"order_id"`
}
type OrderSearchByReceiverData struct {
	// 页数，从0开始
	Page int64 `json:"page"`
	// 总订单数
	Total int64 `json:"total"`
	// 单页大小
	Size int64 `json:"size"`
	// 订单信息
	ShopOrderList []ShopOrderListItem `json:"shop_order_list"`
}
