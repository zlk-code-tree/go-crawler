package topup_result_response

import (
	"doudian.com/open/sdk_golang/core"
)

type TopupResultResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *TopupResultData `json:"data"`
}
type TopupResultData struct {
	// 平台订单号.
	TradeOrderNo string `json:"trade_order_no"`
	// 充值业务
	TopupBiz string `json:"topup_biz"`
	// 平台接口充值结果失败
	Result string `json:"result"`
	// 商家充值单号
	SellerOrderNo string `json:"seller_order_no"`
	// 错误码
	ErrCode string `json:"err_code"`
	// 错误信息
	ErrDesc string `json:"err_desc"`
}
