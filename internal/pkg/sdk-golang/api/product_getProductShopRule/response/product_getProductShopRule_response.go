package product_getProductShopRule_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductGetProductShopRuleResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductGetProductShopRuleData `json:"data"`
}
type AuditAutoRectifyRule struct {
	// true：已授权，false：未授权/拒绝授权
	IsGrant bool `json:"is_grant"`
	// 未授权原因，1：商家不支持该功能，2:未确定：用户不知道此功能，3:待定:用户知道此功能，但未作出是否授权的决定，4:拒绝授权
	UngrantedReason int32 `json:"ungranted_reason"`
}
type ProductGetProductShopRuleData struct {
	// 自动整改规则
	AuditAutoRectifyRule *AuditAutoRectifyRule `json:"audit_auto_rectify_rule"`
}
