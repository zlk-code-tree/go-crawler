package topup_query_response

import (
	"doudian.com/open/sdk_golang/core"
)

type TopupQueryResponse struct {
	doudian_sdk.BaseDoudianOpSpiResponse
	Data *TopupQueryData `json:"data"`
}
func (c *TopupQueryResponse) GetData() interface{}{
	 return c.Data
}


type TopupQueryData struct {
	// 充值平台交易单号  按照入参数返回
	TradeOrderNo string `json:"trade_order_no"`
	// 业务类型 按照入参数返回
	TopupBiz string `json:"topup_biz"`
	// 商家自有充值系统单号，与trade_order_no保持一一对应
	SellerOrderNo string `json:"seller_order_no"`
	// 订单状态，可选范围： SUCCESS 订单充值成功 FAILED 订单充值失败 IN_PROCESS 订单充值中 如果返回状态非以上状态则将进行接口重试
	SellerOrderStatus string `json:"seller_order_status"`
	// 错误码，当seller_order_status为FAILED时，需要填写充值失败的原因
	ErrCode string `json:"err_code"`
	// 错误描述
	ErrDesc string `json:"err_desc"`
}
