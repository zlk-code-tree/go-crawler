package topup_resultCUC_response

import (
	"doudian.com/open/sdk_golang/core"
)

type TopupResultCUCResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *TopupResultCUCData `json:"data"`
}
type TopupResultCUCData struct {
	// 响应固定字符串："SUCCESS"
	Result string `json:"result"`
}
