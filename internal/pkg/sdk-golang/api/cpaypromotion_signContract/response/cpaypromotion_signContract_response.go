package cpaypromotion_signContract_response

import (
	"doudian.com/open/sdk_golang/core"
)

type CpaypromotionSignContractResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *CpaypromotionSignContractData `json:"data"`
}
type BizBaseResponse struct {
	// 错误码
	RetCode string `json:"ret_code"`
	// 错误信息
	RetMsg string `json:"ret_msg"`
	// 标识失败情况下，是否应该重试
	CanRetry bool `json:"can_retry"`
	// 标识请求是否成功
	Success bool `json:"success"`
}
type CpaypromotionSignContractData struct {
	// 签约后，月付签约状态，0表示未签约，1表示已经签约
	CpaySignStatus int32 `json:"cpay_sign_status"`
	// 签约后，营销签约状态，0表示未签约，1表示已经签约
	PromotionSignStatus int32 `json:"promotion_sign_status"`
	// 业务请求基础信息
	BizBaseResponse *BizBaseResponse `json:"biz_base_response"`
}
