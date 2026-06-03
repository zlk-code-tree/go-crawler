package topup_refundNoticeCUC_response

import (
	"doudian.com/open/sdk_golang/core"
)

type TopupRefundNoticeCUCResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *TopupRefundNoticeCUCData `json:"data"`
}
type TopupRefundNoticeCUCData struct {
	// 响应固定字符串："SUCCESS"
	Result string `json:"result"`
}
