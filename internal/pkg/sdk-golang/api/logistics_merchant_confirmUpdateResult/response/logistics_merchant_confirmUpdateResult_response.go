package logistics_merchant_confirmUpdateResult_response

import (
	"doudian.com/open/sdk_golang/core"
)

type LogisticsMerchantConfirmUpdateResultResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *LogisticsMerchantConfirmUpdateResultData `json:"data"`
}
type LogisticsMerchantConfirmUpdateResultData struct {
	// 是否处理成功
	IsSuccess bool `json:"is_success"`
	// 是否可重试
	CanRetry bool `json:"can_retry"`
	// 处理失败错误码
	FailedReasonCode string `json:"failed_reason_code"`
}
