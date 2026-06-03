package cpaypromotion_ecom_downloadMerchantSettleBill_response

import (
	"doudian.com/open/sdk_golang/core"
)

type CpaypromotionEcomDownloadMerchantSettleBillResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *CpaypromotionEcomDownloadMerchantSettleBillData `json:"data"`
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
type CpaypromotionEcomDownloadMerchantSettleBillData struct {
	// 文件下载url
	Url string `json:"url"`
	// 响应体
	BizBaseResponse *BizBaseResponse `json:"biz_base_response"`
}
