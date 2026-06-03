package topup_notify_response

import (
	"doudian.com/open/sdk_golang/core"
)

type TopupNotifyResponse struct {
	doudian_sdk.BaseDoudianOpSpiResponse
	Data *TopupNotifyData `json:"data"`
}
func (c *TopupNotifyResponse) GetData() interface{}{
	 return c.Data
}


type TopupNotifyData struct {
	// 充值平台交易单号  按照入参数返回
	TradeOrderNo string `json:"trade_order_no"`
	// 业务类型  按照入参数返回
	TopupBiz string `json:"topup_biz"`
	// 商家自有充值系统单号，与trade_order_no保持一一对应
	SellerOrderNo string `json:"seller_order_no"`
	// 订单状态，如果返回状态非以下状态则将进行接口重试，可选范围： SUCCESS 订单充值成功 FAILED 订单充值失败 IN_PROCESS 订单充值中
	SellerOrderStatus string `json:"seller_order_status"`
	// 错误码，当 seller_order_status 为FAILED时，需要填写充值失败的原因
	ErrCode string `json:"err_code"`
	// 错误描述
	ErrDesc string `json:"err_desc"`
	// 展示给用户查看的充值失败原因错误码，目前仅支持以下枚举值，若不传则默认使用1009810001：手机号码无效 10002：商家缺货 10003：命中运营商风控策略10098：其他商家处理时的异常
	TopupFailureReasonCode string `json:"topup_failure_reason_code"`
}
