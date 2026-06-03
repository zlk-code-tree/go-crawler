package topup_QueryOrderInfoForCreateOrder_response

import (
	"doudian.com/open/sdk_golang/core"
)

type TopupQueryOrderInfoForCreateOrderResponse struct {
	doudian_sdk.BaseDoudianOpSpiResponse
	Data *TopupQueryOrderInfoForCreateOrderData `json:"data"`
}
func (c *TopupQueryOrderInfoForCreateOrderResponse) GetData() interface{}{
	 return c.Data
}


type TopupQueryOrderInfoForCreateOrderData struct {
	// 外部商家订单号
	OuterOrderId string `json:"outer_order_id"`
	// 用户应该支付金额（单位分）
	PayAmount int64 `json:"pay_amount"`
	// 优惠金额（单位分）；优惠金额+支付金额=商品价格
	PromotionAmount int64 `json:"promotion_amount"`
}
