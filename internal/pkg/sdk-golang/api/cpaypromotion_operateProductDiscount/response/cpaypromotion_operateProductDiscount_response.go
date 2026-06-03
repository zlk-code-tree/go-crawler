package cpaypromotion_operateProductDiscount_response

import (
	"doudian.com/open/sdk_golang/core"
)

type CpaypromotionOperateProductDiscountResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *CpaypromotionOperateProductDiscountData `json:"data"`
}
type BizBaseResponse struct {
	// 错误码
	RetCode string `json:"ret_code"`
	// 错误信息
	RetMsg string `json:"ret_msg"`
	// 标识请求是否成功
	Success bool `json:"success"`
	// 标识失败情况下，是否应该重试
	CanRetry bool `json:"can_retry"`
}
type CpaypromotionOperateProductDiscountData struct {
	// 返回操作结果
	BizBaseResponse *BizBaseResponse `json:"biz_base_response"`
}
