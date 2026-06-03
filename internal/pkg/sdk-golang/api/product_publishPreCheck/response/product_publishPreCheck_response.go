package product_publishPreCheck_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductPublishPreCheckResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductPublishPreCheckData `json:"data"`
}
type CommonCheckResultsItem struct {
	// 校验类型
	CheckType string `json:"check_type"`
	// 校验结果code，0：通过，其他：拦截
	CheckResultCode int32 `json:"check_result_code"`
	// 文案提示
	CheckResultMsg string `json:"check_result_msg"`
}
type ProductPublishPreCheckData struct {
	// 校验
	CommonCheckResults []CommonCheckResultsItem `json:"common_check_results"`
}
