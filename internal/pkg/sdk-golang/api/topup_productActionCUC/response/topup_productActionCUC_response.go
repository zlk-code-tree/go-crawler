package topup_productActionCUC_response

import (
	"doudian.com/open/sdk_golang/core"
)

type TopupProductActionCUCResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *TopupProductActionCUCData `json:"data"`
}
type TopupProductActionCUCData struct {
	// 返回码：0-成功，其他-失败
	ReturnCode string `json:"returnCode"`
	// 返回信息
	Message string `json:"message"`
}
