package cpaypromotion_ecom_applyMerchantSettleBill_response

import (
	"doudian.com/open/sdk_golang/core"
)

type CpaypromotionEcomApplyMerchantSettleBillResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *CpaypromotionEcomApplyMerchantSettleBillData `json:"data"`
}
type BizBaseResponse struct {
	// 响应码
	RetCode string `json:"ret_code"`
	// 响应体基本信息
	RetMsg string `json:"ret_msg"`
	// 本次请求是否成功
	Success bool `json:"success"`
	// 能否重试
	CanRetry bool `json:"can_retry"`
}
type CpaypromotionEcomApplyMerchantSettleBillData struct {
	// 下载id，后续商家需要通过这个下载id获取文件下载链接
	DownloadId string `json:"download_id"`
	// 响应体
	BizBaseResponse *BizBaseResponse `json:"biz_base_response"`
}
