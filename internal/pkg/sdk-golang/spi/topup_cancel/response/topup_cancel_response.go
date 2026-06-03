package topup_cancel_response

import (
	"doudian.com/open/sdk_golang/core"
)

type TopupCancelResponse struct {
	doudian_sdk.BaseDoudianOpSpiResponse
	Data *TopupCancelData `json:"data"`
}
func (c *TopupCancelResponse) GetData() interface{}{
	 return c.Data
}


type TopupCancelData struct {
	// 电商平台交易单号 按照入参数返回
	TradeOrderNo string `json:"trade_order_no"`
	// 业务类型 按照入参数返回
	TopupBiz string `json:"topup_biz"`
	// 商家自有充值系统单号，与trade_order_no保持一一对应
	SellerOrderNo string `json:"seller_order_no"`
	// 可选范围： CANCEL
	SellerOrderStatus string `json:"seller_order_status"`
	// 错误码
	ErrCode string `json:"err_code"`
	// 错误信息
	ErrDesc string `json:"err_desc"`
}
